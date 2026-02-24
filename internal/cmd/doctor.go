package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/ghauth"
	"github.com/dotbrains/gh-identity/internal/gitconfig"
)

func newDoctorCmd(auth ghauth.Auth) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate the full gh-identity setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(auth)
		},
	}
}

func runDoctor(auth ghauth.Auth) error {
	fmt.Println("🩺 gh-identity doctor")
	fmt.Println()

	issues := 0

	// Check 1: Config directory exists.
	configDir, err := config.Dir()
	if err != nil {
		fmt.Printf("❌ Cannot determine config directory: %v\n", err)
		issues++
	} else if _, err := os.Stat(configDir); os.IsNotExist(err) {
		fmt.Printf("❌ Config directory does not exist: %s\n", configDir)
		fmt.Println("   Run `gh identity init` to set up.")
		issues++
	} else {
		fmt.Printf("✅ Config directory: %s\n", configDir)
	}

	// Check 2: Profiles file.
	profiles, err := config.LoadProfiles()
	if err != nil {
		fmt.Printf("❌ Cannot load profiles: %v\n", err)
		issues++
	} else if len(profiles.Profiles) == 0 {
		fmt.Println("⚠️  No profiles configured.")
		issues++
	} else {
		fmt.Printf("✅ %d profile(s) configured.\n", len(profiles.Profiles))

		// Validate required fields.
		if errs := profiles.Validate(); len(errs) > 0 {
			for _, e := range errs {
				fmt.Printf("❌ %s\n", e)
				issues++
			}
		}
	}

	// Check 3: All profiles reference authenticated gh accounts.
	if profiles != nil {
		authedUsers, err := auth.AuthenticatedUsers()
		if err != nil {
			fmt.Printf("⚠️  Cannot list authenticated users: %v\n", err)
		} else {
			authedSet := make(map[string]bool)
			for _, u := range authedUsers {
				authedSet[u] = true
			}
			for name, p := range profiles.Profiles {
				if !authedSet[p.GHUser] {
					fmt.Printf("❌ Profile %q references user %q which is not authenticated.\n", name, p.GHUser)
					fmt.Printf("   Run `gh auth login` to authenticate as %s.\n", p.GHUser)
					issues++
				}
			}
		}
	}

	// Check 4: SSH keys exist.
	if profiles != nil {
		for name, p := range profiles.Profiles {
			if p.SSHKey != "" {
				expanded, err := config.ExpandPath(p.SSHKey)
				if err != nil {
					fmt.Printf("❌ Profile %q: cannot expand SSH key path %q: %v\n", name, p.SSHKey, err)
					issues++
					continue
				}
				info, err := os.Stat(expanded)
				if os.IsNotExist(err) {
					fmt.Printf("❌ Profile %q: SSH key not found: %s\n", name, expanded)
					issues++
				} else if err != nil {
					fmt.Printf("❌ Profile %q: cannot stat SSH key: %v\n", name, err)
					issues++
				} else if info.Mode().Perm()&0o077 != 0 {
					fmt.Printf("⚠️  Profile %q: SSH key %s has overly permissive permissions (%o).\n", name, expanded, info.Mode().Perm())
					fmt.Println("   Run: chmod 600", expanded)
					issues++
				} else {
					fmt.Printf("✅ Profile %q: SSH key OK (%s)\n", name, expanded)
				}
			}
		}
	}

	// Check 5: Shell hook binary.
	binDir, err := config.BinDir()
	if err == nil {
		hookBin := filepath.Join(binDir, "gh-identity-hook")
		if _, err := os.Stat(hookBin); os.IsNotExist(err) {
			fmt.Printf("❌ Hook binary not found: %s\n", hookBin)
			fmt.Println("   Run `gh identity init` to install it.")
			issues++
		} else {
			fmt.Printf("✅ Hook binary: %s\n", hookBin)
		}
	}

	// Check 6: Shell hook installed.
	home, err := os.UserHomeDir()
	if err == nil {
		hookInstalled := false
		shellConfigs := []string{
			filepath.Join(home, ".config", "fish", "conf.d", "gh-identity.fish"),
			filepath.Join(home, ".bashrc"),
			filepath.Join(home, ".zshrc"),
		}
		for _, rc := range shellConfigs {
			content, err := os.ReadFile(rc)
			if err == nil && contains(string(content), "gh-identity") {
				hookInstalled = true
				fmt.Printf("✅ Shell hook installed in %s\n", rc)
			}
		}
		if !hookInstalled {
			fmt.Println("⚠️  Shell hook not detected in any shell config.")
			fmt.Println("   Run `gh identity init` to install it.")
			issues++
		}
	}

	// Check 7: Bindings reference valid profiles.
	bindings, err := config.LoadBindings()
	if err != nil {
		fmt.Printf("⚠️  Cannot load bindings: %v\n", err)
	} else if profiles != nil {
		for _, b := range bindings.Bindings {
			if _, exists := profiles.Profiles[b.Profile]; !exists {
				fmt.Printf("❌ Binding %s → %q references non-existent profile.\n", b.Path, b.Profile)
				issues++
			}
		}
	}

	// Check 8: includeIf directives.
	gcPath, err := gitconfig.GlobalGitconfigPath()
	if err == nil {
		managed, err := gitconfig.ListManagedIncludeIfs(gcPath)
		if err == nil && len(managed) > 0 {
			fmt.Printf("✅ %d managed includeIf directive(s) in %s\n", len(managed), gcPath)
		}
	}

	fmt.Println()
	if issues == 0 {
		fmt.Println("✅ All checks passed!")
	} else {
		fmt.Printf("Found %d issue(s).\n", issues)
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
