package cmd

import (
	"fmt"
	"os"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/ghauth"
	"github.com/dotbrains/gh-identity/internal/resolve"
	"github.com/spf13/cobra"
)

func newStatusCmd(auth ghauth.Auth) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display the active identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(auth)
		},
	}
}

func runStatus(auth ghauth.Auth) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}

	bindings, err := config.LoadBindings()
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	result, err := resolve.ForDirectory(pwd, bindings, profiles.Default)
	if err != nil {
		return err
	}

	// Check if there's an override from environment.
	envProfile := os.Getenv("GH_IDENTITY_PROFILE")
	if envProfile != "" {
		result.Profile = envProfile
	}

	if result.Profile == "" {
		fmt.Println("No active profile.")
		fmt.Println("Run `gh identity bind <profile>` or `gh identity switch <profile>` to activate one.")
		return nil
	}

	profile, err := profiles.GetProfile(result.Profile)
	if err != nil {
		return fmt.Errorf("profile %q configured but not found in profiles.yml", result.Profile)
	}

	fmt.Printf("  Profile:  %s\n", result.Profile)
	fmt.Printf("  Account:  %s\n", profile.GHUser)
	fmt.Printf("  Name:     %s\n", profile.GitName)
	fmt.Printf("  Email:    %s\n", profile.GitEmail)
	if profile.SSHKey != "" {
		fmt.Printf("  SSH Key:  %s\n", profile.SSHKey)
	}
	if result.BoundPath != "" {
		fmt.Printf("  Bound by: %s\n", result.BoundPath)
	} else if result.IsDefault {
		fmt.Printf("  Source:   default profile\n")
	} else if envProfile != "" {
		fmt.Printf("  Source:   environment (GH_IDENTITY_PROFILE)\n")
	}

	return nil
}
