package adaptive

import (
	"context"
	"fmt"
	"sort"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// ===========================================================================
// WorkspaceSettings  (adaptive:index:WorkspaceSettings) — singleton
// ===========================================================================

const workspaceSettingsID = "workspace-settings"

type WorkspaceSettings struct{}

type WorkspaceSettingsArgs struct {
	DefaultCluster                     *string `pulumi:"defaultCluster,optional"`
	DefaultLoginEnabled                *bool   `pulumi:"defaultLoginEnabled,optional"`
	MfaEnabled                         *bool   `pulumi:"mfaEnabled,optional"`
	RequireCommentsOnAccessRequests    *bool   `pulumi:"requireCommentsOnAccessRequests,optional"`
	DisableNewAccessRequest            *bool   `pulumi:"disableNewAccessRequest,optional"`
	MaxJitRequestDuration              *string `pulumi:"maxJitRequestDuration,optional"`
	PendingRequestExpiry               *string `pulumi:"pendingRequestExpiry,optional"`
	Timezone                           *string `pulumi:"timezone,optional"`
	ExpirationNotificationsEnabled     *bool   `pulumi:"expirationNotificationsEnabled,optional"`
	SsoSyncEnabled                     *bool   `pulumi:"ssoSyncEnabled,optional"`
	SsoSyncInterval                    *string `pulumi:"ssoSyncInterval,optional"`
	RemoveSsoDeletedUsersFromEndpoints *bool   `pulumi:"removeSsoDeletedUsersFromEndpoints,optional"`
	DefaultChatProviderId              *string `pulumi:"defaultChatProviderId,optional"`
	DefaultTicketProviderId            *string `pulumi:"defaultTicketProviderId,optional"`
	DefaultLogForwarder                *string `pulumi:"defaultLogForwarder,optional"`
	SessionTimeout                     *string `pulumi:"sessionTimeout,optional"`
	EndpointSleepTimeout               *string `pulumi:"endpointSleepTimeout,optional"`
	IntegrationProbeInterval           *string `pulumi:"integrationProbeInterval,optional"`
	DefaultCpu                         *string `pulumi:"defaultCpu,optional"`
	DefaultMemory                      *string `pulumi:"defaultMemory,optional"`
}

type WorkspaceSettingsState struct {
	WorkspaceSettingsArgs
}

func (s *WorkspaceSettingsArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.DefaultCluster, "Default cluster for new endpoints: \"0\", \"1\" or a Kubernetes resource id.")
	a.Describe(&s.DefaultLoginEnabled, "Whether password/default login is enabled. Setting false while no SSO login method is enabled is rejected by the server (lock-out guard).")
	a.Describe(&s.MfaEnabled, "Whether multi-factor authentication is enabled (also sets MFA on access, as in the UI).")
	a.Describe(&s.RequireCommentsOnAccessRequests, "Whether access requests must include a comment.")
	a.Describe(&s.DisableNewAccessRequest, "Whether new access requests are disabled.")
	a.Describe(&s.MaxJitRequestDuration, "Maximum duration of a just-in-time access request (e.g. 8h, 1d). Set to an empty string to clear the limit.")
	a.Describe(&s.PendingRequestExpiry, "How long a pending access request lives before it expires (e.g. 24h). Set to an empty string to clear.")
	a.Describe(&s.Timezone, "Workspace IANA timezone. Set to an empty string to use the server zone.")
	a.Describe(&s.ExpirationNotificationsEnabled, "Whether access-expiration notifications are sent.")
	a.Describe(&s.SsoSyncEnabled, "Whether automatic SSO user/group sync is enabled.")
	a.Describe(&s.SsoSyncInterval, "Interval of the automatic SSO sync.")
	a.Describe(&s.RemoveSsoDeletedUsersFromEndpoints, "Whether users deleted at the IdP are removed from endpoints.")
	a.Describe(&s.DefaultChatProviderId, "Id of the default Slack/Teams integration. Set to an empty string to clear.")
	a.Describe(&s.DefaultTicketProviderId, "Id of the default FreshService integration. Set to an empty string to clear.")
	a.Describe(&s.DefaultLogForwarder, "Id of the default logger integration (a JSON array string for several). Set to an empty string to clear.")
	a.Describe(&s.SessionTimeout, "Web session timeout.")
	a.Describe(&s.EndpointSleepTimeout, "Idle time before endpoints are put to sleep.")
	a.Describe(&s.IntegrationProbeInterval, "Interval between integration health probes.")
	a.Describe(&s.DefaultCpu, "Default CPU for endpoint pods, a Kubernetes quantity (e.g. 0.5, 500m).")
	a.Describe(&s.DefaultMemory, "Default memory for endpoint pods (e.g. 500Mi, 2Gi).")
}

