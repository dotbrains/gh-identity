package gitconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotbrains/gh-identity/internal/config"
)

func TestWriteProfileFragmentTo(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "work.gitconfig")

	p := config.Profile{
		GitName:  "Test User",
		GitEmail: "test@example.com",
	}

	if err := WriteProfileFragmentTo(path, p); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "name = Test User") {
		t.Error("fragment missing name")
	}
	if !strings.Contains(content, "email = test@example.com") {
		t.Error("fragment missing email")
	}
}

func TestAddIncludeIf(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	// Start with an existing config.
	if err := os.WriteFile(gcPath, []byte("[user]\n    name = Default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := AddIncludeIf(gcPath, "/home/user/code/work", "/home/user/.config/gh-identity/git/work.gitconfig"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(gcPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, `[includeIf "gitdir:/home/user/code/work/"]`) {
		t.Error("includeIf directive not added")
	}
	if !strings.Contains(content, "path = /home/user/.config/gh-identity/git/work.gitconfig") {
		t.Error("path line not added")
	}
	if !strings.Contains(content, marker) {
		t.Error("marker not added")
	}
}

func TestAddIncludeIf_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	if err := os.WriteFile(gcPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if err := AddIncludeIf(gcPath, "/code/work", "/cfg/work.gitconfig"); err != nil {
			t.Fatal(err)
		}
	}

	data, _ := os.ReadFile(gcPath)
	count := strings.Count(string(data), `[includeIf "gitdir:/code/work/"]`)
	if count != 1 {
		t.Errorf("expected 1 includeIf directive, got %d", count)
	}
}

func TestRemoveIncludeIf(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	if err := AddIncludeIf(gcPath, "/code/work", "/cfg/work.gitconfig"); err != nil {
		t.Fatal(err)
	}

	if err := RemoveIncludeIf(gcPath, "/code/work"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(gcPath)
	if strings.Contains(string(data), "includeIf") {
		t.Error("includeIf not removed")
	}
}

func TestListManagedIncludeIfs(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	_ = AddIncludeIf(gcPath, "/code/work", "/cfg/work.gitconfig")
	_ = AddIncludeIf(gcPath, "/code/personal", "/cfg/personal.gitconfig")

	dirs, err := ListManagedIncludeIfs(gcPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirs) != 2 {
		t.Errorf("expected 2 managed dirs, got %d", len(dirs))
	}
}

func TestRemoveIncludeIf_NonExistent(t *testing.T) {
	// Removing from nonexistent file should not error.
	if err := RemoveIncludeIf("/nonexistent/.gitconfig", "/some/path"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
