package search

import (
	"reflect"
	"testing"

	"github.com/RKelln/agentmap/internal/parser"
)

func TestScore_Exact(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMin float64
	}{
		{
			name:    "exact match",
			query:   "Budget Justification",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "exact match different case",
			query:   "budget justification",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "exact match extra spaces",
			query:   "  budget   justification  ",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got < tt.wantMin {
				t.Errorf("Score(%q, %q) = %v, want >= %v", tt.query, tt.heading, got, tt.wantMin)
			}
		})
	}
}

func TestScore_Substring(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		want    float64
	}{
		{
			name:    "heading contains query with shared token",
			query:   "Budget",
			heading: "Budget Justification and Timeline",
			want:    0.95,
		},
		{
			name:    "query contains heading with shared token",
			query:   "The Budget Justification process",
			heading: "Budget Justification",
			want:    0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got != tt.want {
				t.Errorf("Score(%q, %q) = %v, want %v", tt.query, tt.heading, got, tt.want)
			}
		})
	}
}

func TestScore_SubstringNoSharedToken(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMax float64
	}{
		{
			name:    "substring without shared token",
			query:   "get",
			heading: "Budget Justification",
			wantMax: 0.45,
		},
		{
			name:    "shared prefix only, no full token",
			query:   "just",
			heading: "Budget Justification",
			wantMax: 0.60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got > tt.wantMax {
				t.Errorf("Score(%q, %q) = %v, want <= %v", tt.query, tt.heading, got, tt.wantMax)
			}
		})
	}
}

func TestScore_WordReorder(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMin float64
	}{
		{
			name:    "reordered words exact",
			query:   "justification budget",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "reordered with extra words",
			query:   "budget timeline",
			heading: "Project Timeline and Budget Summary",
			wantMin: 0.65,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got < tt.wantMin {
				t.Errorf("Score(%q, %q) = %v, want >= %v", tt.query, tt.heading, got, tt.wantMin)
			}
		})
	}
}

func TestScore_PartialWord(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMin float64
	}{
		{
			name:    "single word match in longer heading",
			query:   "budget",
			heading: "Project Budget and Timeline Overview",
			wantMin: 0.60,
		},
		{
			name:    "partial overlap extra query words",
			query:   "budget justification 2024",
			heading: "Budget Justification",
			wantMin: 0.70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got < tt.wantMin {
				t.Errorf("Score(%q, %q) = %v, want >= %v", tt.query, tt.heading, got, tt.wantMin)
			}
		})
	}
}

func TestScore_Typo(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMin float64
	}{
		{
			name:    "single character typo",
			query:   "Budget Justifcation",
			heading: "Budget Justification",
			wantMin: 0.85,
		},
		{
			name:    "missing letter",
			query:   "Budgt Justification",
			heading: "Budget Justification",
			wantMin: 0.80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got < tt.wantMin {
				t.Errorf("Score(%q, %q) = %v, want >= %v", tt.query, tt.heading, got, tt.wantMin)
			}
		})
	}
}

func TestScore_NoMatch(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMax float64
	}{
		{
			name:    "completely different",
			query:   "budget",
			heading: "Timeline Overview",
			wantMax: 0.15,
		},
		{
			name:    "no common words",
			query:   "budget justification",
			heading: "Timeline and Schedule",
			wantMax: 0.20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if got > tt.wantMax {
				t.Errorf("Score(%q, %q) = %v, want <= %v", tt.query, tt.heading, got, tt.wantMax)
			}
		})
	}
}

func TestScore_ORTerms(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		heading string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "first term matches exactly",
			query:   "budget|timeline|schedule",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "second term matches",
			query:   "alpha|budget|gamma",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "exact match via OR",
			query:   "alpha|Budget Justification|gamma",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "second term matches",
			query:   "alpha|budget|gamma",
			heading: "Budget Justification",
			wantMin: 0.60,
		},
		{
			name:    "exact match via OR",
			query:   "alpha|Budget Justification|gamma",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "no variant matches",
			query:   "xyzzy|quux|flob",
			heading: "Budget Justification",
			wantMax: 0.18,
		},
		{
			name:    "spaces around pipes",
			query:   "alpha | budget | gamma",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "trailing pipe",
			query:   "budget|",
			heading: "Budget Justification",
			wantMin: 0.90,
		},
		{
			name:    "only pipe",
			query:   "|",
			heading: "Budget",
			wantMax: 0.0,
		},
		{
			name:    "OR fallback: only second variant has moderate match",
			query:   "xyzzy|justifcation",
			heading: "Budget Justification",
			wantMin: 0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Score(tt.query, tt.heading)
			if tt.wantMin > 0 && got < tt.wantMin {
				t.Errorf("Score(%q, %q) = %v, want >= %v", tt.query, tt.heading, got, tt.wantMin)
			}
			if tt.wantMax > 0 && got > tt.wantMax {
				t.Errorf("Score(%q, %q) = %v, want <= %v", tt.query, tt.heading, got, tt.wantMax)
			}
		})
	}
}

