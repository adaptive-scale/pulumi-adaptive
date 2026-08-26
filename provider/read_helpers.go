package adaptive

import "fmt"

// Helpers shared by the per-resource Read implementations.
//
// Refresh policy: an optional input the user never set stays unset even when
// the server reports a computed default — adopting server defaults into inputs
// would create perpetual diffs. An optional input the user did set is
// overwritten when the server value differs (real drift). Required inputs are
// always taken from the server. On import, inputs are built entirely from the
// server response.

// strOpt reconciles an optional string input against the server value.
func strOpt(prior *string, server string, isImport bool) *string {
	if isImport {
		if server == "" {
			return nil
		}
		return &server
	}
	if prior == nil {
		return nil
	}
	if *prior == server {
		return prior
	}
	if server == "" {
		return nil
	}
	return &server
}

// boolOpt reconciles an optional bool input against the server value.
func boolOpt(prior *bool, server bool, isImport bool) *bool {
	if isImport {
		if !server {
			return nil
		}
		return &server
	}
	if prior == nil {
		// Never adopt a server-side true into an unset input: for several
		// endpoint toggles the effective value can be forced by workspace
		// settings, and unset must stay unset to avoid perpetual diffs.
		return nil
	}
	if *prior == server {
		return prior
	}
	return &server
}

// sameSet reports whether two string slices contain the same elements,
// ignoring order and duplicates.
func sameSet(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	as := make(map[string]struct{}, len(a))
	for _, v := range a {
		as[v] = struct{}{}
	}
	bs := make(map[string]struct{}, len(b))
	for _, v := range b {
		bs[v] = struct{}{}
	}
	if len(as) != len(bs) {
		return false
	}
	for v := range as {
		if _, ok := bs[v]; !ok {
			return false
		}
	}
	return true
}

// setList reconciles a set-valued input against the server value: when the sets
// are equal the user's ordering is kept, otherwise the server value wins.
//
// An input the program left unset is the exception, and it is the same rule
// strOpt and boolOpt already follow: the server populates several of these
// itself — an endpoint's session users default to its creator — and writing that
// back produces a diff on every preview that no apply can settle, because the
// program will never contain the value. Import is the other exception: there
// are no prior inputs to preserve, so the server view is the only one there is.
//
// The cost is that a list the program does not manage no longer reports
// out-of-band membership changes. That is the trade this file's policy already
// makes for every other optional input, and a silent diff nobody can resolve is
// worse than a change nobody asked to track.
func setList(prior, server []string, isImport bool) []string {
	if isImport {
		if len(server) == 0 {
			return nil
		}
		return server
	}
	if sameSet(prior, server) {
		return prior
	}
	if len(prior) == 0 || len(server) == 0 {
		return nil
	}
	return server
}

// intOpt reconciles an optional int input against the server value.
func intOpt(prior *int, server int, isImport bool) *int {
	if isImport {
		if server == 0 {
			return nil
		}
		return &server
	}
	if prior == nil {
		return nil
	}
	if *prior == server {
		return prior
	}
	return &server
}

// mapOpt reconciles an optional string-map input against the server value.
func mapOpt(prior, server map[string]string) map[string]string {
	if len(prior) == len(server) {
		same := true
		for k, v := range prior {
			if sv, ok := server[k]; !ok || sv != v {
				same = false
				break
			}
		}
		if same {
			return prior
		}
	}
	if len(server) == 0 {
		return nil
	}
	return server
}

// ---------------------------------------------------------------------------
// Per-resource read mappings (pure functions, unit-testable).
// ---------------------------------------------------------------------------

