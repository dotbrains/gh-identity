package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir_EnvVar(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != tmp {
		t.Errorf("Dir() = %q, want %q", dir, tmp)
	}
}

func TestDir_XDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, DefaultConfigDir)
	if dir != want {
		t.Errorf("Dir() = %q, want %q", dir, want)
	}
}

func TestEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", filepath.Join(tmp, "sub", "dir"))
	dir, err := EnsureDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("EnsureDir() did not create directory %q", dir)
	}
}

func TestGitConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	dir, err := GitConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "git")
	if dir != want {
		t.Errorf("GitConfigDir() = %q, want %q", dir, want)
	}
}

func TestBinDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	dir, err := BinDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "bin")
	if dir != want {
		t.Errorf("BinDir() = %q, want %q", dir, want)
	}
}

func TestAskPassPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GH_IDENTITY_CONFIG_DIR", tmp)
	p, err := AskPassPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "bin", "gh-identity-askpass")
	if p != want {
		t.Errorf("AskPassPath() = %q, want %q", p, want)
	}
}
