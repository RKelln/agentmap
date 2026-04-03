package templates_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/ryankelln/agentmap/internal/templates"
)

// mdTemplates lists the .md.tmpl files that must have <!-- agentmap:init --> markers.
var mdTemplates = []string{
	"agents.md.tmpl",
	"cursor.md.tmpl",
	"windsurf.md.tmpl",
	"continue.md.tmpl",
	"roo.md.tmpl",
	"amazonq.md.tmpl",
	"opencode-skill.md.tmpl",
}

// frontmatterTemplates lists the templates that carry YAML frontmatter.
var frontmatterTemplates = []string{
	"cursor.md.tmpl",
	"windsurf.md.tmpl",
	"continue.md.tmpl",
	"opencode-skill.md.tmpl",
}

func TestAllNamesReturnsNames(t *testing.T) {
	names := templates.AllNames()
	if len(names) == 0 {
		t.Fatal("AllNames() returned empty slice; expected at least one template")
	}
}

func TestGetReturnsContentForAllNames(t *testing.T) {
	for _, name := range templates.AllNames() {
		t.Run(name, func(t *testing.T) {
			got, err := templates.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) returned error: %v", name, err)
			}
			if len(got) == 0 {
				t.Fatalf("Get(%q) returned empty content", name)
			}
		})
	}
}

func TestMarkdownTemplatesHaveInitMarker(t *testing.T) {
	for _, name := range mdTemplates {
		t.Run(name, func(t *testing.T) {
			data, err := templates.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			content := string(data)
			if !strings.Contains(content, "<!-- agentmap:init -->") {
				t.Errorf("%s: missing opening marker <!-- agentmap:init -->", name)
			}
			if !strings.Contains(content, "<!-- /agentmap:init -->") {
				t.Errorf("%s: missing closing marker <!-- /agentmap:init -->", name)
			}
		})
	}
}

func TestFrontmatterTemplatesHaveValidYAML(t *testing.T) {
	for _, name := range frontmatterTemplates {
		t.Run(name, func(t *testing.T) {
			data, err := templates.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			content := string(data)

			// Frontmatter must start with ---
			if !strings.HasPrefix(content, "---\n") {
				t.Fatalf("%s: expected frontmatter starting with ---", name)
			}

			// Find the closing ---
			rest := content[4:] // skip opening "---\n"
			idx := strings.Index(rest, "\n---\n")
			if idx == -1 {
				t.Fatalf("%s: could not find closing frontmatter delimiter ---", name)
			}

			fm := rest[:idx]
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(fm), &parsed); err != nil {
				t.Errorf("%s: YAML frontmatter parse error: %v", name, err)
			}
		})
	}
}

func TestTemplateBodyHasNoCommas(t *testing.T) {
	for _, name := range mdTemplates {
		t.Run(name, func(t *testing.T) {
			data, err := templates.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			content := string(data)

			// Determine where the body starts: after frontmatter if present,
			// otherwise from the start.
			body := content
			if strings.HasPrefix(content, "---\n") {
				rest := content[4:]
				idx := strings.Index(rest, "\n---\n")
				if idx != -1 {
					// body begins after the closing "---\n"
					body = rest[idx+5:]
				}
			}

			if strings.Contains(body, ",") {
				t.Errorf("%s: body contains a comma (use semicolons instead)", name)
			}
		})
	}
}

func TestGetNonexistentReturnsError(t *testing.T) {
	_, err := templates.Get("nonexistent.tmpl")
	if err == nil {
		t.Error("Get(\"nonexistent.tmpl\") expected error, got nil")
	}
}
