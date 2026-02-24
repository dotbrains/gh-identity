// Package resolve implements binding resolution: given a working directory,
// find the deepest matching binding and return the associated profile name.
package resolve

import (
	"path/filepath"
	"strings"

	"github.com/dotbrains/gh-identity/internal/config"
)

// Result holds the outcome of a binding resolution.
type Result struct {
	Profile   string // profile name, or "" if no match
	BoundPath string // the binding path that matched, or ""
	IsDefault bool   // true if the default profile was used (no binding match)
}

// ForDirectory resolves the active profile for the given directory.
// It walks up from dir to /, finding the deepest binding match.
// If no binding matches, it falls back to the default profile.
func ForDirectory(dir string, bindings *config.BindingsFile, defaultProfile string) (Result, error) {
	expanded, err := config.ExpandPath(dir)
	if err != nil {
		return Result{}, err
	}

	var bestMatch string
	var bestPath string
	bestDepth := -1

	for _, b := range bindings.Bindings {
		bPath, err := config.ExpandPath(b.Path)
		if err != nil {
			continue
		}

		if isSubpath(expanded, bPath) {
			depth := strings.Count(bPath, string(filepath.Separator))
			if depth > bestDepth {
				bestDepth = depth
				bestMatch = b.Profile
				bestPath = b.Path
			}
		}
	}

	if bestMatch != "" {
		return Result{
			Profile:   bestMatch,
			BoundPath: bestPath,
		}, nil
	}

	return Result{
		Profile:   defaultProfile,
		IsDefault: defaultProfile != "",
	}, nil
}

// isSubpath reports whether child is equal to or a subdirectory of parent.
func isSubpath(child, parent string) bool {
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)

	if child == parent {
		return true
	}

	// Ensure parent ends with separator for prefix check.
	parentPrefix := parent + string(filepath.Separator)
	return strings.HasPrefix(child, parentPrefix)
}
