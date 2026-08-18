package adaptive

import (
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
	t.Run("raw string", func(t *testing.T) {
		tok, url, err := resolveToken("rawtoken", "https://x")
		if err != nil || tok != "rawtoken" || url != "https://x" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("simple json", func(t *testing.T) {
		tok, url, err := resolveToken(`{"token":"t1","url":"u1"}`, "ignored")
		if err != nil || tok != "t1" || url != "u1" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("deployments json picks default", func(t *testing.T) {
		in := `{"deployments":{"a":{"token":"ta","url":"ua","default":false},"b":{"token":"tb","url":"ub","default":true}}}`
		tok, url, err := resolveToken(in, "ignored")
		if err != nil || tok != "tb" || url != "ub" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("single deployment needs no default marker", func(t *testing.T) {
		in := `{"deployments":{"only":{"token":"t1","url":"u1"}}}`
		tok, url, err := resolveToken(in, "ignored")
		if err != nil || tok != "t1" || url != "u1" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("multiple deployments without default errors", func(t *testing.T) {
		in := `{"deployments":{"demo":{"token":"td","url":"ud"},"staging":{"token":"ts","url":"us"}}}`
		_, _, err := resolveToken(in, "ignored")
		if err == nil {
			t.Fatal("expected error for ambiguous deployments, got nil")
		}
		for _, want := range []string{"demo", "staging", "default"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q should mention %q", err, want)
			}
		}
	})
	t.Run("env vars used when config is empty", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", "env-token")
		t.Setenv("ADAPTIVE_URL", "http://env-host:8080")
		tok, url, err := resolveToken("", "")
		if err != nil || tok != "env-token" || url != "http://env-host:8080" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("runtime env overrides saved provider config", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", "env-token")
		t.Setenv("ADAPTIVE_URL", "http://env-host:8080")
		tok, url, err := resolveToken("explicit-token", "http://explicit:9090")
		if err != nil || tok != "env-token" || url != "http://env-host:8080" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("runtime URL overrides URL embedded in token", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", `{"token":"env-token","url":"http://embedded:8080"}`)
		t.Setenv("ADAPTIVE_URL", "http://env-host:8080")
		tok, url, err := resolveToken("explicit-token", "http://explicit:9090")
		if err != nil || tok != "env-token" || url != "http://env-host:8080" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
		}
	})
	t.Run("env token may itself be a deployments config", func(t *testing.T) {
		t.Setenv("ADAPTIVE_SVC_TOKEN", `{"deployments":{"x":{"token":"tx","url":"ux","default":true}}}`)
		t.Setenv("ADAPTIVE_URL", "")
		tok, url, err := resolveToken("", "")
		if err != nil || tok != "tx" || url != "ux" {
			t.Fatalf("got (%q,%q,%v)", tok, url, err)
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
