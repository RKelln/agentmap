// Package guide provides the embedded nav writing guide.
package guide

import _ "embed"

// Content is the full text of the nav writing guide, embedded at build time.
//
//go:embed nav-writing-guide.md
var Content string
