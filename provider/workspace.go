package adaptive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"sort"
	"strings"

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
		DefaultLogForwarder:                ptrOpt(normalizeLogForwarder(prior.DefaultLogForwarder), normalizeLogForwarder(r.DefaultLogForwarder), isImport),
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
	args.DefaultLogForwarder = normalizeLogForwarder(args.DefaultLogForwarder)
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

// WorkspaceUserRole manages a local workspace role for an existing member.
type WorkspaceUserRole struct{}

type WorkspaceUserRoleArgs struct {
	Email string `pulumi:"email" provider:"replaceOnChanges"`
	Role  string `pulumi:"role"`
}

type WorkspaceUserRoleState struct {
	WorkspaceUserRoleArgs
	UserId string `pulumi:"userId"`
}

var workspaceRoles = map[string]bool{
	"workspaceadmin": true, "workspaceuser": true, "workspaceauditor": true,
	"workspaceapprover": true, "workspaceoperator": true, "workspacesoc": true,
}

func (a *WorkspaceUserRoleArgs) Annotate(n infer.Annotator) {
	n.Describe(&a.Email, "Email of an existing workspace member. Changing it replaces this resource.")
	n.Describe(&a.Role, "Local workspace role: workspaceadmin, workspaceuser, workspaceauditor, workspaceapprover, workspaceoperator, or workspacesoc. Independent of IdP roles. Updates are blocked if any enabled workspace IdP owns effective role synchronization, regardless of the member's IdP. Disable role sync separately for all such providers first; this resource never disables it automatically.")
}

func (*WorkspaceUserRole) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[WorkspaceUserRoleArgs], error) {
	a, failures, err := infer.DefaultCheck[WorkspaceUserRoleArgs](ctx, req.NewInputs)
	if err != nil {
		return infer.CheckResponse[WorkspaceUserRoleArgs]{}, err
	}
	for key := range req.NewInputs.AsMap() {
		if key != "email" && key != "role" {
			failures = append(failures, p.CheckFailure{Property: key, Reason: "unknown input; supported inputs are email and role"})
		}
	}
	inputs := req.NewInputs.AsMap()
	emailUnknown := inputs["email"].HasComputed()
	roleUnknown := inputs["role"].HasComputed()
	if !emailUnknown {
		a.Email = strings.ToLower(strings.TrimSpace(a.Email))
		parsedEmail, emailErr := mail.ParseAddress(a.Email)
		if emailErr != nil || parsedEmail.Address != a.Email {
			failures = append(failures, p.CheckFailure{Property: "email", Reason: "must be a valid email address"})
		}
	}
	if !roleUnknown && !workspaceRoles[a.Role] {
		failures = append(failures, p.CheckFailure{Property: "role", Reason: "must be one of workspaceadmin, workspaceuser, workspaceauditor, workspaceapprover, workspaceoperator, workspacesoc"})
	}
	return infer.CheckResponse[WorkspaceUserRoleArgs]{Inputs: a, Failures: failures}, nil
}

