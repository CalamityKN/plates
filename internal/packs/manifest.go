package packs

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"plates/internal/plates"

	"gopkg.in/yaml.v3"
)

type PackManifest struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Version     string    `yaml:"version"`
	Created     time.Time `yaml:"created"`
	Author      string    `yaml:"author,omitempty"`
	Plates      []string  `yaml:"plates"`
}

type ExportOptions struct {
	Name        string
	Description string
	Version     string
	Author      string
	Category    string
	Tag         string
	Plate       string
}

type InspectResult struct {
	Manifest PackManifest
	Plates   []plates.Plate
	Results  []plates.LintResult
}

type ImportResult struct {
	Imported    int
	Skipped     int
	Overwritten int
	Conflicts   []string
}

func Export(rackDir, exportedDir string, index *plates.RackIndex, opts ExportOptions) (string, PackManifest, error) {
	if opts.Name == "" {
		return "", PackManifest{}, errors.New("pack name is required")
	}
	selected := selectPlates(index.Plates, opts)
	if len(selected) == 0 {
		return "", PackManifest{}, errors.New("no plates matched export selection")
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}
	if opts.Description == "" {
		opts.Description = "PLATES plate pack"
	}
	manifest := PackManifest{
		Name:        opts.Name,
		Description: opts.Description,
		Version:     opts.Version,
		Created:     time.Now().UTC(),
		Author:      opts.Author,
		Plates:      plateKeys(selected),
	}
	if err := os.MkdirAll(exportedDir, 0o755); err != nil {
		return "", PackManifest{}, err
	}
	path := filepath.Join(exportedDir, safePackName(opts.Name)+".zip")
	file, err := os.Create(path)
	if err != nil {
		return "", PackManifest{}, err
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	defer zw.Close()
	if err := writeYAML(zw, "pack.yaml", manifest); err != nil {
		return "", PackManifest{}, err
	}
	for _, plate := range selected {
		name := "rack/" + plate.Key() + ".yml"
		if err := writeYAML(zw, name, plate); err != nil {
			return "", PackManifest{}, err
		}
	}
	return filepath.ToSlash(path), manifest, nil
}

func Inspect(path string) (InspectResult, error) {
	manifest, packPlates, err := readPack(path)
	if err != nil {
		return InspectResult{}, err
	}
	index := plates.NewRackIndex("rack", packPlates)
	results := plates.LintRack(index)
	return InspectResult{Manifest: manifest, Plates: packPlates, Results: results}, nil
}

func Validate(path string) ([]plates.LintResult, error) {
	inspected, err := Inspect(path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(inspected.Manifest.Name) == "" {
		return inspected.Results, errors.New("manifest missing name")
	}
	if strings.TrimSpace(inspected.Manifest.Version) == "" {
		return inspected.Results, errors.New("manifest missing version")
	}
	if len(inspected.Manifest.Plates) == 0 {
		return inspected.Results, errors.New("manifest contains no plates")
	}
	for _, result := range inspected.Results {
		if result.HasErrors() {
			return inspected.Results, errors.New("pack contains failing plates")
		}
	}
	return inspected.Results, nil
}

func Import(path, rackDir string, force bool) (ImportResult, error) {
	manifest, packPlates, err := readPack(path)
	if err != nil {
		return ImportResult{}, err
	}
	if len(manifest.Plates) == 0 {
		return ImportResult{}, errors.New("manifest contains no plates")
	}
	index := plates.NewRackIndex("rack", packPlates)
	for _, result := range plates.LintRack(index) {
		if result.HasErrors() {
			return ImportResult{}, fmt.Errorf("invalid plate: %s", result.Plate.Key())
		}
	}
	result := ImportResult{}
	var conflicts []string
	for _, plate := range packPlates {
		dest := filepath.Join(rackDir, filepath.FromSlash(plate.Category), plate.Name+".yml")
		if _, err := os.Stat(dest); err == nil && !force {
			conflicts = append(conflicts, plate.Key())
		} else if err != nil && !os.IsNotExist(err) {
			return ImportResult{}, err
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		result.Conflicts = conflicts
		return result, ErrConflicts
	}
	for _, plate := range packPlates {
		dest := filepath.Join(rackDir, filepath.FromSlash(plate.Category), plate.Name+".yml")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return ImportResult{}, err
		}
		if _, err := os.Stat(dest); err == nil && force {
			result.Overwritten++
		} else {
			result.Imported++
		}
		data, err := yaml.Marshal(plate)
		if err != nil {
			return ImportResult{}, err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return ImportResult{}, err
		}
	}
	return result, nil
}

var ErrConflicts = errors.New("pack import conflicts")

func readPack(path string) (PackManifest, []plates.Plate, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return PackManifest{}, nil, err
	}
	defer zr.Close()
	var manifest PackManifest
	foundManifest := false
	var packPlates []plates.Plate
	for _, file := range zr.File {
		if err := validateEntryName(file.Name); err != nil {
			return PackManifest{}, nil, err
		}
		if file.Name == "pack.yaml" {
			data, err := readZipFile(file)
			if err != nil {
				return PackManifest{}, nil, err
			}
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return PackManifest{}, nil, err
			}
			foundManifest = true
			continue
		}
		if strings.HasPrefix(file.Name, "rack/") && isPlateFile(file.Name) {
			data, err := readZipFile(file)
			if err != nil {
				return PackManifest{}, nil, err
			}
			var plate plates.Plate
			if err := yaml.Unmarshal(data, &plate); err != nil {
				return PackManifest{}, nil, err
			}
			if plate.Ingredients == nil {
				plate.Ingredients = map[string]plates.Ingredient{}
			}
			plate.Path = file.Name
			packPlates = append(packPlates, plate)
		}
	}
	if !foundManifest {
		return PackManifest{}, nil, errors.New("missing pack.yaml")
	}
	sort.SliceStable(packPlates, func(i, j int) bool {
		return packPlates[i].Key() < packPlates[j].Key()
	})
	return manifest, packPlates, nil
}

func validateEntryName(name string) error {
	clean := filepath.ToSlash(name)
	if clean != name || strings.HasPrefix(clean, "/") || filepath.IsAbs(name) || strings.Contains(clean, "../") || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("unsafe pack entry: %s", name)
	}
	if clean == "pack.yaml" {
		return nil
	}
	if strings.HasPrefix(clean, "rack/") && isPlateFile(clean) {
		return nil
	}
	return fmt.Errorf("unsupported pack entry: %s", name)
}

func selectPlates(all []plates.Plate, opts ExportOptions) []plates.Plate {
	var selected []plates.Plate
	for _, plate := range all {
		switch {
		case opts.Plate != "":
			if plate.Key() == opts.Plate {
				selected = append(selected, plate)
			}
		case opts.Category != "":
			if plate.Category == opts.Category {
				selected = append(selected, plate)
			}
		case opts.Tag != "":
			for _, tag := range plate.Tags {
				if tag == opts.Tag {
					selected = append(selected, plate)
					break
				}
			}
		default:
			selected = append(selected, plate)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].Key() < selected[j].Key() })
	return selected
}

func plateKeys(packPlates []plates.Plate) []string {
	keys := make([]string, 0, len(packPlates))
	for _, plate := range packPlates {
		keys = append(keys, plate.Key())
	}
	sort.Strings(keys)
	return keys
}

func writeYAML(zw *zip.Writer, name string, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(data))
	return err
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func isPlateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

func safePackName(name string) string {
	var b strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			b.WriteRune(ch)
		}
	}
	if b.Len() == 0 {
		return "pack"
	}
	return b.String()
}
