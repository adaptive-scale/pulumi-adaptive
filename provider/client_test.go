package adaptive

import "testing"

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
