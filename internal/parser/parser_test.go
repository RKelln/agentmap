package parser

import (
	"reflect"
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
			got := ParseHeadings(tt.input, tt.maxDepth)
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
			got := ParseHeadings(tt.input, tt.maxDepth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseHeadings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHeadings_HTMLComments(t *testing.T) {
	input := "<!--\n## Hidden heading\n-->\n## Visible heading\n"
	want := []Heading{{Line: 4, Depth: 2, Text: "Visible heading"}}

	got := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
}

func TestParseHeadings_Frontmatter(t *testing.T) {
	input := "---\ntitle: Test\n---\n# Heading\n"
	want := []Heading{{Line: 4, Depth: 1, Text: "Heading"}}

	got := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
}

func TestParseHeadings_DuplicateHeadings(t *testing.T) {
	input := "## Examples\n\nSome text.\n\n## Examples\n\nMore text.\n"
	want := []Heading{
		{Line: 1, Depth: 2, Text: "Examples"},
		{Line: 5, Depth: 2, Text: "Examples"},
	}

	got := ParseHeadings(input, 3)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHeadings() = %v, want %v", got, want)
	}
}
