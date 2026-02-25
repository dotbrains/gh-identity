package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/ghauth"
)

func newInitCmd(auth ghauth.Auth) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive first-time setup",
		Long:  "Discovers existing gh authenticated accounts, creates profiles for each, and installs the shell hook.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(auth)
		},
	}
}

func runInit(auth ghauth.Auth) error {
	fmt.Println("🔧 gh-identity init")
	fmt.Println()

	// Step 1: Discover authenticated accounts.
	users, err := auth.AuthenticatedUsers()
	if err != nil {
		return fmt.Errorf("listing authenticated accounts: %w", err)
	}
	if len(users) == 0 {
		fmt.Println("No authenticated gh accounts found.")
		fmt.Println("Run `gh auth login` to authenticate, then re-run `gh identity init`.")
		return nil
	}

	fmt.Printf("Found %d authenticated account(s): %s\n", len(users), strings.Join(users, ", "))
	fmt.Println()

	// Step 2: Ensure config directory exists.
	dir, err := config.EnsureDir()
	if err != nil {
		return err
	}
	fmt.Printf("Config directory: %s\n", dir)

	// Step 3: Create profiles for each account.
	profiles, err := config.LoadProfiles()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	for _, user := range users {
		fmt.Printf("\n--- Profile for %s ---\n", user)

		// Infer defaults
		defaultGitName, defaultGitEmail := inferGitDetails(auth, user)
		defaultSSHKey := detectSSHKey()

		fmt.Printf("Profile name [%s]: ", user)
		name := readLine(reader)
		if name == "" {
			name = user
		}

		fmt.Printf("Git name [%s]: ", defaultGitName)
		gitName := readLine(reader)
		if gitName == "" {
			gitName = defaultGitName
		}

		fmt.Printf("Git email [%s]: ", defaultGitEmail)
		gitEmail := readLine(reader)
		if gitEmail == "" {
			gitEmail = defaultGitEmail
		}

		fmt.Printf("SSH key path [%s]: ", defaultSSHKey)
		sshKey := readLine(reader)
		if sshKey == "" {
			sshKey = defaultSSHKey
		}

		profiles.AddProfile(name, config.Profile{
			GHUser:   user,
			GitName:  gitName,
			GitEmail: gitEmail,
			SSHKey:   sshKey,
		})
	}

	// Set default profile.
	if len(profiles.Profiles) > 0 && profiles.Default == "" {
		fmt.Printf("\nDefault profile name: ")
		profiles.Default = readLine(reader)
	}

	if err := profiles.Save(); err != nil {
		return fmt.Errorf("saving profiles: %w", err)
	}
	fmt.Println("\n✅ Profiles saved.")

	// Step 4: Install shell hook.
	if err := installShellHook(); err != nil {
		fmt.Printf("⚠️  Could not install shell hook: %v\n", err)
		fmt.Println("   You can install it manually later. See `gh identity doctor` for details.")
	} else {
		fmt.Println("✅ Shell hook installed.")
	}

	// Step 5: Install hook binary.
	if err := installHookBinary(); err != nil {
		fmt.Printf("⚠️  Could not install hook binary: %v\n", err)
	} else {
		fmt.Println("✅ Hook binary installed.")
	}

	fmt.Println("\n🎉 Setup complete! Open a new terminal or source your shell config to activate.")
	return nil
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func installShellHook() error {
	shell := detectShell()
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}
	hookBinary := filepath.Join(binDir, "gh-identity-hook")

	var rcFile, hookLine string
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	switch shell {
	case "fish":
		rcFile = filepath.Join(home, ".config", "fish", "conf.d", "gh-identity.fish")
		hookLine = fmt.Sprintf(`# gh-identity hook
function __gh_identity_hook --on-variable PWD
    eval (%s --shell fish)
end
__gh_identity_hook
`, hookBinary)
		// For fish, write directly to conf.d.
		if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
			return err
		}
		return os.WriteFile(rcFile, []byte(hookLine), 0o644)
	case "bash":
		rcFile = filepath.Join(home, ".bashrc")
		hookLine = fmt.Sprintf("\n# gh-identity hook\neval \"$(%s --shell bash)\"\n", hookBinary)
	case "zsh":
		rcFile = filepath.Join(home, ".zshrc")
		hookLine = fmt.Sprintf("\n# gh-identity hook\neval \"$(%s --shell zsh)\"\n", hookBinary)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	// Check if hook is already installed.
	content, err := os.ReadFile(rcFile)
	if err == nil && strings.Contains(string(content), "gh-identity hook") {
		return nil // Already installed.
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(hookLine)
	return err
}

func installHookBinary() error {
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	// Check if we can find the hook binary next to the current executable.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current executable: %w", err)
	}

	hookSrc := filepath.Join(filepath.Dir(exe), "gh-identity-hook")
	if runtime.GOOS == "windows" {
		hookSrc += ".exe"
	}

	if _, err := os.Stat(hookSrc); os.IsNotExist(err) {
		return fmt.Errorf("hook binary not found at %s — build it with `make build`", hookSrc)
	}

	hookDst := filepath.Join(binDir, "gh-identity-hook")
	if runtime.GOOS == "windows" {
		hookDst += ".exe"
	}

	src, err := os.ReadFile(hookSrc)
	if err != nil {
		return err
	}
	return os.WriteFile(hookDst, src, 0o755)
}

func detectShell() string {
	// Check SHELL env var.
	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		base := filepath.Base(shellPath)
		switch base {
		case "fish", "bash", "zsh":
			return base
		}
	}
	return "bash" // default fallback
}

// inferGitDetails tries to infer git name and email from:
// 1. GitHub API
// 2. Global git config
func inferGitDetails(auth ghauth.Auth, username string) (string, string) {
	var name, email string

	// Try GitHub API first
	if ghAuth, ok := auth.(*ghauth.GHAuth); ok {
		if info, err := ghAuth.GetUserInfo(username); err == nil {
			if info.Name != "" {
				name = info.Name
			}
			if info.Email != "" {
				email = info.Email
			}
		}
	}

	// Fallback to global git config if not found
	if name == "" {
		if output, err := exec.Command("git", "config", "--global", "user.name").Output(); err == nil {
			name = strings.TrimSpace(string(output))
		}
	}
	if email == "" {
		if output, err := exec.Command("git", "config", "--global", "user.email").Output(); err == nil {
			email = strings.TrimSpace(string(output))
		}
	}

	return name, email
}

// detectSSHKey tries to find a default SSH key in ~/.ssh/
func detectSSHKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	sshDir := filepath.Join(home, ".ssh")
	// Check common key names in order of preference
	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	for _, keyName := range keyNames {
		keyPath := filepath.Join(sshDir, keyName)
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath
		}
	}

	return ""
}
