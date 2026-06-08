package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Banner             bool   `yaml:"banner"`
	Theme              string `yaml:"theme"`
	PromptStyle        string `yaml:"prompt_style"`
	Tips               bool   `yaml:"tips"`
	StoreSecretOutputs bool   `yaml:"store_secret_outputs"`
}

type AppConfigStore struct {
	path string
}

func NewAppConfigStore(path string) *AppConfigStore {
	return &AppConfigStore{path: path}
}

func DefaultAppConfig() AppConfig {
	return AppConfig{
		Banner:             true,
		Theme:              "default",
		PromptStyle:        "full",
		Tips:               true,
		StoreSecretOutputs: false,
	}
}

func (s *AppConfigStore) Ensure() error {
	if _, err := os.Stat(s.path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return s.Save(DefaultAppConfig())
}

func (s *AppConfigStore) Load() (AppConfig, error) {
	if err := s.Ensure(); err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return AppConfig{}, err
	}
	cfg := DefaultAppConfig()
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return AppConfig{}, err
		}
	}
	if err := cfg.Validate(); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}

func (s *AppConfigStore) Save(cfg AppConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *AppConfigStore) Set(key, value string) (AppConfig, error) {
	cfg, err := s.Load()
	if err != nil {
		return AppConfig{}, err
	}
	switch key {
	case "banner":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return AppConfig{}, fmt.Errorf("banner must be true or false")
		}
		cfg.Banner = parsed
	case "tips":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return AppConfig{}, fmt.Errorf("tips must be true or false")
		}
		cfg.Tips = parsed
	case "store_secret_outputs":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return AppConfig{}, fmt.Errorf("store_secret_outputs must be true or false")
		}
		cfg.StoreSecretOutputs = parsed
	case "theme":
		cfg.Theme = value
	case "prompt_style":
		cfg.PromptStyle = value
	default:
		return AppConfig{}, fmt.Errorf("unknown config key: %s", key)
	}
	if err := s.Save(cfg); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}

func (c AppConfig) Validate() error {
	if c.Theme != "default" && c.Theme != "minimal" {
		return fmt.Errorf("theme must be default or minimal")
	}
	if c.PromptStyle != "full" && c.PromptStyle != "compact" {
		return fmt.Errorf("prompt_style must be full or compact")
	}
	return nil
}