func (*WorkspaceUserRole) Create(ctx context.Context, req infer.CreateRequest[WorkspaceUserRoleArgs]) (infer.CreateResponse[WorkspaceUserRoleState], error) {
	out := infer.CreateResponse[WorkspaceUserRoleState]{ID: req.Inputs.Email, Output: WorkspaceUserRoleState{WorkspaceUserRoleArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	r, err := c.UpdateWorkspaceUserRole(ctx, req.Inputs.Email, req.Inputs.Role)
	if err != nil {
		return out, err
	}
	out.ID = r.Email
	out.Output = WorkspaceUserRoleState{WorkspaceUserRoleArgs: WorkspaceUserRoleArgs{Email: r.Email, Role: r.Role}, UserId: r.UserID}
	return out, nil
}

func (*WorkspaceUserRole) Update(ctx context.Context, req infer.UpdateRequest[WorkspaceUserRoleArgs, WorkspaceUserRoleState]) (infer.UpdateResponse[WorkspaceUserRoleState], error) {
	out := infer.UpdateResponse[WorkspaceUserRoleState]{Output: WorkspaceUserRoleState{WorkspaceUserRoleArgs: req.Inputs, UserId: req.State.UserId}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	r, err := c.UpdateWorkspaceUserRole(ctx, req.Inputs.Email, req.Inputs.Role)
	if err != nil {
		return out, err
	}
	out.Output = WorkspaceUserRoleState{WorkspaceUserRoleArgs: WorkspaceUserRoleArgs{Email: r.Email, Role: r.Role}, UserId: r.UserID}
	return out, nil
}

func (*WorkspaceUserRole) Read(ctx context.Context, req infer.ReadRequest[WorkspaceUserRoleArgs, WorkspaceUserRoleState]) (infer.ReadResponse[WorkspaceUserRoleArgs, WorkspaceUserRoleState], error) {
	type response = infer.ReadResponse[WorkspaceUserRoleArgs, WorkspaceUserRoleState]
	c, err := clientFromConfig(ctx)
	if err != nil {
		return response{}, err
	}
	r, err := c.ReadWorkspaceUserRole(ctx, req.ID)
	if err != nil {
		return response{}, err
	}
	if r == nil {
		if req.Inputs.Email == "" {
			return response{}, notFoundOnImport("workspace user role", req.ID)
		}
		return response{}, nil
	}
	a := WorkspaceUserRoleArgs{Email: r.Email, Role: r.Role}
	return response{ID: r.Email, Inputs: a, State: WorkspaceUserRoleState{WorkspaceUserRoleArgs: a, UserId: r.UserID}}, nil
}

func (*WorkspaceUserRole) Delete(ctx context.Context, _ infer.DeleteRequest[WorkspaceUserRoleState]) (infer.DeleteResponse, error) {
	p.GetLogger(ctx).Info("adaptive:index:WorkspaceUserRole: delete is a no-op; the member role remains unchanged")
	return infer.DeleteResponse{}, nil
}

// ===========================================================================
// SsoSync  (adaptive:index:SsoSync) — one per SSO provider
// ===========================================================================

type SsoSync struct{}

type SsoSyncArgs struct {
	ProviderId      string `pulumi:"providerId" provider:"replaceOnChanges"`
	LoginEnabled    *bool  `pulumi:"loginEnabled,optional"`
	AutoSyncEnabled *bool  `pulumi:"autoSyncEnabled,optional"`
	RoleSyncEnabled *bool  `pulumi:"roleSyncEnabled,optional"`
	// Groups is a pointer so nil (omitted) is distinct from &[]string{} (explicit empty).
	// infer allocates the pointer when the property key is present, even for an empty array.
	Groups      *[]string `pulumi:"groups,optional"`
	SyncOnApply *bool     `pulumi:"syncOnApply,optional"`
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
	a.Describe(&s.Groups, "IdP group ids included in SSO sync. Order does not matter. When set, the list replaces and manages the persisted selection, including an explicit empty list. Unset leaves the selection untouched and unmanaged. An empty persisted selection makes auto sync fall back to its legacy behaviour; it does not disable synchronization. Supported provider types: google, azureactivedirectory, jumpcloud, onelogin; okta and LDAP are rejected by the server.")
	a.Describe(&s.SyncOnApply, "Trigger a sync after applying (default true).")
}

func (s *SsoSyncState) Annotate(a infer.Annotator) {
	a.Describe(&s.ProviderType, "The IdP type of the provider (e.g. okta).")
	a.Describe(&s.GroupNames, "Map of synced group id to group name.")
}

// normalizeLogForwarder canonicalizes a defaultLogForwarder value to match the
// backend's FormatDefaultLogForwarderIDs contract so that ordering, whitespace,
// and singleton-vs-array formatting never produce a spurious diff.
//
//   - nil        → nil        (field unset; ptrOpt leaves it alone)
//   - ""         → ""         (explicit clear; sent to backend as-is)
//   - plain str  → trimmed    (single id stored as a bare string by the backend)
//   - JSON array → dedupe + trim + sort → plain string if 1 element, compact
//     JSON array if 2+
//
// The backend stores a single forwarder as a bare string, not `["id"]`, so we
// collapse a singleton array to a plain string here so both sides compare equal.
func normalizeLogForwarder(s *string) *string {
	if s == nil || *s == "" {
		return s
	}
	raw := strings.TrimSpace(*s)
	// Try to parse as a JSON array.
	var parsed []string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		// Plain string (single id). Trim and return.
		out := raw
		return &out
	}
	// Deduplicate and trim individual elements.
	seen := make(map[string]struct{}, len(parsed))
	ids := make([]string, 0, len(parsed))
	for _, id := range parsed {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		out := ""
		return &out
	}
	sort.Strings(ids)
	if len(ids) == 1 {
		// Backend stores a single forwarder as a bare string.
		return &ids[0]
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return s
	}
	out := string(b)
	return &out
}

// normalizeGroups sorts and de-duplicates ids. A nil pointer (omitted) is
// returned unchanged; a non-nil pointer (explicit, possibly empty) is normalised in place.
func normalizeGroups(in *[]string) *[]string {
	if in == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(*in))
	out := make([]string, 0, len(*in))
	for _, g := range *in {
		if _, ok := seen[g]; ok {
			continue
		}
		seen[g] = struct{}{}
		out = append(out, g)
	}
	sort.Strings(out)
	return &out
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
	var groups *[]string
	if a.Groups != nil {
		// Ensure a non-nil pointer always carries a non-nil slice (infer may
		// allocate the pointer but leave the inner slice nil for an empty array).
		g := *a.Groups
		if g == nil {
			g = []string{}
		}
		groups = &g
	}
	return SsoSyncUpdateRequest{
		LoginEnabled: a.LoginEnabled, AutoSyncEnabled: a.AutoSyncEnabled,
		RoleSyncEnabled: a.RoleSyncEnabled, Groups: groups,
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
	normIDs := normalizeGroups(&ids)
	// nil Groups means unmanaged; non-nil (even &[]) means managed.
	if !isImport && prior.Groups == nil {
		a.Groups = nil // not managed by program
	} else if !isImport && sameSet(*prior.Groups, *normIDs) {
		a.Groups = prior.Groups
	} else {
		a.Groups = normIDs
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
