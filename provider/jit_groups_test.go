package adaptive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/pulumi/pulumi-go-provider/infer"
)

func TestJITGroupPresenceAndRefresh(t *testing.T) {
	for _, tc := range []struct {
		groups   []string
		expected string
	}{{nil, ""}, {[]string{}, `"access_approver_groups":[]`}, {[]string{"On call"}, `"access_approver_groups":["On call"]`}} {
		args := EndpointArgs{Name: "prod", Resource: "database", JitApproverGroups: tc.groups}
		req, err := args.toSessionRequest()
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		if tc.expected == "" {
			if strings.Contains(string(data), "access_approver_groups") {
				t.Fatal(string(data))
			}
		} else if !strings.Contains(string(data), tc.expected) {
			t.Fatal(string(data))
		}
	}
	cleared := applyEndpointRead(EndpointArgs{JitApproverGroups: []string{}}, &SessionReadResponse{}, false)
	if cleared.JitApproverGroups == nil {
		t.Fatal("refresh lost explicit clear")
	}
	server := &SessionReadResponse{AccessApproverGroups: []string{"On call"}}
	if got := applyEndpointRead(EndpointArgs{}, server, false); got.JitApproverGroups != nil {
		t.Fatal("refresh adopted unmanaged groups")
	}
	if got := applyEndpointRead(EndpointArgs{}, server, true); !reflect.DeepEqual(got.JitApproverGroups, server.AccessApproverGroups) {
		t.Fatal("import lost groups")
	}
	if got := applyEndpointRead(EndpointArgs{JitApproverGroups: []string{"old"}}, server, false); !reflect.DeepEqual(got.JitApproverGroups, server.AccessApproverGroups) {
		t.Fatal("refresh lost drift")
	}
}

func TestJITGroupRefreshMissingFieldCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name    string
		groups  []string
		body    string
		wantErr bool
	}{
		{"legacy server unmanaged", nil, `{"id":"ep","name":"prod"}`, false},
		{"legacy server managed", []string{"On call"}, `{"id":"ep","name":"prod"}`, true},
		{"null managed", []string{"On call"}, `{"id":"ep","name":"prod","accessApproverGroups":null}`, true},
		{"clear on legacy server", []string{}, `{"id":"ep","name":"prod"}`, true},
		{"cleared on server", []string{"On call"}, `{"id":"ep","name":"prod","accessApproverGroups":[]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Error("refresh must not mutate the endpoint")
				}
				_, _ = w.Write([]byte(tc.body))
			}))
			defer api.Close()
			t.Setenv("ADAPTIVE_URL", api.URL)
			t.Setenv("ADAPTIVE_SVC_TOKEN", "test-token")
			result, err := (&Endpoint{}).Read(context.Background(), infer.ReadRequest[EndpointArgs, EndpointState]{
				ID: "ep", Inputs: EndpointArgs{Name: "prod", Resource: "database", JitApproverGroups: tc.groups},
			})
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "omitted JIT approver groups") {
					t.Fatalf("missing group readback must fail rather than clear managed state: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.ID != "ep" || (tc.groups == nil) != (result.Inputs.JitApproverGroups == nil) || len(result.Inputs.JitApproverGroups) != 0 {
				t.Fatalf("unexpected refresh result: %#v", result)
			}
		})
	}
}

func TestJITGroupsRejectUnsupportedServerBeforeMutation(t *testing.T) {
	mutations := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			mutations++
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := NewClient("test-token", server.URL)
	groups := []string{"On call"}
	for _, update := range []bool{false, true} {
		req := CreateSessionRequest{AccessApproverGroups: &groups}
		var err error
		if update {
			_, err = client.UpdateSession(context.Background(), "ep", req)
		} else {
			_, err = client.CreateSession(context.Background(), req)
		}
		if err == nil {
			t.Fatal("unsupported groups accepted")
		}
	}
	if mutations != 0 {
		t.Fatal("mutated old server")
	}
}
