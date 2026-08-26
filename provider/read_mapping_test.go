package adaptive

import (
	"reflect"
	"testing"
)

// strp/boolp are declared in integrations_test.go.

func TestApplyEndpointReadTypeEquivalence(t *testing.T) {
	// "direct" (and unset) resolve to wire "cli": no drift.
	for _, prior := range []*string{nil, strp("direct"), strp("cli")} {
		got := applyEndpointRead(EndpointArgs{Name: "ep", Resource: "db", Type: prior},
			&SessionReadResponse{Name: "ep", Resource: "db", SessionType: "cli"}, false)
		if !reflect.DeepEqual(got.Type, prior) {
			t.Errorf("prior %v: type changed to %v", sv(prior), sv(got.Type))
		}
	}
	// Real drift: client -> services.
	got := applyEndpointRead(EndpointArgs{Name: "ep", Resource: "db", Type: strp("client")},
		&SessionReadResponse{Name: "ep", Resource: "db", SessionType: "services"}, false)
	if sv(got.Type) != "services" {
		t.Errorf("expected drift to services, got %v", sv(got.Type))
	}
}

func TestApplyEndpointReadServerDefaults(t *testing.T) {
	// Unset idleTimeout reading back the server default is not drift.
	got := applyEndpointRead(EndpointArgs{Name: "ep", Resource: "db"},
		&SessionReadResponse{Name: "ep", Resource: "db", SessionType: "cli", IdleTimeout: "15m", Memory: "512Mi", CPU: "0.5", Cluster: "default"}, false)
	if got.IdleTimeout != nil {
		t.Errorf("idleTimeout adopted server default: %v", sv(got.IdleTimeout))
	}
	if got.Memory != nil || got.CPU != nil || got.Cluster != nil {
		t.Errorf("unset optional inputs adopted server defaults: mem=%v cpu=%v cluster=%v",
			sv(got.Memory), sv(got.CPU), sv(got.Cluster))
	}
	// A previously-set idleTimeout does track drift.
	got = applyEndpointRead(EndpointArgs{Name: "ep", Resource: "db", IdleTimeout: strp("30m")},
		&SessionReadResponse{Name: "ep", Resource: "db", SessionType: "cli", IdleTimeout: "1h"}, false)
	if sv(got.IdleTimeout) != "1h" {
		t.Errorf("expected idleTimeout drift to 1h, got %v", sv(got.IdleTimeout))
	}
}

func TestApplyEndpointReadImportAdoptsServerValues(t *testing.T) {
	got := applyEndpointRead(EndpointArgs{}, &SessionReadResponse{
		Name: "ep", Resource: "db", SessionType: "client", TTL: "7d",
		Memory: "512Mi", IsJITEnabled: true, JITTotalApprovers: 2,
		SessionUsers: []string{"a@x.co"}, UserTags: []string{"k=v"},
	}, true)
	if got.Name != "ep" || got.Resource != "db" || sv(got.Type) != "client" ||
		sv(got.TTL) != "7d" || sv(got.Memory) != "512Mi" || !bv(got.IsJitEnabled) ||
		iv(got.JitTotalApprovers) != 2 ||
		!reflect.DeepEqual(got.Users, []string{"a@x.co"}) ||
		!reflect.DeepEqual(got.Tags, []string{"k=v"}) {
		t.Errorf("import did not adopt server values: %+v", got)
	}
}

func TestApplyEndpointReadSetSemantics(t *testing.T) {
	prior := EndpointArgs{Name: "ep", Resource: "db", Users: []string{"b@x.co", "a@x.co"}}
	got := applyEndpointRead(prior, &SessionReadResponse{
		Name: "ep", Resource: "db", SessionType: "cli",
		SessionUsers: []string{"a@x.co", "b@x.co"},
	}, false)
	// Same set: user ordering preserved.
	if !reflect.DeepEqual(got.Users, prior.Users) {
		t.Errorf("ordering not preserved: %v", got.Users)
	}
	// Different set: server wins.
	got = applyEndpointRead(prior, &SessionReadResponse{
		Name: "ep", Resource: "db", SessionType: "cli",
		SessionUsers: []string{"c@x.co"},
	}, false)
	if !reflect.DeepEqual(got.Users, []string{"c@x.co"}) {
		t.Errorf("server set not adopted: %v", got.Users)
	}
}

func TestApplyEndpointReadWorkspaceForcedToggle(t *testing.T) {
	// disableOutputCapture unset + server true (workspace-forced): stays unset.
	got := applyEndpointRead(EndpointArgs{Name: "ep", Resource: "db"},
		&SessionReadResponse{Name: "ep", Resource: "db", SessionType: "cli", DisableOutputCapture: true}, false)
	if got.DisableOutputCapture != nil {
		t.Errorf("workspace-forced toggle adopted into inputs: %v", bv(got.DisableOutputCapture))
	}
}

func TestApplyScriptRead(t *testing.T) {
	// Refresh: command untouched, description drift detected.
	prior := ScriptArgs{Name: "s", Command: "ls -la", Endpoint: "ep", Description: strp("old")}
	got := applyScriptRead(prior, &ScriptReadResponse{
		Name: "s", Endpoint: "ep", Description: "new", CommandOmitted: true,
	}, false)
	if got.Command != "ls -la" {
		t.Errorf("refresh clobbered command: %q", got.Command)
	}
	if sv(got.Description) != "new" {
		t.Errorf("description drift missed: %v", sv(got.Description))
	}
	// Import: command left empty.
	got = applyScriptRead(ScriptArgs{}, &ScriptReadResponse{
		Name: "s", Endpoint: "ep", ParameterDescriptions: map[string]string{"p": "d"}, CommandOmitted: true,
	}, true)
	if got.Command != "" || got.Name != "s" || got.ParameterDescriptions["p"] != "d" {
		t.Errorf("import mapping wrong: %+v", got)
	}
}

