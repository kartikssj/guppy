package auth

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/tasks/v1"

	"guppy/internal/config"
)

// Embedded OAuth client credentials for guppy's shared Google Cloud project.
//
// These are NOT secrets and shipping them in release binaries is intentional:
// desktop/native OAuth clients are "public clients" per RFC 8252 — anything
// shipped in a binary can be extracted from it. The authorization flow is
// protected by PKCE and user consent, not by hiding these strings.
//
// The values are not committed to the repo because GitHub push protection
// blocks pushes containing them. Release builds inject them via -ldflags -X
// from repository secrets (see .github/workflows/release.yml); maintainers
// create the project in Google Cloud Console (see README.md, "Bring your own
// Google project"). A binary built from source has no embedded credentials —
// use GUPPY_CLIENT_ID/GUPPY_CLIENT_SECRET or a credentials.json instead.
var (
	embeddedClientID     string
	embeddedClientSecret string
)

// Credentials is an OAuth client identity used to authorize with Google.
type Credentials struct {
	ClientID     string
	ClientSecret string
}

// ResolveCredentials picks the OAuth client identity, highest priority first:
//
//  1. GUPPY_CLIENT_ID / GUPPY_CLIENT_SECRET environment variables
//  2. A Google Cloud credentials.json at $GUPPY_CREDENTIALS or
//     ~/.config/guppy/credentials.json
//  3. The credentials embedded in the binary
func ResolveCredentials() (Credentials, error) {
	if id := os.Getenv("GUPPY_CLIENT_ID"); id != "" {
		return Credentials{
			ClientID:     id,
			ClientSecret: os.Getenv("GUPPY_CLIENT_SECRET"),
		}, nil
	}

	path := os.Getenv("GUPPY_CREDENTIALS")
	if path == "" {
		path = config.CredentialsPath()
	}
	if b, err := os.ReadFile(path); err == nil {
		cfg, err := google.ConfigFromJSON(b, tasks.TasksScope)
		if err != nil {
			return Credentials{}, fmt.Errorf("invalid credentials file %s: %w", path, err)
		}
		return Credentials{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret}, nil
	}

	if embeddedClientID == "" {
		return Credentials{}, errors.New("no Google OAuth credentials configured yet — " +
			"see README.md (\"Bring your own Google project\") to set them up")
	}
	return Credentials{
		ClientID:     embeddedClientID,
		ClientSecret: embeddedClientSecret,
	}, nil
}

// OAuthConfig builds an oauth2.Config for these credentials. redirectURL may
// be empty for flows that don't need it (e.g. token refresh).
func (c Credentials) OAuthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{tasks.TasksScope}, // minimal scope: tasks read/write
		RedirectURL:  redirectURL,
	}
}
