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
		IndexInlineMax:  20,
		Exclude:         []string{".agentmap", ".agentmap/**", "AGENTMAP.md", "AGENTS.md", "CLAUDE.md", "LICENSE.md"},
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
				IndexInlineMax:  20,
				Exclude:         []string{".agentmap", ".agentmap/**", "AGENTMAP.md", "AGENTS.md", "CLAUDE.md", "LICENSE.md"},
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
				IndexInlineMax:  20,
				// User patterns prepended; default protected patterns always appended.
				Exclude: []string{"dist/**", "CHANGELOG.md", ".agentmap", ".agentmap/**", "AGENTMAP.md", "AGENTS.md", "CLAUDE.md", "LICENSE.md"},
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
				IndexInlineMax:  20,
				// User patterns prepended; default protected patterns always appended.
				Exclude: []string{"vendor/**", ".agentmap", ".agentmap/**", "AGENTMAP.md", "AGENTS.md", "CLAUDE.md", "LICENSE.md"},
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

func TestLoad_ExcludePreservesDefaults(t *testing.T) {
	// When user specifies exclude: in agentmap.yml, default protected patterns
	// must still be present in the merged result.
	dir := t.TempDir()
	path := filepath.Join(dir, "agentmap.yml")
	yaml := "exclude:\n  - CHANGELOG.md\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	hasAgentmap := false
	hasAgentmapGlob := false
	hasAgentmapMD := false
	hasAgentsMD := false
	hasClaudeMD := false
	hasLicenseMD := false
	for _, p := range cfg.Exclude {
		if p == ".agentmap" {
			hasAgentmap = true
		}
		if p == ".agentmap/**" {
			hasAgentmapGlob = true
		}
		if p == "AGENTMAP.md" {
			hasAgentmapMD = true
		}
		if p == "AGENTS.md" {
			hasAgentsMD = true
		}
		if p == "CLAUDE.md" {
			hasClaudeMD = true
		}
		if p == "LICENSE.md" {
			hasLicenseMD = true
		}
	}
	if !hasAgentmap {
		t.Errorf("Exclude should contain .agentmap; got %v", cfg.Exclude)
	}
	if !hasAgentmapGlob {
		t.Errorf("Exclude should contain .agentmap/**; got %v", cfg.Exclude)
	}
	if !hasAgentmapMD {
		t.Errorf("Exclude should contain AGENTMAP.md; got %v", cfg.Exclude)
	}
	if !hasAgentsMD {
		t.Errorf("Exclude should contain AGENTS.md; got %v", cfg.Exclude)
	}
	if !hasClaudeMD {
		t.Errorf("Exclude should contain CLAUDE.md; got %v", cfg.Exclude)
	}
	if !hasLicenseMD {
		t.Errorf("Exclude should contain LICENSE.md; got %v", cfg.Exclude)
	}
	// User pattern must also be present.
	hasChangelog := false
	for _, p := range cfg.Exclude {
		if p == "CHANGELOG.md" {
			hasChangelog = true
		}
	}
	if !hasChangelog {
		t.Errorf("Exclude should contain CHANGELOG.md; got %v", cfg.Exclude)
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
