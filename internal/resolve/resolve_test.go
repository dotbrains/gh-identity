package resolve

import (
	"path/filepath"
	"testing"

	"github.com/dotbrains/gh-identity/internal/config"
)

func TestForDirectory_ExactMatch(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "code", "personal")

	bf := &config.BindingsFile{
		Bindings: []config.Binding{
			{Path: dir, Profile: "personal"},
		},
	}

	result, err := ForDirectory(dir, bf, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile != "personal" {
		t.Errorf("Profile = %q, want %q", result.Profile, "personal")
	}
	if result.BoundPath != dir {
		t.Errorf("BoundPath = %q, want %q", result.BoundPath, dir)
	}
}

func TestForDirectory_ChildMatch(t *testing.T) {
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "code", "org")
	child := filepath.Join(parent, "repo", "src")

	bf := &config.BindingsFile{
		Bindings: []config.Binding{
			{Path: parent, Profile: "work"},
		},
	}

	result, err := ForDirectory(child, bf, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile != "work" {
		t.Errorf("Profile = %q, want %q", result.Profile, "work")
	}
}

func TestForDirectory_DeepestMatch(t *testing.T) {
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "code")
	child := filepath.Join(tmp, "code", "org")
	grandchild := filepath.Join(tmp, "code", "org", "repo")

	bf := &config.BindingsFile{
		Bindings: []config.Binding{
			{Path: parent, Profile: "default"},
			{Path: child, Profile: "org"},
		},
	}

	result, err := ForDirectory(grandchild, bf, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile != "org" {
		t.Errorf("Profile = %q, want %q (deepest match)", result.Profile, "org")
	}
}

func TestForDirectory_NoMatch_Default(t *testing.T) {
	bf := &config.BindingsFile{}

	result, err := ForDirectory("/some/random/dir", bf, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile != "fallback" {
		t.Errorf("Profile = %q, want %q", result.Profile, "fallback")
	}
	if !result.IsDefault {
		t.Error("expected IsDefault = true")
	}
}

func TestForDirectory_NoMatch_NoDefault(t *testing.T) {
	bf := &config.BindingsFile{}

	result, err := ForDirectory("/some/random/dir", bf, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile != "" {
		t.Errorf("Profile = %q, want empty", result.Profile)
	}
}

func TestIsSubpath(t *testing.T) {
	tests := []struct {
		child  string
		parent string
		want   bool
	}{
		{"/a/b/c", "/a/b", true},
		{"/a/b", "/a/b", true},
		{"/a/b", "/a/bc", false},
		{"/a/bc", "/a/b", false},
		{"/x/y/z", "/a/b", false},
	}
	for _, tt := range tests {
		got := isSubpath(tt.child, tt.parent)
		if got != tt.want {
			t.Errorf("isSubpath(%q, %q) = %v, want %v", tt.child, tt.parent, got, tt.want)
		}
	}
}
