package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"golang.org/x/oauth2"
)

const (
	envGitHubAppID             = "GITHUB_APP_ID"
	envGitHubAppInstallationID = "GITHUB_APP_INSTALLATION_ID"
	envGitHubAppPrivateKey     = "GITHUB_APP_PRIVATE_KEY"
	envGitHubKey               = "GITHUB_KEY"
)

type githubAuthMode string

const (
	githubAuthApp   githubAuthMode = "github-app"
	githubAuthToken githubAuthMode = "token"
)

// githubAuthConfig is the validated GitHub credential configuration. Its
// fields hold secrets, so it must never be logged; use mode for that.
type githubAuthConfig struct {
	mode githubAuthMode

	appID          int64
	installationID int64
	privateKey     []byte

	token string
}

// githubAuthFromEnv selects the GitHub authentication mode. GitHub App
// credentials win when all three App variables are set; GITHUB_KEY is the
// fallback. Partially set App credentials are an error, even if GITHUB_KEY
// is set, so a misconfigured rollout fails loudly instead of silently
// falling back to the old token.
func githubAuthFromEnv(getenv func(string) string) (githubAuthConfig, error) {
	appVars := []string{envGitHubAppID, envGitHubAppInstallationID, envGitHubAppPrivateKey}

	var set, missing []string
	for _, name := range appVars {
		if strings.TrimSpace(getenv(name)) != "" {
			set = append(set, name)
		} else {
			missing = append(missing, name)
		}
	}

	switch {
	case len(set) == len(appVars):
		appID, err := strconv.ParseInt(strings.TrimSpace(getenv(envGitHubAppID)), 10, 64)
		if err != nil || appID <= 0 {
			return githubAuthConfig{}, fmt.Errorf("%s must be a positive integer", envGitHubAppID)
		}
		installationID, err := strconv.ParseInt(strings.TrimSpace(getenv(envGitHubAppInstallationID)), 10, 64)
		if err != nil || installationID <= 0 {
			return githubAuthConfig{}, fmt.Errorf("%s must be a positive integer", envGitHubAppInstallationID)
		}
		return githubAuthConfig{
			mode:           githubAuthApp,
			appID:          appID,
			installationID: installationID,
			privateKey:     []byte(getenv(envGitHubAppPrivateKey)),
		}, nil

	case len(set) > 0:
		return githubAuthConfig{}, fmt.Errorf(
			"incomplete GitHub App configuration: %s set but %s missing",
			strings.Join(set, ", "), strings.Join(missing, ", "))
	}

	if token := strings.TrimSpace(getenv(envGitHubKey)); token != "" {
		return githubAuthConfig{mode: githubAuthToken, token: token}, nil
	}

	return githubAuthConfig{}, errors.New(
		"no GitHub credentials: set " + strings.Join(appVars, ", ") + " (GitHub App) or " + envGitHubKey + " (token)")
}

// httpClient builds the HTTP client that authenticates requests. In App mode
// the transport mints and refreshes installation tokens itself.
func (a githubAuthConfig) httpClient(ctx context.Context) (*http.Client, error) {
	if a.mode == githubAuthApp {
		tr, err := ghinstallation.New(http.DefaultTransport, a.appID, a.installationID, a.privateKey)
		if err != nil {
			return nil, fmt.Errorf("creating GitHub App installation transport: %w", err)
		}
		return &http.Client{Transport: tr}, nil
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: a.token})
	return oauth2.NewClient(ctx, ts), nil
}
