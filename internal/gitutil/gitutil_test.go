package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDiffOutput_HunkParsing(t *testing.T) {
	// Direct test of the hunk parser with sample git diff -U0 output.
	tests := []struct {
		name       string
		input      string
		path       string
		wantRanges []LineRange
	}{
		{
			name: "single hunk single line",
			input: `diff --git a/foo.md b/foo.md
index abc..def 100644
--- a/foo.md
+++ b/foo.md
@@ -5 +5 @@ some context
+new line
`,
			path:       "foo.md",
			wantRanges: []LineRange{{Start: 5, End: 5}},
		},
		{
			name: "single hunk multiple lines",
			input: `diff --git a/docs/auth.md b/docs/auth.md
--- a/docs/auth.md
+++ b/docs/auth.md
@@ -10,3 +10,5 @@ context
+line1
+line2
+line3
+line4
+line5
`,
			path:       "docs/auth.md",
			wantRanges: []LineRange{{Start: 10, End: 14}},
		},
		{
			name: "multiple hunks",
			input: `diff --git a/doc.md b/doc.md
--- a/doc.md
+++ b/doc.md
@@ -1,2 +1,3 @@
+added
 line1
 line2
@@ -20,0 +21,2 @@
+new1
+new2
`,
			path:       "doc.md",
			wantRanges: []LineRange{{Start: 1, End: 3}, {Start: 21, End: 22}},
		},
		{
			name: "pure deletion (count=0 means no added lines)",
			input: `diff --git a/doc.md b/doc.md
--- a/doc.md
+++ b/doc.md
@@ -5,3 +5,0 @@
-line1
-line2
-line3
`,
			path:       "doc.md",
			wantRanges: nil, // count=0, skip
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRepoDiff(tt.input)
			got := result[tt.path]
			if len(got) != len(tt.wantRanges) {
				t.Errorf("len(ranges) = %d, want %d; got %v, want %v",
					len(got), len(tt.wantRanges), got, tt.wantRanges)
				return
			}
			for i, want := range tt.wantRanges {
				if got[i] != want {
					t.Errorf("range[%d] = %v, want %v", i, got[i], want)
				}
			}
		})
	}
}

func TestRepoChanges_TempRepo(t *testing.T) {
	// Create a temp git repo, commit a file, make changes, verify RepoChanges works.
	dir := t.TempDir()

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cmd %s %v: %v\n%s", name, args, err, out)
		}
	}

	// Init repo
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")

	// Create and commit initial file
	initialContent := strings.Repeat("line\n", 20)
	if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	// Modify the file
	modContent := strings.Repeat("line\n", 10) + "CHANGED\n" + strings.Repeat("line\n", 9)
	if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte(modContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// RepoChanges should work by passing dir directly — no os.Chdir needed.
	changes, err := RepoChanges(dir)
	if err != nil {
		t.Fatalf("RepoChanges() error = %v", err)
	}

	if changes == nil {
		t.Fatal("RepoChanges() returned nil map")
	}

	// Should contain doc.md with at least one range
	ranges, ok := changes["doc.md"]
	if !ok {
		t.Errorf("doc.md not in changes map; got keys: %v", mapKeys(changes))
		return
	}
	if len(ranges) == 0 {
		t.Error("doc.md ranges should not be empty")
	}
	// Line 11 should be changed
	found := false
	for _, r := range ranges {
		if r.Start <= 11 && r.End >= 11 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected line 11 to be in changed ranges, got %v", ranges)
	}
}

func mapKeys(m map[string][]LineRange) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
