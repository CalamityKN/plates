package config

import (
	"path/filepath"
	"testing"
)

func TestYAMLStorePersistsGlobalAndWorkspaceValues(t *testing.T) {
	root := t.TempDir()
	paths := Paths{
		RootDir:       root,
		DataDir:       filepath.Join(root, DataDir),
		PantryDir:     filepath.Join(root, DataDir, PantryDir),
		WorkspacesDir: filepath.Join(root, DataDir, WorkspacesDir),
		RackDir:       filepath.Join(root, DataDir, RackDir),
		GlobalsFile:   filepath.Join(root, DataDir, PantryDir, GlobalsFile),
	}
	store := NewYAMLStore(paths)

	if err := store.EnsureBase(); err != nil {
		t.Fatalf("EnsureBase() error = %v", err)
	}
	if err := store.SetGlobal("my_ip", "10.10.14.3"); err != nil {
		t.Fatalf("SetGlobal() error = %v", err)
	}
	if err := store.EnsureWorkspace("devhub"); err != nil {
		t.Fatalf("EnsureWorkspace() error = %v", err)
	}
	if err := store.SetWorkspaceValue("devhub", "target", "10.129.202.242"); err != nil {
		t.Fatalf("SetWorkspaceValue() error = %v", err)
	}

	globals, err := store.Globals()
	if err != nil {
		t.Fatalf("Globals() error = %v", err)
	}
	if globals["my_ip"] != "10.10.14.3" {
		t.Fatalf("Globals()[my_ip] = %q", globals["my_ip"])
	}

	values, err := store.WorkspaceValues("devhub")
	if err != nil {
		t.Fatalf("WorkspaceValues() error = %v", err)
	}
	if values["target"] != "10.129.202.242" {
		t.Fatalf("WorkspaceValues()[target] = %q", values["target"])
	}
}
