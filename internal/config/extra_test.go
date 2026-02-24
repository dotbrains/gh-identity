package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureGitConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	dir, err := EnsureGitConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "git")
	if dir != want {
		t.Errorf("EnsureGitConfigDir() = %q, want %q", dir, want)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("directory should have been created")
	}
}

func TestProfilesPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	path, err := ProfilesPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "profiles.yml")
	if path != want {
		t.Errorf("ProfilesPath() = %q, want %q", path, want)
	}
}

func TestBindingsPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	path, err := BindingsPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "bindings.yml")
	if path != want {
		t.Errorf("BindingsPath() = %q, want %q", path, want)
	}
}

func TestLoadProfiles_ViaDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)

	// No file yet — should return empty.
	pf, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(pf.Profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(pf.Profiles))
	}

	// Save and reload.
	pf.AddProfile("test", Profile{GHUser: "u", GitName: "n", GitEmail: "e"})
	if err := pf.Save(); err != nil {
		t.Fatal(err)
	}

	pf2, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(pf2.Profiles) != 1 {
		t.Errorf("expected 1 profile after save/load, got %d", len(pf2.Profiles))
	}
}

func TestLoadBindings_ViaDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)

	bf, err := LoadBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 0 {
		t.Errorf("expected 0 bindings, got %d", len(bf.Bindings))
	}

	// Save and reload.
	_ = bf.AddBinding("/test/path", "prof")
	if err := bf.Save(); err != nil {
		t.Fatal(err)
	}

	bf2, err := LoadBindings()
	if err != nil {
		t.Fatal(err)
	}
	if len(bf2.Bindings) != 1 {
		t.Errorf("expected 1 binding after save/load, got %d", len(bf2.Bindings))
	}
}

func TestDir_DefaultHome(t *testing.T) {
	t.Setenv("GH_IDENTITY_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Error("Dir() returned empty string")
	}
}

func TestSaveTo_CreatesDirIfNeeded(t *testing.T) {
	tmp := t.TempDir()
	nested := filepath.Join(tmp, "a", "b", "c", "profiles.yml")

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"test": {GHUser: "u", GitName: "n", GitEmail: "e"},
		},
	}

	if err := pf.SaveTo(nested); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("file should have been created with parent dirs")
	}
}

func TestBindingsSaveTo_CreatesDirIfNeeded(t *testing.T) {
	tmp := t.TempDir()
	nested := filepath.Join(tmp, "x", "y", "bindings.yml")

	bf := &BindingsFile{
		Bindings: []Binding{{Path: "/test", Profile: "p"}},
	}

	if err := bf.SaveTo(nested); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("file should have been created with parent dirs")
	}
}

func TestExpandPath_Tilde(t *testing.T) {
	result, err := ExpandPath("~/test/path")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" || result == "~/test/path" {
		t.Error("expected tilde to be expanded")
	}
}

func TestExpandPath_Absolute(t *testing.T) {
	result, err := ExpandPath("/absolute/path")
	if err != nil {
		t.Fatal(err)
	}
	if result != "/absolute/path" {
		t.Errorf("expected /absolute/path, got %q", result)
	}
}

func TestExpandPath_Relative(t *testing.T) {
	result, err := ExpandPath("relative/path")
	if err != nil {
		t.Fatal(err)
	}
	if result == "relative/path" {
		t.Error("expected relative path to be made absolute")
	}
}

func TestExpandPath_TildeOnly(t *testing.T) {
	result, err := ExpandPath("~")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if result != home {
		t.Errorf("expected %q, got %q", home, result)
	}
}

func TestAddBinding_ReplacesExisting(t *testing.T) {
	tmp := t.TempDir()
	bf := &BindingsFile{
		Bindings: []Binding{{Path: tmp, Profile: "old"}},
	}
	if err := bf.AddBinding(tmp, "new"); err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(bf.Bindings))
	}
	if bf.Bindings[0].Profile != "new" {
		t.Errorf("expected profile 'new', got %q", bf.Bindings[0].Profile)
	}
}

func TestLoadProfilesFrom_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "profiles.yml")
	os.WriteFile(badFile, []byte("{{{invalid yaml"), 0o644)

	_, err := LoadProfilesFrom(badFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadBindingsFrom_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "bindings.yml")
	os.WriteFile(badFile, []byte("{{{invalid yaml"), 0o644)

	_, err := LoadBindingsFrom(badFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
