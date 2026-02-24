package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mockAuth implements ghauth.Auth for testing.
type mockAuth struct {
	users      []string
	activeUser string
	tokens     map[string]string
	err        error
}

func (m *mockAuth) Token(username string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if tok, ok := m.tokens[username]; ok {
		return tok, nil
	}
	return "mock-token-" + username, nil
}

func (m *mockAuth) AuthenticatedUsers() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}

func (m *mockAuth) ActiveUser() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.activeUser, nil
}

func setupTestEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", dir)
	return dir
}

func writeProfiles(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "profiles.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeBindings(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "bindings.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestNewRootCmd verifies the command tree is properly wired.
func TestNewRootCmd(t *testing.T) {
	root := NewRootCmd()
	if root == nil {
		t.Fatal("NewRootCmd() returned nil")
	}
	if root.Use != "identity" {
		t.Errorf("Use = %q, want %q", root.Use, "identity")
	}

	// Verify all subcommands are registered.
	wantCmds := []string{"init", "profile", "bind", "unbind", "switch", "status", "clone", "doctor"}
	cmds := make(map[string]bool)
	for _, c := range root.Commands() {
		cmds[c.Use] = true
	}
	for _, want := range wantCmds {
		found := false
		for use := range cmds {
			if len(use) >= len(want) && use[:len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

// TestRepoToDir tests the clone directory name extraction.
func TestRepoToDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"owner/repo", "repo"},
		{"https://github.com/owner/repo.git", "repo"},
		{"https://github.com/owner/repo", "repo"},
		{"myrepo", "myrepo"},
		{"org/sub/repo.git", "repo"},
	}
	for _, tt := range tests {
		got := repoToDir(tt.input)
		if got != tt.want {
			t.Errorf("repoToDir(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestContains tests the contains helper function.
func TestContains(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "", true},
		{"short", "longer string", false},
	}
	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

// TestContainsStr tests the containsStr helper.
func TestContainsStr(t *testing.T) {
	if !containsStr("foobar", "oba") {
		t.Error("expected true for substring match")
	}
	if containsStr("foo", "bar") {
		t.Error("expected false for non-match")
	}
}

// TestDetectShell tests shell detection from SHELL env.
func TestDetectShell(t *testing.T) {
	tests := []struct {
		shellEnv string
		want     string
	}{
		{"/usr/bin/fish", "fish"},
		{"/bin/bash", "bash"},
		{"/bin/zsh", "zsh"},
		{"/usr/local/bin/fish", "fish"},
		{"", "bash"},
		{"/bin/sh", "bash"},
	}
	for _, tt := range tests {
		t.Run(tt.shellEnv, func(t *testing.T) {
			t.Setenv("SHELL", tt.shellEnv)
			got := detectShell()
			if got != tt.want {
				t.Errorf("detectShell() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRunProfileList tests the profile list command.
func TestRunProfileList(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  personal:
    gh_user: user1
    git_name: User One
    git_email: user1@example.com
  work:
    gh_user: user2
    git_name: User Two
    git_email: user2@company.com
default: personal`)

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProfileList()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "personal") {
		t.Error("expected 'personal' in output")
	}
	if !containsStr(output, "work") {
		t.Error("expected 'work' in output")
	}
	if !containsStr(output, "user1") {
		t.Error("expected 'user1' in output")
	}
}

// TestRunProfileList_Empty tests list with no profiles.
func TestRunProfileList_Empty(t *testing.T) {
	setupTestEnv(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProfileList()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "No profiles configured") {
		t.Error("expected 'No profiles configured' message")
	}
}

// TestRunBind tests binding a directory to a profile.
func TestRunBind(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  work:
    gh_user: user2
    git_name: User Two
    git_email: user2@company.com`)

	// Create a temp gitconfig to avoid modifying the real one.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	bindDir := t.TempDir()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runBind(bindDir, "work")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	// Verify binding was created.
	data, err := os.ReadFile(filepath.Join(dir, "bindings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "work") {
		t.Error("expected 'work' in bindings.yml")
	}
}

// TestRunBind_InvalidProfile tests binding with nonexistent profile.
func TestRunBind_InvalidProfile(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles: {}`)

	err := runBind("/some/dir", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

// TestRunUnbind tests unbinding a directory.
func TestRunUnbind(t *testing.T) {
	dir := setupTestEnv(t)
	bindDir := t.TempDir()
	writeProfiles(t, dir, `profiles:
  work:
    gh_user: user2
    git_name: User Two
    git_email: user2@company.com`)
	writeBindings(t, dir, `bindings:
  - path: `+bindDir+`
    profile: work`)

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runUnbind(bindDir)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}
}

// TestRunUnbind_NotBound tests unbinding a directory that isn't bound.
func TestRunUnbind_NotBound(t *testing.T) {
	setupTestEnv(t)
	t.Setenv("HOME", t.TempDir())

	err := runUnbind("/some/unbound/dir")
	if err == nil {
		t.Error("expected error unbinding unbound directory")
	}
}

// TestRunSwitch tests the switch command output.
func TestRunSwitch(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  personal:
    gh_user: user1
    git_name: User One
    git_email: user1@example.com`)

	auth := &mockAuth{
		tokens: map[string]string{"user1": "test-token-123"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSwitch(auth, "personal")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "test-token-123") {
		t.Error("expected token in switch output")
	}
	if !containsStr(output, "GH_IDENTITY_PROFILE") {
		t.Error("expected profile env var in output")
	}
}

// TestRunSwitch_InvalidProfile tests switch with nonexistent profile.
func TestRunSwitch_InvalidProfile(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles: {}`)

	auth := &mockAuth{}
	err := runSwitch(auth, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

// TestRunStatus tests the status command.
func TestRunStatus(t *testing.T) {
	dir := setupTestEnv(t)
	pwd, _ := os.Getwd()
	writeProfiles(t, dir, `profiles:
  work:
    gh_user: user2
    git_name: User Two
    git_email: user2@company.com
    ssh_key: ~/.ssh/id_test
default: work`)
	writeBindings(t, dir, `bindings:
  - path: `+pwd+`
    profile: work`)

	auth := &mockAuth{activeUser: "user2"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "work") {
		t.Error("expected 'work' profile in status")
	}
	if !containsStr(output, "user2") {
		t.Error("expected 'user2' in status")
	}
}

// TestRunStatus_NoProfile tests status with no active profile.
func TestRunStatus_NoProfile(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles: {}`)
	writeBindings(t, dir, `bindings: []`)
	t.Setenv("GH_IDENTITY_PROFILE", "")

	auth := &mockAuth{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "No active profile") {
		t.Error("expected 'No active profile' message")
	}
}

// TestRunStatus_EnvOverride tests status with GH_IDENTITY_PROFILE env override.
func TestRunStatus_EnvOverride(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  override:
    gh_user: user3
    git_name: User Three
    git_email: user3@example.com`)
	writeBindings(t, dir, `bindings: []`)
	t.Setenv("GH_IDENTITY_PROFILE", "override")

	auth := &mockAuth{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "override") {
		t.Error("expected 'override' profile from env")
	}
	if !containsStr(output, "environment") {
		t.Error("expected 'environment' source indicator")
	}
}

// TestRunProfileRemove tests removing a profile.
func TestRunProfileRemove(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  todelete:
    gh_user: user1
    git_name: Test
    git_email: test@test.com`)
	writeBindings(t, dir, `bindings:
  - path: /some/path
    profile: todelete`)

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runProfileRemove("todelete")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	// Verify profile was removed.
	data, _ := os.ReadFile(filepath.Join(dir, "profiles.yml"))
	if containsStr(string(data), "todelete") {
		t.Error("profile should have been removed")
	}
}

// TestRunProfileRemove_NotFound tests removing nonexistent profile.
func TestRunProfileRemove_NotFound(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles: {}`)

	err := runProfileRemove("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent profile")
	}
}

// TestReadLine tests the readLine helper.
func TestReadLine(t *testing.T) {
	input := bytes.NewBufferString("hello world\n")
	reader := bufio.NewReader(input)
	got := readLine(reader)
	if got != "hello world" {
		t.Errorf("readLine() = %q, want %q", got, "hello world")
	}
}

// TestRunProfileAdd tests the profile add command with stdin input.
func TestRunProfileAdd(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	writeProfiles(t, dir, `profiles: {}`)

	// Provide stdin input for the interactive prompts.
	oldStdin := os.Stdin
	input := "testuser\nTest User\ntest@example.com\n~/.ssh/id_test\n"
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	auth := &mockAuth{users: []string{"testuser"}}

	oldOut := os.Stdout
	_, outW, _ := os.Pipe()
	os.Stdout = outW

	err := runProfileAdd(auth, "newprofile")

	outW.Close()
	os.Stdout = oldOut

	if err != nil {
		t.Fatal(err)
	}

	// Verify profile was saved.
	data, err := os.ReadFile(filepath.Join(dir, "profiles.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "newprofile") {
		t.Error("expected 'newprofile' in profiles.yml")
	}
	if !containsStr(string(data), "testuser") {
		t.Error("expected 'testuser' in profiles.yml")
	}
}

// TestRunProfileAdd_Duplicate tests adding a profile that already exists.
func TestRunProfileAdd_Duplicate(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  existing:
    gh_user: user1
    git_name: Existing
    git_email: e@e.com`)

	auth := &mockAuth{}
	err := runProfileAdd(auth, "existing")
	if err == nil {
		t.Error("expected error for duplicate profile")
	}
	if !containsStr(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got %v", err)
	}
}

// TestInstallShellHook_Bash tests shell hook installation for bash.
func TestInstallShellHook_Bash(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELL", "/bin/bash")

	// Create bin dir with config dir.
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)

	err := installShellHook()
	if err != nil {
		t.Fatal(err)
	}

	// Verify .bashrc was created with hook.
	data, err := os.ReadFile(filepath.Join(tmpHome, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "gh-identity hook") {
		t.Error("expected 'gh-identity hook' in .bashrc")
	}
}

// TestInstallShellHook_Zsh tests shell hook installation for zsh.
func TestInstallShellHook_Zsh(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELL", "/bin/zsh")

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)

	err := installShellHook()
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpHome, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "gh-identity hook") {
		t.Error("expected 'gh-identity hook' in .zshrc")
	}
}

// TestInstallShellHook_Fish tests shell hook installation for fish.
func TestInstallShellHook_Fish(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELL", "/usr/bin/fish")

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)

	err := installShellHook()
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpHome, ".config", "fish", "conf.d", "gh-identity.fish"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "gh-identity hook") {
		t.Error("expected 'gh-identity hook' in fish config")
	}
}

// TestInstallShellHook_AlreadyInstalled tests idempotency.
func TestInstallShellHook_AlreadyInstalled(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELL", "/bin/bash")

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)

	// Pre-create .bashrc with existing hook.
	os.WriteFile(filepath.Join(tmpHome, ".bashrc"), []byte("# gh-identity hook\neval ...\n"), 0o644)

	err := installShellHook()
	if err != nil {
		t.Fatal(err)
	}

	// Should not duplicate the hook.
	data, _ := os.ReadFile(filepath.Join(tmpHome, ".bashrc"))
	count := 0
	for i := 0; i <= len(string(data))-len("gh-identity hook"); i++ {
		if string(data)[i:i+len("gh-identity hook")] == "gh-identity hook" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 hook entry, got %d", count)
	}
}

// TestInstallHookBinary_NotFound tests installHookBinary when binary doesn't exist.
func TestInstallHookBinary_NotFound(t *testing.T) {
	setupTestEnv(t)

	err := installHookBinary()
	if err == nil {
		t.Error("expected error when hook binary not found")
	}
}

// TestRunProfileList_ActiveProfile tests list highlighting active profile.
func TestRunProfileList_ActiveProfile(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  work:
    gh_user: user2
    git_name: User Two
    git_email: user2@company.com
    ssh_key: ~/.ssh/id_work`)
	t.Setenv("GH_IDENTITY_PROFILE", "work")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProfileList()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "*") {
		t.Error("expected '*' indicator for active profile")
	}
	if !containsStr(output, "ssh_key") {
		t.Error("expected ssh_key in output")
	}
}

// TestRunSwitch_TokenError tests switch when token retrieval fails.
func TestRunSwitch_TokenError(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  broken:
    gh_user: baduser
    git_name: Broken
    git_email: broken@test.com`)

	auth := &mockAuth{
		err: fmt.Errorf("token error"),
	}

	err := runSwitch(auth, "broken")
	if err == nil {
		t.Error("expected error when token fails")
	}
}

// TestRunDoctor tests the doctor command with various setups.
func TestRunDoctor_NoConfig(t *testing.T) {
	// Point to a non-existent config dir.
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", filepath.Join(tmp, "nonexistent"))
	t.Setenv("HOME", t.TempDir())

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "Config directory does not exist") {
		t.Error("expected config dir missing message")
	}
}

// TestRunDoctor_ValidSetup tests doctor with a valid configuration.
func TestRunDoctor_ValidSetup(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles:
  test:
    gh_user: user1
    git_name: Test
    git_email: test@test.com`)
	writeBindings(t, dir, `bindings:
  - path: /valid/path
    profile: test`)

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "1 profile(s) configured") {
		t.Error("expected profiles configured message")
	}
}

// TestRunDoctor_InvalidProfile tests doctor with invalid profile (missing auth).
func TestRunDoctor_InvalidProfile(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles:
  bad:
    gh_user: unknown_user
    git_name: Bad
    git_email: bad@bad.com`)
	writeBindings(t, dir, `bindings: []`)

	// Auth knows about user1 but profile references unknown_user.
	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "not authenticated") {
		t.Error("expected unauthenticated user warning")
	}
}