func (a WorkspaceSettingsArgs) toBody() WorkspaceSettingsBody {
	return WorkspaceSettingsBody{
		DefaultCluster: a.DefaultCluster, DefaultLoginEnabled: a.DefaultLoginEnabled,
		MfaEnabled:                      a.MfaEnabled,
		RequireCommentsOnAccessRequests: a.RequireCommentsOnAccessRequests,
		DisableNewAccessRequest:         a.DisableNewAccessRequest,
		MaxJitRequestDuration:           a.MaxJitRequestDuration, PendingRequestExpiry: a.PendingRequestExpiry,
		Timezone:                       a.Timezone,
		ExpirationNotificationsEnabled: a.ExpirationNotificationsEnabled,
		SsoSyncEnabled:                 a.SsoSyncEnabled, SsoSyncInterval: a.SsoSyncInterval,
		RemoveSsoDeletedUsersFromEndpoints: a.RemoveSsoDeletedUsersFromEndpoints,
		DefaultChatProviderId:              a.DefaultChatProviderId,
		DefaultTicketProviderId:            a.DefaultTicketProviderId,
		DefaultLogForwarder:                a.DefaultLogForwarder, SessionTimeout: a.SessionTimeout,
		EndpointSleepTimeout: a.EndpointSleepTimeout, IntegrationProbeInterval: a.IntegrationProbeInterval,
		DefaultCpu: a.DefaultCpu, DefaultMemory: a.DefaultMemory,
	}
}

func (a WorkspaceSettingsArgs) isEmpty() bool {
	return a.toBody() == WorkspaceSettingsBody{}
}

// ptrOpt reconciles an optional value: unset stays unset (never diffs); a set
// value is replaced by the server's when they differ (drift). A field the server
// does not report keeps the prior value.
func ptrOpt[T comparable](prior, server *T, isImport bool) *T {
	if isImport {
		return server
	}
	if prior == nil {
		return nil
	}
	if server == nil || *prior == *server {
		return prior
	}
	return server
}

func applyWorkspaceSettingsRead(prior WorkspaceSettingsArgs, r *WorkspaceSettingsBody, isImport bool) WorkspaceSettingsArgs {
	return WorkspaceSettingsArgs{
		DefaultCluster:                     ptrOpt(prior.DefaultCluster, r.DefaultCluster, isImport),
		DefaultLoginEnabled:                ptrOpt(prior.DefaultLoginEnabled, r.DefaultLoginEnabled, isImport),
		MfaEnabled:                         ptrOpt(prior.MfaEnabled, r.MfaEnabled, isImport),
		RequireCommentsOnAccessRequests:    ptrOpt(prior.RequireCommentsOnAccessRequests, r.RequireCommentsOnAccessRequests, isImport),
		DisableNewAccessRequest:            ptrOpt(prior.DisableNewAccessRequest, r.DisableNewAccessRequest, isImport),
		MaxJitRequestDuration:              ptrOpt(prior.MaxJitRequestDuration, r.MaxJitRequestDuration, isImport),
		PendingRequestExpiry:               ptrOpt(prior.PendingRequestExpiry, r.PendingRequestExpiry, isImport),
		Timezone:                           ptrOpt(prior.Timezone, r.Timezone, isImport),
		ExpirationNotificationsEnabled:     ptrOpt(prior.ExpirationNotificationsEnabled, r.ExpirationNotificationsEnabled, isImport),
		SsoSyncEnabled:                     ptrOpt(prior.SsoSyncEnabled, r.SsoSyncEnabled, isImport),
		SsoSyncInterval:                    ptrOpt(prior.SsoSyncInterval, r.SsoSyncInterval, isImport),
		RemoveSsoDeletedUsersFromEndpoints: ptrOpt(prior.RemoveSsoDeletedUsersFromEndpoints, r.RemoveSsoDeletedUsersFromEndpoints, isImport),
		DefaultChatProviderId:              ptrOpt(prior.DefaultChatProviderId, r.DefaultChatProviderId, isImport),
		DefaultTicketProviderId:            ptrOpt(prior.DefaultTicketProviderId, r.DefaultTicketProviderId, isImport),
		DefaultLogForwarder:                ptrOpt(prior.DefaultLogForwarder, r.DefaultLogForwarder, isImport),
		SessionTimeout:                     ptrOpt(prior.SessionTimeout, r.SessionTimeout, isImport),
		EndpointSleepTimeout:               ptrOpt(prior.EndpointSleepTimeout, r.EndpointSleepTimeout, isImport),
		IntegrationProbeInterval:           ptrOpt(prior.IntegrationProbeInterval, r.IntegrationProbeInterval, isImport),
		DefaultCpu:                         ptrOpt(prior.DefaultCpu, r.DefaultCpu, isImport),
		DefaultMemory:                      ptrOpt(prior.DefaultMemory, r.DefaultMemory, isImport),
	}
}