func TestApplyTeamRead(t *testing.T) {
	prior := GroupArgs{Name: "g", Members: []string{"a@x.co"}, SlackChannelID: strp("C1")}
	got := applyTeamRead(prior, &TeamReadResponse{
		Name: "g2", Members: []string{"a@x.co"}, Endpoints: []string{"ep"}, SlackChannelID: "C2",
	}, false)
	if got.Name != "g2" || sv(got.SlackChannelID) != "C2" {
		t.Errorf("team mapping wrong: %+v", got)
	}
	// endpoints was unset in the program, so a server-side value is NOT adopted.
	// This case used to assert the opposite, which is the bug: the program will
	// never contain that value, so writing it into state produces a diff on
	// every preview that no apply can settle.
	if got.Endpoints != nil {
		t.Errorf("server endpoints adopted into an unset input: %v", got.Endpoints)
	}

	// A list the program does set still reports real drift.
	managed := GroupArgs{Name: "g", Endpoints: []string{"ep-a"}}
	got = applyTeamRead(managed, &TeamReadResponse{
		Name: "g", Endpoints: []string{"ep-a", "ep-b"},
	}, false)
	if !reflect.DeepEqual(got.Endpoints, []string{"ep-a", "ep-b"}) {
		t.Errorf("drift on a managed list should be adopted, got %v", got.Endpoints)
	}

	// On import there are no prior inputs, so the server view is all there is.
	got = applyTeamRead(GroupArgs{}, &TeamReadResponse{
		Name: "g", Endpoints: []string{"ep"}, Members: []string{"a@x.co"},
	}, true)
	if !reflect.DeepEqual(got.Endpoints, []string{"ep"}) || !reflect.DeepEqual(got.Members, []string{"a@x.co"}) {
		t.Errorf("import should take the server lists, got %+v", got)
	}
}

// The regression test for the perpetual diff this policy exists to prevent: the
// platform seeds an endpoint's session users with its creator, so a program that
// never set `users` would otherwise adopt that on the first refresh and show an
// unresolvable update on every preview afterwards.
func TestApplyEndpointReadDoesNotAdoptServerSeededUsers(t *testing.T) {
	got := applyEndpointRead(EndpointArgs{Name: "ep", Resource: "db"},
		&SessionReadResponse{
			Name: "ep", Resource: "db", SessionType: "cli",
			SessionUsers: []string{"creator@x.co"},
		}, false)
	if got.Users != nil {
		t.Errorf("server-seeded users adopted into an unset input: %v", got.Users)
	}
}

func TestApplyAuthorizationRead(t *testing.T) {
	got := applyAuthorizationRead(AuthorizationArgs{}, &AuthorizationReadResponse{
		Name: "ro", ResourceType: "postgres", Permissions: "allow:\n- database: d\n", Description: "desc",
	}, true)
	if got.Name != "ro" || got.ResourceType != "postgres" || sv(got.Description) != "desc" {
		t.Errorf("authorization mapping wrong: %+v", got)
	}
}

func TestApplyScheduleReadNormalization(t *testing.T) {
	prior := ScheduleArgs{
		Name: "s", ScheduleType: "custom",
		Weekdays:      []string{"Monday", "TUESDAY"},
		SpecificDates: []string{"2026-08-13T10:00:00+05:30"},
		ExpiresAt:     strp("2026-12-31T18:30:00+05:30"),
	}
	got := applyScheduleRead(prior, &ScheduleReadResponse{
		Name: "s", ScheduleType: "custom", IsActive: true,
		Weekdays:      []string{"monday", "tuesday"},
		SpecificDates: []string{"2026-08-13T04:30:00Z"},
		ExpiresAt:     "2026-12-31T13:00:00Z",
	}, false)
	if !reflect.DeepEqual(got.Weekdays, prior.Weekdays) {
		t.Errorf("weekday casing not preserved: %v", got.Weekdays)
	}
	if !reflect.DeepEqual(got.SpecificDates, prior.SpecificDates) {
		t.Errorf("equal instants not preserved: %v", got.SpecificDates)
	}
	if got.ExpiresAt != prior.ExpiresAt {
		t.Errorf("equal expiry instant not preserved: %v", sv(got.ExpiresAt))
	}
	// Unset isActive reading back server default true is not drift.
	if got.IsActive != nil {
		t.Errorf("isActive adopted server default: %v", bv(got.IsActive))
	}
	// operationType unset + server autoapprove: not drift.
	if got.OperationType != nil {
		t.Errorf("operationType adopted default: %v", sv(got.OperationType))
	}
	// Real weekday drift.
	got = applyScheduleRead(prior, &ScheduleReadResponse{
		Name: "s", ScheduleType: "custom", Weekdays: []string{"friday"},
	}, false)
	if !reflect.DeepEqual(got.Weekdays, []string{"friday"}) {
		t.Errorf("weekday drift missed: %v", got.Weekdays)
	}
}
