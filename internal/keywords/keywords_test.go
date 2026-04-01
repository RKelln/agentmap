package keywords

import (
	"strings"
	"testing"
)

func TestExtractKeywords_Basic(t *testing.T) {
	tests := []struct {
		name string
		text string
		max  int
		want string
	}{
		{
			name: "simple authentication text",
			text: "OAuth2 token exchange with PKCE proof key. Token lifecycle management for authentication platform.",
			max:  5,
			want: "token;authentication;exchange;key;lifecycle",
		},
		{
			name: "all stopwords returns empty",
			text: "the and or but if then else when this that these those",
			max:  5,
			want: "",
		},
		{
			name: "empty text returns empty",
			text: "",
			max:  5,
			want: "",
		},
		{
			name: "short tokens filtered out",
			text: "a an be is it my API OAuth",
			max:  5,
			want: "api;oauth",
		},
		{
			name: "respects max keywords",
			text: "alpha beta gamma delta epsilon zeta eta theta",
			max:  3,
			want: "alpha;beta;delta",
		},
		{
			name: "deduplicates and counts frequency",
			text: "token token token refresh refresh retry",
			max:  5,
			want: "token;refresh;retry",
		},
		{
			name: "handles punctuation",
			text: "OAuth2.0! Token-exchange: refresh; retry...",
			max:  5,
			want: "exchange;oauth2;refresh;retry;token",
		},
		{
			name: "case insensitive",
			text: "Token TOKEN token Auth AUTH",
			max:  5,
			want: "token;auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.text, tt.max)
			if got != tt.want {
				t.Errorf("ExtractKeywords() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractKeywords_Stopwords(t *testing.T) {
	tests := []struct {
		name string
		word string
	}{
		{name: "english: the", word: "the"},
		{name: "english: and", word: "and"},
		{name: "english: because", word: "because"},
		{name: "english: therefore", word: "therefore"},
		{name: "markdown: section", word: "section"},
		{name: "markdown: overview", word: "overview"},
		{name: "markdown: describes", word: "describes"},
		{name: "markdown: provides", word: "provides"},
		{name: "markdown: including", word: "including"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.word, 5)
			if got != "" {
				t.Errorf("stopword %q should be filtered, got %q", tt.word, got)
			}
		})
	}
}

func TestExtractPurpose(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "multi-section document",
			text: strings.Repeat("authentication token OAuth2 PKCE refresh revoke ", 10),
		},
		{
			name: "empty document",
			text: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPurpose(tt.text)
			if tt.text == "" && got != "" {
				t.Errorf("ExtractPurpose() = %q, want empty", got)
			}
			if tt.text != "" && got == "" {
				t.Errorf("ExtractPurpose() returned empty for non-empty text")
			}
			if strings.Contains(got, ",") {
				t.Errorf("purpose should not contain commas: %q", got)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "simple words",
			text: "hello world",
			want: []string{"hello", "world"},
		},
		{
			name: "with punctuation",
			text: "hello, world!",
			want: []string{"hello", "world"},
		},
		{
			name: "mixed case",
			text: "Hello WORLD",
			want: []string{"hello", "world"},
		},
		{
			name: "with numbers",
			text: "OAuth2 token",
			want: []string{"oauth2", "token"},
		},
		{
			name: "empty",
			text: "",
			want: nil,
		},
		{
			name: "only punctuation",
			text: "... ,,, ;;;",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
