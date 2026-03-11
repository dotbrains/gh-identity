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

func TestSyncIncludeIfs(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	// Simulate: managed blocks, then [user] at the bottom (the bug).
	content := `[includeIf "gitdir:/code/dotbrains/"] # managed by gh-identity
    path = /cfg/personal.gitconfig
[credential "https://github.com"]
    helper = !/opt/homebrew/bin/gh auth git-credential
[user]
    email = work@company.com
    name = Work User
[commit]
    gpgsign = true
`
	if err := os.WriteFile(gcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SyncIncludeIfs(gcPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(gcPath)
	result := string(data)

	// The managed block should be at the very end.
	userIdx := strings.Index(result, "[user]")
	includeIdx := strings.Index(result, `[includeIf "gitdir:/code/dotbrains/"]`)
	if includeIdx <= userIdx {
		t.Errorf("managed includeIf should appear after [user]; user at %d, includeIf at %d", userIdx, includeIdx)
	}

	// Credential and commit sections should still be present.
	if !strings.Contains(result, `[credential "https://github.com"]`) {
		t.Error("credential section was lost")
	}
	if !strings.Contains(result, "[commit]") {
		t.Error("commit section was lost")
	}
}

func TestSyncIncludeIfs_NoOp(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	// Already correct: [user] first, managed blocks last.
	content := `[user]
    name = Default
    email = default@test.com

[includeIf "gitdir:/code/work/"] # managed by gh-identity
    path = /cfg/work.gitconfig
`
	if err := os.WriteFile(gcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SyncIncludeIfs(gcPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(gcPath)
	result := string(data)

	// [user] should still be before includeIf.
	userIdx := strings.Index(result, "[user]")
	includeIdx := strings.Index(result, `[includeIf "gitdir:/code/work/"]`)
	if includeIdx <= userIdx {
		t.Errorf("includeIf should remain after [user]")
	}
}

func TestSyncIncludeIfs_ExternalSectionsInterleaved(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	// Multiple managed blocks with external sections in between.
	content := `[includeIf "gitdir:/code/work/"] # managed by gh-identity
    path = /cfg/work.gitconfig
[credential "https://github.com"]
    helper = gh auth git-credential
[includeIf "gitdir:/code/personal/"] # managed by gh-identity
    path = /cfg/personal.gitconfig
[gpg]
    format = ssh
[user]
    name = Global
`
	if err := os.WriteFile(gcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SyncIncludeIfs(gcPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(gcPath)
	result := string(data)

	// Both managed blocks should be after [user] and [gpg].
	userIdx := strings.Index(result, "[user]")
	workIdx := strings.Index(result, `[includeIf "gitdir:/code/work/"]`)
	personalIdx := strings.Index(result, `[includeIf "gitdir:/code/personal/"]`)

	if workIdx <= userIdx {
		t.Errorf("work includeIf should be after [user]")
	}
	if personalIdx <= userIdx {
		t.Errorf("personal includeIf should be after [user]")
	}

	// Non-managed sections preserved.
	if !strings.Contains(result, `[credential "https://github.com"]`) {
		t.Error("credential section lost")
	}
	if !strings.Contains(result, "[gpg]") {
		t.Error("gpg section lost")
	}
}

func TestHasUserAfterIncludeIfs(t *testing.T) {
	tmp := t.TempDir()

	// Positive case: [user] after managed block.
	badPath := filepath.Join(tmp, "bad.gitconfig")
	os.WriteFile(badPath, []byte(`[includeIf "gitdir:/x/"] # managed by gh-identity
    path = /y.gitconfig
[user]
    name = Override
`), 0o644)

	has, err := HasUserAfterIncludeIfs(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("expected true for [user] after managed block")
	}

	// Negative case: [user] before managed block.
	goodPath := filepath.Join(tmp, "good.gitconfig")
	os.WriteFile(goodPath, []byte(`[user]
    name = Default
[includeIf "gitdir:/x/"] # managed by gh-identity
    path = /y.gitconfig
`), 0o644)

	has, err = HasUserAfterIncludeIfs(goodPath)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected false for [user] before managed block")
	}

	// No managed blocks at all.
	noManagedPath := filepath.Join(tmp, "none.gitconfig")
	os.WriteFile(noManagedPath, []byte(`[user]
    name = Test
`), 0o644)

	has, err = HasUserAfterIncludeIfs(noManagedPath)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected false when no managed blocks")
	}
}

func TestAddIncludeIf_SyncsOrdering(t *testing.T) {
	tmp := t.TempDir()
	gcPath := filepath.Join(tmp, ".gitconfig")

	// Start with [user] at top, then add a managed block.
	// Simulate: a managed block already exists, then [user] was appended by another tool.
	content := `[includeIf "gitdir:/code/old/"] # managed by gh-identity
    path = /cfg/old.gitconfig
[user]
    name = Global User
    email = global@test.com
`
	if err := os.WriteFile(gcPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Adding a new includeIf should trigger sync.
	if err := AddIncludeIf(gcPath, "/code/new", "/cfg/new.gitconfig"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(gcPath)
	result := string(data)

	// Both managed blocks should be after [user].
	userIdx := strings.Index(result, "[user]")
	oldIdx := strings.Index(result, `[includeIf "gitdir:/code/old/"]`)
	newIdx := strings.Index(result, `[includeIf "gitdir:/code/new/"]`)

	if oldIdx <= userIdx {
		t.Errorf("old includeIf should be after [user]; user=%d old=%d", userIdx, oldIdx)
	}
	if newIdx <= userIdx {
		t.Errorf("new includeIf should be after [user]; user=%d new=%d", userIdx, newIdx)
	}
}
