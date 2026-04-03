// Package templates provides embedded agent skill template files.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
)

// FS is the embedded filesystem containing all .tmpl template files.
//
//go:embed *.tmpl
var FS embed.FS

// Get returns the contents of the named template file.
// Name must be one of the .tmpl filenames (e.g. "agents.md.tmpl").
func Get(name string) ([]byte, error) {
	return FS.ReadFile(name)
}

// AllNames returns the names of all embedded template files.
// Panics if the embedded filesystem is unreadable (indicates a corrupt binary).
func AllNames() []string {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		panic(fmt.Sprintf("templates: embedded FS unreadable: %v", err))
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}

	return names
}
