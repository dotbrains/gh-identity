package config

import (
	"path/filepath"
	"testing"
)

func TestBindingsRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bindings.yml")

	bf := &BindingsFile{
		Bindings: []Binding{
			{Path: "/home/user/code/personal", Profile: "personal"},
			{Path: "/home/user/code/work", Profile: "work"},
		},
	}

	if err := bf.SaveTo(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadBindingsFrom(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(loaded.Bindings))
	}
	if loaded.Bindings[0].Profile != "personal" {
		t.Errorf("expected profile %q, got %q", "personal", loaded.Bindings[0].Profile)
	}
}

func TestLoadBindingsFrom_NotExist(t *testing.T) {
	bf, err := LoadBindingsFrom("/nonexistent/bindings.yml")
	if err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 0 {
		t.Errorf("expected 0 bindings, got %d", len(bf.Bindings))
	}
}

func TestAddBinding(t *testing.T) {
	tmp := t.TempDir()
	dir1 := filepath.Join(tmp, "proj1")
	dir2 := filepath.Join(tmp, "proj2")

	bf := &BindingsFile{}

	if err := bf.AddBinding(dir1, "personal"); err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bf.Bindings))
	}

	// Adding same path should replace, not duplicate.
	if err := bf.AddBinding(dir1, "work"); err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 1 {
		t.Fatalf("expected 1 binding after replace, got %d", len(bf.Bindings))
	}
	if bf.Bindings[0].Profile != "work" {
		t.Errorf("expected profile %q, got %q", "work", bf.Bindings[0].Profile)
	}

	// Adding different path.
	if err := bf.AddBinding(dir2, "personal"); err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bf.Bindings))
	}
}

func TestRemoveBinding(t *testing.T) {
	tmp := t.TempDir()
	dir1 := filepath.Join(tmp, "proj1")

	bf := &BindingsFile{}
	_ = bf.AddBinding(dir1, "personal")

	if err := bf.RemoveBinding(dir1); err != nil {
		t.Fatal(err)
	}
	if len(bf.Bindings) != 0 {
		t.Fatalf("expected 0 bindings, got %d", len(bf.Bindings))
	}

	if err := bf.RemoveBinding(dir1); err == nil {
		t.Error("expected error removing nonexistent binding")
	}
}

func TestFindBinding(t *testing.T) {
	tmp := t.TempDir()
	dir1 := filepath.Join(tmp, "proj1")

	bf := &BindingsFile{}
	_ = bf.AddBinding(dir1, "personal")

	if profile := bf.FindBinding(dir1); profile != "personal" {
		t.Errorf("FindBinding() = %q, want %q", profile, "personal")
	}
	if profile := bf.FindBinding("/nonexistent"); profile != "" {
		t.Errorf("FindBinding() = %q, want empty", profile)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "absolute", input: "/usr/local/bin"},
		{name: "relative", input: "some/path"},
		{name: "tilde", input: "~/code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !filepath.IsAbs(result) {
				t.Errorf("ExpandPath(%q) = %q, expected absolute path", tt.input, result)
			}
		})
	}
}
