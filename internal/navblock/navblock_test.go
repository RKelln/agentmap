package navblock

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeHeading(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips leading hashes and whitespace",
			input: "##Setup Configuration",
			want:  "Setup Configuration",
		},
		{
			name:  "strips commas",
			input: "##Setup, Configuration",
			want:  "Setup Configuration",
		},
		{
			name:  "strips leading whitespace after hash removal",
			input: "## Setup, Configuration",
			want:  "Setup Configuration",
		},
		{
			name:  "h1 hash",
			input: "#Authentication",
			want:  "Authentication",
		},
		{
			name:  "h3 with comma",
			input: "###Token Exchange, Refresh",
			want:  "Token Exchange Refresh",
		},
		{
			name:  "no hash prefix",
			input: "Setup Configuration",
			want:  "Setup Configuration",
		},
		{
			name:  "multiple commas",
			input: "##A, B, C",
			want:  "A B C",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only hashes",
			input: "##",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeHeading(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeHeading(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseNavBlock_ReturnsParseResult(t *testing.T) {
	// Verify ParseNavBlock returns a ParseResult struct (not bare multi-return).
	tests := []struct {
		name      string
		content   string
		wantFound bool
		wantCorr  bool
		wantStart int // 1-indexed
		wantEnd   int // 1-indexed
	}{
		{
			name: "valid block returns correct struct",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,10,#Heading,desc
-->
`,
			wantFound: true,
			wantCorr:  false,
			wantStart: 1,
			wantEnd:   5,
		},
		{
			name:      "no block returns Found=false Start=-1 End=-1",
			content:   "# No nav block\n",
			wantFound: false,
			wantCorr:  false,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name: "corrupted block returns Corrupted=true",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
badentry
-->
`,
			wantFound: true,
			wantCorr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseNavBlock(tt.content)
			if result.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", result.Found, tt.wantFound)
			}
			if result.Corrupted != tt.wantCorr {
				t.Errorf("Corrupted = %v, want %v", result.Corrupted, tt.wantCorr)
			}
			if tt.wantStart != 0 {
				if result.Start != tt.wantStart {
					t.Errorf("Start = %d, want %d", result.Start, tt.wantStart)
				}
			}
			if tt.wantEnd != 0 {
				if result.End != tt.wantEnd {
					t.Errorf("End = %d, want %d", result.End, tt.wantEnd)
				}
			}
		})
	}
}

func TestParseNavBlock_Corrupted(t *testing.T) {
	// §11.6: corrupted nav block → corrupted=true, treat as no block.
	tests := []struct {
		name      string
		content   string
		wantFound bool
		wantCorr  bool
	}{
		{
			name: "truncated block missing closing -->",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,10,#Heading,desc
`,
			wantFound: false, // no closing -->, so not found at all
			wantCorr:  false,
		},
		{
			name: "nav entry with only 1 field (missing n, name, about)",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
badentry
-->
`,
			wantFound: true,
			wantCorr:  true,
		},
		{
			name: "nav entry with only 2 fields",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,10
-->
`,
			wantFound: true,
			wantCorr:  true,
		},
		{
			name: "n < 1 in a block",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,0,#Heading,desc
-->
`,
			wantFound: true,
			wantCorr:  true,
		},
		{
			name: "valid block is not corrupted",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,10,#Heading,desc
-->
`,
			wantFound: true,
			wantCorr:  false,
		},
		{
			name: "purpose-only block is not corrupted",
			content: `<!-- AGENT:NAV
purpose:test
-->
`,
			wantFound: true,
			wantCorr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseNavBlock(tt.content)
			if result.Found != tt.wantFound {
				t.Errorf("found = %v, want %v", result.Found, tt.wantFound)
			}
			if result.Corrupted != tt.wantCorr {
				t.Errorf("corrupted = %v, want %v", result.Corrupted, tt.wantCorr)
			}
		})
	}
}

func TestParseNavBlock_Full(t *testing.T) {
	content := `<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange
nav[3]{s,n,name,about}:
12,54,#Authentication,token lifecycle management
14,22,##Token Exchange,OAuth2 code-for-token flow
36,15,##Token Refresh,silent rotation and expiry
see[1]{path,why}:
src/config.py,default timeout and token TTL values
-->
# Real content starts here
`

	result := ParseNavBlock(content)
	block := result.Block
	start := result.Start
	end := result.End
	found := result.Found
	if !found {
		t.Fatal("expected nav block to be found")
	}
	if start != 1 {
		t.Errorf("startLine = %d, want 1", start)
	}
	if end != 9 {
		t.Errorf("endLine = %d, want 9", end)
	}
	if block.Purpose != "token lifecycle; OAuth2 exchange" {
		t.Errorf("purpose = %q, want %q", block.Purpose, "token lifecycle; OAuth2 exchange")
	}
	if len(block.Nav) != 3 {
		t.Fatalf("len(Nav) = %d, want 3", len(block.Nav))
	}
	want := NavEntry{Start: 12, N: 54, Name: "#Authentication", About: "token lifecycle management"}
	if !reflect.DeepEqual(block.Nav[0], want) {
		t.Errorf("nav[0] = %+v, want %+v", block.Nav[0], want)
	}
	if len(block.See) != 1 {
		t.Fatalf("len(See) = %d, want 1", len(block.See))
	}
	if block.See[0].Path != "src/config.py" {
		t.Errorf("see[0].Path = %q, want %q", block.See[0].Path, "src/config.py")
	}
}

func TestParseNavBlock_PurposeOnly(t *testing.T) {
	content := `<!-- AGENT:NAV
purpose:helper utilities
-->
# Tiny file
`

	result2 := ParseNavBlock(content)
	block2 := result2.Block
	found2 := result2.Found
	if !found2 {
		t.Fatal("expected nav block to be found")
	}
	if block2.Purpose != "helper utilities" {
		t.Errorf("purpose = %q, want %q", block2.Purpose, "helper utilities")
	}
	if len(block2.Nav) != 0 {
		t.Errorf("expected no nav entries, got %d", len(block2.Nav))
	}
	if len(block2.See) != 0 {
		t.Errorf("expected no see entries, got %d", len(block2.See))
	}
}

func TestParseNavBlock_NotFound(t *testing.T) {
	content := "# No nav block here\n"
	result := ParseNavBlock(content)
	if result.Found {
		t.Error("expected nav block not found")
	}
}

func TestParseNavBlock_EmptyAbout(t *testing.T) {
	content := `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,10,#Heading,
-->
`

	result3 := ParseNavBlock(content)
	if !result3.Found {
		t.Fatal("expected nav block to be found")
	}
	if result3.Block.Nav[0].About != "" {
		t.Errorf("about = %q, want empty", result3.Block.Nav[0].About)
	}
}

func TestRenderNavBlock(t *testing.T) {
	block := NavBlock{
		Purpose: "test purpose",
		Nav: []NavEntry{
			{Start: 5, N: 41, Name: "#Authentication", About: "token lifecycle"},
			{Start: 8, N: 13, Name: "##Token Exchange", About: "OAuth2 flow"},
		},
		See: []SeeEntry{
			{Path: "src/config.py", Why: "default timeout values"},
		},
	}

	got := RenderNavBlock(block)
	want := `<!-- AGENT:NAV
purpose:test purpose
nav[2]{s,n,name,about}:
5,41,#Authentication,token lifecycle
8,13,##Token Exchange,OAuth2 flow
see[1]{path,why}:
src/config.py,default timeout values
-->`

	if got != want {
		t.Errorf("RenderNavBlock() =\n%s\n\nwant:\n%s", got, want)
	}
}

func TestRenderPurposeOnly(t *testing.T) {
	got := RenderPurposeOnly("helper utilities")
	want := "<!-- AGENT:NAV\npurpose:helper utilities\n-->"
	if got != want {
		t.Errorf("RenderPurposeOnly() = %q, want %q", got, want)
	}
}

func TestParseNavBlock_RoundTrip(t *testing.T) {
	original := NavBlock{
		Purpose: "round trip test",
		Nav: []NavEntry{
			{Start: 10, N: 41, Name: "#Section", About: "content description"},
			{Start: 12, N: 19, Name: "##Subsection", About: ""},
		},
		See: []SeeEntry{
			{Path: "other.md", Why: "related info"},
		},
	}

	rendered := RenderNavBlock(original)
	pr := ParseNavBlock(rendered)
	parsed := pr.Block
	if !pr.Found {
		t.Fatal("expected nav block to be found after render")
	}
	if parsed.Purpose != original.Purpose {
		t.Errorf("purpose = %q, want %q", parsed.Purpose, original.Purpose)
	}
	if len(parsed.Nav) != len(original.Nav) {
		t.Fatalf("nav count = %d, want %d", len(parsed.Nav), len(original.Nav))
	}
	for i := range original.Nav {
		if !reflect.DeepEqual(parsed.Nav[i], original.Nav[i]) {
			t.Errorf("nav[%d] = %+v, want %+v", i, parsed.Nav[i], original.Nav[i])
		}
	}
	if len(parsed.See) != len(original.See) {
		t.Fatalf("see count = %d, want %d", len(parsed.See), len(original.See))
	}
	if !reflect.DeepEqual(parsed.See[0], original.See[0]) {
		t.Errorf("see[0] = %+v, want %+v", parsed.See[0], original.See[0])
	}
}

func TestNavBlockRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		block NavBlock
	}{
		{
			name: "full block with h1 h2 h3 and see",
			block: NavBlock{
				Purpose: "authentication module; handles token lifecycle",
				Nav: []NavEntry{
					{Start: 10, N: 41, Name: "#Authentication", About: "token lifecycle management"},
					{Start: 12, N: 19, Name: "##Token Exchange", About: "OAuth2 code-for-token flow"},
					{Start: 36, N: 15, Name: "###Token Validation", About: "JWT signature and expiry checks"},
				},
				See: []SeeEntry{
					{Path: "src/config.py", Why: "default timeout values"},
					{Path: "docs/oauth2.md", Why: "protocol specification"},
				},
			},
		},
		{
			name: "purpose only",
			block: NavBlock{
				Purpose: "helper utilities",
			},
		},
		{
			name: "nav only no see",
			block: NavBlock{
				Purpose: "parser module",
				Nav: []NavEntry{
					{Start: 1, N: 100, Name: "#Parser", About: "markdown heading parser"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := RenderNavBlock(tt.block)
			pr2 := ParseNavBlock(first)
			if !pr2.Found {
				t.Fatal("expected nav block to be found after first render")
			}
			second := RenderNavBlock(pr2.Block)
			if first != second {
				t.Errorf("round-trip render mismatch:\nfirst:\n%s\n\nsecond:\n%s", first, second)
			}
		})
	}
}

func TestNavBlockRoundTrip_EmptyAbout(t *testing.T) {
	block := NavBlock{
		Purpose: "test empty about fields",
		Nav: []NavEntry{
			{Start: 1, N: 10, Name: "#Heading", About: ""},
			{Start: 2, N: 5, Name: "##Sub", About: ""},
			{Start: 3, N: 3, Name: "###Deep", About: ""},
		},
	}

	first := RenderNavBlock(block)
	pr3 := ParseNavBlock(first)
	if !pr3.Found {
		t.Fatal("expected nav block to be found")
	}
	second := RenderNavBlock(pr3.Block)
	if first != second {
		t.Errorf("round-trip with empty about mismatch:\nfirst:\n%s\n\nsecond:\n%s", first, second)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty string", input: "", want: 0},
		{name: "whitespace only", input: "  ", want: 0},
		{name: "single word", input: "hello", want: 1},
		{name: "two words", input: "hello world", want: 2},
		{name: "heading prefix stripped", input: "# My Heading", want: 2},
		{name: "multiline string", input: "# My Heading\nsome content here", want: 5},
		{name: "blank lines ignored", input: "hello\n\nworld", want: 2},
		{name: "extra whitespace", input: "  foo   bar  ", want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountWords(tt.input)
			if got != tt.want {
				t.Errorf("CountWords(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNavEntry_WordCountNotSerialized(t *testing.T) {
	entry := NavEntry{Start: 10, N: 5, Name: "#Section", About: "desc", WordCount: 42}
	block := NavBlock{
		Purpose: "test",
		Nav:     []NavEntry{entry},
	}
	rendered := RenderNavBlock(block)
	pr := ParseNavBlock(rendered)
	if !pr.Found {
		t.Fatal("expected nav block to be found after render")
	}
	if len(pr.Block.Nav) != 1 {
		t.Fatalf("expected 1 nav entry, got %d", len(pr.Block.Nav))
	}
	if pr.Block.Nav[0].WordCount != 0 {
		t.Errorf("WordCount after round-trip = %d, want 0 (must not be serialized)", pr.Block.Nav[0].WordCount)
	}
}

func TestSectionWordCount(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		start int // 1-indexed
		n     int // line count
		want  int
	}{
		{
			name:  "zero lines (n=0)",
			lines: []string{"# Heading", "some content"},
			start: 1,
			n:     0,
			want:  0,
		},
		{
			name:  "heading only (n=1)",
			lines: []string{"# Heading", "next line"},
			start: 1,
			n:     1,
			want:  0,
		},
		{
			name:  "single content line",
			lines: []string{"# Heading", "hello world"},
			start: 1,
			n:     2,
			want:  2,
		},
		{
			name:  "multi-line section",
			lines: []string{"# Heading", "one two", "three four five", "six"},
			start: 1,
			n:     4,
			want:  6,
		},
		{
			name:  "start beyond slice (out-of-bounds safety)",
			lines: []string{"# Heading"},
			start: 5,
			n:     3,
			want:  0,
		},
		{
			name:  "n exceeds available lines (clamps to end)",
			lines: []string{"# Heading", "word1 word2"},
			start: 1,
			n:     100,
			want:  2,
		},
		{
			name:  "empty lines slice",
			lines: []string{},
			start: 1,
			n:     5,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SectionWordCount(tt.lines, tt.start, tt.n)
			if got != tt.want {
				t.Errorf("SectionWordCount(lines, %d, %d) = %d, want %d", tt.start, tt.n, got, tt.want)
			}
		})
	}
}

func TestIsAutoGenerated(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty string", input: "", want: false},
		{name: "tilde prefix", input: "~token refresh", want: true},
		{name: "no prefix", input: "token refresh", want: false},
		{name: "double tilde", input: "~~double", want: true},
		{name: "tilde only", input: "~", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAutoGenerated(tt.input)
			if got != tt.want {
				t.Errorf("IsAutoGenerated(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimAutoGenerated(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "tilde prefix", input: "~token refresh", want: "token refresh"},
		{name: "no prefix", input: "token refresh", want: "token refresh"},
		{name: "double tilde", input: "~~double", want: "~double"},
		{name: "tilde only", input: "~", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimAutoGenerated(tt.input)
			if got != tt.want {
				t.Errorf("TrimAutoGenerated(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderFilesBlock(t *testing.T) {
	tests := []struct {
		name  string
		block FilesBlock
		want  string
	}{
		{
			name: "root-level files only",
			block: FilesBlock{
				Purpose: "project file index for myrepo",
				Entries: []FilesEntry{
					{RelPath: "README.md", About: "project overview and quickstart"},
					{RelPath: "AGENTS.md", About: "agent workflow instructions"},
				},
			},
			want: "<!-- AGENT:NAV\npurpose:project file index for myrepo\nfiles[2]{path,about}:\nREADME.md,project overview and quickstart\nAGENTS.md,agent workflow instructions\n-->",
		},
		{
			name: "files grouped by directory",
			block: FilesBlock{
				Purpose: "project file index for myrepo",
				Entries: []FilesEntry{
					{RelPath: "README.md", About: "project overview"},
					{RelPath: "docs/authentication.md", About: "token lifecycle; OAuth2 exchange"},
					{RelPath: "docs/api/endpoints.md", About: "REST endpoint catalog"},
				},
			},
			want: "<!-- AGENT:NAV\npurpose:project file index for myrepo\nfiles[3]{path,about}:\nREADME.md,project overview\ndocs/\nauthentication.md,token lifecycle; OAuth2 exchange\ndocs/api/\nendpoints.md,REST endpoint catalog\n-->",
		},
		{
			name: "N count is total file entries not directory prefix lines",
			block: FilesBlock{
				Purpose: "project file index",
				Entries: []FilesEntry{
					{RelPath: "docs/a.md", About: "doc a"},
					{RelPath: "docs/b.md", About: "doc b"},
				},
			},
			want: "<!-- AGENT:NAV\npurpose:project file index\nfiles[2]{path,about}:\ndocs/\na.md,doc a\nb.md,doc b\n-->",
		},
		{
			name:  "empty entries",
			block: FilesBlock{Purpose: "empty project"},
			want:  "<!-- AGENT:NAV\npurpose:empty project\nfiles[0]{path,about}:\n-->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderFilesBlock(tt.block)
			if got != tt.want {
				t.Errorf("RenderFilesBlock() =\n%s\n\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestParseFilesBlock(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantFound bool
		wantBlock FilesBlock
	}{
		{
			name: "root-level files only",
			content: `<!-- AGENT:NAV
purpose:project file index for myrepo
files[2]{path,about}:
README.md,project overview and quickstart
AGENTS.md,agent workflow instructions
-->`,
			wantFound: true,
			wantBlock: FilesBlock{
				Purpose: "project file index for myrepo",
				Entries: []FilesEntry{
					{RelPath: "README.md", About: "project overview and quickstart"},
					{RelPath: "AGENTS.md", About: "agent workflow instructions"},
				},
			},
		},
		{
			name: "directory prefix lines",
			content: `<!-- AGENT:NAV
purpose:project file index
files[3]{path,about}:
README.md,project overview
docs/
authentication.md,token lifecycle; OAuth2 exchange
docs/api/
endpoints.md,REST endpoint catalog
-->`,
			wantFound: true,
			wantBlock: FilesBlock{
				Purpose: "project file index",
				Entries: []FilesEntry{
					{RelPath: "README.md", About: "project overview"},
					{RelPath: "docs/authentication.md", About: "token lifecycle; OAuth2 exchange"},
					{RelPath: "docs/api/endpoints.md", About: "REST endpoint catalog"},
				},
			},
		},
		{
			name:      "no files block",
			content:   "# Not a nav block\n",
			wantFound: false,
		},
		{
			name: "empty entries",
			content: `<!-- AGENT:NAV
purpose:empty project
files[0]{path,about}:
-->`,
			wantFound: true,
			wantBlock: FilesBlock{
				Purpose: "empty project",
				Entries: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := ParseFilesBlock(tt.content)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
				return
			}
			if !found {
				return
			}
			if got.Purpose != tt.wantBlock.Purpose {
				t.Errorf("Purpose = %q, want %q", got.Purpose, tt.wantBlock.Purpose)
			}
			if len(got.Entries) != len(tt.wantBlock.Entries) {
				t.Fatalf("len(Entries) = %d, want %d\ngot: %+v\nwant: %+v",
					len(got.Entries), len(tt.wantBlock.Entries), got.Entries, tt.wantBlock.Entries)
			}
			for i := range tt.wantBlock.Entries {
				if got.Entries[i] != tt.wantBlock.Entries[i] {
					t.Errorf("Entries[%d] = %+v, want %+v", i, got.Entries[i], tt.wantBlock.Entries[i])
				}
			}
		})
	}
}

func TestRenderFilesBlock_RoundTrip(t *testing.T) {
	block := FilesBlock{
		Purpose: "project file index for myrepo",
		Entries: []FilesEntry{
			{RelPath: "README.md", About: "project overview"},
			{RelPath: "docs/authentication.md", About: "token lifecycle; OAuth2 exchange"},
			{RelPath: "docs/api/endpoints.md", About: "REST endpoint catalog"},
		},
	}
	rendered := RenderFilesBlock(block)
	got, found := ParseFilesBlock(rendered)
	if !found {
		t.Fatal("ParseFilesBlock did not find block after render")
	}
	if got.Purpose != block.Purpose {
		t.Errorf("Purpose = %q, want %q", got.Purpose, block.Purpose)
	}
	if len(got.Entries) != len(block.Entries) {
		t.Fatalf("len(Entries) = %d, want %d", len(got.Entries), len(block.Entries))
	}
	for i := range block.Entries {
		if got.Entries[i] != block.Entries[i] {
			t.Errorf("Entries[%d] = %+v, want %+v", i, got.Entries[i], block.Entries[i])
		}
	}
}

func TestParseNavBlock_InvalidLineCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantN   []int
	}{
		{
			name: "n is zero",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,0,#Heading,desc
-->
`,
			wantN: []int{0},
		},
		{
			name: "n is negative",
			content: `<!-- AGENT:NAV
purpose:test
nav[1]{s,n,name,about}:
1,-5,#Heading,desc
-->
`,
			wantN: []int{0},
		},
		{
			name: "mixed valid and invalid",
			content: `<!-- AGENT:NAV
purpose:test
nav[3]{s,n,name,about}:
1,10,#Valid,desc
2,0,#Zero,desc
3,-1,#Negative,desc
-->
`,
			wantN: []int{10, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			pr4 := ParseNavBlock(tt.content)
			block := pr4.Block
			found := pr4.Found
			if err := w.Close(); err != nil {
				t.Fatalf("pipe close: %v", err)
			}
			os.Stderr = old

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("pipe copy: %v", err)
			}
			stderr := buf.String()

			if !found {
				t.Fatal("expected nav block to be found")
			}
			if len(block.Nav) != len(tt.wantN) {
				t.Fatalf("nav count = %d, want %d", len(block.Nav), len(tt.wantN))
			}
			for i, want := range tt.wantN {
				if block.Nav[i].N != want {
					t.Errorf("nav[%d].N = %d, want %d", i, block.Nav[i].N, want)
				}
			}
			if !strings.Contains(stderr, "warning") {
				t.Errorf("expected warning on stderr, got: %q", stderr)
			}
		})
	}
}
