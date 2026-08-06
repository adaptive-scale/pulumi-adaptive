package adaptive

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"
	"gopkg.in/yaml.v2"
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

	DisableOutputCapture *bool `pulumi:"disableOutputCapture,optional"`

	Tags []string `pulumi:"tags,optional"`
}

type EndpointState struct {
	EndpointArgs
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
	a.Describe(&e.DisableOutputCapture, "Whether to stop retaining terminal output for sessions on this endpoint. Commands are still audited; the recorded output is purged and session replay shows a notice instead. The workspace-level setting is OR'd with this, so leaving it false does not re-enable capture for a workspace that has turned it off.")
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
	}, nil
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
	return out, nil
}

func (*Endpoint) Update(ctx context.Context, req infer.UpdateRequest[EndpointArgs, EndpointState]) (infer.UpdateResponse[EndpointState], error) {
	out := infer.UpdateResponse[EndpointState]{Output: EndpointState{EndpointArgs: req.Inputs}}
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
	return out, nil
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
	Name      string   `pulumi:"name"`
	Members   []string `pulumi:"members,optional"`
	Endpoints []string `pulumi:"endpoints,optional"`
}

type GroupState struct {
	GroupArgs
}

func (g *GroupArgs) Annotate(a infer.Annotator) {
	a.Describe(&g.Name, "Name of the group. Must be unique.")
	a.Describe(&g.Members, "Emails of users to add to the group.")
	a.Describe(&g.Endpoints, "Names of endpoints to add to this group.")
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
	return out, c.UpdateTeam(ctx, req.ID, req.Inputs.Name, req.Inputs.Members, req.Inputs.Endpoints)
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
	Name     string `pulumi:"name"`
	Command  string `pulumi:"command"`
	Endpoint string `pulumi:"endpoint"`
}

type ScriptState struct {
	ScriptArgs
}

func (s *ScriptArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.Name, "The name of the script.")
	a.Describe(&s.Command, "The command the script runs.")
	a.Describe(&s.Endpoint, "The endpoint the script is attached to. Cannot be changed after creation.")
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
	resp, err := c.CreateScript(ctx, req.Inputs.Name, req.Inputs.Command, req.Inputs.Endpoint)
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	return out, nil
}

func (*Script) Update(ctx context.Context, req infer.UpdateRequest[ScriptArgs, ScriptState]) (infer.UpdateResponse[ScriptState], error) {
	out := infer.UpdateResponse[ScriptState]{Output: ScriptState{ScriptArgs: req.Inputs}}
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
	return out, c.UpdateScript(ctx, req.ID, req.Inputs.Name, req.Inputs.Command, req.Inputs.Endpoint)
}

func (*Script) Delete(ctx context.Context, req infer.DeleteRequest[ScriptState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteScript(ctx, req.ID, req.State.Name)
}

// ===========================================================================
// MSTeamsWorkflow  (adaptive:index:MsTeamsWorkflow)
// ===========================================================================

type MSTeamsWorkflow struct{}

type MSTeamsWorkflowArgs struct {
	Name       string `pulumi:"name"`
	WebhookURL string `pulumi:"webhookUrl"`
}

type MSTeamsWorkflowState struct {
	MSTeamsWorkflowArgs
}

type msTeamsWorkflowConfig struct {
	Name       string `yaml:"name"`
	WebhookURL string `yaml:"webhookURL"`
}

func (m *MSTeamsWorkflowArgs) Annotate(a infer.Annotator) {
	a.Describe(&m.Name, "Name of the MS Teams workflow integration.")
	a.Describe(&m.WebhookURL, "The webhook URL for the MS Teams workflow.")
}

func (*MSTeamsWorkflow) Create(ctx context.Context, req infer.CreateRequest[MSTeamsWorkflowArgs]) (infer.CreateResponse[MSTeamsWorkflowState], error) {
	out := infer.CreateResponse[MSTeamsWorkflowState]{Output: MSTeamsWorkflowState{MSTeamsWorkflowArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	cfg, err := yaml.Marshal(msTeamsWorkflowConfig{Name: req.Inputs.Name, WebhookURL: req.Inputs.WebhookURL})
	if err != nil {
		return out, err
	}
	resp, err := c.CreateResource(ctx, req.Inputs.Name, "msteams_workflow", cfg, []string{}, "")
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	return out, nil
}

func (*MSTeamsWorkflow) Update(ctx context.Context, req infer.UpdateRequest[MSTeamsWorkflowArgs, MSTeamsWorkflowState]) (infer.UpdateResponse[MSTeamsWorkflowState], error) {
	out := infer.UpdateResponse[MSTeamsWorkflowState]{Output: MSTeamsWorkflowState{MSTeamsWorkflowArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	cfg, err := yaml.Marshal(msTeamsWorkflowConfig{Name: req.Inputs.Name, WebhookURL: req.Inputs.WebhookURL})
	if err != nil {
		return out, err
	}
	_, err = c.UpdateResource(ctx, req.ID, "msteams_workflow", cfg, []string{}, "")
	return out, err
}

func (*MSTeamsWorkflow) Delete(ctx context.Context, req infer.DeleteRequest[MSTeamsWorkflowState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteResource(ctx, req.ID, req.State.Name)
}
