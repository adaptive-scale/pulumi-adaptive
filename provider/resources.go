package adaptive

import (
	"context"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ===========================================================================
// Endpoint  (adaptive:index:Endpoint) — "session" in the Adaptive API
// ===========================================================================

type Endpoint struct{}

type EndpointArgs struct {
	Name             string   `pulumi:"name"`
	Resource         string   `pulumi:"resource"`
	Type             *string  `pulumi:"type,optional"`
	TTL              *string  `pulumi:"ttl,optional"`
	Authorization    *string  `pulumi:"authorization,optional"`
	Cluster          *string  `pulumi:"cluster,optional"`
	IdleTimeout      *string  `pulumi:"idleTimeout,optional"`
	Users            []string `pulumi:"users,optional"`
	Groups           []string `pulumi:"groups,optional"`
	IsJitEnabled     *bool    `pulumi:"isJitEnabled,optional"`
	JitApprovers     []string `pulumi:"jitApprovers,optional"`
	PauseTimeout     *string  `pulumi:"pauseTimeout,optional"`
	Memory           *string  `pulumi:"memory,optional"`
	CPU              *string  `pulumi:"cpu,optional"`
	ScriptOnlyAccess *bool    `pulumi:"scriptOnlyAccess,optional"`
	Tags             []string `pulumi:"tags,optional"`

	DisableOutputCapture *bool   `pulumi:"disableOutputCapture,optional"`
	DisableDataStudio    *bool   `pulumi:"disableDataStudio,optional"`
	DisableWebCli        *bool   `pulumi:"disableWebCli,optional"`
	JitMode              *string `pulumi:"jitMode,optional"`
	AutoApproval         *bool   `pulumi:"autoApproval,optional"`
	JitMultiApprover     *bool   `pulumi:"jitMultiApprover,optional"`
	JitTotalApprovers    *int    `pulumi:"jitTotalApprovers,optional"`
}

type EndpointState struct {
	EndpointArgs
	Status       string `pulumi:"status,optional"`
	Exposed      bool   `pulumi:"exposed,optional"`
	ExposeType   string `pulumi:"exposeType,optional"`
	ExposeStatus string `pulumi:"exposeStatus,optional"`
}

func (e *EndpointArgs) Annotate(a infer.Annotator) {
	a.Describe(&e.Name, "The name of the endpoint to create.")
	a.Describe(&e.Resource, "The resource (by name) this endpoint grants access to.")
	a.Describe(&e.Type, "The type of session: direct, client, cli, or services.")
	a.SetDefault(&e.Type, "direct")
	a.Describe(&e.TTL, "Time-to-live for the endpoint, e.g. 8h, 7d, 90d.")
	a.Describe(&e.Authorization, "The authorization (by name) to apply to this endpoint.")
	a.Describe(&e.IsJitEnabled, "Whether Just-In-Time access approval is enabled.")
	a.Describe(&e.JitApprovers, "Emails of users who can approve Just-In-Time access requests.")
	a.Describe(&e.ScriptOnlyAccess, "Whether the endpoint is only accessible via script.")
	a.Describe(&e.DisableOutputCapture, "Do not retain terminal output for this endpoint. "+
		"The effective value is OR'd with the workspace-level setting: an endpoint cannot re-enable "+
		"capture that the workspace disabled.")
	a.Describe(&e.DisableDataStudio, "Hide Data Studio for this endpoint (cli endpoints only).")
	a.Describe(&e.DisableWebCli, "Hide 'Connect via Web CLI' for this endpoint (cli endpoints only).")
	a.Describe(&e.JitMode, "Which access paths require Just-In-Time approval: session, script, or both.")
	a.Describe(&e.AutoApproval, "Automatically approve Just-In-Time access requests.")
	a.Describe(&e.JitMultiApprover, "Require multiple approvers for Just-In-Time access requests.")
	a.Describe(&e.JitTotalApprovers, "Number of approvals required when multi-approver is enabled.")
}

func (e *EndpointState) Annotate(a infer.Annotator) {
	a.Describe(&e.Status, "Current lifecycle status of the endpoint.")
	a.Describe(&e.Exposed, "Whether the endpoint is exposed as a service.")
	a.Describe(&e.ExposeType, "How the endpoint is exposed (e.g. LoadBalancer).")
	a.Describe(&e.ExposeStatus, "Provisioning status of the service exposure.")
}

func getSessionType(t string) (string, bool) {
	switch t {
	case "", "direct", "cli":
		return "cli", true
	case "client":
		return "client", true
	case "services":
		return "services", true
	default:
		return "", false
	}
}

func (a EndpointArgs) toSessionRequest() (CreateSessionRequest, error) {
	st, ok := getSessionType(sv(a.Type))
	if !ok {
		return CreateSessionRequest{}, fmt.Errorf("invalid session type: %s", sv(a.Type))
	}
	return CreateSessionRequest{
		SessionName:       a.Name,
		ResourceName:      a.Resource,
		AuthorizationName: sv(a.Authorization),
		ClusterName:       sv(a.Cluster),
		SessionTTL:        sv(a.TTL),
		SessionType:       st,
		SessionUsers:      a.Users,
		IsJITEnabled:      bv(a.IsJitEnabled),
		AccessApprovers:   a.JitApprovers,
		PauseTimeout:      sv(a.PauseTimeout),
		Memory:            sv(a.Memory),
		CPU:               sv(a.CPU),
		UsersTags:         a.Tags,
		Groups:            a.Groups,
		IdleTimeout:       sv(a.IdleTimeout),
		ScriptOnlyAccess:  bv(a.ScriptOnlyAccess),

		DisableOutputCapture: bv(a.DisableOutputCapture),
		DisableDataStudio:    bv(a.DisableDataStudio),
		DisableWebCLI:        bv(a.DisableWebCli),
		JITMode:              sv(a.JitMode),
		AutoApproval:         a.AutoApproval,
		JITMultiApprover:     a.JitMultiApprover,
		JITTotalApprovers:    a.JitTotalApprovers,
	}, nil
}

// fillEndpointOutputs copies the server-computed output fields onto the state.
func fillEndpointOutputs(s *EndpointState, r *SessionReadResponse) {
	if r == nil {
		return
	}
	s.Status = r.Status
	s.Exposed = r.Exposed
	s.ExposeType = r.ExposeType
	s.ExposeStatus = r.ExposeStatus
}

func (*Endpoint) Create(ctx context.Context, req infer.CreateRequest[EndpointArgs]) (infer.CreateResponse[EndpointState], error) {
	out := infer.CreateResponse[EndpointState]{Output: EndpointState{EndpointArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	sreq, err := req.Inputs.toSessionRequest()
	if err != nil {
		return out, err
	}
	resp, err := c.CreateSession(ctx, sreq)
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	// Best-effort: surface the server-computed outputs right away.
	if r, rerr := c.ReadSession(ctx, resp.ID); rerr == nil {
		fillEndpointOutputs(&out.Output, r)
	}
	return out, nil
}

func (*Endpoint) Update(ctx context.Context, req infer.UpdateRequest[EndpointArgs, EndpointState]) (infer.UpdateResponse[EndpointState], error) {
	out := infer.UpdateResponse[EndpointState]{Output: EndpointState{
		EndpointArgs: req.Inputs,
		Status:       req.State.Status,
		Exposed:      req.State.Exposed,
		ExposeType:   req.State.ExposeType,
		ExposeStatus: req.State.ExposeStatus,
	}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	sreq, err := req.Inputs.toSessionRequest()
	if err != nil {
		return out, err
	}
	if _, err := c.UpdateSession(ctx, req.ID, sreq); err != nil {
		return out, err
	}
	// Best-effort refresh of the server-computed outputs.
	if r, rerr := c.ReadSession(ctx, req.ID); rerr == nil && r != nil {
		fillEndpointOutputs(&out.Output, r)
	}
	return out, nil
}

func (*Endpoint) Read(ctx context.Context, req infer.ReadRequest[EndpointArgs, EndpointState]) (infer.ReadResponse[EndpointArgs, EndpointState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, err
	}
	r, err := c.ReadSession(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, err
	}
	isImport := req.Inputs.Name == ""
	if r == nil {
		if isImport {
			return infer.ReadResponse[EndpointArgs, EndpointState]{}, notFoundOnImport("endpoint", req.ID)
		}
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, nil
	}
	// A 200 carrying no identity is not a real record — a server that answers
	// unknown ids with a zero-valued body would otherwise fabricate a resource
	// out of nothing. Only enforced on import, where there is no prior state to
	// lose.
	if isImport && r.Name == "" {
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, notFoundOnImport("endpoint", req.ID)
	}
	inputs := applyEndpointRead(req.Inputs, r, isImport)
	state := EndpointState{EndpointArgs: inputs}
	fillEndpointOutputs(&state, r)
	return infer.ReadResponse[EndpointArgs, EndpointState]{ID: req.ID, Inputs: inputs, State: state}, nil
}

func (*Endpoint) Delete(ctx context.Context, req infer.DeleteRequest[EndpointState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteSession(ctx, req.ID)
}

// ===========================================================================
// Authorization  (adaptive:index:Authorization)
// ===========================================================================

type Authorization struct{}

type AuthorizationArgs struct {
	Name         string  `pulumi:"name"`
	ResourceType string  `pulumi:"resourceType"`
	Permissions  string  `pulumi:"permissions"`
	Description  *string `pulumi:"description,optional"`
}

type AuthorizationState struct {
	AuthorizationArgs
	Status string `pulumi:"status,optional"`
}

func (a *AuthorizationState) Annotate(an infer.Annotator) {
	an.Describe(&a.Status, "Current lifecycle status of the authorization.")
}

func (a *AuthorizationArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Name, "The name of the authorization object.")
	an.Describe(&a.ResourceType, "Resource type to grant permission on, e.g. postgres, mysql, mongodb, kubernetes, ssh.")
	an.Describe(&a.Permissions, "The permission policy. The required format depends on resource_type: "+
		"structured YAML for postgres/mysql/sqlserver/cockroachdb/yugabytedb (an `allow:` list of database/privileges/objects) "+
		"and for kubernetes (an RBAC Role manifest); a free-form string for mongodb, ssh, and elasticsearch. "+
		"Note: the SQL and kubernetes types are validated as YAML server-side, so a bare value like \"SELECT\" is rejected.")
	an.Describe(&a.Description, "An optional description of the authorization object.")
}

func (*Authorization) Create(ctx context.Context, req infer.CreateRequest[AuthorizationArgs]) (infer.CreateResponse[AuthorizationState], error) {
	out := infer.CreateResponse[AuthorizationState]{Output: AuthorizationState{AuthorizationArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.CreateAuthorization(ctx, req.Inputs.Name, sv(req.Inputs.Description), req.Inputs.Permissions, req.Inputs.ResourceType)
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	return out, nil
}

func (*Authorization) Update(ctx context.Context, req infer.UpdateRequest[AuthorizationArgs, AuthorizationState]) (infer.UpdateResponse[AuthorizationState], error) {
	out := infer.UpdateResponse[AuthorizationState]{Output: AuthorizationState{AuthorizationArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	_, err = c.UpdateAuthorization(ctx, req.ID, req.Inputs.Name, sv(req.Inputs.Description), req.Inputs.Permissions, req.Inputs.ResourceType)
	return out, err
}

func (*Authorization) Read(ctx context.Context, req infer.ReadRequest[AuthorizationArgs, AuthorizationState]) (infer.ReadResponse[AuthorizationArgs, AuthorizationState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, err
	}
	r, err := c.ReadAuthorization(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, err
	}
	isImport := req.Inputs.Name == ""
	if r == nil {
		if isImport {
			return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, notFoundOnImport("authorization", req.ID)
		}
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, nil
	}
	// A 200 carrying no identity is not a real record — a server that answers
	// unknown ids with a zero-valued body would otherwise fabricate a resource
	// out of nothing. Only enforced on import, where there is no prior state to
	// lose.
	if isImport && r.Name == "" {
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, notFoundOnImport("authorization", req.ID)
	}
	inputs := applyAuthorizationRead(req.Inputs, r, isImport)
	return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{
		ID:     req.ID,
		Inputs: inputs,
		State:  AuthorizationState{AuthorizationArgs: inputs, Status: r.Status},
	}, nil
}

func (*Authorization) Delete(ctx context.Context, req infer.DeleteRequest[AuthorizationState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteAuthorization(ctx, req.ID)
}

// ===========================================================================
// Group  (adaptive:index:Group) — "team" in the Adaptive API
// ===========================================================================

type Group struct{}

type GroupArgs struct {
	Name           string   `pulumi:"name"`
	Members        []string `pulumi:"members,optional"`
	Endpoints      []string `pulumi:"endpoints,optional"`
	SlackChannelID *string  `pulumi:"slackChannelId,optional"`
}

type GroupState struct {
	GroupArgs
}

func (g *GroupArgs) Annotate(a infer.Annotator) {
	a.Describe(&g.Name, "Name of the group. Must be unique.")
	a.Describe(&g.Members, "Emails of users to add to the group.")
	a.Describe(&g.Endpoints, "Names of endpoints to add to this group.")
	a.Describe(&g.SlackChannelID, "Slack channel ID associated with this group.")
}

func (*Group) Create(ctx context.Context, req infer.CreateRequest[GroupArgs]) (infer.CreateResponse[GroupState], error) {
	out := infer.CreateResponse[GroupState]{Output: GroupState{GroupArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.CreateTeam(ctx, req.Inputs.Name, req.Inputs.Members, req.Inputs.Endpoints)
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	// The create route does not accept a Slack channel; set it via update.
	if sv(req.Inputs.SlackChannelID) != "" {
		if err := c.UpdateTeam(ctx, resp.ID, req.Inputs.Name, req.Inputs.Members, req.Inputs.Endpoints, sv(req.Inputs.SlackChannelID)); err != nil {
			return out, fmt.Errorf("group %s created but setting slackChannelId failed: %w", req.Inputs.Name, err)
		}
	}
	return out, nil
}

func (*Group) Update(ctx context.Context, req infer.UpdateRequest[GroupArgs, GroupState]) (infer.UpdateResponse[GroupState], error) {
	out := infer.UpdateResponse[GroupState]{Output: GroupState{GroupArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	return out, c.UpdateTeam(ctx, req.ID, req.Inputs.Name, req.Inputs.Members, req.Inputs.Endpoints, sv(req.Inputs.SlackChannelID))
}

func (*Group) Read(ctx context.Context, req infer.ReadRequest[GroupArgs, GroupState]) (infer.ReadResponse[GroupArgs, GroupState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[GroupArgs, GroupState]{}, err
	}
	r, err := c.ReadTeam(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[GroupArgs, GroupState]{}, err
	}
	isImport := req.Inputs.Name == ""
	if r == nil {
		if isImport {
			return infer.ReadResponse[GroupArgs, GroupState]{}, notFoundOnImport("group", req.ID)
		}
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[GroupArgs, GroupState]{}, nil
	}
	// A 200 carrying no identity is not a real record — a server that answers
	// unknown ids with a zero-valued body would otherwise fabricate a resource
	// out of nothing. Only enforced on import, where there is no prior state to
	// lose.
	if isImport && r.Name == "" {
		return infer.ReadResponse[GroupArgs, GroupState]{}, notFoundOnImport("group", req.ID)
	}
	inputs := applyTeamRead(req.Inputs, r, isImport)
	return infer.ReadResponse[GroupArgs, GroupState]{
		ID:     req.ID,
		Inputs: inputs,
		State:  GroupState{GroupArgs: inputs},
	}, nil
}

func (*Group) Delete(ctx context.Context, req infer.DeleteRequest[GroupState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteTeam(ctx, req.ID, req.State.Name)
}

// ===========================================================================
// Script  (adaptive:index:Script)
// ===========================================================================

type Script struct{}

type ScriptArgs struct {
	Name                  string            `pulumi:"name"`
	Command               string            `pulumi:"command" provider:"secret"`
	Endpoint              string            `pulumi:"endpoint"`
	Description           *string           `pulumi:"description,optional"`
	ParameterDescriptions map[string]string `pulumi:"parameterDescriptions,optional"`
}

type ScriptState struct {
	ScriptArgs
	IsAutoGen     bool   `pulumi:"isAutoGen,optional"`
	CommandDigest string `pulumi:"commandDigest,optional"`
}

func (s *ScriptArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.Name, "The name of the script.")
	a.Describe(&s.Command, "The command the script runs. Write-only: the Adaptive API never returns "+
		"script bodies, so drift in the command cannot be detected and `pulumi import` leaves it empty.")
	a.Describe(&s.Endpoint, "The endpoint the script is attached to. Cannot be changed after creation.")
	a.Describe(&s.Description, "An optional description of the script.")
	a.Describe(&s.ParameterDescriptions, "Descriptions for the script's parameters, keyed by parameter name.")
}

func (s *ScriptState) Annotate(a infer.Annotator) {
	a.Describe(&s.IsAutoGen, "Whether the script was auto-generated by Adaptive.")
	a.Describe(&s.CommandDigest, "Opaque server fingerprint of the script body, used to detect "+
		"out-of-band command changes on refresh.")
}

func (a ScriptArgs) toScriptRequest() ScriptRequest {
	return ScriptRequest{
		Name:                  a.Name,
		Command:               a.Command,
		Endpoint:              a.Endpoint,
		Description:           sv(a.Description),
		ParameterDescriptions: a.ParameterDescriptions,
	}
}

func (*Script) Create(ctx context.Context, req infer.CreateRequest[ScriptArgs]) (infer.CreateResponse[ScriptState], error) {
	out := infer.CreateResponse[ScriptState]{Output: ScriptState{ScriptArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.CreateScript(ctx, req.Inputs.toScriptRequest())
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	// Best-effort: capture the server's command fingerprint for drift detection.
	if r, rerr := c.ReadScript(ctx, resp.ID); rerr == nil && r != nil {
		out.Output.CommandDigest = r.CommandDigest
	}
	return out, nil
}

func (*Script) Update(ctx context.Context, req infer.UpdateRequest[ScriptArgs, ScriptState]) (infer.UpdateResponse[ScriptState], error) {
	out := infer.UpdateResponse[ScriptState]{Output: ScriptState{
		ScriptArgs:    req.Inputs,
		IsAutoGen:     req.State.IsAutoGen,
		CommandDigest: req.State.CommandDigest,
	}}
	if req.DryRun {
		return out, nil
	}
	// The Adaptive API does not allow changing a script's endpoint after creation.
	if req.Inputs.Endpoint != req.State.Endpoint {
		return out, fmt.Errorf("endpoint cannot be updated for an existing script")
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	if err := c.UpdateScript(ctx, req.ID, req.Inputs.toScriptRequest()); err != nil {
		return out, err
	}
	// Best-effort: refresh the command fingerprint after the write.
	if r, rerr := c.ReadScript(ctx, req.ID); rerr == nil && r != nil {
		out.Output.CommandDigest = r.CommandDigest
	}
	return out, nil
}

func (*Script) Read(ctx context.Context, req infer.ReadRequest[ScriptArgs, ScriptState]) (infer.ReadResponse[ScriptArgs, ScriptState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, err
	}
	r, err := c.ReadScript(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, err
	}
	isImport := req.Inputs.Name == ""
	if r == nil {
		if isImport {
			return infer.ReadResponse[ScriptArgs, ScriptState]{}, notFoundOnImport("script", req.ID)
		}
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, nil
	}
	// A 200 carrying no identity is not a real record — a server that answers
	// unknown ids with a zero-valued body would otherwise fabricate a resource
	// out of nothing. Only enforced on import, where there is no prior state to
	// lose.
	if isImport && r.Name == "" {
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, notFoundOnImport("script", req.ID)
	}
	inputs := applyScriptRead(req.Inputs, r, isImport)
	// The script body is write-only, but the server returns an opaque
	// fingerprint of it: a mismatch with the recorded one means the command
	// changed out-of-band. Clearing it makes the next preview show the
	// program's command being re-applied.
	if !isImport && req.State.CommandDigest != "" && r.CommandDigest != "" &&
		req.State.CommandDigest != r.CommandDigest {
		inputs.Command = ""
	}
	if isImport {
		p.GetLogger(ctx).Warning("the Adaptive API does not return script bodies; " +
			"set `command` in your program before the first `pulumi up` (the first update will rewrite it)")
	}
	return infer.ReadResponse[ScriptArgs, ScriptState]{
		ID:     req.ID,
		Inputs: inputs,
		State:  ScriptState{ScriptArgs: inputs, IsAutoGen: r.IsAutoGen, CommandDigest: r.CommandDigest},
	}, nil
}

func (*Script) Delete(ctx context.Context, req infer.DeleteRequest[ScriptState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteScript(ctx, req.ID, req.State.Name)
}
