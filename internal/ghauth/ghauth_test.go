package ghauth

import (
	"bytes"
	"fmt"
	"testing"
)

// mockExec returns a mock execFn for testing.
func mockExec(stdout, stderr string, err error) execFn {
	return func(args ...string) (bytes.Buffer, bytes.Buffer, error) {
		var outBuf, errBuf bytes.Buffer
		outBuf.WriteString(stdout)
		errBuf.WriteString(stderr)
		return outBuf, errBuf, err
	}
}

func TestParseAuthUsers(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "single account",
			output: "  Logged in to github.com account user1 (keyring)",
			want:   []string{"user1"},
		},
		{
			name: "multiple accounts",
			output: `  Logged in to github.com account user1 (keyring)
  Logged in to github.com account user2 (keyring)`,
			want: []string{"user1", "user2"},
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
		{
			name:   "no account keyword",
			output: "  Some random output without the keyword",
			want:   nil,
		},
		{
			name: "deduplicates",
			output: `  Logged in to github.com account user1 (keyring)
  Logged in to github.com account user1 (token)`,
			want: []string{"user1"},
		},
		{
			name:   "strips trailing parens",
			output: "  Logged in to github.com account user1 (keyring)",
			want:   []string{"user1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAuthUsers(tt.output)
			if len(got) != len(tt.want) {
				t.Errorf("parseAuthUsers() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseAuthUsers()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewGHAuth(t *testing.T) {
	auth := NewGHAuth()
	if auth == nil {
		t.Error("NewGHAuth() returned nil")
	}
}

func TestGHAuth_Token(t *testing.T) {
	g := &GHAuth{exec: mockExec("  gho_abc123\n", "", nil)}
	tok, err := g.Token("user1")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "gho_abc123" {
		t.Errorf("Token() = %q, want %q", tok, "gho_abc123")
	}
}

func TestGHAuth_Token_Error(t *testing.T) {
	g := &GHAuth{exec: mockExec("", "no token found", fmt.Errorf("exit 1"))}
	_, err := g.Token("baduser")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGHAuth_AuthenticatedUsers(t *testing.T) {
	output := "  Logged in to github.com account user1 (keyring)\n  Logged in to github.com account user2 (token)\n"
	g := &GHAuth{exec: mockExec(output, "", nil)}
	users, err := g.AuthenticatedUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0] != "user1" || users[1] != "user2" {
		t.Errorf("users = %v, want [user1 user2]", users)
	}
}

func TestGHAuth_AuthenticatedUsers_NotLoggedIn(t *testing.T) {
	g := &GHAuth{exec: mockExec("", "You are not logged in to any GitHub hosts. Run gh auth login to authenticate.", fmt.Errorf("exit 1"))}
	users, err := g.AuthenticatedUsers()
	// "not logged in" substring in stderr triggers nil, nil return.
	if err != nil {
		t.Fatal(err)
	}
	if users != nil {
		t.Errorf("expected nil users, got %v", users)
	}
}

func TestGHAuth_AuthenticatedUsers_Error(t *testing.T) {
	g := &GHAuth{exec: mockExec("", "some other error", fmt.Errorf("exit 1"))}
	_, err := g.AuthenticatedUsers()
	if err == nil {
		t.Error("expected error")
	}
}

func TestGHAuth_ActiveUser(t *testing.T) {
	output := "github.com\n  Logged in to github.com account activeuser (keyring)\n"
	g := &GHAuth{exec: mockExec(output, "", nil)}
	user, err := g.ActiveUser()
	if err != nil {
		t.Fatal(err)
	}
	if user != "activeuser" {
		t.Errorf("ActiveUser() = %q, want %q", user, "activeuser")
	}
}

func TestGHAuth_ActiveUser_Error(t *testing.T) {
	g := &GHAuth{exec: mockExec("", "error output", fmt.Errorf("exit 1"))}
	_, err := g.ActiveUser()
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseActiveUser(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:   "standard output",
			output: "  Logged in to github.com account user1 (keyring)",
			want:   "user1",
		},
		{
			name: "multiline with active",
			output: `github.com
  Logged in to github.com account myuser (token)`,
			want: "myuser",
		},
		{
			name:    "no account keyword",
			output:  "Some random status output here",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:   "strips parens from user",
			output: "  Logged in to github.com account user2 (keyring)",
			want:   "user2",
		},
		{
			name:    "account at end of line without user",
			output:  "  something account",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseActiveUser(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("parseActiveUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseNameFromJSON(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		want   string
	}{
		{
			name: "valid name",
			json: `{
  "login": "octocat",
  "name": "The Octocat",
  "email": null
}`,
			want: "The Octocat",
		},
		{
			name: "name with comma",
			json: `{
  "login": "user",
  "name": "John Doe",
  "email": null
}`,
			want: "John Doe",
		},
		{
			name: "no name field",
			json: `{
  "login": "user",
  "email": "user@example.com"
}`,
			want: "",
		},
		{
			name: "null name",
			json: `{
  "name": null
}`,
			want: "",
		},
		{
			name: "empty json",
			json: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNameFromJSON(tt.json)
			if got != tt.want {
				t.Errorf("parseNameFromJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePrimaryEmailFromJSON(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		want   string
	}{
		{
			name: "primary email",
			json: `[
  {
    "email": "user@example.com",
    "primary": true,
    "verified": true
  },
  {
    "email": "other@example.com",
    "primary": false,
    "verified": true
  }
]`,
			want: "user@example.com",
		},
		{
			name: "single email",
			json: `[
  {
    "email": "test@example.com",
    "primary": true,
    "verified": true
  }
]`,
			want: "test@example.com",
		},
		{
			name: "no primary email",
			json: `[
  {
    "email": "user@example.com",
    "primary": false,
    "verified": true
  }
]`,
			want: "",
		},
		{
			name: "empty array",
			json: "[]",
			want: "",
		},
		{
			name: "empty json",
			json: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePrimaryEmailFromJSON(tt.json)
			if got != tt.want {
				t.Errorf("parsePrimaryEmailFromJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGHAuth_GetUserInfo(t *testing.T) {
	tests := []struct {
		name         string
		userJSON     string
		userErr      error
		emailsJSON   string
		emailsErr    error
		wantName     string
		wantEmail    string
		wantErr      bool
	}{
		{
			name: "successful fetch",
			userJSON: `{
  "login": "octocat",
  "name": "The Octocat"
}`,
			emailsJSON: `[
  {
    "email": "octocat@github.com",
    "primary": true,
    "verified": true
  }
]`,
			wantName:  "The Octocat",
			wantEmail: "octocat@github.com",
		},
		{
			name:     "user API error",
			userErr:  fmt.Errorf("API error"),
			wantErr:  true,
		},
		{
			name:       "emails API error",
			userJSON:   `{"name": "Test User"}`,
			emailsErr:  fmt.Errorf("API error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			g := &GHAuth{
				exec: func(args ...string) (bytes.Buffer, bytes.Buffer, error) {
					var stdout, stderr bytes.Buffer
					callCount++
					if callCount == 1 {
						// First call: gh api user
						stdout.WriteString(tt.userJSON)
						return stdout, stderr, tt.userErr
					}
					// Second call: gh api user/emails
					stdout.WriteString(tt.emailsJSON)
					return stdout, stderr, tt.emailsErr
				},
			}

			info, err := g.GetUserInfo("testuser")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if info.Name != tt.wantName {
				t.Errorf("GetUserInfo().Name = %q, want %q", info.Name, tt.wantName)
			}
			if info.Email != tt.wantEmail {
				t.Errorf("GetUserInfo().Email = %q, want %q", info.Email, tt.wantEmail)
			}
		})
	}
}