func (*WorkspaceSettings) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[WorkspaceSettingsArgs], error) {
	args, failures, err := infer.DefaultCheck[WorkspaceSettingsArgs](ctx, req.NewInputs)
	if err != nil {
		return infer.CheckResponse[WorkspaceSettingsArgs]{}, err
	}
	if len(failures) == 0 && args.isEmpty() {
		failures = append(failures, p.CheckFailure{
			Property: "defaultCluster",
			Reason:   "set at least one workspace setting; an empty WorkspaceSettings manages nothing",
		})
	}
	return infer.CheckResponse[WorkspaceSettingsArgs]{Inputs: args, Failures: failures}, nil
}

func (*WorkspaceSettings) Create(ctx context.Context, req infer.CreateRequest[WorkspaceSettingsArgs]) (infer.CreateResponse[WorkspaceSettingsState], error) {
	out := infer.CreateResponse[WorkspaceSettingsState]{ID: workspaceSettingsID, Output: WorkspaceSettingsState{req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	if _, err := c.UpdateWorkspaceSettings(ctx, req.Inputs.toBody()); err != nil {
		return out, err
	}
	return out, nil
}

func (*WorkspaceSettings) Update(ctx context.Context, req infer.UpdateRequest[WorkspaceSettingsArgs, WorkspaceSettingsState]) (infer.UpdateResponse[WorkspaceSettingsState], error) {
	out := infer.UpdateResponse[WorkspaceSettingsState]{Output: WorkspaceSettingsState{req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	if _, err := c.UpdateWorkspaceSettings(ctx, req.Inputs.toBody()); err != nil {
		return out, err
	}
	return out, nil
}

func (*WorkspaceSettings) Read(ctx context.Context, req infer.ReadRequest[WorkspaceSettingsArgs, WorkspaceSettingsState]) (infer.ReadResponse[WorkspaceSettingsArgs, WorkspaceSettingsState], error) {
	type resp = infer.ReadResponse[WorkspaceSettingsArgs, WorkspaceSettingsState]
	c, err := clientFromConfig(ctx)
	if err != nil {
		return resp{}, err
	}
	r, err := c.ReadWorkspaceSettings(ctx)
	if err != nil {
		return resp{}, err
	}
	// Check rejects an empty program resource, so empty inputs mean import.
	isImport := req.Inputs.isEmpty()
	if r == nil {
		if isImport {
			return resp{}, notFoundOnImport("workspace settings", req.ID)
		}
		return resp{}, nil
	}
	inputs := applyWorkspaceSettingsRead(req.Inputs, r, isImport)
	return resp{ID: workspaceSettingsID, Inputs: inputs, State: WorkspaceSettingsState{inputs}}, nil
}

// Delete only stops managing the settings; the workspace keeps its values.
func (*WorkspaceSettings) Delete(ctx context.Context, _ infer.DeleteRequest[WorkspaceSettingsState]) (infer.DeleteResponse, error) {
	p.GetLogger(ctx).Info("adaptive:index:WorkspaceSettings: delete is a no-op; the workspace keeps its current settings and is simply no longer managed by Pulumi")
	return infer.DeleteResponse{}, nil
}

// ===========================================================================
// SsoSync  (adaptive:index:SsoSync) — one per SSO provider
// ===========================================================================

type SsoSync struct{}

type SsoSyncArgs struct {
	ProviderId      string   `pulumi:"providerId" provider:"replaceOnChanges"`
	LoginEnabled    *bool    `pulumi:"loginEnabled,optional"`
	AutoSyncEnabled *bool    `pulumi:"autoSyncEnabled,optional"`
	RoleSyncEnabled *bool    `pulumi:"roleSyncEnabled,optional"`
	Groups          []string `pulumi:"groups,optional"`
	SyncOnApply     *bool    `pulumi:"syncOnApply,optional"`
}

type SsoSyncState struct {
	SsoSyncArgs
	ProviderType string            `pulumi:"providerType,optional"`
	GroupNames   map[string]string `pulumi:"groupNames,optional"`
}

func (s *SsoSyncArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.ProviderId, "Id of the SSO integration (resource) whose group sync is managed (google, azureactivedirectory, jumpcloud or onelogin). Changing it replaces the resource.")
	a.Describe(&s.LoginEnabled, "Whether login through this SSO provider is enabled. Unset leaves it untouched.")
	a.Describe(&s.AutoSyncEnabled, "Whether automatic sync is enabled for this provider. Unset leaves it untouched.")
	a.Describe(&s.RoleSyncEnabled, "Whether roles are synced from the IdP. Unset leaves it untouched.")
	a.Describe(&s.Groups, "IdP group ids included in SSO sync. Order does not matter. The set replaces the persisted selection. An empty list (or unset) means no persisted selection, so auto sync falls back to its legacy behaviour. Supported provider types: google, azureactivedirectory, jumpcloud, onelogin; okta and LDAP are rejected by the server.")
	a.Describe(&s.SyncOnApply, "Trigger a sync after applying (default true).")
}

func (s *SsoSyncState) Annotate(a infer.Annotator) {
	a.Describe(&s.ProviderType, "The IdP type of the provider (e.g. okta).")
	a.Describe(&s.GroupNames, "Map of synced group id to group name.")
}

// normalizeGroups sorts and de-duplicates ids so ordering never produces a diff.
func normalizeGroups(in []string) []string {
	if in == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, g := range in {
		if _, ok := seen[g]; ok {
			continue
		}
		seen[g] = struct{}{}
		out = append(out, g)
	}
	sort.Strings(out)
	return out
}

func (*SsoSync) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[SsoSyncArgs], error) {
	args, failures, err := infer.DefaultCheck[SsoSyncArgs](ctx, req.NewInputs)
	if err != nil {
		return infer.CheckResponse[SsoSyncArgs]{}, err
	}
	args.Groups = normalizeGroups(args.Groups)
	return infer.CheckResponse[SsoSyncArgs]{Inputs: args, Failures: failures}, nil
}

func (a SsoSyncArgs) toRequest() SsoSyncUpdateRequest {
	return SsoSyncUpdateRequest{
		LoginEnabled: a.LoginEnabled, AutoSyncEnabled: a.AutoSyncEnabled,
		RoleSyncEnabled: a.RoleSyncEnabled, Groups: groupsPtr(a.Groups),
		SyncNow: a.SyncOnApply == nil || *a.SyncOnApply,
	}
}

func ssoState(args SsoSyncArgs, r *SsoSyncReadResponse) SsoSyncState {
	names := make(map[string]string, len(r.Groups))
	for _, g := range r.Groups {
		names[g.ID] = g.Name
	}
	return SsoSyncState{SsoSyncArgs: args, ProviderType: r.ProviderType, GroupNames: names}
}

func (*SsoSync) Create(ctx context.Context, req infer.CreateRequest[SsoSyncArgs]) (infer.CreateResponse[SsoSyncState], error) {
	out := infer.CreateResponse[SsoSyncState]{ID: req.Inputs.ProviderId, Output: SsoSyncState{SsoSyncArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	r, err := c.UpdateSsoSync(ctx, req.Inputs.ProviderId, req.Inputs.toRequest())
	if err != nil {
		return out, err
	}
	out.Output = ssoState(req.Inputs, r)
	return out, nil
}

func (*SsoSync) Update(ctx context.Context, req infer.UpdateRequest[SsoSyncArgs, SsoSyncState]) (infer.UpdateResponse[SsoSyncState], error) {
	out := infer.UpdateResponse[SsoSyncState]{Output: SsoSyncState{SsoSyncArgs: req.Inputs, ProviderType: req.State.ProviderType, GroupNames: req.State.GroupNames}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	body := req.Inputs.toRequest()
	// Removing groups from the program (or setting []) must clear the persisted
	// list; infer decodes an empty list as nil, so send [] explicitly.
	if body.Groups == nil && len(req.State.Groups) > 0 {
		empty := []string{}
		body.Groups = &empty
	}
	r, err := c.UpdateSsoSync(ctx, req.Inputs.ProviderId, body)
	if err != nil {
		return out, err
	}
	out.Output = ssoState(req.Inputs, r)
	return out, nil
}

func applySsoSyncRead(prior SsoSyncArgs, r *SsoSyncReadResponse, isImport bool) SsoSyncArgs {
	a := prior
	if isImport {
		a = SsoSyncArgs{}
	}
	a.ProviderId = r.ProviderID
	if r.ProviderID == "" {
		a.ProviderId = prior.ProviderId
	}
	lg, ag, rg := r.LoginEnabled, r.AutoSyncEnabled, r.RoleSyncEnabled
	a.LoginEnabled = ptrOpt(prior.LoginEnabled, &lg, isImport)
	a.AutoSyncEnabled = ptrOpt(prior.AutoSyncEnabled, &ag, isImport)
	a.RoleSyncEnabled = ptrOpt(prior.RoleSyncEnabled, &rg, isImport)
	ids := make([]string, 0, len(r.Groups))
	for _, g := range r.Groups {
		ids = append(ids, g.ID)
	}
	ids = normalizeGroups(ids)
	// groups are always managed once the resource exists; order never matters.
	if !isImport && prior.Groups == nil {
		a.Groups = nil // groups not managed by the program
	} else if !isImport && sameSet(prior.Groups, ids) {
		a.Groups = prior.Groups
	} else {
		a.Groups = ids
	}
	return a
}

func (*SsoSync) Read(ctx context.Context, req infer.ReadRequest[SsoSyncArgs, SsoSyncState]) (infer.ReadResponse[SsoSyncArgs, SsoSyncState], error) {
	type resp = infer.ReadResponse[SsoSyncArgs, SsoSyncState]
	c, err := clientFromConfig(ctx)
	if err != nil {
		return resp{}, err
	}
	r, err := c.ReadSsoSync(ctx, req.ID)
	if err != nil {
		return resp{}, err
	}
	isImport := req.Inputs.ProviderId == ""
	if r == nil {
		if isImport {
			return resp{}, notFoundOnImport("sso sync", req.ID)
		}
		return resp{}, nil
	}
	prior := req.Inputs
	if prior.ProviderId == "" {
		prior.ProviderId = req.ID
	}
	inputs := applySsoSyncRead(prior, r, isImport)
	if inputs.ProviderId == "" {
		return resp{}, fmt.Errorf("sso sync %q: server returned no provider id", req.ID)
	}
	return resp{ID: req.ID, Inputs: inputs, State: ssoState(inputs, r)}, nil
}

func (*SsoSync) Delete(ctx context.Context, req infer.DeleteRequest[SsoSyncState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteSsoSync(ctx, req.ID)
}

// groupsPtr maps unset (nil) to omitted, so the server leaves the persisted
// list alone, while an explicit empty list is sent and empties it.
func groupsPtr(g []string) *[]string {
	if g == nil {
		return nil
	}
	return &g
}
