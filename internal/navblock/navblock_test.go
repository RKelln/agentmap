package navblock

import (
	"reflect"
	"testing"
)

func TestParseNavBlock_Full(t *testing.T) {
	content := `<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange
nav[3]{s,e,name,about}:
12,65,#Authentication,token lifecycle management
14,35,##Token Exchange,OAuth2 code-for-token flow
36,50,##Token Refresh,silent rotation and expiry
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
	want := NavEntry{Start: 12, End: 65, Name: "#Authentication", About: "token lifecycle management"}
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
nav[1]{s,e,name,about}:
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
			{Start: 5, End: 45, Name: "#Authentication", About: "token lifecycle"},
			{Start: 8, End: 20, Name: "##Token Exchange", About: "OAuth2 flow"},
		},
		See: []SeeEntry{
			{Path: "src/config.py", Why: "default timeout values"},
		},
	}

	got := RenderNavBlock(block)
	want := `<!-- AGENT:NAV
purpose:test purpose
nav[2]{s,e,name,about}:
5,45,#Authentication,token lifecycle
8,20,##Token Exchange,OAuth2 flow
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
			{Start: 10, End: 50, Name: "#Section", About: "content description"},
			{Start: 12, End: 30, Name: "##Subsection", About: ""},
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
