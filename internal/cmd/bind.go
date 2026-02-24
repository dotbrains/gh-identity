package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/gitconfig"
	"github.com/spf13/cobra"
)

func newBindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bind [<path>] <profile>",
		Short: "Bind a directory to an identity profile",
		Long:  "Bind a directory (defaults to $PWD) to a profile. All gh/git operations inside that tree will use the bound identity.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var dirPath, profileName string
			if len(args) == 2 {
				dirPath = args[0]
				profileName = args[1]
			} else {
				dirPath = "."
				profileName = args[0]
			}
			return runBind(dirPath, profileName)
		},
	}
}

func runBind(dirPath, profileName string) error {
	// Validate profile exists.
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}
	profile, err := profiles.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("profile %q not found — run `gh identity profile list` to see available profiles", profileName)
	}

	// Expand and resolve the directory path.
	expanded, err := config.ExpandPath(dirPath)
	if err != nil {
		return err
	}

	// Add the binding.
	bindings, err := config.LoadBindings()
	if err != nil {
		return err
	}
	if err := bindings.AddBinding(expanded, profileName); err != nil {
		return err
	}
	if err := bindings.Save(); err != nil {
		return err
	}

	// Write gitconfig fragment.
	if err := gitconfig.WriteProfileFragment(profileName, profile); err != nil {
		return fmt.Errorf("writing gitconfig fragment: %w", err)
	}

	// Add includeIf to global gitconfig.
	gcPath, err := gitconfig.GlobalGitconfigPath()
	if err != nil {
		return err
	}
	gitDir, err := config.GitConfigDir()
	if err != nil {
		return err
	}
	fragmentPath := filepath.Join(gitDir, profileName+".gitconfig")
	if err := gitconfig.AddIncludeIf(gcPath, expanded, fragmentPath); err != nil {
		return fmt.Errorf("adding includeIf directive: %w", err)
	}

	fmt.Printf("✅ Bound %s → %s\n", expanded, profileName)
	return nil
}
