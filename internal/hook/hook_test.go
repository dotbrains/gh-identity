package hook

import (
	"strings"
	"testing"
)

func TestFormatOutput_Fish(t *testing.T) {
	env := EnvOutput{
		GHUser:            "testuser",
		GitAuthorName:     "Test User",
		GitAuthorEmail:    "test@example.com",
		GitCommitterName:  "Test User",
		GitCommitterEmail: "test@example.com",
		GHIdentityProfile: "personal",
	}

	output := formatOutput(Fish, env)

	if !strings.Contains(output, "set -e GH_TOKEN") {
		t.Error("missing fish GH_TOKEN unset")
	}
	if !strings.Contains(output, "gh auth switch --user testuser") {
		t.Error("missing gh auth switch command")
	}
	if !strings.Contains(output, "set -gx GH_IDENTITY_PROFILE") {
		t.Error("missing fish GH_IDENTITY_PROFILE export")
	}
	if strings.Contains(output, "export ") {
		t.Error("fish output should not contain 'export'")
	}
}

func TestFormatOutput_Bash(t *testing.T) {
	env := EnvOutput{
		GHUser:            "testuser",
		GitAuthorName:     "Test User",
		GitAuthorEmail:    "test@example.com",
		GitCommitterName:  "Test User",
		GitCommitterEmail: "test@example.com",
		GHIdentityProfile: "personal",
	}

	output := formatOutput(Bash, env)

	if !strings.Contains(output, "unset GH_TOKEN") {
		t.Error("missing bash GH_TOKEN unset")
	}
	if !strings.Contains(output, "gh auth switch --user testuser") {
		t.Error("missing gh auth switch command")
	}
	if !strings.Contains(output, "export GH_IDENTITY_PROFILE=") {
		t.Error("missing bash GH_IDENTITY_PROFILE export")
	}
	if strings.Contains(output, "set -gx") {
		t.Error("bash output should not contain 'set -gx'")
	}
}

func TestFormatOutput_SSHCommand(t *testing.T) {
	env := EnvOutput{
		GHUser:            "testuser",
		GitAuthorName:     "Test",
		GitAuthorEmail:    "test@test.com",
		GitCommitterName:  "Test",
		GitCommitterEmail: "test@test.com",
		GHIdentityProfile: "work",
		GHSSHCommand:      "ssh -i /home/user/.ssh/id_work -o IdentitiesOnly=yes",
	}

	output := formatOutput(Fish, env)
	if !strings.Contains(output, "GIT_SSH_COMMAND") {
		t.Error("missing GIT_SSH_COMMAND export when SSH key is set")
	}
}

func TestFormatOutput_NoSSHCommand(t *testing.T) {
	env := EnvOutput{
		GHUser:            "testuser",
		GitAuthorName:     "Test",
		GitAuthorEmail:    "test@test.com",
		GitCommitterName:  "Test",
		GitCommitterEmail: "test@test.com",
		GHIdentityProfile: "work",
	}

	output := formatOutput(Fish, env)
	if strings.Contains(output, "GIT_SSH_COMMAND") {
		t.Error("GIT_SSH_COMMAND should not be set when SSH key is empty")
	}
}
