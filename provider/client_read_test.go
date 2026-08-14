package adaptive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at an httptest server handler.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("test-token", srv.URL), srv
}

func TestReadSessionDecodes(t *testing.T) {
	body := `{
		"id": "s-1", "name": "ep", "resource": "db", "cluster": "c1",
		"authorization": "ro", "status": "connected", "sessionType": "cli",
		"ttl": "7d", "idleTimeout": "15m", "pauseTimeout": "1h",
		"memory": "512Mi", "cpu": "0.5",
		"isJitEnabled": true, "jitMode": "session", "jitMultiApprover": true,
		"jitTotalApprovers": 2, "autoApproval": true,
		"sessionUsers": ["a@x.co"], "accessApprovers": ["b@x.co"],
		"groups": ["g1"], "userTags": ["k=v"],
		"scriptOnlyAccess": true, "disableOutputCapture": true,
		"disableDataStudio": true, "disableWebCli": true,
		"exposed": true, "exposeType": "LoadBalancer", "exposeStatus": "ready"
	}`
	for _, code := range []int{http.StatusOK, http.StatusAccepted} {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/terraform/session/read/s-1" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			w.WriteHeader(code)
			_, _ = w.Write([]byte(body))
		})
		got, err := c.ReadSession(context.Background(), "s-1")
		if err != nil {
			t.Fatalf("status %d: %v", code, err)
		}
		if got == nil {
			t.Fatalf("status %d: expected response, got nil", code)
		}
		if got.Name != "ep" || got.Resource != "db" || got.SessionType != "cli" ||
			!got.JITMultiApprover || got.JITTotalApprovers != 2 ||
			!got.DisableDataStudio || !got.DisableWebCLI || !got.AutoApproval ||
			got.ExposeType != "LoadBalancer" || got.ExposeStatus != "ready" {
			t.Errorf("status %d: bad decode: %+v", code, got)
		}
	}
}

func TestReadNotFoundIsNilNil(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	ctx := context.Background()
	if r, err := c.ReadSession(ctx, "x"); r != nil || err != nil {
		t.Errorf("ReadSession 404: got (%v, %v), want (nil, nil)", r, err)
	}
	if r, err := c.ReadResource(ctx, "x"); r != nil || err != nil {
		t.Errorf("ReadResource 404: got (%v, %v), want (nil, nil)", r, err)
	}
	if r, err := c.ReadAuthorization(ctx, "x"); r != nil || err != nil {
		t.Errorf("ReadAuthorization 404: got (%v, %v), want (nil, nil)", r, err)
	}
	if r, err := c.ReadTeam(ctx, "x"); r != nil || err != nil {
		t.Errorf("ReadTeam 404: got (%v, %v), want (nil, nil)", r, err)
	}
	if r, err := c.ReadScript(ctx, "x"); r != nil || err != nil {
		t.Errorf("ReadScript 404: got (%v, %v), want (nil, nil)", r, err)
	}
	if r, err := c.ReadSchedule(ctx, "x"); r != nil || err != nil {
		t.Errorf("ReadSchedule 404: got (%v, %v), want (nil, nil)", r, err)
	}
}

func TestReadServerErrorIsError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	if _, err := c.ReadSession(context.Background(), "x"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestReadResourceDecodes(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ResourceReadResponse{
			ID: "r-1", Name: "pg", IntegrationType: "postgres",
			UserTags: []string{"env=dev"}, DefaultCluster: "c1",
			Configuration: map[string]any{"username": "app", "hostname": "h"},
			RedactedKeys:  []string{"password"},
		})
	})
	got, err := c.ReadResource(context.Background(), "r-1")
	if err != nil || got == nil {
		t.Fatalf("got (%v, %v)", got, err)
	}
	if got.IntegrationType != "postgres" || got.Configuration["username"] != "app" ||
		len(got.RedactedKeys) != 1 || got.RedactedKeys[0] != "password" {
		t.Errorf("bad decode: %+v", got)
	}
}

