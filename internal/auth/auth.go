// Package auth handles Google OAuth2 for guppy: a PKCE-protected browser
// consent flow over a localhost loopback redirect, plus a disk-cached,
// auto-refreshing token.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Client returns an HTTP client authorized for Google Tasks. It loads the
// cached token from tokenPath when possible, otherwise it runs the browser
// consent flow. Refreshed tokens are persisted back to tokenPath.
func Client(ctx context.Context, creds Credentials, tokenPath string) (*http.Client, error) {
	tok, err := loadToken(tokenPath)
	if err != nil {
		tok, err = flow(ctx, creds)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, tok); err != nil {
			return nil, err
		}
	}
	src := &persistSource{
		base: creds.OAuthConfig("").TokenSource(ctx, tok),
		path: tokenPath,
		last: tok.AccessToken,
	}
	return oauth2.NewClient(ctx, src), nil
}

// persistSource wraps a TokenSource and saves refreshed tokens to disk so the
// user stays signed in across runs.
type persistSource struct {
	base oauth2.TokenSource
	path string

	mu   sync.Mutex
	last string
}

func (p *persistSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if tok.AccessToken != p.last {
		// Google omits the refresh token on refresh responses; keep ours.
		if tok.RefreshToken == "" {
			if old, err := loadToken(p.path); err == nil {
				tok.RefreshToken = old.RefreshToken
			}
		}
		if err := saveToken(p.path, tok); err == nil {
			p.last = tok.AccessToken
		}
	}
	return tok, nil
}

// flow runs the Authorization Code + PKCE flow: it starts a loopback HTTP
// server on a random port, opens the system browser to Google's consent
// screen, and waits for the redirect.
func flow(ctx context.Context, creds Credentials) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("cannot start local callback server: %w", err)
	}
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/", ln.Addr().(*net.TCPAddr).Port)

	cfg := creds.OAuthConfig(redirectURL)
	verifier := oauth2.GenerateVerifier()
	state, err := randomState()
	if err != nil {
		ln.Close()
		return nil, err
	}
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("error") != "":
			errCh <- fmt.Errorf("authorization failed: %s", q.Get("error"))
			http.Error(w, "authorization failed — you can close this tab", http.StatusBadRequest)
		case q.Get("state") != state:
			errCh <- errors.New("authorization state mismatch")
			http.Error(w, "state mismatch — you can close this tab", http.StatusBadRequest)
		case q.Get("code") != "":
			codeCh <- q.Get("code")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, "<html><body style=\"font-family:sans-serif;text-align:center;padding-top:4em\">"+
				"<h2>guppy is signed in</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		default:
			http.NotFound(w, r) // e.g. favicon requests
		}
	})}

	go func() { _ = srv.Serve(ln) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Println("Opening your browser to sign in with Google…")
	fmt.Printf("If it doesn't open, visit this URL yourself:\n\n  %s\n\n", authURL)
	_ = openBrowser(authURL) // best-effort; the URL above is the fallback

	select {
	case code := <-codeCh:
		return cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	case err := <-errCh:
		return nil, err
	case <-time.After(3 * time.Minute):
		return nil, errors.New("sign-in timed out — run guppy again to retry")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func loadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

func saveToken(path string, tok *oauth2.Token) error {
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("encoding token: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("caching token: %w", err)
	}
	// WriteFile keeps existing permissions; enforce owner-only regardless.
	_ = os.Chmod(path, 0o600)
	return nil
}
