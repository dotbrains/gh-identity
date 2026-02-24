// Package hook implements the shell hook resolution logic.
// It is designed to be fast (<5ms) and is used by the gh-identity-hook binary.
package hook

import (
	"fmt"
	"os"
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
	GHToken           string
	GitAuthorName     string
	GitAuthorEmail    string
	GitCommitterName  string
	GitCommitterEmail string
	GHIdentityProfile string
	GHSSHCommand      string // optional
	GitAskPass        string // optional: path to askpass helper for HTTPS auth
}

// Resolve loads config, resolves the binding for dir, and returns shell export statements.
// tokenFn is called to obtain the GH_TOKEN for the resolved profile's gh_user.
// It is separated to allow the hook binary to call gh auth token itself.
func Resolve(dir string, shell ShellType, tokenFn func(ghUser string) (string, error)) (string, error) {
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

	token, err := tokenFn(profile.GHUser)
	if err != nil {
		return "", fmt.Errorf("getting token for %s: %w", profile.GHUser, err)
	}

	env := EnvOutput{
		GHToken:           token,
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

	// Set GIT_ASKPASS so git HTTPS operations use the resolved GH_TOKEN.
	askpassPath, err := config.AskPassPath()
	if err == nil {
		if _, statErr := os.Stat(askpassPath); statErr == nil {
			env.GitAskPass = askpassPath
		}
	}

	return formatExports(shell, env), nil
}

func formatExports(shell ShellType, env EnvOutput) string {
	var b strings.Builder

	switch shell {
	case Fish:
		writeFishExport(&b, "GH_TOKEN", env.GHToken)
		writeFishExport(&b, "GIT_AUTHOR_NAME", env.GitAuthorName)
		writeFishExport(&b, "GIT_AUTHOR_EMAIL", env.GitAuthorEmail)
		writeFishExport(&b, "GIT_COMMITTER_NAME", env.GitCommitterName)
		writeFishExport(&b, "GIT_COMMITTER_EMAIL", env.GitCommitterEmail)
		writeFishExport(&b, "GH_IDENTITY_PROFILE", env.GHIdentityProfile)
		if env.GHSSHCommand != "" {
			writeFishExport(&b, "GIT_SSH_COMMAND", env.GHSSHCommand)
		}
		if env.GitAskPass != "" {
			writeFishExport(&b, "GIT_ASKPASS", env.GitAskPass)
		}
	default: // bash, zsh
		writePosixExport(&b, "GH_TOKEN", env.GHToken)
		writePosixExport(&b, "GIT_AUTHOR_NAME", env.GitAuthorName)
		writePosixExport(&b, "GIT_AUTHOR_EMAIL", env.GitAuthorEmail)
		writePosixExport(&b, "GIT_COMMITTER_NAME", env.GitCommitterName)
		writePosixExport(&b, "GIT_COMMITTER_EMAIL", env.GitCommitterEmail)
		writePosixExport(&b, "GH_IDENTITY_PROFILE", env.GHIdentityProfile)
		if env.GHSSHCommand != "" {
			writePosixExport(&b, "GIT_SSH_COMMAND", env.GHSSHCommand)
		}
		if env.GitAskPass != "" {
			writePosixExport(&b, "GIT_ASKPASS", env.GitAskPass)
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
