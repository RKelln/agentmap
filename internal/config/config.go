// Package config loads agentmap.yml configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds agentmap configuration values.
type Config struct {
	MinLines        int      `yaml:"min_lines"`
	SubThreshold    int      `yaml:"sub_threshold"`
	ExpandThreshold int      `yaml:"expand_threshold"`
	MaxDepth        int      `yaml:"max_depth"`
	Exclude         []string `yaml:"exclude"`
}

// Defaults returns a Config with all default values applied.
func Defaults() Config {
	return Config{
		MinLines:        50,
		SubThreshold:    50,
		ExpandThreshold: 150,
		MaxDepth:        3,
		Exclude:         []string{},
	}
}

// Load reads a YAML config file and merges it with defaults.
// If the file doesn't exist, returns Defaults() with no error.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return Config{}, fmt.Errorf("config: read file: %w", err)
	}

	cfg := Defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse yaml: %w", err)
	}

	if cfg.Exclude == nil {
		cfg.Exclude = []string{}
	}

	return cfg, nil
}

// FindConfig searches upward from startDir for agentmap.yml.
// Returns the path to the file, or empty string if not found.
func FindConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("config: resolve path: %w", err)
	}

	for {
		path := filepath.Join(dir, "agentmap.yml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}
