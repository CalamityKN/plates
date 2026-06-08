package plates

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Discoverer interface {
	List() ([]string, error)
}

type Loader interface {
	Load(selector string) (Plate, error)
}

type Browser interface {
	Index() (*RackIndex, error)
}

type RackRepository struct {
	rootDir string
	rackDir string
}

func NewRackRepository(rootDir, rackDir string) *RackRepository {
	return &RackRepository{rootDir: rootDir, rackDir: rackDir}
}

func NewRackDiscoverer(rackDir string) *RackRepository {
	return NewRackRepository(filepath.Dir(filepath.Dir(rackDir)), rackDir)
}

func (r *RackRepository) List() ([]string, error) {
	index, err := r.Index()
	if err != nil {
		return nil, err
	}
	items := make([]string, 0, len(index.Plates))
	for _, plate := range index.Plates {
		items = append(items, plate.Key())
	}
	return items, nil
}

func (r *RackRepository) Index() (*RackIndex, error) {
	loaded, err := r.loadAll()
	if err != nil {
		return nil, err
	}
	return NewRackIndex(r.displayPath(r.rackDir), loaded), nil
}

func (r *RackRepository) Load(selector string) (Plate, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return Plate{}, errors.New("usage: use <plate>")
	}

	index, err := r.Index()
	if err != nil {
		return Plate{}, err
	}

	matches := index.ByName[selector]
	if strings.Contains(selector, "/") {
		matches = index.ByName[selector]
	}
	if len(matches) == 0 {
		return Plate{}, fmt.Errorf("plate %q not found", selector)
	}
	if len(matches) > 1 {
		keys := make([]string, 0, len(matches))
		for _, match := range matches {
			keys = append(keys, match.Key())
		}
		sort.Strings(keys)
		return Plate{}, fmt.Errorf("multiple plates named %q found; use one of: %s", selector, strings.Join(keys, ", "))
	}
	return matches[0], nil
}

func (r *RackRepository) loadAll() ([]Plate, error) {
	var found []Plate
	err := filepath.WalkDir(r.rackDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isPlateFile(path) {
			return nil
		}
		plate, err := r.loadFile(path)
		if err != nil {
			return err
		}
		found = append(found, plate)
		return nil
	})
	if os.IsNotExist(err) {
		return []Plate{}, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(found, func(i, j int) bool {
		return found[i].Key() < found[j].Key()
	})
	return found, nil
}

func (r *RackRepository) loadFile(path string) (Plate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plate{}, err
	}
	var plate Plate
	if err := yaml.Unmarshal(data, &plate); err != nil {
		return Plate{}, fmt.Errorf("%s: %w", path, err)
	}
	plate.Path = r.displayPath(path)
	if plate.Ingredients == nil {
		plate.Ingredients = map[string]Ingredient{}
	}
	if err := plate.Validate(); err != nil {
		return Plate{}, fmt.Errorf("%s: %w", plate.Path, err)
	}
	return plate, nil
}

func (r *RackRepository) displayPath(path string) string {
	rel, err := filepath.Rel(r.rootDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func isPlateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}
