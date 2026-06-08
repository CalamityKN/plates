package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

type VariableStore interface {
	EnsureBase() error
	SetGlobal(key, value string) error
	Globals() (map[string]string, error)
	EnsureWorkspace(name string) error
	SetWorkspaceValue(workspace, key, value string) error
	WorkspaceValues(workspace string) (map[string]string, error)
}

type YAMLStore struct {
	paths Paths
}

func NewYAMLStore(paths Paths) *YAMLStore {
	return &YAMLStore{paths: paths}
}

func (s *YAMLStore) EnsureBase() error {
	for _, dir := range []string{s.paths.PantryDir, s.paths.WorkspacesDir, s.paths.RackDir, s.paths.PacksDir, s.paths.ImportedPacksDir, s.paths.ExportedPacksDir, s.paths.SecretsDir} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *YAMLStore) SetGlobal(key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	values, err := s.Globals()
	if err != nil {
		return err
	}
	values[key] = value
	return writeYAMLMap(s.paths.GlobalsFile, values)
}

func (s *YAMLStore) Globals() (map[string]string, error) {
	return readYAMLMap(s.paths.GlobalsFile)
}

func (s *YAMLStore) EnsureWorkspace(name string) error {
	if err := validateWorkspaceName(name); err != nil {
		return err
	}
	if err := s.EnsureBase(); err != nil {
		return err
	}
	path := s.workspacePath(name)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeYAMLMap(path, map[string]string{})
}

func (s *YAMLStore) SetWorkspaceValue(workspace, key, value string) error {
	if workspace == "" {
		return errors.New("no active workspace; run 'workspace <name>' first")
	}
	if err := validateKey(key); err != nil {
		return err
	}
	if err := s.EnsureWorkspace(workspace); err != nil {
		return err
	}
	values, err := s.WorkspaceValues(workspace)
	if err != nil {
		return err
	}
	values[key] = value
	return writeYAMLMap(s.workspacePath(workspace), values)
}

func (s *YAMLStore) WorkspaceValues(workspace string) (map[string]string, error) {
	if workspace == "" {
		return nil, errors.New("no active workspace; run 'workspace <name>' first")
	}
	if err := validateWorkspaceName(workspace); err != nil {
		return nil, err
	}
	return readYAMLMap(s.workspacePath(workspace))
}

func SortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *YAMLStore) workspacePath(name string) string {
	return filepath.Join(s.paths.WorkspacesDir, name+".yaml")
}

func readYAMLMap(path string) (map[string]string, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]string{}, nil
	}

	values := map[string]string{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func writeYAMLMap(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

var (
	keyPattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	workspacePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func validateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("invalid key %q; use letters, numbers, and underscores, starting with a letter or underscore", key)
	}
	return nil
}

func validateWorkspaceName(name string) error {
	if !workspacePattern.MatchString(name) {
		return fmt.Errorf("invalid workspace %q; use letters, numbers, hyphens, and underscores", name)
	}
	return nil
}
