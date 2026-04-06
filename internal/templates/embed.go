// Package templates provides embedded agent skill template files.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

// FS is the embedded filesystem containing all .tmpl template files.
//
//go:embed *.tmpl
var FS embed.FS

const bodyPlaceholder = "{{AGENTMAP_BODY}}"

// Get returns the contents of the named template file.
// Name must be one of the .tmpl filenames (e.g. "agents.md.tmpl").
func Get(name string) ([]byte, error) {
	b, err := FS.ReadFile(name)
	if err != nil {
		return nil, err
	}
	s := string(b)
	if strings.Contains(s, bodyPlaceholder) {
		body, err := Body()
		if err != nil {
			return nil, err
		}
		s = strings.ReplaceAll(s, bodyPlaceholder, body)
		return []byte(s), nil
	}
	return b, nil
}

// Body returns the shared agent instruction body that every tool template
// embeds between its <!-- agentmap:init --> markers. It is read from
// _body.md.tmpl, which is the single source of truth for that content.
func Body() (string, error) {
	b, err := FS.ReadFile("_body.md.tmpl")
	if err != nil {
		return "", fmt.Errorf("templates: read body: %w", err)
	}
	return string(b), nil
}

// AllNames returns the names of all embedded tool template files.
// Files whose names begin with "_" (e.g. _body.md.tmpl) are excluded because
// they are internal fragments, not standalone tool templates.
// Panics if the embedded filesystem is unreadable (indicates a corrupt binary).
func AllNames() []string {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		panic(fmt.Sprintf("templates: embedded FS unreadable: %v", err))
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), "_") {
			names = append(names, e.Name())
		}
	}

	return names
}
