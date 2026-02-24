package gitconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotbrains/gh-identity/internal/config"
)

func TestWriteProfileFragment(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)

	p := config.Profile{
		GitName:  "Test User",
		GitEmail: "test@example.com",
	}

	if err := WriteProfileFragment("testprofile", p); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmp, "git", "testprofile.gitconfig")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Test User") {
		t.Error("missing name in fragment")
	}
	if !strings.Contains(content, "test@example.com") {
		t.Error("missing email in fragment")
	}
}

func TestRemoveProfileFragment(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)

	// Create git dir and fragment.
	gitDir := filepath.Join(tmp, "git")
	os.MkdirAll(gitDir, 0o755)
	fragPath := filepath.Join(gitDir, "test.gitconfig")
	os.WriteFile(fragPath, []byte("[user]\n    name = Test\n"), 0o644)

	if err := RemoveProfileFragment("test"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(fragPath); !os.IsNotExist(err) {
		t.Error("fragment should have been removed")
	}
}

func TestRemoveProfileFragment_NotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	os.MkdirAll(filepath.Join(tmp, "git"), 0o755)

	// Should not error if file doesn't exist.
	if err := RemoveProfileFragment("nonexistent"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGlobalGitconfigPath(t *testing.T) {
	path, err := GlobalGitconfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !strings.HasSuffix(path, ".gitconfig") {
		t.Errorf("expected path to end with .gitconfig, got %q", path)
	}
}

func TestWriteProfileFragmentTo_CreatesDirs(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a", "b", "profile.gitconfig")

	p := config.Profile{GitName: "N", GitEmail: "e@e.com"}
	if err := WriteProfileFragmentTo(path, p); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should have been created")
	}
}

func TestListManagedIncludeIfs_Empty(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")
	os.WriteFile(gcPath, []byte("[user]\n    name = Test\n"), 0o644)

	dirs, err := ListManagedIncludeIfs(gcPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 0 {
		t.Errorf("expected 0 managed dirs, got %d", len(dirs))
	}
}

func TestAddIncludeIf_NewFile(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")
	// File doesn't exist yet.

	if err := AddIncludeIf(gcPath, "/test/dir", "/cfg/test.gitconfig"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(gcPath)
	content := string(data)
	if !strings.Contains(content, `[includeIf "gitdir:/test/dir/"]`) {
		t.Error("includeIf directive not written")
	}
}
