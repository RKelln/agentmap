package navblock

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

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

	block, start, end, found := ParseNavBlock(content)
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

	block, _, _, found := ParseNavBlock(content)
	if !found {
		t.Fatal("expected nav block to be found")
	}
	if block.Purpose != "helper utilities" {
		t.Errorf("purpose = %q, want %q", block.Purpose, "helper utilities")
	}
	if len(block.Nav) != 0 {
		t.Errorf("expected no nav entries, got %d", len(block.Nav))
	}
	if len(block.See) != 0 {
		t.Errorf("expected no see entries, got %d", len(block.See))
	}
}

func TestParseNavBlock_NotFound(t *testing.T) {
	content := "# No nav block here\n"
	_, _, _, found := ParseNavBlock(content)
	if found {
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

	block, _, _, found := ParseNavBlock(content)
	if !found {
		t.Fatal("expected nav block to be found")
	}
	if block.Nav[0].About != "" {
		t.Errorf("about = %q, want empty", block.Nav[0].About)
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
	parsed, _, _, found := ParseNavBlock(rendered)
	if !found {
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
			parsed, _, _, found := ParseNavBlock(first)
			if !found {
				t.Fatal("expected nav block to be found after first render")
			}
			second := RenderNavBlock(parsed)
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
	parsed, _, _, found := ParseNavBlock(first)
	if !found {
		t.Fatal("expected nav block to be found")
	}
	second := RenderNavBlock(parsed)
	if first != second {
		t.Errorf("round-trip with empty about mismatch:\nfirst:\n%s\n\nsecond:\n%s", first, second)
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

			block, _, _, found := ParseNavBlock(tt.content)
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
