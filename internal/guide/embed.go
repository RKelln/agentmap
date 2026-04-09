// Package guide provides the embedded nav writing guide.
package guide

import (
	_ "embed"
	"strings"
)

// Content is the full text of the nav writing guide, embedded at build time.
//
//go:embed nav-writing-guide.md
var Content string

// RulesContent is the writing-rules portion of the guide only (sections 3–7).
// Sections 1 and 2 (workflow/scratch guidance) are omitted because they are
// redundant when the task list already provides a concrete workflow up front.
var RulesContent string

func init() {
	const marker = "<!-- rules-start -->\n"
	if idx := strings.Index(Content, marker); idx >= 0 {
		RulesContent = Content[idx+len(marker):]
	} else {
		// Fallback: use full content if marker is ever removed.
		RulesContent = Content
	}
}
