package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseHeadings_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxDepth int
		want     []Heading
	}{
		{
			name:     "single h1",
			input:    "# Hello\n",
			maxDepth: 3,
			want:     []Heading{{Line: 1, Depth: 1, Text: "Hello"}},
		},
		{
			name:     "h1 and h2",
			input:    "# Title\n\n## Section\n",
			maxDepth: 3,
			want: []Heading{
				{Line: 1, Depth: 1, Text: "Title"},
				{Line: 3, Depth: 2, Text: "Section"},
			},
		},
		{
			name:     "three levels",
			input:    "# A\n## B\n### C\n",
			maxDepth: 3,
			want: []Heading{
				{Line: 1, Depth: 1, Text: "A"},
				{Line: 2, Depth: 2, Text: "B"},
				{Line: 3, Depth: 3, Text: "C"},
			},
		},
		{
			name:     "respects maxDepth",
			input:    "# A\n## B\n### C\n#### D\n",
			maxDepth: 2,
			want: []Heading{
				{Line: 1, Depth: 1, Text: "A"},
				{Line: 2, Depth: 2, Text: "B"},
			},
		},
		{
			name:     "heading with extra spaces",
			input:    "##  Spaced  \n",
			maxDepth: 3,
			want:     []Heading{{Line: 1, Depth: 2, Text: "Spaced"}},
		},
		{
			name:     "no headings",
			input:    "just some text\n",
			maxDepth: 3,
			want:     nil,
		},
		{
			name:     "empty input",
			input:    "",
			maxDepth: 3,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseHeadings(tt.input, tt.maxDepth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseHeadings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHeadings_CodeFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxDepth int
		want     []Heading
	}{
		{
			name:     "heading inside backtick fence",
			input:    "```\n## Not a heading\n```\n## Real heading\n",
			maxDepth: 3,
			want:     []Heading{{Line: 4, Depth: 2, Text: "Real heading"}},
		},
		{
			name:     "heading inside tilde fence",
			input:    "~~~\n## Not a heading\n~~~\n## Real heading\n",
			maxDepth: 3,
			want:     []Heading{{Line: 4, Depth: 2, Text: "Real heading"}},
		},
		{
			name:     "fence with language hint",
			input:    "```markdown\n## Not a heading\n```\n## Real\n",
			maxDepth: 3,
			want:     []Heading{{Line: 4, Depth: 2, Text: "Real"}},
		},
		{
			name:     "multiple fences",
			input:    "## Before\n```\n# Inside\n```\n## After\n",
			maxDepth: 3,
			want: []Heading{
				{Line: 1, Depth: 2, Text: "Before"},
				{Line: 5, Depth: 2, Text: "After"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseHeadings(tt.input, tt.maxDepth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseHeadings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHeadings_HTMLComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Heading
	}{
		{
			name:  "block comment hides heading",
			input: "<!--\n## Hidden heading\n-->\n## Visible heading\n",
			want:  []Heading{{Line: 4, Depth: 2, Text: "Visible heading"}},
		},
		{
			name:  "inline <!-- in prose does not open comment",
			input: "Find files with `<!-- AGENT:NAV` block.\n## Still visible\n",
			want:  []Heading{{Line: 2, Depth: 2, Text: "Still visible"}},
		},
		{
			name:  "inline <!-- and --> on same prose line does not swallow next headings",
			input: "Blocks like `<!-- -->` are skipped.\n## Still visible\n",
			want:  []Heading{{Line: 2, Depth: 2, Text: "Still visible"}},
		},
		{
			name: "inline <!-- without --> on prose line does not swallow subsequent headings",
			input: "See `<!-- AGENT:NAV` for the format.\n" +
				"## Section A\n" +
				"Content about `-->` syntax.\n" +
				"## Section B\n",
			want: []Heading{
				{Line: 2, Depth: 2, Text: "Section A"},
				{Line: 4, Depth: 2, Text: "Section B"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseHeadings(tt.input, 3)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseHeadings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHeadings_Frontmatter(t *testing.T) {
	input := "---\ntitle: Test\n---\n# Heading\n"
	want := []Heading{{Line: 4, Depth: 1, Text: "Heading"}}

	got, _ := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
}

func TestParseHeadings_RegressionInlineCommentMarker(t *testing.T) {
	// Prose lines containing inline `<!-- AGENT:NAV` (backtick-quoted) were
	// wrongly opening an HTML comment state, swallowing all headings until the
	// next --> appeared anywhere in the document.
	input := "## Section 4.2\n" +
		"Find files that have an existing `<!-- AGENT:NAV` block.\n" +
		"## Section 4.3\n" +
		"Content referencing `-->` syntax.\n" +
		"## Section 5\n"
	want := []Heading{
		{Line: 1, Depth: 2, Text: "Section 4.2"},
		{Line: 3, Depth: 2, Text: "Section 4.3"},
		{Line: 5, Depth: 2, Text: "Section 5"},
	}

	got, _ := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
}

func TestParseHeadings_UnclosedFenceWarning(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantHeadings []Heading
		wantWarning  bool
	}{
		{
			name:         "properly closed fence has no warning",
			input:        "```bash\n# in fence\n```\n## Real\n",
			wantHeadings: []Heading{{Line: 4, Depth: 2, Text: "Real"}},
			wantWarning:  false,
		},
		{
			name:         "unclosed fence at EOF produces warning",
			input:        "## Before\n```bash\n# not a heading\n",
			wantHeadings: []Heading{{Line: 1, Depth: 2, Text: "Before"}},
			wantWarning:  true,
		},
		{
			name:         "empty file has no warning",
			input:        "",
			wantHeadings: nil,
			wantWarning:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warnings := ParseHeadings(tt.input, 3)
			if !reflect.DeepEqual(got, tt.wantHeadings) {
				t.Errorf("ParseHeadings() headings = %v, want %v", got, tt.wantHeadings)
			}
			hasWarning := len(warnings) > 0
			if hasWarning != tt.wantWarning {
				t.Errorf("ParseHeadings() warnings = %v, wantWarning = %v", warnings, tt.wantWarning)
			}
		})
	}
}

func TestParseHeadings_FenceClosingRules(t *testing.T) {
	// Per CommonMark: only a bare closing fence (no info string) can close a
	// fenced code block. A line like ```bash cannot close an open fence.
	tests := []struct {
		name     string
		input    string
		maxDepth int
		want     []Heading
	}{
		{
			name: "lang-fence inside open fence is content not closer",
			// ```bash opens, then ```bash inside cannot close it, bare ``` closes
			input:    "```bash\n# not a heading\n```bash\n# still not a heading\n```\n## Real\n",
			maxDepth: 3,
			want:     []Heading{{Line: 6, Depth: 2, Text: "Real"}},
		},
		{
			name: "stray bare fence then lang-fence does not expose shell comments",
			// Normal block closes. Stray ``` opens phantom fence.
			// ```bash inside phantom fence must NOT close it (would expose # comments).
			// Second bare ``` closes phantom fence. ## Visible is then outside.
			input:    "```bash\n# in fence\n```\n## After\n```\n## Phantom content\n```bash\n# not a heading\n```\n## Visible\n",
			maxDepth: 3,
			want: []Heading{
				{Line: 4, Depth: 2, Text: "After"},
				{Line: 10, Depth: 2, Text: "Visible"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseHeadings(tt.input, tt.maxDepth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseHeadings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHeadings_DuplicateHeadings(t *testing.T) {
	input := "## Examples\n\nSome text.\n\n## Examples\n\nMore text.\n"
	want := []Heading{
		{Line: 1, Depth: 2, Text: "Examples"},
		{Line: 5, Depth: 2, Text: "Examples"},
	}

	got, _ := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
}

func TestParseHeadings_DirtyFrontmatterClose(t *testing.T) {
	input := "---\nfoo\n---<!-- AGENT:NAV\n# Heading\n"
	want := []Heading{{Line: 4, Depth: 1, Text: "Heading"}}
	got, warnings := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "frontmatter close delimiter has trailing content") {
		t.Errorf("expected dirty frontmatter warning, got %v", warnings)
	}
}
