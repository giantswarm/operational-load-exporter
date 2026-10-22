package main

import (
	"strings"
	"testing"
)

const (
	testToken = "tok"
	errNoAuth = "no GitHub credentials"
)

func TestGithubAuthFromEnv(t *testing.T) {
	const pem = "-----BEGIN " + "RSA PRIVATE KEY-----\nabc\n-----END " + "RSA PRIVATE KEY-----\n" // not a real key

	tests := []struct {
		name         string
		env          map[string]string
		wantMode     githubAuthMode
		wantAppID    int64
		wantInstall  int64
		wantToken    string
		wantErrParts []string
	}{
		{
			name: "app credentials",
			env: map[string]string{
				envGitHubAppID: "12345", envGitHubAppInstallationID: "678", envGitHubAppPrivateKey: pem,
			},
			wantMode: githubAuthApp, wantAppID: 12345, wantInstall: 678,
		},
		{
			name: "app credentials win over token",
			env: map[string]string{
				envGitHubAppID: "1", envGitHubAppInstallationID: "2", envGitHubAppPrivateKey: pem,
				envGitHubKey: testToken,
			},
			wantMode: githubAuthApp, wantAppID: 1, wantInstall: 2,
		},
		{
			name:      "token only",
			env:       map[string]string{envGitHubKey: testToken},
			wantMode:  githubAuthToken,
			wantToken: testToken,
		},
		{
			name:      "token is trimmed",
			env:       map[string]string{envGitHubKey: " tok\n"},
			wantMode:  githubAuthToken,
			wantToken: testToken,
		},
		{
			name:         "whitespace-only token counts as unset",
			env:          map[string]string{envGitHubKey: " \n"},
			wantErrParts: []string{errNoAuth},
		},
		{
			name:         "nothing set",
			env:          map[string]string{},
			wantErrParts: []string{errNoAuth, envGitHubAppID, envGitHubKey},
		},
		{
			name:         "whitespace only counts as unset",
			env:          map[string]string{envGitHubAppID: " ", envGitHubKey: ""},
			wantErrParts: []string{errNoAuth},
		},
		{
			name:         "only app id",
			env:          map[string]string{envGitHubAppID: "1"},
			wantErrParts: []string{"incomplete", envGitHubAppID + " set", envGitHubAppInstallationID, envGitHubAppPrivateKey},
		},
		{
			name: "partial app does not fall back to token",
			env: map[string]string{
				envGitHubAppID: "1", envGitHubAppInstallationID: "2", envGitHubKey: testToken,
			},
			wantErrParts: []string{"incomplete", envGitHubAppPrivateKey + " missing"},
		},
		{
			name: "non-numeric app id",
			env: map[string]string{
				envGitHubAppID: "abc", envGitHubAppInstallationID: "2", envGitHubAppPrivateKey: pem,
			},
			wantErrParts: []string{envGitHubAppID, "positive integer"},
		},
		{
			name: "zero installation id",
			env: map[string]string{
				envGitHubAppID: "1", envGitHubAppInstallationID: "0", envGitHubAppPrivateKey: pem,
			},
			wantErrParts: []string{envGitHubAppInstallationID, "positive integer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubAuthFromEnv(func(k string) string { return tt.env[k] })

			if len(tt.wantErrParts) > 0 {
				if err == nil {
					t.Fatalf("expected error, got config mode %q", got.mode)
				}
				for _, part := range tt.wantErrParts {
					if !strings.Contains(err.Error(), part) {
						t.Errorf("error %q does not contain %q", err, part)
					}
				}
				if strings.Contains(err.Error(), "BEGIN RSA") {
					t.Errorf("error leaks a secret: %q", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", got.mode, tt.wantMode)
			}
			if got.appID != tt.wantAppID || got.installationID != tt.wantInstall {
				t.Errorf("ids = %d/%d, want %d/%d", got.appID, got.installationID, tt.wantAppID, tt.wantInstall)
			}
			if got.token != tt.wantToken {
				t.Errorf("token mismatch")
			}
		})
	}
}

func TestHTTPClientInvalidPrivateKey(t *testing.T) {
	cfg := githubAuthConfig{mode: githubAuthApp, appID: 1, installationID: 2, privateKey: []byte("not a pem")}
	if _, err := cfg.httpClient(t.Context()); err == nil {
		t.Fatal("expected error for invalid private key")
	}
}

func TestHTTPClientToken(t *testing.T) {
	cfg := githubAuthConfig{mode: githubAuthToken, token: testToken}
	c, err := cfg.httpClient(t.Context())
	if err != nil || c == nil {
		t.Fatalf("unexpected result: %v, %v", c, err)
	}
}
