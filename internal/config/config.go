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
	MaxNavEntries   int      `yaml:"max_nav_entries"`  // default 20
	NavStubWords    int      `yaml:"nav_stub_words"`   // default 20
	IndexInlineMax  int      `yaml:"index_inline_max"` // default 20
	Exclude         []string `yaml:"exclude"`
}

// Defaults returns a Config with all default values applied.
func Defaults() Config {
	return Config{
		MinLines:        50,
		SubThreshold:    50,
		ExpandThreshold: 150,
		MaxDepth:        3,
		MaxNavEntries:   20,
		NavStubWords:    20,
		IndexInlineMax:  20,
		Exclude:         []string{".agentmap", ".agentmap/**", "AGENTMAP.md", "AGENTS.md", "CLAUDE.md", "LICENSE.md"},
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

	// Unmarshal into a zero-value struct to detect which fields were set.
	// Numeric zero values are ambiguous, but for Exclude we can detect nil
	// (not specified) vs. non-nil (user provided a list, possibly empty).
	var user Config
	if err := yaml.Unmarshal(data, &user); err != nil {
		return Config{}, fmt.Errorf("config: parse yaml: %w", err)
	}

	cfg := Defaults()

	// Merge scalar fields: use user value if non-zero.
	if user.MinLines != 0 {
		cfg.MinLines = user.MinLines
	}
	if user.SubThreshold != 0 {
		cfg.SubThreshold = user.SubThreshold
	}
	if user.ExpandThreshold != 0 {
		cfg.ExpandThreshold = user.ExpandThreshold
	}
	if user.MaxDepth != 0 {
		cfg.MaxDepth = user.MaxDepth
	}
	if user.MaxNavEntries != 0 {
		cfg.MaxNavEntries = user.MaxNavEntries
	}
	if user.NavStubWords != 0 {
		cfg.NavStubWords = user.NavStubWords
	}
	if user.IndexInlineMax != 0 {
		cfg.IndexInlineMax = user.IndexInlineMax
	}

	// Merge exclude: if user specified exclude:, prepend their patterns then
	// always append the defaults so .agentmap protection is never silently lost.
	if user.Exclude != nil {
		seen := make(map[string]bool)
		var merged []string
		for _, p := range append(user.Exclude, cfg.Exclude...) {
			if !seen[p] {
				seen[p] = true
				merged = append(merged, p)
			}
		}
		cfg.Exclude = merged
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
