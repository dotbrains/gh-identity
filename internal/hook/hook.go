// Package hook implements the shell hook resolution logic.
// It is designed to be fast (<5ms) and is used by the gh-identity-hook binary.
package hook

import (
	"fmt"
	"strings"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/resolve"
)

// ShellType represents a supported shell.
type ShellType string

const (
	Fish ShellType = "fish"
	Bash ShellType = "bash"
	Zsh  ShellType = "zsh"
)

// EnvOutput holds the environment variables to export.
type EnvOutput struct {
	GHUser            string // gh auth account to switch to
	GitAuthorName     string
	GitAuthorEmail    string
	GitCommitterName  string
	GitCommitterEmail string
	GHIdentityProfile string
	GHSSHCommand      string // optional
}

// Resolve loads config, resolves the binding for dir, and returns shell statements.
func Resolve(dir string, shell ShellType) (string, error) {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return "", fmt.Errorf("loading profiles: %w", err)
	}

	bindings, err := config.LoadBindings()
	if err != nil {
		return "", fmt.Errorf("loading bindings: %w", err)
	}

	result, err := resolve.ForDirectory(dir, bindings, profiles.Default)
	if err != nil {
		return "", fmt.Errorf("resolving binding: %w", err)
	}

	if result.Profile == "" {
		// No profile resolved; emit nothing.
		return "", nil
	}

	profile, err := profiles.GetProfile(result.Profile)
	if err != nil {
		return "", fmt.Errorf("getting profile %q: %w", result.Profile, err)
	}

	env := EnvOutput{
		GHUser:            profile.GHUser,
		GitAuthorName:     profile.GitName,
		GitAuthorEmail:    profile.GitEmail,
		GitCommitterName:  profile.GitName,
		GitCommitterEmail: profile.GitEmail,
		GHIdentityProfile: result.Profile,
	}

	if profile.SSHKey != "" {
		expanded, err := config.ExpandPath(profile.SSHKey)
		if err == nil {
			env.GHSSHCommand = fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", expanded)
		}
	}

	return formatOutput(shell, env), nil
}

func formatOutput(shell ShellType, env EnvOutput) string {
	var b strings.Builder

	switch shell {
	case Fish:
		// Unset GH_TOKEN so it doesn't override gh auth's keyring token.
		b.WriteString("set -e GH_TOKEN 2>/dev/null\n")
		// Switch gh CLI to the correct account.
		fmt.Fprintf(&b, "gh auth switch --user %s 2>/dev/null\n", env.GHUser)
		writeFishExport(&b, "GIT_AUTHOR_NAME", env.GitAuthorName)
		writeFishExport(&b, "GIT_AUTHOR_EMAIL", env.GitAuthorEmail)
		writeFishExport(&b, "GIT_COMMITTER_NAME", env.GitCommitterName)
		writeFishExport(&b, "GIT_COMMITTER_EMAIL", env.GitCommitterEmail)
		writeFishExport(&b, "GH_IDENTITY_PROFILE", env.GHIdentityProfile)
		if env.GHSSHCommand != "" {
			writeFishExport(&b, "GIT_SSH_COMMAND", env.GHSSHCommand)
		}
	default: // bash, zsh
		// Unset GH_TOKEN so it doesn't override gh auth's keyring token.
		b.WriteString("unset GH_TOKEN 2>/dev/null\n")
		// Switch gh CLI to the correct account.
		fmt.Fprintf(&b, "gh auth switch --user %s 2>/dev/null\n", env.GHUser)
		writePosixExport(&b, "GIT_AUTHOR_NAME", env.GitAuthorName)
		writePosixExport(&b, "GIT_AUTHOR_EMAIL", env.GitAuthorEmail)
		writePosixExport(&b, "GIT_COMMITTER_NAME", env.GitCommitterName)
		writePosixExport(&b, "GIT_COMMITTER_EMAIL", env.GitCommitterEmail)
		writePosixExport(&b, "GH_IDENTITY_PROFILE", env.GHIdentityProfile)
		if env.GHSSHCommand != "" {
			writePosixExport(&b, "GIT_SSH_COMMAND", env.GHSSHCommand)
		}
	}

	return b.String()
}

func writeFishExport(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "set -gx %s %q\n", key, value)
}

func writePosixExport(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "export %s=%q\n", key, value)
}
