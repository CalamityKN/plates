package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RenderRecord struct {
	ID              string            `json:"id" yaml:"id"`
	Timestamp       time.Time         `json:"timestamp" yaml:"timestamp"`
	Workspace       string            `json:"workspace" yaml:"workspace"`
	PlateName       string            `json:"plate_name" yaml:"plate_name"`
	Category        string            `json:"category" yaml:"category"`
	Variables       map[string]string `json:"variables" yaml:"variables"`
	Output          string            `json:"output" yaml:"output"`
	RawOutput       string            `json:"raw_output,omitempty" yaml:"raw_output,omitempty"`
	ContainsSecrets bool              `json:"contains_secrets" yaml:"contains_secrets"`
}

type Statistics struct {
	TotalRenders  int
	TopPlates     []Count
	TopCategories []Count
	MostRecent    *time.Time
}

type Count struct {
	Name  string
	Count int
}

type Store struct {
	outputDir string
	history   string
	rendered  string
	exports   string
	now       func() time.Time
}

func NewStore(dataDir string) *Store {
	outputDir := filepath.Join(dataDir, "output")
	return &Store{
		outputDir: outputDir,
		history:   filepath.Join(outputDir, "history.json"),
		rendered:  filepath.Join(outputDir, "rendered"),
		exports:   filepath.Join(outputDir, "exports"),
		now:       time.Now,
	}
}

func (s *Store) Add(record RenderRecord) (RenderRecord, error) {
	records, err := s.Load()
	if err != nil {
		return RenderRecord{}, err
	}
	record.ID = nextID(records)
	if record.Timestamp.IsZero() {
		record.Timestamp = s.now()
	}
	if record.Variables == nil {
		record.Variables = map[string]string{}
	}
	records = append(records, record)
	if err := s.save(records); err != nil {
		return RenderRecord{}, err
	}
	return record, nil
}

func Redact(text string, secrets []string) string {
	redacted := text
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "********")
	}
	return redacted
}

func (s *Store) Ensure() error {
	return s.ensure()
}

func (s *Store) Load() ([]RenderRecord, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.history)
	if errors.Is(err, os.ErrNotExist) {
		return []RenderRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []RenderRecord{}, nil
	}
	var records []RenderRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Store) Recent() ([]RenderRecord, error) {
	records, err := s.Load()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})
	return records, nil
}

func (s *Store) Latest() (RenderRecord, error) {
	records, err := s.Recent()
	if err != nil {
		return RenderRecord{}, err
	}
	if len(records) == 0 {
		return RenderRecord{}, errors.New("no rendered output available")
	}
	return records[0], nil
}

func (s *Store) Get(id string) (RenderRecord, error) {
	records, err := s.Load()
	if err != nil {
		return RenderRecord{}, err
	}
	for _, record := range records {
		if record.ID == id {
			return record, nil
		}
	}
	return RenderRecord{}, fmt.Errorf("render record not found: %s", id)
}

func (s *Store) Delete(id string) error {
	records, err := s.Load()
	if err != nil {
		return err
	}
	filtered := make([]RenderRecord, 0, len(records))
	found := false
	for _, record := range records {
		if record.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, record)
	}
	if !found {
		return fmt.Errorf("render record not found: %s", id)
	}
	return s.save(filtered)
}

func (s *Store) Clear() error {
	return s.save([]RenderRecord{})
}

func (s *Store) SaveOutput(record RenderRecord, filename string) (string, error) {
	if err := os.MkdirAll(s.rendered, 0o755); err != nil {
		return "", err
	}
	if filename == "" {
		filename = record.Timestamp.Format("2006-01-02_150405") + "_" + safeName(record.PlateName) + ".txt"
	} else {
		filename = filepath.Base(filename)
	}
	path := filepath.Join(s.rendered, filename)
	return filepath.ToSlash(path), os.WriteFile(path, []byte(record.Output), 0o644)
}

func (s *Store) Export(record RenderRecord, format string) (string, error) {
	if err := os.MkdirAll(s.exports, 0o755); err != nil {
		return "", err
	}
	name := record.Timestamp.Format("2006-01-02_150405") + "_" + safeName(record.PlateName)
	var data []byte
	var filename string
	var err error
	switch format {
	case "json":
		filename = name + ".json"
		data, err = json.MarshalIndent(record, "", "  ")
	case "yaml":
		filename = name + ".yaml"
		data, err = yaml.Marshal(record)
	case "markdown":
		filename = name + ".md"
		data = []byte(markdown(record))
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.exports, filename)
	return filepath.ToSlash(path), os.WriteFile(path, data, 0o644)
}

func (s *Store) Stats() (Statistics, error) {
	records, err := s.Recent()
	if err != nil {
		return Statistics{}, err
	}
	stats := Statistics{TotalRenders: len(records)}
	if len(records) > 0 {
		t := records[0].Timestamp
		stats.MostRecent = &t
	}
	plateCounts := map[string]int{}
	categoryCounts := map[string]int{}
	for _, record := range records {
		plateCounts[record.Category+"/"+record.PlateName]++
		categoryCounts[record.Category]++
	}
	stats.TopPlates = topCounts(plateCounts)
	stats.TopCategories = topCounts(categoryCounts)
	return stats, nil
}

func (s *Store) ensure() error {
	for _, dir := range []string{s.outputDir, s.rendered, s.exports} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) save(records []RenderRecord) error {
	if err := s.ensure(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.history, data, 0o644)
}

func nextID(records []RenderRecord) string {
	maxID := 0
	for _, record := range records {
		id, err := strconv.Atoi(record.ID)
		if err == nil && id > maxID {
			maxID = id
		}
	}
	return strconv.Itoa(maxID + 1)
}

func markdown(record RenderRecord) string {
	return fmt.Sprintf(`# Rendered Plate

Plate: %s/%s

Timestamp: %s

## Output

`+"```bash"+`
%s
`+"```"+`
`, record.Category, record.PlateName, record.Timestamp.Format("2006-01-02 15:04"), strings.TrimRight(record.Output, "\n"))
}

func safeName(name string) string {
	var b strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			b.WriteRune(ch)
		}
	}
	if b.Len() == 0 {
		return "rendered"
	}
	return b.String()
}

func topCounts(counts map[string]int) []Count {
	items := make([]Count, 0, len(counts))
	for name, count := range counts {
		if name == "" {
			continue
		}
		items = append(items, Count{Name: name, Count: count})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Name < items[j].Name
	})
	return items
}