func TestCreateSessionSendsNewFields(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/session/create") {
			_ = json.NewDecoder(r.Body).Decode(&captured)
			_ = json.NewEncoder(w).Encode(IDResponse{ID: "s-1"})
			return
		}
		// waitForSession read after create
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"s-1","status":"connected"}`))
	})
	tr, ap := true, 3
	_, err := c.CreateSession(context.Background(), CreateSessionRequest{
		SessionName: "ep", ResourceName: "db", SessionType: "cli",
		DisableOutputCapture: true, DisableDataStudio: true, DisableWebCLI: true,
		JITMode: "session", AutoApproval: &tr, JITMultiApprover: &tr, JITTotalApprovers: &ap,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"disable_output_capture", "disable_data_studio", "disable_web_cli", "jit_mode", "auto_approval", "jit_multi_approver", "jit_total_approvers"} {
		if _, ok := captured[k]; !ok {
			t.Errorf("create payload missing %q: %v", k, captured)
		}
	}
}

func TestCreateScriptSendsOptionalFields(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(IDResponse{ID: "sc-1"})
	})
	_, err := c.CreateScript(context.Background(), ScriptRequest{
		Name: "s", Command: "ls", Endpoint: "ep",
		Description:           "desc",
		ParameterDescriptions: map[string]string{"p1": "d1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured["Description"] != "desc" {
		t.Errorf("payload missing Description: %v", captured)
	}
	if pd, ok := captured["ParameterDescriptions"].(map[string]any); !ok || pd["p1"] != "d1" {
		t.Errorf("payload missing ParameterDescriptions: %v", captured)
	}
}

func TestUpdateTeamSendsSlackChannel(t *testing.T) {
	var captured map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.UpdateTeam(context.Background(), "t-1", "team", nil, nil, "C123"); err != nil {
		t.Fatal(err)
	}
	if captured["SlackChannelID"] != "C123" {
		t.Errorf("payload missing SlackChannelID: %v", captured)
	}
}

func TestScheduleCRUD(t *testing.T) {
	var createBody map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/schedule/create"):
			_ = json.NewDecoder(r.Body).Decode(&createBody)
			_ = json.NewEncoder(w).Encode(ScheduleReadResponse{ID: "sch-1", Name: "s", ScheduleType: "custom", UpdatedAt: "now"})
		case strings.Contains(r.URL.Path, "/schedule/read/sch-1"):
			_ = json.NewEncoder(w).Encode(ScheduleReadResponse{ID: "sch-1", Name: "s", ScheduleType: "custom", Weekdays: []string{"monday"}})
		case strings.Contains(r.URL.Path, "/schedule/update/sch-1"):
			_ = json.NewEncoder(w).Encode(ScheduleReadResponse{ID: "sch-1", Name: "s2", ScheduleType: "custom"})
		case strings.Contains(r.URL.Path, "/schedule/delete/sch-1"):
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	ctx := context.Background()
	created, err := c.CreateSchedule(ctx, ScheduleRequest{Name: "s", ScheduleType: "custom", Weekdays: []string{"monday"}, OperationType: "autoapprove"})
	if err != nil || created.ID != "sch-1" {
		t.Fatalf("create: (%v, %v)", created, err)
	}
	if createBody["operationType"] != "autoapprove" || createBody["scheduleType"] != "custom" {
		t.Errorf("create payload: %v", createBody)
	}
	got, err := c.ReadSchedule(ctx, "sch-1")
	if err != nil || got == nil || got.Weekdays[0] != "monday" {
		t.Fatalf("read: (%+v, %v)", got, err)
	}
	if _, err := c.UpdateSchedule(ctx, "sch-1", ScheduleRequest{Name: "s2", ScheduleType: "custom"}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteSchedule(ctx, "sch-1"); err != nil {
		t.Fatal(err)
	}
}
