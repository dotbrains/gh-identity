package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/ghauth"
	"github.com/dotbrains/gh-identity/internal/gitconfig"
)

func newProfileCmd(auth ghauth.Auth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage identity profiles",
	}

	cmd.AddCommand(
		newProfileAddCmd(auth),
		newProfileListCmd(),
		newProfileRemoveCmd(),
	)

	return cmd
}

func newProfileAddCmd(auth ghauth.Auth) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new identity profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileAdd(auth, args[0])
		},
	}
}

func runProfileAdd(auth ghauth.Auth, name string) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}

	if _, exists := profiles.Profiles[name]; exists {
		return fmt.Errorf("profile %q already exists", name)
	}

	// List authenticated users for reference.
	users, err := auth.AuthenticatedUsers()
	if err == nil && len(users) > 0 {
		fmt.Printf("Authenticated accounts: %s\n", strings.Join(users, ", "))
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("GitHub username (gh_user): ")
	ghUser := readLine(reader)

	fmt.Printf("Git name: ")
	gitName := readLine(reader)

	fmt.Printf("Git email: ")
	gitEmail := readLine(reader)

	fmt.Printf("SSH key path (optional): ")
	sshKey := readLine(reader)

	p := config.Profile{
		GHUser:   ghUser,
		GitName:  gitName,
		GitEmail: gitEmail,
		SSHKey:   sshKey,
	}

	profiles.AddProfile(name, p)
	if err := profiles.Save(); err != nil {
		return err
	}

	// Write gitconfig fragment.
	if err := gitconfig.WriteProfileFragment(name, p); err != nil {
		return fmt.Errorf("writing gitconfig fragment: %w", err)
	}

	fmt.Printf("✅ Profile %q created.\n", name)
	return nil
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all configured profiles",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileList()
		},
	}
}

func runProfileList() error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}

	if len(profiles.Profiles) == 0 {
		fmt.Println("No profiles configured. Run `gh identity profile add <name>` to create one.")
		return nil
	}

	activeProfile := os.Getenv("GH_IDENTITY_PROFILE")

	// Sort profile names for consistent output.
	names := make([]string, 0, len(profiles.Profiles))
	for name := range profiles.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		p := profiles.Profiles[name]
		indicator := "  "
		if name == activeProfile {
			indicator = "* "
		} else if name == profiles.Default {
			indicator = "→ "
		}
		fmt.Printf("%s%s\n", indicator, name)
		fmt.Printf("    gh_user:   %s\n", p.GHUser)
		fmt.Printf("    git_name:  %s\n", p.GitName)
		fmt.Printf("    git_email: %s\n", p.GitEmail)
		if p.SSHKey != "" {
			fmt.Printf("    ssh_key:   %s\n", p.SSHKey)
		}
	}

	return nil
}

func newProfileRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove a profile and its associated bindings",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileRemove(args[0])
		},
	}
}

func runProfileRemove(name string) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}

	if err := profiles.RemoveProfile(name); err != nil {
		return err
	}
	if err := profiles.Save(); err != nil {
		return err
	}

	// Remove associated bindings.
	bindings, err := config.LoadBindings()
	if err != nil {
		return err
	}

	var remaining []config.Binding
	var removedPaths []string
	for _, b := range bindings.Bindings {
		if b.Profile == name {
			removedPaths = append(removedPaths, b.Path)
		} else {
			remaining = append(remaining, b)
		}
	}
	bindings.Bindings = remaining
	if err := bindings.Save(); err != nil {
		return err
	}

	// Remove gitconfig fragment and includeIf entries.
	if err := gitconfig.RemoveProfileFragment(name); err != nil {
		fmt.Printf("⚠️  Could not remove gitconfig fragment: %v\n", err)
	}

	gcPath, err := gitconfig.GlobalGitconfigPath()
	if err == nil {
		for _, p := range removedPaths {
			expanded, err := config.ExpandPath(p)
			if err != nil {
				continue
			}
			_ = gitconfig.RemoveIncludeIf(gcPath, expanded)
		}
	}

	fmt.Printf("✅ Profile %q removed.\n", name)
	if len(removedPaths) > 0 {
		fmt.Printf("   Also removed %d binding(s).\n", len(removedPaths))
	}
	return nil
}