// applyEndpointRead reconciles endpoint inputs against a session read.
func applyEndpointRead(prior EndpointArgs, r *SessionReadResponse, isImport bool) EndpointArgs {
	a := prior
	if isImport {
		a = EndpointArgs{}
	}
	a.Name = r.Name
	a.Resource = r.Resource

	// The user-facing type "direct" (and unset) maps to wire type "cli", so
	// keep the user's spelling whenever it resolves to what the server has.
	if isImport {
		a.Type = strPtrOrNil(r.SessionType)
	} else if wire, ok := getSessionType(sv(prior.Type)); !ok || wire != r.SessionType {
		a.Type = strPtrOrNil(r.SessionType)
	}

	a.TTL = strOpt(prior.TTL, r.TTL, isImport)
	a.Authorization = strOpt(prior.Authorization, r.Authorization, isImport)
	a.Cluster = strOpt(prior.Cluster, r.Cluster, isImport)

	// The server normalizes an unset idle timeout to its default "15m"; an
	// unset input reading back the default is not drift.
	if !isImport && prior.IdleTimeout == nil && (r.IdleTimeout == "" || r.IdleTimeout == "15m") {
		a.IdleTimeout = nil
	} else {
		a.IdleTimeout = strOpt(prior.IdleTimeout, r.IdleTimeout, isImport)
	}

	a.Users = setList(prior.Users, r.SessionUsers, isImport)
	a.Groups = setList(prior.Groups, r.Groups, isImport)
	a.IsJitEnabled = boolOpt(prior.IsJitEnabled, r.IsJITEnabled, isImport)
	a.JitApprovers = setList(prior.JitApprovers, r.AccessApprovers, isImport)
	a.PauseTimeout = strOpt(prior.PauseTimeout, r.PauseTimeout, isImport)
	a.Memory = strOpt(prior.Memory, r.Memory, isImport)
	a.CPU = strOpt(prior.CPU, r.CPU, isImport)
	a.ScriptOnlyAccess = boolOpt(prior.ScriptOnlyAccess, r.ScriptOnlyAccess, isImport)
	a.Tags = setList(prior.Tags, r.UserTags, isImport)
	a.DisableOutputCapture = boolOpt(prior.DisableOutputCapture, r.DisableOutputCapture, isImport)
	a.DisableDataStudio = boolOpt(prior.DisableDataStudio, r.DisableDataStudio, isImport)
	a.DisableWebCli = boolOpt(prior.DisableWebCli, r.DisableWebCLI, isImport)
	a.JitMode = strOpt(prior.JitMode, r.JITMode, isImport)
	a.AutoApproval = boolOpt(prior.AutoApproval, r.AutoApproval, isImport)
	a.JitMultiApprover = boolOpt(prior.JitMultiApprover, r.JITMultiApprover, isImport)
	a.JitTotalApprovers = intOpt(prior.JitTotalApprovers, r.JITTotalApprovers, isImport)
	return a
}

// applyAuthorizationRead reconciles authorization inputs against a read.
func applyAuthorizationRead(prior AuthorizationArgs, r *AuthorizationReadResponse, isImport bool) AuthorizationArgs {
	a := prior
	if isImport {
		a = AuthorizationArgs{}
	}
	a.Name = r.Name
	a.ResourceType = r.ResourceType
	a.Permissions = r.Permissions
	a.Description = strOpt(prior.Description, r.Description, isImport)
	return a
}

// applyTeamRead reconciles group inputs against a team read.
func applyTeamRead(prior GroupArgs, r *TeamReadResponse, isImport bool) GroupArgs {
	a := prior
	if isImport {
		a = GroupArgs{}
	}
	a.Name = r.Name
	a.Members = setList(prior.Members, r.Members, isImport)
	a.Endpoints = setList(prior.Endpoints, r.Endpoints, isImport)
	a.SlackChannelID = strOpt(prior.SlackChannelID, r.SlackChannelID, isImport)
	return a
}

// applyScriptRead reconciles script inputs against a read. The script body
// (command) is write-only server-side: on refresh the prior value is kept, on
// import it stays empty and the caller must warn.
func applyScriptRead(prior ScriptArgs, r *ScriptReadResponse, isImport bool) ScriptArgs {
	a := prior
	if isImport {
		a = ScriptArgs{}
	}
	a.Name = r.Name
	a.Endpoint = r.Endpoint
	a.Description = strOpt(prior.Description, r.Description, isImport)
	if isImport {
		a.ParameterDescriptions = mapOpt(nil, r.ParameterDescriptions)
	} else {
		a.ParameterDescriptions = mapOpt(prior.ParameterDescriptions, r.ParameterDescriptions)
	}
	return a
}

// strPtrOrNil returns a pointer to s, or nil when s is empty.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// notFoundOnImport is the error a Read returns when the object named by an
// import does not exist.
//
// Refresh and import learn "it's gone" from the same 404 but need opposite
// signals. Refresh wants an empty ReadResponse: the engine reads a blank ID as
// "delete from state". Import cannot use that — its only existence check is a
// nil output map, and infer always encodes the zero-valued inputs/state it is
// handed into a non-nil map, so an empty response is indistinguishable from a
// successful read of an object whose every field happens to be empty. The
// engine then falls back to the ID the user typed and writes those empty inputs
// into state, leaving a diff on every later preview. An error is the only signal
// that fails the import.
func notFoundOnImport(kind, id string) error {
	return fmt.Errorf("cannot import %s %q: no such %s in this workspace "+
		"(check the id, and that the service token targets the right workspace)", kind, id, kind)
}