func TestScore_Empty(t *testing.T) {
	if got := Score("", "Budget"); got != 0 {
		t.Errorf("Score empty query = %v, want 0", got)
	}
	if got := Score("Budget", ""); got != 0 {
		t.Errorf("Score empty heading = %v, want 0", got)
	}
	if got := Score("", ""); got != 0 {
		t.Errorf("Score both empty = %v, want 0", got)
	}
}

func TestScore_Punctuation(t *testing.T) {
	got := Score("budget justification", "Budget: Justification (Revised)")
	if got < 0.70 {
		t.Errorf("Score with punctuation = %v, want >= 0.70", got)
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  Hello   World  ", "hello world"},
		{"BUDGET", "budget"},
		{"Budget\nJustification", "budget justification"},
		{"\tTab\tTest", "tab test"},
	}

	for _, tt := range tests {
		got := normalize(tt.input)
		if got != tt.want {
			t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"budget: justification", []string{"budget", "justification"}},
		{"one, two; three", []string{"one", "two", "three"}},
		{"  spaces   everywhere  ", []string{"spaces", "everywhere"}},
	}

	for _, tt := range tests {
		got := tokenize(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLevenshteinDist(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{"hello", "hello", 0},
		{"hello", "hallo", 1},
		{"abc", "", 3},
		{"", "", 0},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		got := levenshteinDist(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshteinDist(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestEditRatio(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want float64
	}{
		{"hello", "hello", 1.0},
		{"hello", "hallo", 0.8},
		{"abc", "xyz", 0.0},
		{"", "", 1.0},
	}

	for _, tt := range tests {
		got := editRatio(tt.a, tt.b)
		if !floatEqual(got, tt.want) {
			t.Errorf("editRatio(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestHasNavBlock(t *testing.T) {
	if !hasNavBlock("foo <!-- AGENT:NAV\nbar\n-->") {
		t.Error("hasNavBlock should detect AGENT:NAV")
	}
	if hasNavBlock("no nav block here") {
		t.Error("hasNavBlock should return false for content without AGENT:NAV")
	}
}

func TestLineCount(t *testing.T) {
	tests := []struct {
		content string
		want    int
	}{
		{"", 0},
		{"one line", 0},
		{"line1\nline2", 1},
		{"line1\nline2\n", 2},
	}

	for _, tt := range tests {
		got := lineCount(tt.content)
		if got != tt.want {
			t.Errorf("lineCount(%q) = %d, want %d", tt.content, got, tt.want)
		}
	}
}

func TestFuzzyTokenMatch(t *testing.T) {
	tests := []struct {
		name   string
		source []string
		target []string
		want   float64
	}{
		{
			name:   "exact tokens",
			source: []string{"budget", "justification"},
			target: []string{"budget", "justification"},
			want:   1.0,
		},
		{
			name:   "reordered",
			source: []string{"justification", "budget"},
			target: []string{"budget", "justification"},
			want:   1.0,
		},
		{
			name:   "typo",
			source: []string{"budget", "justifcation"},
			target: []string{"budget", "justification"},
			want:   0.96,
		},
		{
			name:   "no match",
			source: []string{"budget"},
			target: []string{"timeline", "overview"},
			want:   0.13,
		},
		{
			name:   "empty source",
			source: []string{},
			target: []string{"a", "b"},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyTokenMatch(tt.source, tt.target)
			if !floatEqualApprox(got, tt.want, 0.05) {
				t.Errorf("fuzzyTokenMatch(%v, %v) = %v, want ~%v", tt.source, tt.target, got, tt.want)
			}
		})
	}
}

func TestExtractContent(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\n"

	tests := []struct {
		name  string
		start int
		end   int
		want  string
	}{
		{
			name:  "full section",
			start: 2,
			end:   4,
			want:  "line2\nline3\nline4",
		},
		{
			name:  "single line section",
			start: 3,
			end:   3,
			want:  "line3",
		},
		{
			name:  "first line",
			start: 1,
			end:   2,
			want:  "line1\nline2",
		},
		{
			name:  "end beyond file",
			start: 4,
			end:   10,
			want:  "line4\nline5\n",
		},
		{
			name:  "start zero",
			start: 0,
			end:   3,
			want:  "",
		},
		{
			name:  "start beyond file",
			start: 10,
			end:   12,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := parser.Section{Heading: parser.Heading{Line: tt.start, Text: "test"}, Start: tt.start, End: tt.end}
			got := extractContent(content, s)
			if got != tt.want {
				t.Errorf("extractContent(start=%d, end=%d) = %q, want %q", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

func TestSharesExactToken(t *testing.T) {
	tests := []struct {
		a    []string
		b    []string
		want bool
	}{
		{[]string{"budget"}, []string{"budget", "justification"}, true},
		{[]string{"get"}, []string{"budget", "justification"}, false},
		{[]string{"a", "b"}, []string{"c", "d"}, false},
		{[]string{}, []string{"a"}, false},
	}

	for _, tt := range tests {
		got := sharesExactToken(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("sharesExactToken(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func floatEqual(a, b float64) bool {
	return floatEqualApprox(a, b, 0.0001)
}

func floatEqualApprox(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
