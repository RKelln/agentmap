// Package search implements fuzzy header search across agentmapped markdown files.
package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/RKelln/agentmap/internal/discovery"
	"github.com/RKelln/agentmap/internal/parser"
)

// Result is a single search match.
type Result struct {
	FilePath string
	Heading  string
	Depth    int
	Start    int
	End      int
	Score    float64
	Content  string
}

// Options control search behavior.
type Options struct {
	Query      string
	Threshold  float64
	MaxResults int
	MaxDepth   int
	Root       string
	Exclude    []string
	Paths      []string
}

// Search discovers indexed markdown files and returns fuzzy-matched heading sections.
// If Options.Paths is non-nil, those specific files are searched; otherwise all
// .md files under Options.Root are discovered.
func Search(opts Options) ([]Result, error) {
	var files []string
	if opts.Paths != nil {
		files = opts.Paths
	} else {
		var err error
		files, err = discovery.DiscoverFiles(opts.Root, opts.Exclude)
		if err != nil {
			return nil, fmt.Errorf("search: discover files: %w", err)
		}
	}

	var results []Result
	for _, relPath := range files {
		fullPath := filepath.Join(opts.Root, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)

		if !hasNavBlock(content) {
			continue
		}

		totalLines := lineCount(content)
		headings, parseWarnings := parser.ParseHeadings(content, opts.MaxDepth)
		for _, w := range parseWarnings {
			fmt.Fprintf(os.Stderr, "warning: %s: %s\n", relPath, w)
		}
		sections := parser.ComputeSections(headings, totalLines)

		for _, s := range sections {
			score := Score(opts.Query, s.Text)
			if score < opts.Threshold {
				continue
			}
			results = append(results, Result{
				FilePath: relPath,
				Heading:  s.Text,
				Depth:    s.Depth,
				Start:    s.Start,
				End:      s.End,
				Score:    score,
				Content:  extractContent(content, s),
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if opts.MaxResults > 0 && len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	return results, nil
}

// Score computes how well a query matches a heading.
// Uses fuzzy token matching: each query token is matched to its best heading token by edit ratio.
// Substring containment is only a bonus when the strings share at least one exact token.
//
// Pipe (|) acts as OR: "budget|timeline|schedule" scores each variant independently
// and returns the best match. Spaces around pipes are trimmed (both "a|b" and "a | b" work).
func Score(query, heading string) float64 {
	if strings.Contains(query, "|") {
		variants := strings.Split(query, "|")
		best := 0.0
		for _, v := range variants {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			s := scoreSingle(v, heading)
			if s > best {
				best = s
			}
		}
		return best
	}
	return scoreSingle(query, heading)
}

// scoreSingle scores a single (non-OR) query against a heading.
func scoreSingle(query, heading string) float64 {
	q := normalize(query)
	h := normalize(heading)

	if q == "" || h == "" {
		return 0
	}

	qTokens := tokenize(q)
	hTokens := tokenize(h)

	if len(qTokens) == 0 {
		return 0
	}

	if strings.Contains(h, q) || strings.Contains(q, h) {
		if sharesExactToken(qTokens, hTokens) {
			return 0.95
		}
	}

	tokenMatch := fuzzyTokenMatch(qTokens, hTokens)
	headingCoverage := fuzzyTokenMatch(hTokens, qTokens)

	if len(hTokens) == 0 {
		return tokenMatch
	}

	return 0.6*tokenMatch + 0.4*headingCoverage
}

// sharesExactToken reports whether a and b share at least one identical token.
func sharesExactToken(a, b []string) bool {
	bSet := make(map[string]struct{}, len(b))
	for _, t := range b {
		bSet[t] = struct{}{}
	}
	for _, t := range a {
		if _, ok := bSet[t]; ok {
			return true
		}
	}
	return false
}

// fuzzyTokenMatch returns average best edit-ratio for each source token against target tokens.
func fuzzyTokenMatch(source, target []string) float64 {
	if len(source) == 0 {
		return 0
	}
	total := 0.0
	for _, st := range source {
		best := 0.0
		for _, tt := range target {
			r := editRatio(st, tt)
			if r > best {
				best = r
			}
		}
		total += best
	}
	return total / float64(len(source))
}

func normalize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

func tokenize(s string) []string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var tokens []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func editRatio(a, b string) float64 {
	dist := levenshteinDist(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

func levenshteinDist(a, b string) int {
	la, lb := len(a), len(b)
	if la < lb {
		a, b = b, a
		la, lb = lb, la
	}

	prev := make([]int, lb+1)
	cur := make([]int, lb+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			mn := prev[j] + 1
			if cur[j-1]+1 < mn {
				mn = cur[j-1] + 1
			}
			if prev[j-1]+cost < mn {
				mn = prev[j-1] + cost
			}
			cur[j] = mn
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func hasNavBlock(content string) bool {
	return strings.Contains(content, "<!-- AGENT:NAV")
}

func lineCount(content string) int {
	return strings.Count(content, "\n")
}

func extractContent(content string, section parser.Section) string {
	lines := strings.Split(content, "\n")
	if section.End > len(lines) {
		section.End = len(lines)
	}
	if section.Start < 1 || section.Start > len(lines) {
		return ""
	}
	return strings.Join(lines[section.Start-1:section.End], "\n")
}
