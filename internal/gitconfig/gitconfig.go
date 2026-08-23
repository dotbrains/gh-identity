// Package gitconfig manages per-profile gitconfig fragments and
// includeIf directives in the user's global ~/.gitconfig.
package gitconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smeltery/gh-identity/internal/config"
)

const (
	// marker is used to identify lines managed by gh-identity.
	marker = "# managed by gh-identity"
)

// WriteProfileFragment writes a gitconfig fragment for the given profile.
// e.g. ~/.config/gh-identity/git/work.gitconfig
func WriteProfileFragment(profileName string, p config.Profile) error {
	dir, err := config.EnsureGitConfigDir()
	if err != nil {
		return err
	}
	return WriteProfileFragmentTo(filepath.Join(dir, profileName+".gitconfig"), p)
}

// WriteProfileFragmentTo writes a profile gitconfig fragment to a specific path.
func WriteProfileFragmentTo(path string, p config.Profile) error {
	content := fmt.Sprintf("[user]\n    name = %s\n    email = %s\n", p.GitName, p.GitEmail)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing gitconfig fragment: %w", err)
	}
	return nil
}

// RemoveProfileFragment deletes the gitconfig fragment for a profile.
func RemoveProfileFragment(profileName string) error {
	dir, err := config.GitConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, profileName+".gitconfig")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing gitconfig fragment: %w", err)
	}
	return nil
}

// AddIncludeIf adds an includeIf directive to the global gitconfig.
// gitconfigPath is the path to ~/.gitconfig (or equivalent).
// dirPath is the bound directory, fragmentPath is the profile gitconfig fragment.
func AddIncludeIf(gitconfigPath, dirPath, fragmentPath string) error {
	// Ensure dirPath ends with / for gitdir matching.
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	directive := fmt.Sprintf("[includeIf \"gitdir:%s\"]", dirPath)
	pathLine := fmt.Sprintf("    path = %s", fragmentPath)

	lines, err := readLines(gitconfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if the directive already exists.
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match with or without the marker suffix.
		bare := strings.TrimSuffix(trimmed, " "+marker)
		if bare == directive {
			// Update the path line if it's the next line.
			if i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "path = ") {
				lines[i+1] = pathLine
				return writeLines(gitconfigPath, lines)
			}
		}
	}

	// Append new directive.
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		lines = append(lines, "")
	}
	lines = append(lines, directive+" "+marker)
	lines = append(lines, pathLine)

	if err := writeLines(gitconfigPath, lines); err != nil {
		return err
	}
	return SyncIncludeIfs(gitconfigPath)
}

// RemoveIncludeIf removes an includeIf directive for the given directory from the global gitconfig.
func RemoveIncludeIf(gitconfigPath, dirPath string) error {
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	directive := fmt.Sprintf("[includeIf \"gitdir:%s\"]", dirPath)

	lines, err := readLines(gitconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var result []string
	skip := false
	for _, line := range lines {
		if strings.TrimSpace(strings.TrimSuffix(line, " "+marker)) == directive ||
			strings.TrimSpace(line) == directive {
			skip = true
			continue
		}
		if skip {
			// Skip the path = line that follows the directive.
			if strings.HasPrefix(strings.TrimSpace(line), "path = ") {
				skip = false
				continue
			}
			skip = false
		}
		result = append(result, line)
	}

	// Remove trailing blank lines.
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	return writeLines(gitconfigPath, result)
}

// ListManagedIncludeIfs returns all includeIf dirPaths managed by gh-identity.
func ListManagedIncludeIfs(gitconfigPath string) ([]string, error) {
	lines, err := readLines(gitconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dirs []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, marker) {
			// Extract dirPath from [includeIf "gitdir:<path>"]
			start := strings.Index(trimmed, "gitdir:")
			if start == -1 {
				continue
			}
			end := strings.Index(trimmed[start:], "\"]")
			if end == -1 {
				continue
			}
			dirs = append(dirs, trimmed[start+7:start+end])
		}
	}
	return dirs, nil
}

// SyncIncludeIfs ensures all managed includeIf blocks are at the very end of
// the gitconfig file. This prevents other tools (e.g. gh auth login) from
// appending [user] sections that override the identity set by includeIf.
func SyncIncludeIfs(gitconfigPath string) error {
	lines, err := readLines(gitconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type block struct {
		directive string
		pathLine  string
	}

	var managed []block
	var rest []string

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.Contains(trimmed, marker) && strings.Contains(trimmed, "includeIf") {
			b := block{directive: lines[i]}
			if i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "path = ") {
				b.pathLine = lines[i+1]
				i++ // skip path line
			}
			managed = append(managed, b)
		} else {
			rest = append(rest, lines[i])
		}
	}

	if len(managed) == 0 {
		return nil // nothing to sync
	}

	// Remove trailing blank lines from rest.
	for len(rest) > 0 && rest[len(rest)-1] == "" {
		rest = rest[:len(rest)-1]
	}

	// Re-append managed blocks at the end.
	for _, b := range managed {
		rest = append(rest, "")
		rest = append(rest, b.directive)
		if b.pathLine != "" {
			rest = append(rest, b.pathLine)
		}
	}

	return writeLines(gitconfigPath, rest)
}

// HasUserAfterIncludeIfs reports whether any [user] name or email setting
// appears after the last managed includeIf block, which would override it.
func HasUserAfterIncludeIfs(gitconfigPath string) (bool, error) {
	lines, err := readLines(gitconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	lastManagedIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, marker) && strings.Contains(trimmed, "includeIf") {
			lastManagedIdx = i
		}
	}

	if lastManagedIdx == -1 {
		return false, nil // no managed blocks
	}

	inUserSection := false
	for i := lastManagedIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "[user]" {
			inUserSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inUserSection = false
			continue
		}
		if inUserSection {
			if strings.HasPrefix(trimmed, "name = ") || strings.HasPrefix(trimmed, "email = ") {
				return true, nil
			}
		}
	}

	return false, nil
}

// GlobalGitconfigPath returns the path to the user's global gitconfig.
func GlobalGitconfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".gitconfig"), nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLines(path string, lines []string) error {
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
