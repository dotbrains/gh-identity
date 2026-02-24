package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Binding ties a directory path to a profile name.
type Binding struct {
	Path    string `yaml:"path"`
	Profile string `yaml:"profile"`
}

// BindingsFile is the top-level structure of bindings.yml.
type BindingsFile struct {
	Bindings []Binding `yaml:"bindings"`
}

// BindingsPath returns the path to bindings.yml.
func BindingsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bindings.yml"), nil
}

// LoadBindings reads and parses bindings.yml.
// Returns an empty BindingsFile (not an error) if the file does not exist.
func LoadBindings() (*BindingsFile, error) {
	path, err := BindingsPath()
	if err != nil {
		return nil, err
	}
	return LoadBindingsFrom(path)
}

// LoadBindingsFrom reads bindings from the given path.
func LoadBindingsFrom(path string) (*BindingsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &BindingsFile{}, nil
		}
		return nil, fmt.Errorf("reading bindings: %w", err)
	}

	var bf BindingsFile
	if err := yaml.Unmarshal(data, &bf); err != nil {
		return nil, fmt.Errorf("parsing bindings: %w", err)
	}
	return &bf, nil
}

// Save writes the bindings file to disk.
func (bf *BindingsFile) Save() error {
	path, err := BindingsPath()
	if err != nil {
		return err
	}
	return bf.SaveTo(path)
}

// SaveTo writes the bindings file to the given path.
func (bf *BindingsFile) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := yaml.Marshal(bf)
	if err != nil {
		return fmt.Errorf("marshalling bindings: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing bindings: %w", err)
	}
	return nil
}

// ExpandPath resolves ~ and cleans a path for storage.
func ExpandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		p = filepath.Join(home, p[1:])
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}
	return filepath.Clean(abs), nil
}

// AddBinding adds or replaces a binding for the given path.
func (bf *BindingsFile) AddBinding(dirPath, profile string) error {
	expanded, err := ExpandPath(dirPath)
	if err != nil {
		return err
	}

	// Replace existing binding for the same path.
	for i, b := range bf.Bindings {
		existingExpanded, err := ExpandPath(b.Path)
		if err != nil {
			continue
		}
		if existingExpanded == expanded {
			bf.Bindings[i].Profile = profile
			return nil
		}
	}

	bf.Bindings = append(bf.Bindings, Binding{Path: expanded, Profile: profile})
	return nil
}

// RemoveBinding removes the binding for the given path.
func (bf *BindingsFile) RemoveBinding(dirPath string) error {
	expanded, err := ExpandPath(dirPath)
	if err != nil {
		return err
	}

	for i, b := range bf.Bindings {
		existingExpanded, err := ExpandPath(b.Path)
		if err != nil {
			continue
		}
		if existingExpanded == expanded {
			bf.Bindings = append(bf.Bindings[:i], bf.Bindings[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("no binding found for %q", dirPath)
}

// FindBinding returns the profile name bound to the given path, or "".
func (bf *BindingsFile) FindBinding(dirPath string) string {
	expanded, err := ExpandPath(dirPath)
	if err != nil {
		return ""
	}

	for _, b := range bf.Bindings {
		existingExpanded, err := ExpandPath(b.Path)
		if err != nil {
			continue
		}
		if existingExpanded == expanded {
			return b.Profile
		}
	}
	return ""
}
