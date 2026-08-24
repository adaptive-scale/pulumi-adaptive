package adaptive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	presource "github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// readCases covers every resource that implements Read. `refresh` holds the
// minimum inputs that make the resource's import sentinel read as "this is a
// refresh, not an import".
var readCases = []struct {
	token   string
	refresh map[string]property.Value
}{
	{"adaptive:index:Endpoint", map[string]property.Value{
		"name": property.New("ep"), "resource": property.New("db"),
	}},
	{"adaptive:index:Resource", map[string]property.Value{
		"name": property.New("db"), "type": property.New("postgres"),
	}},
	{"adaptive:index:Authorization", map[string]property.Value{
		"name": property.New("ro"), "resourceType": property.New("postgres"),
		"permissions": property.New("read"),
	}},
	{"adaptive:index:Group", map[string]property.Value{
		"name": property.New("team"),
	}},
	{"adaptive:index:Script", map[string]property.Value{
		"name": property.New("s"), "endpoint": property.New("ep"),
		"command": property.New("echo hi"),
	}},
	{"adaptive:index:Schedule", map[string]property.Value{
		"name": property.New("sch"), "scheduleType": property.New("custom"),
	}},
	{"adaptive:index:DataProtection", map[string]property.Value{
		"resource": property.New("db"),
	}},
}

// newReadServer runs the real provider in-process against an httptest stand-in
// for the Adaptive API. serviceToken is set explicitly so resolveToken never
// falls back to ~/.adaptive/token on the developer's machine.
func newReadServer(t *testing.T, handler http.HandlerFunc) integration.Server {
	t.Helper()
	api := httptest.NewServer(handler)
	t.Cleanup(api.Close)

	prov, err := Provider()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := integration.NewServer(context.Background(), "adaptive",
		semver.MustParse("0.2.0"), integration.WithProvider(prov))
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Configure(p.ConfigureRequest{Args: property.NewMap(map[string]property.Value{
		"serviceToken": property.New("test-token"),
		"workspaceUrl": property.New(api.URL),
	})}); err != nil {
		t.Fatal(err)
	}
	return srv
}

func urnFor(token string) presource.URN {
	return presource.URN(fmt.Sprintf("urn:pulumi:dev::proj::%s::r", token))
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`"not found"`))
}

// TestImportUnknownIDFails is the regression test for imports of ids the server
// does not have. Returning an empty ReadResponse is not enough: the engine's
// only existence check for an import step is a nil output map, and infer always
// encodes the zero-valued inputs it is handed into a non-nil map, so the import
// would silently land in state with empty inputs and diff forever. Read has to
// return an error.
func TestImportUnknownIDFails(t *testing.T) {
	for _, tc := range readCases {
		t.Run(tc.token, func(t *testing.T) {
			srv := newReadServer(t, notFoundHandler)
			_, err := srv.Read(p.ReadRequest{
				ID:  "does-not-exist",
				Urn: urnFor(tc.token),
			})
			if err == nil {
				t.Fatal("import of an unknown id must fail")
			}
			if !strings.Contains(err.Error(), "cannot import") {
				t.Errorf("unhelpful error: %v", err)
			}
		})
	}
}

// TestRefreshUnknownIDDropsFromState pins the other half of the contract: the
// same 404 during a refresh must stay an empty response, which the engine reads
// as "delete from state".
func TestRefreshUnknownIDDropsFromState(t *testing.T) {
	for _, tc := range readCases {
		t.Run(tc.token, func(t *testing.T) {
			srv := newReadServer(t, notFoundHandler)
			state := property.NewMap(tc.refresh)
			resp, err := srv.Read(p.ReadRequest{
				ID:         "gone",
				Urn:        urnFor(tc.token),
				Inputs:     state,
				Properties: state,
			})
			if err != nil {
				t.Fatalf("refresh of a deleted resource must not error: %v", err)
			}
			if resp.ID != "" {
				t.Errorf("expected a blank ID to signal deletion, got %q", resp.ID)
			}
		})
	}
}

// TestImportIdentitylessResponseFails guards against a server that answers an
// unknown id with 200 and a zero-valued body instead of 404: adopting that
// would fabricate a resource out of nothing.
func TestImportIdentitylessResponseFails(t *testing.T) {
	for _, tc := range readCases {
		t.Run(tc.token, func(t *testing.T) {
			srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			})
			_, err := srv.Read(p.ReadRequest{
				ID:  "does-not-exist",
				Urn: urnFor(tc.token),
			})
			if err == nil {
				t.Fatal("a 200 with no identity must not import")
			}
		})
	}
}

// TestReadWrongRecordFails pins the readTyped guard: a response describing a
// different object than the one requested is a fault, not drift.
func TestReadWrongRecordFails(t *testing.T) {
	for _, tc := range readCases {
		t.Run(tc.token, func(t *testing.T) {
			srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"id": "somebody-else", "name": "other", "resourceName": "other"}`))
			})
			state := property.NewMap(tc.refresh)
			_, err := srv.Read(p.ReadRequest{
				ID:         "asked-for-this-one",
				Urn:        urnFor(tc.token),
				Inputs:     state,
				Properties: state,
			})
			if err == nil {
				t.Fatal("a read returning a different record must fail")
			}
			if !strings.Contains(err.Error(), "somebody-else") {
				t.Errorf("error should name the record that came back: %v", err)
			}
		})
	}
}