// TestRunDoctor_BadBinding tests doctor with a binding referencing nonexistent profile.
func TestRunDoctor_BadBinding(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles:
  good:
    gh_user: user1
    git_name: Good
    git_email: good@good.com`)
	writeBindings(t, dir, `bindings:
  - path: /some/path
    profile: nonexistent`)

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "non-existent profile") {
		t.Error("expected non-existent profile warning")
	}
}

// TestRunDoctor_EmptyProfiles tests doctor with no profiles.
func TestRunDoctor_EmptyProfiles(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles: {}`)

	auth := &mockAuth{users: []string{}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "No profiles configured") {
		t.Error("expected no profiles message")
	}
}

// TestRunDoctor_ValidationErrors tests doctor with invalid profile fields.
func TestRunDoctor_ValidationErrors(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles:
  bad:
    gh_user: ""
    git_name: ""
    git_email: ""`)

	auth := &mockAuth{users: []string{}}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "is required") {
		t.Error("expected validation error messages")
	}
}

// TestRunInit tests the init command with mock auth and stdin.
func TestRunInit(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELL", "/bin/bash")

	// Provide stdin for: profile name, git name, git email, ssh key, default profile.
	oldStdin := os.Stdin
	input := "personal\nJohn Doe\njohn@example.com\n\npersonal\n"
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	auth := &mockAuth{users: []string{"user1"}}

	oldOut := os.Stdout
	_, outW, _ := os.Pipe()
	os.Stdout = outW

	err := runInit(auth)

	outW.Close()
	os.Stdout = oldOut

	if err != nil {
		t.Fatal(err)
	}

	// Verify profiles were saved.
	data, err := os.ReadFile(filepath.Join(dir, "profiles.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(data), "personal") {
		t.Error("expected 'personal' in profiles.yml")
	}
	if !containsStr(string(data), "user1") {
		t.Error("expected 'user1' in profiles.yml")
	}
}

// TestRunInit_NoUsers tests init when no gh accounts are authenticated.
func TestRunInit_NoUsers(t *testing.T) {
	setupTestEnv(t)

	auth := &mockAuth{users: []string{}}

	oldOut := os.Stdout
	_, outW, _ := os.Pipe()
	os.Stdout = outW

	err := runInit(auth)

	outW.Close()
	os.Stdout = oldOut

	if err != nil {
		t.Fatal(err)
	}
}

// TestRunInit_AuthError tests init when auth fails.
func TestRunInit_AuthError(t *testing.T) {
	setupTestEnv(t)

	auth := &mockAuth{err: fmt.Errorf("auth failed")}

	oldOut := os.Stdout
	_, outW, _ := os.Pipe()
	os.Stdout = outW

	err := runInit(auth)

	outW.Close()
	os.Stdout = oldOut

	if err == nil {
		t.Error("expected error when auth fails")
	}
}

// TestRunInit_MultipleUsers tests init with multiple authenticated users.
func TestRunInit_MultipleUsers(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELL", "/bin/bash")

	// Input for 2 users: name1, gitname1, email1, sshkey1, name2, gitname2, email2, sshkey2, default
	oldStdin := os.Stdin
	input := "work\nWork User\nwork@company.com\n~/.ssh/id_work\npersonal\nPersonal User\nme@home.com\n\nwork\n"
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	auth := &mockAuth{users: []string{"workuser", "personaluser"}}

	oldOut := os.Stdout
	_, outW, _ := os.Pipe()
	os.Stdout = outW

	err := runInit(auth)

	outW.Close()
	os.Stdout = oldOut

	if err != nil {
		t.Fatal(err)
	}

	// Verify profiles were saved.
	data, _ := os.ReadFile(filepath.Join(dir, "profiles.yml"))
	if !containsStr(string(data), "work") {
		t.Error("expected 'work' in profiles.yml")
	}
	if !containsStr(string(data), "personal") {
		t.Error("expected 'personal' in profiles.yml")
	}
}

// TestRunDoctor_SSHKeyValid tests doctor with a valid SSH key.
func TestRunDoctor_SSHKeyValid(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a fake SSH key with correct permissions.
	sshDir := filepath.Join(tmpHome, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	keyPath := filepath.Join(sshDir, "id_test")
	os.WriteFile(keyPath, []byte("fake-key"), 0o600)

	writeProfiles(t, dir, `profiles:
  sshprof:
    gh_user: user1
    git_name: SSH
    git_email: ssh@test.com
    ssh_key: `+keyPath)
	writeBindings(t, dir, `bindings: []`)

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r2, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r2)
	output := buf.String()

	if !containsStr(output, "SSH key OK") {
		t.Error("expected 'SSH key OK' message")
	}
}

// TestRunDoctor_SSHKeyMissing tests doctor with a missing SSH key.
func TestRunDoctor_SSHKeyMissing(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles:
  sshprof:
    gh_user: user1
    git_name: SSH
    git_email: ssh@test.com
    ssh_key: /nonexistent/key`)
	writeBindings(t, dir, `bindings: []`)

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r2, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r2)
	output := buf.String()

	if !containsStr(output, "SSH key not found") {
		t.Error("expected 'SSH key not found' message")
	}
}

