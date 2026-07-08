//go:build integration

// Package harness provides the integration-test plumbing: config loading and a
// Pulumi Automation API wrapper that deploys/destroys real stacks.
package harness

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Config is the resolved credentials/endpoints an integration test needs.
type Config struct {
	URL          string // Adaptive base URL, e.g. http://localhost:8080
	ServiceToken string // service token used by the Pulumi provider to CREATE
	ClientID     string // Client App id used by the verifier to READ
	ClientSecret string // Client App secret used by the verifier to READ
}

// RequireConfig loads config from the environment (falling back to
// tests/.env.local) and skips the test cleanly if required values are absent.
func RequireConfig(t *testing.T) Config {
	t.Helper()
	loadDotEnv()

	url := os.Getenv("ADAPTIVE_URL")
	if url == "" {
		url = "http://localhost:8080"
	}
	cfg := Config{
		URL:          strings.TrimRight(url, "/"),
		ServiceToken: resolveServiceToken(),
		ClientID:     os.Getenv("ADAPTIVE_CLIENT_ID"),
		ClientSecret: os.Getenv("ADAPTIVE_CLIENT_SECRET"),
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		t.Skip("integration test skipped: set ADAPTIVE_CLIENT_ID and ADAPTIVE_CLIENT_SECRET (e.g. in tests/.env.local)")
	}
	if cfg.ServiceToken == "" {
		t.Skip("integration test skipped: no Adaptive service token (set ADAPTIVE_SVC_TOKEN or run `adaptive login`)")
	}
	preflight(t, cfg)
	return cfg
}

var (
	preflightOnce sync.Once
	preflightErr  error
)

// preflight verifies, once per test binary, that the backend is reachable and
// the service token is accepted. It fails fast with a single actionable message
// instead of letting every test fail deep inside Pulumi with "bad token" — the
// service token expires after ~3h, so a stale token is the most common cause of
// a whole-suite failure. Any non-401 response means the token was accepted (the
// provider itself treats only 401 as a bad token), so we don't care about the
// exact status of this throwaway read.
func preflight(t *testing.T, cfg Config) {
	t.Helper()
	preflightOnce.Do(func() {
		url := cfg.URL + "/api/v1/terraform/authorization/read/preflight-check"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			preflightErr = err
			return
		}
		req.Header.Set("Authorization", cfg.ServiceToken)
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			preflightErr = fmt.Errorf("cannot reach Adaptive backend at %s: %v", cfg.URL, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			preflightErr = errors.New("Adaptive service token rejected (401) — run `adaptive login` to refresh (tokens expire after ~3h)")
		}
	})
	if preflightErr != nil {
		t.Fatalf("integration preflight: %v", preflightErr)
	}
}

// loadDotEnv loads KEY=VALUE lines from tests/.env.local (one directory above
// the integration package) into the process environment, without overwriting
// values already set. Lines may be blank, `# comments`, or optionally prefixed
// with `export `. Values may be quoted.
func loadDotEnv() {
	for _, p := range []string{"../.env.local", ".env.local"} {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.Trim(strings.TrimSpace(v), `"'`)
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
		f.Close()
		return
	}
}

// resolveServiceToken mirrors the provider's token resolution: ADAPTIVE_SVC_TOKEN
// if set, otherwise the token from ~/.adaptive/token (raw, {token,...}, or
// {deployments:{...}} shapes).
func resolveServiceToken() string {
	if t := os.Getenv("ADAPTIVE_SVC_TOKEN"); t != "" {
		return t
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, ".adaptive", "token"))
	if err != nil {
		return ""
	}
	var simple struct {
		Token string `json:"token"`
	}
	if json.Unmarshal(b, &simple) == nil && simple.Token != "" {
		return simple.Token
	}
	var dep struct {
		Deployments map[string]struct {
			Token   string `json:"token"`
			Default bool   `json:"default"`
		} `json:"deployments"`
	}
	if json.Unmarshal(b, &dep) == nil && len(dep.Deployments) > 0 {
		for _, d := range dep.Deployments {
			if d.Default {
				return d.Token
			}
		}
		for _, d := range dep.Deployments {
			return d.Token
		}
	}
	return strings.TrimSpace(string(b))
}
