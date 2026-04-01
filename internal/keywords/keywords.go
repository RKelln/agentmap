// Package keywords implements Tier 1 keyword extraction using term frequency.
package keywords

import (
	"sort"
	"strings"
	"unicode"
)

// Stopwords are English function words and common markdown/doc terms
// that carry no distinctive meaning for section descriptions.
var stopwords = map[string]bool{
	// English stopwords
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"if": true, "then": true, "else": true, "when": true, "at": true, "by": true,
	"for": true, "with": true, "about": true, "against": true, "between": true,
	"into": true, "through": true, "during": true, "before": true, "after": true,
	"above": true, "below": true, "to": true, "from": true, "up": true, "down": true,
	"in": true, "out": true, "on": true, "off": true, "over": true, "under": true,
	"again": true, "further": true, "once": true, "here": true, "there": true,
	"all": true, "each": true, "few": true, "more": true, "most": true, "other": true,
	"some": true, "such": true, "no": true, "nor": true, "not": true, "only": true,
	"own": true, "same": true, "so": true, "than": true, "too": true, "very": true,
	"can": true, "will": true, "just": true, "don": true, "should": true, "now": true,
	"is": true, "it": true, "its": true, "was": true, "are": true, "were": true,
	"been": true, "being": true, "have": true, "has": true, "had": true, "having": true,
	"do": true, "does": true, "did": true, "doing": true, "would": true, "could": true,
	"might": true, "must": true, "shall": true, "this": true, "that": true, "these": true,
	"those": true, "i": true, "me": true, "my": true, "myself": true, "we": true,
	"our": true, "ours": true, "ourselves": true, "you": true, "your": true, "yours": true,
	"yourself": true, "yourselves": true, "he": true, "him": true, "his": true,
	"himself": true, "she": true, "her": true, "hers": true, "herself": true,
	"they": true, "them": true, "their": true, "theirs": true, "themselves": true,
	"what": true, "which": true, "who": true, "whom": true, "how": true, "why": true,
	"as": true, "of": true, "be": true, "am": true,
	// Common markdown/doc words
	"section": true, "following": true, "example": true, "note": true, "see": true,
	"also": true, "describes": true, "provides": true, "contains": true, "overview": true,
	"using": true, "used": true, "use": true, "set": true, "get": true, "make": true,
	"like": true, "one": true, "two": true, "first": true, "second": true, "third": true,
	"may": true, "need": true, "needs": true, "way": true, "ways": true, "many": true,
	"much": true, "well": true, "back": true, "even": true, "still": true, "already": true,
	"including": true, "include": true, "includes": true, "based": true, "within": true,
	"without": true, "however": true, "whether": true, "while": true, "where": true,
	"both": true, "any": true, "every": true, "either": true, "neither": true,
	"although": true, "because": true, "unless": true, "until": true, "since": true,
	"therefore": true, "thus": true, "hence": true, "yet": true,
	// Markdown syntax artifacts
	"true": true, "false": true, "null": true, "yes": true,
}

// ExtractKeywords extracts the top distinctive terms from text using term frequency.
// Returns keywords joined by semicolons (no commas, per nav block format).
// Tokens shorter than 3 characters and stopwords are excluded.
func ExtractKeywords(text string, maxKeywords int) string {
	tokens := tokenize(text)
	freq := make(map[string]int)
	for _, t := range tokens {
		if len(t) < 3 {
			continue
		}
		if stopwords[t] {
			continue
		}
		freq[t]++
	}

	if len(freq) == 0 {
		return ""
	}

	type termScore struct {
		term  string
		count int
	}
	scored := make([]termScore, 0, len(freq))
	for term, count := range freq {
		scored = append(scored, termScore{term, count})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].count != scored[j].count {
			return scored[i].count > scored[j].count
		}
		return scored[i].term < scored[j].term
	})

	if maxKeywords > len(scored) {
		maxKeywords = len(scored)
	}

	result := make([]string, maxKeywords)
	for i := 0; i < maxKeywords; i++ {
		result[i] = scored[i].term
	}

	return strings.Join(result, ";")
}

// ExtractPurpose extracts keywords from the entire file content for the purpose line.
// Uses a higher keyword count (6-8 terms) since it summarizes the whole file.
func ExtractPurpose(text string) string {
	return ExtractKeywords(text, 8)
}

// tokenize splits text into lowercase word tokens, stripping punctuation.
func tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(unicode.ToLower(r))
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
