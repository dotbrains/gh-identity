package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gh "github.com/cli/go-gh/v2"
	"github.com/spf13/cobra"

	"github.com/dotbrains/gh-identity/internal/ghauth"
)

func newCloneCmd(auth ghauth.Auth) *cobra.Command {
	var profileFlag string

	cmd := &cobra.Command{
		Use:   "clone <repo>",
		Short: "Clone a repo and bind it to a profile",
		Long:  "Wraps `gh repo clone`. After cloning, automatically binds the new directory to the specified profile (or the currently active one).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClone(auth, args[0], profileFlag)
		},
	}

	cmd.Flags().StringVar(&profileFlag, "profile", "", "Profile to bind the cloned repo to (defaults to active profile)")
	return cmd
}

func runClone(auth ghauth.Auth, repo, profileName string) error {
	// Determine profile.
	if profileName == "" {
		profileName = os.Getenv("GH_IDENTITY_PROFILE")
	}
	if profileName == "" {
		return fmt.Errorf("no profile specified and no active profile — use --profile or activate a profile first")
	}

	// Clone the repo.
	fmt.Printf("Cloning %s...\n", repo)
	_, stderr, err := gh.Exec("repo", "clone", repo)
	if err != nil {
		return fmt.Errorf("cloning repo: %s: %w", stderr.String(), err)
	}

	// Determine the cloned directory name.
	cloneDir := repoToDir(repo)
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fullPath := filepath.Join(pwd, cloneDir)

	// Verify it exists.
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Clone succeeded but directory %q not found. Bind manually with `gh identity bind`.\n", fullPath)
		return nil
	}

	// Bind the directory.
	if err := runBind(fullPath, profileName); err != nil {
		return fmt.Errorf("binding cloned repo: %w", err)
	}

	return nil
}

// repoToDir extracts the directory name from a repo specifier.
// e.g. "owner/repo" → "repo", "https://github.com/owner/repo.git" → "repo"
func repoToDir(repo string) string {
	// Remove .git suffix.
	repo = strings.TrimSuffix(repo, ".git")

	// Handle URL format.
	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		return parts[len(parts)-1]
	}

	return repo
}
