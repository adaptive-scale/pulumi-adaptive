package adaptive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewClientURL(t *testing.T) {
	cases := map[string]string{
		"https://app.adaptive.live":  "https://app.adaptive.live/api/v1",
		"https://app.adaptive.live/": "https://app.adaptive.live/api/v1", // trailing slash trimmed
		"http://localhost:3000":      "http://localhost:3000/api/v1",
		"":                           defaultAdaptiveURL, // empty -> default
	}
	for in, want := range cases {
		if got := NewClient("tok", in).workspaceURL; got != want {
			t.Errorf("NewClient(%q).workspaceURL = %q, want %q", in, got, want)
		}
	}
}

func TestResolveToken(t *testing.T) {
	// The token arrives through ADAPTIVE_SVC_TOKEN (or the token file) and may
	// itself be JSON in either shape, so every case sets the environment — there
	// is no explicit-argument path any more.
	t.Run("raw string", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", "rawtoken")
		t.Setenv("ADAPTIVE_URL", "https://x")
		tok, url, err := resolveToken()
		if err != nil || tok != "rawtoken" || url != "https://x" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("simple json carries its own url", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", `{"token":"t1","url":"u1"}`)
		t.Setenv("ADAPTIVE_URL", "")
		tok, url, err := resolveToken()
		if err != nil || tok != "t1" || url != "u1" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("deployments json picks default", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN",
			`{"deployments":{"a":{"token":"ta","url":"ua","default":false},"b":{"token":"tb","url":"ub","default":true}}}`)
		t.Setenv("ADAPTIVE_URL", "")
		tok, url, err := resolveToken()
		if err != nil || tok != "tb" || url != "ub" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("single deployment needs no default marker", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", `{"deployments":{"only":{"token":"t1","url":"u1"}}}`)
		t.Setenv("ADAPTIVE_URL", "")
		tok, url, err := resolveToken()
		if err != nil || tok != "t1" || url != "u1" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("multiple deployments without default errors", func(t *testing.T) {
		// Picking one at map-iteration order would send credentials to whichever
		// environment came up first, so this must refuse and name the choices.
		t.Setenv("ADAPTIVE_SVC_TOKEN",
			`{"deployments":{"demo":{"token":"td","url":"ud"},"staging":{"token":"ts","url":"us"}}}`)
		t.Setenv("ADAPTIVE_URL", "")
		if _, _, err := resolveToken(); err == nil {
			t.Fatal("expected error for ambiguous deployments, got nil")
		} else {
			for _, want := range []string{"demo", "staging", "default"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q should mention %q", err, want)
				}
			}
		}
	})
	t.Run("falls back to the token file", func(t *testing.T) {
		// One of only two ways to authenticate now that there is no config, so
		// it is worth pinning rather than assuming.
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, ".adaptive"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".adaptive", "token"), []byte("file-token"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", home)
		t.Setenv("ADAPTIVE_SVC_TOKEN", "")
		t.Setenv("ADAPTIVE_URL", "https://from-env")

		tok, url, err := resolveToken()
		if err != nil || tok != "file-token" || url != "https://from-env" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("no token anywhere is an error", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("ADAPTIVE_SVC_TOKEN", "")
		if _, _, err := resolveToken(); err == nil {
			t.Fatal("expected an error when there is no token to be found")
		}
	})
}

func TestDefaultAdaptiveURL(t *testing.T) {
	// Regression: this was app.adaptive.com, which is not the product domain.
	if !strings.Contains(defaultAdaptiveURL, "app.adaptive.live") {
		t.Errorf("defaultAdaptiveURL = %q, want the adaptive.live domain", defaultAdaptiveURL)
	}
}

func TestGetSessionType(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "cli", true},
		{"direct", "cli", true},
		{"cli", "cli", true},
		{"client", "client", true},
		{"services", "services", true},
		{"bogus", "", false},
	}
	for _, c := range cases {
		got, ok := getSessionType(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("getSessionType(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestEndpointToSessionRequest(t *testing.T) {
	args := EndpointArgs{
		Name: "ep", Resource: "res", Type: strp("client"),
		TTL: strp("8h"), Authorization: strp("auth"),
		Users: []string{"a@x.com"}, IsJitEnabled: boolp(true),
		JitApprovers: []string{"b@x.com"}, ScriptOnlyAccess: boolp(true),
	}
	req, err := args.toSessionRequest()
	if err != nil {
		t.Fatal(err)
	}
	if req.SessionType != "client" {
		t.Errorf("SessionType = %q, want client", req.SessionType)
	}
	if req.SessionName != "ep" || req.ResourceName != "res" || req.AuthorizationName != "auth" || req.SessionTTL != "8h" {
		t.Errorf("unexpected request: %+v", req)
	}
	if !req.IsJITEnabled || !req.ScriptOnlyAccess {
		t.Errorf("bool flags not propagated: %+v", req)
	}
	if len(req.SessionUsers) != 1 || req.SessionUsers[0] != "a@x.com" {
		t.Errorf("users = %v", req.SessionUsers)
	}
}
