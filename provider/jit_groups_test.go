package adaptive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
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
