package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile represents a named identity bundle.
type Profile struct {
	GHUser   string `yaml:"gh_user"`
	GitName  string `yaml:"git_name"`
	GitEmail string `yaml:"git_email"`
	SSHKey   string `yaml:"ssh_key,omitempty"`
}

// ProfilesFile is the top-level structure of profiles.yml.
type ProfilesFile struct {
	Profiles map[string]Profile `yaml:"profiles"`
	Default  string             `yaml:"default,omitempty"`
}

// ProfilesPath returns the path to profiles.yml.
func ProfilesPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles.yml"), nil
}

// LoadProfiles reads and parses profiles.yml.
// Returns an empty ProfilesFile (not an error) if the file does not exist.
func LoadProfiles() (*ProfilesFile, error) {
	path, err := ProfilesPath()
	if err != nil {
		return nil, err
	}
	return LoadProfilesFrom(path)
}

// LoadProfilesFrom reads profiles from the given path.
func LoadProfilesFrom(path string) (*ProfilesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProfilesFile{Profiles: make(map[string]Profile)}, nil
		}
		return nil, fmt.Errorf("reading profiles: %w", err)
	}

	var pf ProfilesFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing profiles: %w", err)
	}
	if pf.Profiles == nil {
		pf.Profiles = make(map[string]Profile)
	}
	return &pf, nil
}

// Save writes the profiles file to disk.
func (pf *ProfilesFile) Save() error {
	path, err := ProfilesPath()
	if err != nil {
		return err
	}
	return pf.SaveTo(path)
}

// SaveTo writes the profiles file to the given path.
func (pf *ProfilesFile) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := yaml.Marshal(pf)
	if err != nil {
		return fmt.Errorf("marshalling profiles: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing profiles: %w", err)
	}
	return nil
}

// GetProfile returns the named profile, or an error if not found.
func (pf *ProfilesFile) GetProfile(name string) (Profile, error) {
	p, ok := pf.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// AddProfile adds or updates a named profile.
func (pf *ProfilesFile) AddProfile(name string, p Profile) {
	pf.Profiles[name] = p
}

// RemoveProfile removes a profile by name.
func (pf *ProfilesFile) RemoveProfile(name string) error {
	if _, ok := pf.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(pf.Profiles, name)
	if pf.Default == name {
		pf.Default = ""
	}
	return nil
}

// Validate checks that all profiles have required fields.
func (pf *ProfilesFile) Validate() []string {
	var errs []string
	for name, p := range pf.Profiles {
		if p.GHUser == "" {
			errs = append(errs, fmt.Sprintf("profile %q: gh_user is required", name))
		}
		if p.GitName == "" {
			errs = append(errs, fmt.Sprintf("profile %q: git_name is required", name))
		}
		if p.GitEmail == "" {
			errs = append(errs, fmt.Sprintf("profile %q: git_email is required", name))
		}
	}
	return errs
}
