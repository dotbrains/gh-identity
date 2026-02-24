package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/ghauth"
)

func newSwitchCmd(auth ghauth.Auth) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <profile>",
		Short: "Manually activate a profile for the current session",
		Long:  "Activate a profile for the current session, overriding any directory binding until the next directory change.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSwitch(auth, args[0])
		},
	}
}

func runSwitch(_ ghauth.Auth, profileName string) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}

	profile, err := profiles.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Print commands for the user to eval.
	fmt.Println("unset GH_TOKEN 2>/dev/null")
	fmt.Printf("gh auth switch --user %s 2>/dev/null\n", profile.GHUser)
	fmt.Printf("export GIT_AUTHOR_NAME=%q\n", profile.GitName)
	fmt.Printf("export GIT_AUTHOR_EMAIL=%q\n", profile.GitEmail)
	fmt.Printf("export GIT_COMMITTER_NAME=%q\n", profile.GitName)
	fmt.Printf("export GIT_COMMITTER_EMAIL=%q\n", profile.GitEmail)
	fmt.Printf("export GH_IDENTITY_PROFILE=%q\n", profileName)
	if profile.SSHKey != "" {
		expanded, err := config.ExpandPath(profile.SSHKey)
		if err == nil {
			fmt.Printf("export GIT_SSH_COMMAND=%q\n", fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", expanded))
		}
	}

	return nil
}
