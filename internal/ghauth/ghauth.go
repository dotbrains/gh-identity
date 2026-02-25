// Package ghauth provides an interface for interacting with gh's authentication
// subsystem (listing authenticated accounts, retrieving tokens).
package ghauth

import (
	"bytes"
	"fmt"
	"strings"

	gh "github.com/cli/go-gh/v2"
)

// Auth is the interface for gh authentication operations.
// Use the interface for testability; the default implementation shells out to gh.
type Auth interface {
	// Token returns the auth token for the given username.
	Token(username string) (string, error)
	// AuthenticatedUsers returns a list of authenticated gh usernames.
	AuthenticatedUsers() ([]string, error)
	// ActiveUser returns the currently active gh user.
	ActiveUser() (string, error)
}

// execFn is the function signature for executing gh commands.
type execFn func(args ...string) (bytes.Buffer, bytes.Buffer, error)

// GHAuth is the default implementation using the gh CLI.
type GHAuth struct {
	exec execFn
}

// NewGHAuth returns a new default Auth implementation.
func NewGHAuth() *GHAuth {
	return &GHAuth{exec: ghExec}
}

// ghExec wraps gh.Exec.
func ghExec(args ...string) (bytes.Buffer, bytes.Buffer, error) {
	return gh.Exec(args...)
}

// Token retrieves the auth token for the given username via `gh auth token -u <user>`.
func (g *GHAuth) Token(username string) (string, error) {
	stdout, stderr, err := g.exec("auth", "token", "-u", username)
	if err != nil {
		return "", fmt.Errorf("gh auth token -u %s: %s: %w", username, stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// AuthenticatedUsers returns the list of authenticated users via `gh auth status`.
func (g *GHAuth) AuthenticatedUsers() ([]string, error) {
	stdout, stderr, err := g.exec("auth", "status", "-a")
	if err != nil {
		// gh auth status exits 1 if not logged in; check stderr.
		output := stderr.String()
		if strings.Contains(output, "not logged in") {
			return nil, nil
		}
		return nil, fmt.Errorf("gh auth status: %s: %w", output, err)
	}

	return parseAuthUsers(stdout.String() + stderr.String()), nil
}

// ActiveUser returns the currently active gh user via `gh auth status`.
func (g *GHAuth) ActiveUser() (string, error) {
	stdout, stderr, err := g.exec("auth", "status")
	if err != nil {
		return "", fmt.Errorf("gh auth status: %s: %w", stderr.String(), err)
	}
	combined := stdout.String() + stderr.String()
	return parseActiveUser(combined)
}

// UserInfo holds information about a GitHub user.
type UserInfo struct {
	Name  string
	Email string
}

// GetUserInfo retrieves the user's name and email from GitHub API.
func (g *GHAuth) GetUserInfo(username string) (*UserInfo, error) {
	info := &UserInfo{}

	// Get name from user profile
	stdout, stderr, err := g.exec("api", "user", "-u", username)
	if err != nil {
		return nil, fmt.Errorf("gh api user: %s: %w", stderr.String(), err)
	}
	info.Name = parseNameFromJSON(stdout.String())

	// Get primary email
	stdout, stderr, err = g.exec("api", "user/emails", "-u", username)
	if err != nil {
		return nil, fmt.Errorf("gh api user/emails: %s: %w", stderr.String(), err)
	}
	info.Email = parsePrimaryEmailFromJSON(stdout.String())

	return info, nil
}

// parseActiveUser extracts the active username from gh auth status output.
func parseActiveUser(output string) (string, error) {
	// Look for "Logged in to github.com account <user>"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "account") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "account" && i+1 < len(parts) {
					user := strings.TrimRight(parts[i+1], "()")
					return user, nil
				}
			}
		}
	}
	return "", fmt.Errorf("could not determine active user from gh auth status output")
}

// parseAuthUsers extracts usernames from gh auth status output.
// The format varies across gh versions; we look for "account <user>" patterns.
func parseAuthUsers(output string) []string {
	var users []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "account" && i+1 < len(fields) {
				user := strings.TrimRight(fields[i+1], "()")
				if !seen[user] {
					seen[user] = true
					users = append(users, user)
				}
			}
		}
	}
	return users
}

// parseNameFromJSON extracts the name field from GitHub API /user response.
func parseNameFromJSON(jsonStr string) string {
	// Simple extraction: look for "name": "value"
	for _, line := range strings.Split(jsonStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"name":`) {
			// Extract value between quotes after colon
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, `",`)
				// Return empty if null
				if value == "null" {
					return ""
				}
				return value
			}
		}
	}
	return ""
}

// parsePrimaryEmailFromJSON extracts the primary email from GitHub API /user/emails response.
func parsePrimaryEmailFromJSON(jsonStr string) string {
	// Look for "email": "...", followed by "primary": true
	lines := strings.Split(jsonStr, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"primary":`) && strings.Contains(line, "true") {
			// Look backwards for the email field
			for j := i - 1; j >= 0; j-- {
				emailLine := strings.TrimSpace(lines[j])
				if strings.HasPrefix(emailLine, `"email":`) {
					parts := strings.SplitN(emailLine, ":", 2)
					if len(parts) == 2 {
						value := strings.TrimSpace(parts[1])
						value = strings.Trim(value, `",`)
						return value
					}
				}
			}
		}
	}
	return ""
}
