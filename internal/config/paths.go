package config

import (
	"os"
	"path/filepath"
)

const (
	DataDir       = "data"
	PantryDir     = "pantry"
	WorkspacesDir = "workspaces"
	RackDir       = "rack"
	PacksDir      = "packs"
	SecretsDir    = "secrets"
	GlobalsFile   = "globals.yaml"
	ConfigFile    = "config.yaml"
)

type Paths struct {
	RootDir          string
	DataDir          string
	PantryDir        string
	WorkspacesDir    string
	RackDir          string
	PacksDir         string
	ImportedPacksDir string
	ExportedPacksDir string
	SecretsDir       string
	SecretsFile      string
	GlobalsFile      string
	ConfigFile       string
}

func DefaultPaths() (Paths, error) {
	root, err := os.Getwd()
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		RootDir:          root,
		DataDir:          filepath.Join(root, DataDir),
		PantryDir:        filepath.Join(root, DataDir, PantryDir),
		WorkspacesDir:    filepath.Join(root, DataDir, WorkspacesDir),
		RackDir:          filepath.Join(root, DataDir, RackDir),
		PacksDir:         filepath.Join(root, DataDir, PacksDir),
		ImportedPacksDir: filepath.Join(root, DataDir, PacksDir, "imported"),
		ExportedPacksDir: filepath.Join(root, DataDir, PacksDir, "exported"),
		SecretsDir:       filepath.Join(root, DataDir, SecretsDir),
		SecretsFile:      filepath.Join(root, DataDir, SecretsDir, "secrets.yaml"),
		GlobalsFile:      filepath.Join(root, DataDir, PantryDir, GlobalsFile),
		ConfigFile:       filepath.Join(root, DataDir, ConfigFile),
	}, nil
}
