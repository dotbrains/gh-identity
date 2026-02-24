package hook

import (
	"strings"
	"testing"
)

func TestFormatExports_Fish(t *testing.T) {
	env := EnvOutput{
		GHToken:           "token123",
		GitAuthorName:     "Test User",
		GitAuthorEmail:    "test@example.com",
		GitCommitterName:  "Test User",
		GitCommitterEmail: "test@example.com",
		GHIdentityProfile: "personal",
	}

	output := formatExports(Fish, env)

	if !strings.Contains(output, "set -gx GH_TOKEN") {
		t.Error("missing fish GH_TOKEN export")
	}
	if !strings.Contains(output, "set -gx GH_IDENTITY_PROFILE") {
		t.Error("missing fish GH_IDENTITY_PROFILE export")
	}
	if strings.Contains(output, "export ") {
		t.Error("fish output should not contain 'export'")
	}
}

func TestFormatExports_Bash(t *testing.T) {
	env := EnvOutput{
		GHToken:           "token123",
		GitAuthorName:     "Test User",
		GitAuthorEmail:    "test@example.com",
		GitCommitterName:  "Test User",
		GitCommitterEmail: "test@example.com",
		GHIdentityProfile: "personal",
	}

	output := formatExports(Bash, env)

	if !strings.Contains(output, "export GH_TOKEN=") {
		t.Error("missing bash GH_TOKEN export")
	}
	if !strings.Contains(output, "export GH_IDENTITY_PROFILE=") {
		t.Error("missing bash GH_IDENTITY_PROFILE export")
	}
	if strings.Contains(output, "set -gx") {
		t.Error("bash output should not contain 'set -gx'")
	}
}

func TestFormatExports_SSHCommand(t *testing.T) {
	env := EnvOutput{
		GHToken:           "token123",
		GitAuthorName:     "Test",
		GitAuthorEmail:    "test@test.com",
		GitCommitterName:  "Test",
		GitCommitterEmail: "test@test.com",
		GHIdentityProfile: "work",
		GHSSHCommand:      "ssh -i /home/user/.ssh/id_work -o IdentitiesOnly=yes",
	}

	output := formatExports(Fish, env)
	if !strings.Contains(output, "GIT_SSH_COMMAND") {
		t.Error("missing GIT_SSH_COMMAND export when SSH key is set")
	}
}

func TestFormatExports_NoSSHCommand(t *testing.T) {
	env := EnvOutput{
		GHToken:           "token123",
		GitAuthorName:     "Test",
		GitAuthorEmail:    "test@test.com",
		GitCommitterName:  "Test",
		GitCommitterEmail: "test@test.com",
		GHIdentityProfile: "work",
	}

	output := formatExports(Fish, env)
	if strings.Contains(output, "GIT_SSH_COMMAND") {
		t.Error("GIT_SSH_COMMAND should not be set when SSH key is empty")
	}
}
