package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestConfig(t *testing.T, profilesYAML, bindingsYAML string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", dir)

	if profilesYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "profiles.yml"), []byte(profilesYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if bindingsYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "bindings.yml"), []byte(bindingsYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestResolve_WithBinding(t *testing.T) {
	tmp := t.TempDir()
	boundDir := filepath.Join(tmp, "code", "personal")
	if err := os.MkdirAll(boundDir, 0o755); err != nil {
		t.Fatal(err)
	}

	setupTestConfig(t,
		`profiles:
  personal:
    gh_user: user1
    git_name: User One
    git_email: user1@example.com
    ssh_key: ~/.ssh/id_test
default: personal`,
		`bindings:
  - path: `+boundDir+`
    profile: personal`,
	)

	output, err := Resolve(boundDir, Fish)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "gh auth switch --user user1") {
		t.Error("expected gh auth switch for user1 in output")
	}
	if !strings.Contains(output, "User One") {
		t.Error("expected git name in output")
	}
	if !strings.Contains(output, "user1@example.com") {
		t.Error("expected git email in output")
	}
	if !strings.Contains(output, "GH_IDENTITY_PROFILE") {
		t.Error("expected profile name in output")
	}
	if !strings.Contains(output, "GIT_SSH_COMMAND") {
		t.Error("expected SSH command in output when ssh_key is set")
	}
}

func TestResolve_WithDefault(t *testing.T) {
	setupTestConfig(t,
		`profiles:
  work:
    gh_user: user2
    git_name: User Two
    git_email: user2@company.com
default: work`,
		`bindings: []`,
	)

	output, err := Resolve("/some/random/dir", Bash)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "gh auth switch --user user2") {
		t.Error("expected gh auth switch for user2")
	}
	if !strings.Contains(output, "export GIT_AUTHOR_NAME=") {
		t.Error("expected bash export syntax")
	}
}

func TestResolve_NoProfile(t *testing.T) {
	setupTestConfig(t,
		`profiles: {}`,
		`bindings: []`,
	)

	output, err := Resolve("/some/dir", Fish)
	if err != nil {
		t.Fatal(err)
	}

	if output != "" {
		t.Errorf("expected empty output when no profile resolved, got %q", output)
	}
}

func TestResolve_NoConfig(t *testing.T) {
	// Point to an empty dir (no config files).
	t.Setenv("GH_IDENTITY_CONFIG_DIR", t.TempDir())

	output, err := Resolve("/some/dir", Zsh)
	if err != nil {
		t.Fatal(err)
	}

	if output != "" {
		t.Errorf("expected empty output with no config, got %q", output)
	}
}

func TestResolve_ZshFormat(t *testing.T) {
	setupTestConfig(t,
		`profiles:
  test:
    gh_user: testuser
    git_name: Test
    git_email: test@test.com
default: test`,
		`bindings: []`,
	)

	output, err := Resolve("/any", Zsh)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "unset GH_TOKEN") {
		t.Error("expected GH_TOKEN unset for zsh")
	}
	if !strings.Contains(output, "gh auth switch --user testuser") {
		t.Error("expected gh auth switch for zsh")
	}
}
