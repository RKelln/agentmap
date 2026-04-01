package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	expected := Config{
		MinLines:        50,
		SubThreshold:    50,
		ExpandThreshold: 150,
		MaxDepth:        3,
		MaxNavEntries:   20,
		NavStubWords:    20,
		Exclude:         []string{},
	}
	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("Defaults() = %+v, want %+v", cfg, expected)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "nonexistent.yml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	expected := Defaults()
	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("Load(missing) = %+v, want %+v", cfg, expected)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected Config
	}{
		{
			name:     "empty file returns defaults",
			yaml:     "",
			expected: Defaults(),
		},
		{
			name: "partial config merges with defaults",
			yaml: "min_lines: 100\n",
			expected: Config{
				MinLines:        100,
				SubThreshold:    50,
				ExpandThreshold: 150,
				MaxDepth:        3,
				MaxNavEntries:   20,
				NavStubWords:    20,
				Exclude:         []string{},
			},
		},
		{
			name: "full config overrides all defaults",
			yaml: `min_lines: 30
sub_threshold: 40
expand_threshold: 200
max_depth: 2
exclude:
  - "dist/**"
  - "CHANGELOG.md"
`,
			expected: Config{
				MinLines:        30,
				SubThreshold:    40,
				ExpandThreshold: 200,
				MaxDepth:        2,
				MaxNavEntries:   20,
				NavStubWords:    20,
				Exclude:         []string{"dist/**", "CHANGELOG.md"},
			},
		},
		{
			name: "exclude only overrides",
			yaml: `exclude:
  - "vendor/**"
`,
			expected: Config{
				MinLines:        50,
				SubThreshold:    50,
				ExpandThreshold: 150,
				MaxDepth:        3,
				MaxNavEntries:   20,
				NavStubWords:    20,
				Exclude:         []string{"vendor/**"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "agentmap.yml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !reflect.DeepEqual(cfg, tt.expected) {
				t.Errorf("Load() = %+v, want %+v", cfg, tt.expected)
			}
		})
	}
}

func TestFindConfig(t *testing.T) {
	t.Run("finds in current directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "agentmap.yml")
		if err := os.WriteFile(path, []byte("min_lines: 100\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		found, err := FindConfig(dir)
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		if found != path {
			t.Errorf("FindConfig() = %q, want %q", found, path)
		}
	})

	t.Run("searches upward", func(t *testing.T) {
		root := t.TempDir()
		configPath := filepath.Join(root, "agentmap.yml")
		if err := os.WriteFile(configPath, []byte("min_lines: 100\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		subdir := filepath.Join(root, "docs", "api")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		found, err := FindConfig(subdir)
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		if found != configPath {
			t.Errorf("FindConfig() = %q, want %q", found, configPath)
		}
	})

	t.Run("returns empty when not found", func(t *testing.T) {
		dir := t.TempDir()
		found, err := FindConfig(dir)
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		if found != "" {
			t.Errorf("FindConfig() = %q, want empty", found)
		}
	})
}
