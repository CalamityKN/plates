package secrets

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Ensure() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return s.save(map[string]string{})
	} else {
		return err
	}
}

func (s *Store) Set(key, value string) error {
	values, err := s.All()
	if err != nil {
		return err
	}
	values[key] = value
	return s.save(values)
}

func (s *Store) Get(key string) (string, bool, error) {
	values, err := s.All()
	if err != nil {
		return "", false, err
	}
	value, ok := values[key]
	return value, ok, nil
}

func (s *Store) Delete(key string) error {
	values, err := s.All()
	if err != nil {
		return err
	}
	delete(values, key)
	return s.save(values)
}

func (s *Store) Clear() error {
	return s.save(map[string]string{})
}

func (s *Store) All() (map[string]string, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &values); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (s *Store) Keys() ([]string, error) {
	values, err := s.All()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *Store) save(values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func Mask() string {
	return "********"
}
