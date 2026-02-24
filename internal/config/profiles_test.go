package config

import (
	"path/filepath"
	"testing"
)

func TestProfilesRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "profiles.yml")

	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"personal": {
				GHUser:   "user1",
				GitName:  "User One",
				GitEmail: "user1@example.com",
				SSHKey:   "~/.ssh/id_ed25519",
			},
			"work": {
				GHUser:   "user2",
				GitName:  "User Two",
				GitEmail: "user2@company.com",
			},
		},
		Default: "personal",
	}

	if err := pf.SaveTo(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProfilesFrom(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(loaded.Profiles))
	}
	if loaded.Default != "personal" {
		t.Errorf("expected default %q, got %q", "personal", loaded.Default)
	}
	if loaded.Profiles["personal"].GHUser != "user1" {
		t.Errorf("expected gh_user %q, got %q", "user1", loaded.Profiles["personal"].GHUser)
	}
	if loaded.Profiles["work"].GitEmail != "user2@company.com" {
		t.Errorf("expected git_email %q, got %q", "user2@company.com", loaded.Profiles["work"].GitEmail)
	}
}

func TestLoadProfilesFrom_NotExist(t *testing.T) {
	pf, err := LoadProfilesFrom("/nonexistent/profiles.yml")
	if err != nil {
		t.Fatal(err)
	}
	if len(pf.Profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(pf.Profiles))
	}
}

func TestGetProfile(t *testing.T) {
	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"test": {GHUser: "user1", GitName: "Test", GitEmail: "test@test.com"},
		},
	}

	p, err := pf.GetProfile("test")
	if err != nil {
		t.Fatal(err)
	}
	if p.GHUser != "user1" {
		t.Errorf("expected %q, got %q", "user1", p.GHUser)
	}

	_, err = pf.GetProfile("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

func TestAddRemoveProfile(t *testing.T) {
	pf := &ProfilesFile{
		Profiles: make(map[string]Profile),
		Default:  "test",
	}

	pf.AddProfile("test", Profile{GHUser: "u", GitName: "n", GitEmail: "e"})
	if len(pf.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(pf.Profiles))
	}

	if err := pf.RemoveProfile("test"); err != nil {
		t.Fatal(err)
	}
	if len(pf.Profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(pf.Profiles))
	}
	if pf.Default != "" {
		t.Errorf("expected default to be cleared, got %q", pf.Default)
	}

	if err := pf.RemoveProfile("test"); err == nil {
		t.Error("expected error removing nonexistent profile")
	}
}

func TestValidate(t *testing.T) {
	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"good": {GHUser: "u", GitName: "n", GitEmail: "e"},
			"bad":  {GHUser: "", GitName: "", GitEmail: ""},
		},
	}

	errs := pf.Validate()
	if len(errs) != 3 {
		t.Errorf("expected 3 validation errors, got %d: %v", len(errs), errs)
	}
}
