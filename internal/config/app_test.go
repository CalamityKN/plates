package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigCreation(t *testing.T) {
	store := NewAppConfigStore(filepath.Join(t.TempDir(), "data", "config.yaml"))
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Banner || cfg.Theme != "default" || cfg.PromptStyle != "full" || !cfg.Tips {
		t.Fatalf("cfg = %#v", cfg)
	}
	if cfg.StoreSecretOutputs {
		t.Fatalf("StoreSecretOutputs = true")
	}
}

func TestConfigSetValidation(t *testing.T) {
	store := NewAppConfigStore(filepath.Join(t.TempDir(), "data", "config.yaml"))
	cfg, err := store.Set("banner", "false")
	if err != nil {
		t.Fatalf("Set(banner) error = %v", err)
	}
	if cfg.Banner {
		t.Fatalf("Banner = true")
	}
	if _, err := store.Set("theme", "loud"); err == nil {
		t.Fatal("Set(theme loud) error = nil")
	}
	if _, err := store.Set("prompt_style", "tiny"); err == nil {
		t.Fatal("Set(prompt_style tiny) error = nil")
	}
	if _, err := store.Set("tips", "maybe"); err == nil {
		t.Fatal("Set(tips maybe) error = nil")
	}
	cfg, err = store.Set("store_secret_outputs", "true")
	if err != nil {
		t.Fatalf("Set(store_secret_outputs) error = %v", err)
	}
	if !cfg.StoreSecretOutputs {
		t.Fatal("StoreSecretOutputs = false")
	}
}
