package templates_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/RKelln/agentmap/internal/templates"
)

// mdTemplates lists the .md.tmpl files that must have <!-- agentmap:init --> markers.
var mdTemplates = []string{
	"agents.md.tmpl",
	"amazonq.md.tmpl",
	"continue.md.tmpl",
	"cursor.md.tmpl",
	"opencode-skill.md.tmpl",
	"roo.md.tmpl",
	"windsurf.md.tmpl",
}

// hookTemplates lists the hook templates that must have # agentmap: validate markers.
var hookTemplates = []string{
	"hook-git.sh.tmpl",
	"hook-precommit.yml.tmpl",
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

func TestAllNamesExcludesUnderscorePrefixed(t *testing.T) {
	names := templates.AllNames()
	for _, name := range names {
		if strings.HasPrefix(name, "_") {
			t.Errorf("AllNames() returned %q; files starting with _ should be excluded", name)
		}
	}
}

func TestBodyIsReadable(t *testing.T) {
	body, err := templates.Body()
	if err != nil {
		t.Fatalf("Body() error: %v", err)
	}
	if body == "" {
		t.Fatal("Body() returned empty string")
	}
	// The body must contain the two canonical section headings.
	if !strings.Contains(body, "## Reading Markdown Files") {
		t.Error("Body() missing '## Reading Markdown Files' section")
	}
	if !strings.Contains(body, "## Before Updating Agentmap") {
		t.Error("Body() missing '## Before Updating Agentmap' section")
	}
}

func TestMarkdownTemplatesContainBody(t *testing.T) {
	body, err := templates.Body()
	if err != nil {
		t.Fatalf("Body() error: %v", err)
	}

	for _, name := range mdTemplates {
		t.Run(name, func(t *testing.T) {
			data, err := templates.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			content := string(data)
			if !strings.Contains(content, body) {
				t.Errorf("%s: content does not contain the shared Body() text", name)
			}
		})
	}
}

func TestHookTemplatesHaveValidateMarkers(t *testing.T) {
	for _, name := range hookTemplates {
		t.Run(name, func(t *testing.T) {
			data, err := templates.Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			content := string(data)
			if !strings.Contains(content, "# agentmap: validate") {
				t.Errorf("%s: missing opening marker '# agentmap: validate'", name)
			}
			if !strings.Contains(content, "# /agentmap: validate") {
				t.Errorf("%s: missing closing marker '# /agentmap: validate'", name)
			}
		})
	}
}