// TestRunDoctor_SSHKeyPermissive tests doctor with overly permissive SSH key.
func TestRunDoctor_SSHKeyPermissive(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a fake SSH key with overly permissive permissions.
	sshDir := filepath.Join(tmpHome, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	keyPath := filepath.Join(sshDir, "id_test")
	os.WriteFile(keyPath, []byte("fake-key"), 0o644)

	writeProfiles(t, dir, `profiles:
  sshprof:
    gh_user: user1
    git_name: SSH
    git_email: ssh@test.com
    ssh_key: `+keyPath)
	writeBindings(t, dir, `bindings: []`)

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r2, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r2)
	output := buf.String()

	if !containsStr(output, "permissive") {
		t.Error("expected 'permissive' warning message")
	}
}

// TestRunDoctor_AllChecksPassed tests doctor with everything configured correctly.
func TestRunDoctor_AllChecksPassed(t *testing.T) {
	dir := setupTestEnv(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	writeProfiles(t, dir, `profiles:
  good:
    gh_user: user1
    git_name: Good
    git_email: good@good.com`)
	writeBindings(t, dir, `bindings: []`)

	// Create hook binary.
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, "gh-identity-hook"), []byte("fake"), 0o755)

	// Create shell hook in bashrc.
	os.WriteFile(filepath.Join(tmpHome, ".bashrc"), []byte("# gh-identity hook\neval ..."), 0o644)

	auth := &mockAuth{users: []string{"user1"}}

	old := os.Stdout
	r2, w, _ := os.Pipe()
	os.Stdout = w

	err := runDoctor(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r2)
	output := buf.String()

	if !containsStr(output, "All checks passed") {
		t.Errorf("expected 'All checks passed', got:\n%s", output)
	}
}

// TestRunSwitch_WithSSHKey tests switch with a profile that has an SSH key.
func TestRunSwitch_WithSSHKey(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  sshuser:
    gh_user: user1
    git_name: SSH User
    git_email: ssh@example.com
    ssh_key: ~/.ssh/id_test`)

	auth := &mockAuth{
		tokens: map[string]string{"user1": "ssh-token"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSwitch(auth, "sshuser")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "GIT_SSH_COMMAND") {
		t.Error("expected GIT_SSH_COMMAND in output for profile with SSH key")
	}
}

// TestRunStatus_DefaultProfile tests status with default profile fallback.
func TestRunStatus_DefaultProfile(t *testing.T) {
	dir := setupTestEnv(t)
	writeProfiles(t, dir, `profiles:
  fallback:
    gh_user: user1
    git_name: Fallback
    git_email: fb@example.com
default: fallback`)
	writeBindings(t, dir, `bindings: []`)
	t.Setenv("GH_IDENTITY_PROFILE", "")

	auth := &mockAuth{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(auth)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !containsStr(output, "fallback") {
		t.Error("expected 'fallback' profile")
	}
	if !containsStr(output, "default profile") {
		t.Error("expected 'default profile' source")
	}
}
