package adaptive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// Client is a thin REST client for the Adaptive API. It is a direct port of the
// Terraform provider's internal client, minus the Terraform-specific logging.
type Client struct {
	serviceToken string
	workspaceURL string
	httpClient   *http.Client
}

const defaultAdaptiveURL = "https://app.adaptive.com/api/v1"

// NewClient builds a Client. workspaceURL is the base workspace URL (e.g.
// https://app.adaptive.live); the /api/v1 suffix is appended automatically.
func NewClient(serviceToken, workspaceURL string) *Client {
	if workspaceURL == "" {
		workspaceURL = defaultAdaptiveURL
	} else {
		workspaceURL = fmt.Sprintf("%s/api/v1", strings.TrimRight(workspaceURL, "/"))
	}
	return &Client{
		serviceToken: serviceToken,
		workspaceURL: workspaceURL,
		httpClient:   &http.Client{},
	}
}

func (c *Client) authorizationAPI() string { return c.workspaceURL + "/terraform/authorization" }
func (c *Client) teamAPI() string          { return c.workspaceURL + "/terraform/team" }
func (c *Client) scriptAPI() string        { return c.workspaceURL + "/terraform/script" }
func (c *Client) resourceAPI() string      { return c.workspaceURL + "/terraform/resource" }
func (c *Client) sessionAPI() string       { return c.workspaceURL + "/terraform/session" }
func (c *Client) scheduleAPI() string      { return c.workspaceURL + "/terraform/schedule" }

func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", c.serviceToken)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized {
		res.Body.Close()
		return nil, errors.New("bad token. please check your service token")
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

type CreateResourceRequest struct {
	IntegrationType string   `json:"integrationType"`
	Name            string   `json:"name"`
	Configuration   string   `json:"config"`
	UserTags        []string `json:"userTags"`
	DefaultCluster  string   `json:"defaultCluster,omitempty"`
}

type UpdateResourceRequest struct {
	IntegrationType string   `json:"integrationType"`
	Configuration   string   `json:"config"`
	UserTags        []string `json:"userTags"`
	DefaultCluster  string   `json:"defaultCluster,omitempty"`
}

type IDResponse struct {
	ID string `json:"id"`
}

type CreateSessionRequest struct {
	SessionName       string   `json:"sessionName"`
	ResourceName      string   `json:"resourceName"`
	ClusterName       string   `json:"clusterName,omitempty"`
	AuthorizationName string   `json:"authorizationName,omitempty"`
	SessionTTL        string   `json:"sessionTTL,omitempty"`
	SessionType       string   `json:"sessionType"`
	SessionUsers      []string `json:"sessionUsers,omitempty"`
	IsJITEnabled      bool     `json:"is_jit_enabled"`
	AccessApprovers   []string `json:"access_approvers"`
	Memory            string   `json:"memory"`
	CPU               string   `json:"cpu"`
	UsersTags         []string `json:"usertags"`
	PauseTimeout      string   `json:"pause_timeout,omitempty"`
	Groups            []string `json:"groups,omitempty"`
	IdleTimeout       string   `json:"idle_timeout,omitempty"`
	ScriptOnlyAccess  bool     `json:"script_only_access"`

	DisableOutputCapture bool   `json:"disable_output_capture"`
	DisableDataStudio    bool   `json:"disable_data_studio"`
	DisableWebCLI        bool   `json:"disable_web_cli"`
	JITMode              string `json:"jit_mode,omitempty"`
	AutoApproval         *bool  `json:"auto_approval,omitempty"`
	JITMultiApprover     *bool  `json:"jit_multi_approver,omitempty"`
	JITTotalApprovers    *int   `json:"jit_total_approvers,omitempty"`
}

type ScriptRequest struct {
	Name                  string            `json:"Name"`
	Command               string            `json:"Command"`
	Endpoint              string            `json:"Endpoint"`
	Description           string            `json:"Description,omitempty"`
	ParameterDescriptions map[string]string `json:"ParameterDescriptions,omitempty"`
}

// ScheduleRequest mirrors the server's TerraformCreateScheduleRequest.
type ScheduleRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ScheduleType  string   `json:"scheduleType"`
	IsActive      *bool    `json:"isActive,omitempty"`
	AllDay        bool     `json:"allDay,omitempty"`
	StartHour     int      `json:"startHour"`
	StartMinute   int      `json:"startMinute"`
	EndHour       int      `json:"endHour"`
	EndMinute     int      `json:"endMinute"`
	Weekdays      []string `json:"weekdays,omitempty"`
	StartDay      int      `json:"startDay,omitempty"`
	EndDay        int      `json:"endDay,omitempty"`
	SpecificDates []string `json:"specificDates,omitempty"`
	Users         []string `json:"users,omitempty"`
	Teams         []string `json:"teams,omitempty"`
	Endpoints     []string `json:"endpoints,omitempty"`
	ExpiresAt     *string  `json:"expiresAt,omitempty"`
	MaxAccessTime *int     `json:"maxAccessTime,omitempty"`
	Timezone      string   `json:"timezone,omitempty"`
	OperationType string   `json:"operationType,omitempty"`
}

// ---------------------------------------------------------------------------
// Read models — canonical camelCase keys of the /terraform/<type>/read/:id DTOs.
// ---------------------------------------------------------------------------

type ResourceReadResponse struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	IntegrationType string         `json:"integrationType"`
	UserTags        []string       `json:"userTags"`
	DefaultCluster  string         `json:"defaultCluster"`
	Configuration   map[string]any `json:"configuration"` // secret values are stripped server-side
	RedactedKeys    []string       `json:"redactedKeys"`
	CreatedAt       string         `json:"createdAt"`
	UpdatedAt       string         `json:"updatedAt"`
}

type SessionReadResponse struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Resource             string   `json:"resource"` // integration name, not id
	Cluster              string   `json:"cluster"`
	Authorization        string   `json:"authorization"`
	Status               string   `json:"status"`
	SessionType          string   `json:"sessionType"` // cli | client | services
	TTL                  string   `json:"ttl"`
	IdleTimeout          string   `json:"idleTimeout"`
	PauseTimeout         string   `json:"pauseTimeout"`
	Memory               string   `json:"memory"`
	CPU                  string   `json:"cpu"`
	Storage              string   `json:"storage"`
	IsJITEnabled         bool     `json:"isJitEnabled"`
	JITMode              string   `json:"jitMode"`
	JITMultiApprover     bool     `json:"jitMultiApprover"`
	JITTotalApprovers    int      `json:"jitTotalApprovers"`
	AutoApproval         bool     `json:"autoApproval"`
	SessionUsers         []string `json:"sessionUsers"`
	AccessApprovers      []string `json:"accessApprovers"`
	Groups               []string `json:"groups"`
	UserTags             []string `json:"userTags"`
	ScriptOnlyAccess     bool     `json:"scriptOnlyAccess"`
	DisableOutputCapture bool     `json:"disableOutputCapture"`
	DisableDataStudio    bool     `json:"disableDataStudio"`
	DisableWebCLI        bool     `json:"disableWebCli"`
	Public               bool     `json:"public"`
	Exposed              bool     `json:"exposed"`
	ExposeType           string   `json:"exposeType"`
	ExposeStatus         string   `json:"exposeStatus"`
}

type AuthorizationReadResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ResourceType string `json:"resourceType"`
	Permissions  string `json:"permissions"`
	Status       string `json:"status"`
	Type         string `json:"type"`
}

type TeamReadResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Members        []string `json:"members"`   // emails
	Endpoints      []string `json:"endpoints"` // endpoint names
	SlackChannelID string   `json:"slackChannelId"`
	CreatedBy      string   `json:"createdBy"`
	CreatedAt      string   `json:"createdAt"`
}

type ScriptReadResponse struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Endpoint              string            `json:"endpoint"` // endpoint name (resolved)
	Description           string            `json:"description"`
	ParameterDescriptions map[string]string `json:"parameterDescriptions"`
	IsAutoGen             bool              `json:"isAutoGen"`
	CommandOmitted        bool              `json:"commandOmitted"` // command is write-only
	CreatedBy             string            `json:"createdBy"`
	CreatedAt             string            `json:"createdAt"`
}

type ScheduleReadResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ScheduleType  string   `json:"scheduleType"`
	IsActive      bool     `json:"isActive"`
	AllDay        bool     `json:"allDay"`
	StartHour     int      `json:"startHour"`
	StartMinute   int      `json:"startMinute"`
	EndHour       int      `json:"endHour"`
	EndMinute     int      `json:"endMinute"`
	Weekdays      []string `json:"weekdays"`
	StartDay      int      `json:"startDay"`
	EndDay        int      `json:"endDay"`
	SpecificDates []string `json:"specificDates"`
	Users         []string `json:"users"`
	Teams         []string `json:"teams"`
	Endpoints     []string `json:"endpoints"`
	ExpiresAt     string   `json:"expiresAt"`
	MaxAccessTime *int     `json:"maxAccessTime"`
	Timezone      string   `json:"timezone"`
	OperationType string   `json:"operationType"`
	UpdatedAt     string   `json:"updatedAt"`
}

type CreateAuthorizationRequest struct {
	AuthorizationName string `json:"name"`
	Resource          string `json:"resource"`
	Description       string `json:"description"`
	Permissions       string `json:"permissions"`
}

type UpdateAuthorizationRequest struct {
	AuthorizationName        string `json:"name"`
	AuthorizationDescription string `json:"description"`
	ResourceType             string `json:"resourceType"`
	Permissions              string `json:"permissions"`
}

// ---------------------------------------------------------------------------
// Resources (integrations)
// ---------------------------------------------------------------------------

func (c *Client) CreateResource(ctx context.Context, name, rType string, yamlConfig []byte, tags []string, defaultCluster string) (*IDResponse, error) {
	body, err := encode(CreateResourceRequest{
		IntegrationType: rType,
		Name:            name,
		Configuration:   string(yamlConfig),
		UserTags:        tags,
		DefaultCluster:  defaultCluster,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.resourceAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("duplicate resource with name %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating resource %s: %s", name, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	return &out, nil
}

func (c *Client) UpdateResource(ctx context.Context, resourceID, rType string, yamlConfig []byte, tags []string, defaultCluster string) (*IDResponse, error) {
	body, err := encode(UpdateResourceRequest{
		IntegrationType: rType,
		Configuration:   string(yamlConfig),
		UserTags:        tags,
		DefaultCluster:  defaultCluster,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.resourceAPI(), resourceID), body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error updating resource %s: %s", resourceID, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteResource(ctx context.Context, resourceID, resourceName string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.resourceAPI(), resourceID), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting resource %s: %s", resourceName, readBody(resp))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sessions (endpoints)
// ---------------------------------------------------------------------------

func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*IDResponse, error) {
	body, err := encode(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.sessionAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("duplicate session with name %s", req.SessionName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating session %s, reason %s", req.SessionName, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	// Wait for the session to become functional (best-effort, matches TF behavior).
	_, _ = c.waitForSession(ctx, out.ID)
	return &out, nil
}

func (c *Client) UpdateSession(ctx context.Context, sessionID string, req CreateSessionRequest) (*IDResponse, error) {
	body, err := encode(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.sessionAPI(), sessionID), body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error updating session %s, reason %s", req.SessionName, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteSession fully reaps an endpoint's session so the resource it points at
// can be deleted afterwards. This is a synchronous two-step sequence:
//
//  1. POST /session/delete/:id terminates the session (status -> terminated). On
//     its own this does NOT remove the sessions row — a terminated row has no
//     delete_at and is never picked up by the deferred reaper, so it (and its
//     integration_id association) would linger indefinitely and keep the parent
//     resource undeletable.
//  2. POST /session/forcedelete/:id hard-deletes the row and its child rows in a
//     single committed transaction. Its precondition is the terminal status that
//     step 1 establishes.
//
// Once step 2 returns, the row is gone (single primary DB, read-after-write
// consistent), so a following DeleteResource passes the "no associated endpoint"
// guard without waiting on the 7-day deferred-deletion cool-off.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	// Step 1: terminate (precondition for force delete).
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.sessionAPI(), sessionID), nil))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		msg := readBody(resp)
		resp.Body.Close()
		return fmt.Errorf("error terminating session %s: %s", sessionID, msg)
	}
	resp.Body.Close()

	// Step 2: force delete (synchronously removes the row + resource association).
	resp, err = c.do(ctx, mustReq("POST", fmt.Sprintf("%s/forcedelete/%s", c.sessionAPI(), sessionID), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error force-deleting session %s: %s", sessionID, readBody(resp))
	}
	return nil
}

// readTyped performs a GET against a /read/:id route and decodes the response.
// Not-found (404) is reported as (nil, nil) so callers can treat it as "the
// resource no longer exists" rather than a fault.
func readTyped[T any](c *Client, ctx context.Context, url, what, id string) (*T, error) {
	resp, err := c.do(ctx, mustReq("GET", url, nil))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	// Session read historically responds 202 Accepted; every other type uses 200.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error reading %s %s: %s", what, id, readBody(resp))
	}
	out := new(T)
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("failed to decode %s read response: %w", what, err)
	}
	return out, nil
}

// ReadSession fetches the current state of an endpoint (session) by ID.
// Returns (nil, nil) when the session no longer exists.
func (c *Client) ReadSession(ctx context.Context, sessionID string) (*SessionReadResponse, error) {
	return readTyped[SessionReadResponse](c, ctx, fmt.Sprintf("%s/read/%s", c.sessionAPI(), sessionID), "session", sessionID)
}

// ReadResource fetches an integration (resource) by ID. Secret configuration
// values are stripped server-side; their keys are listed in RedactedKeys.
// Returns (nil, nil) when the resource no longer exists.
func (c *Client) ReadResource(ctx context.Context, resourceID string) (*ResourceReadResponse, error) {
	return readTyped[ResourceReadResponse](c, ctx, fmt.Sprintf("%s/read/%s", c.resourceAPI(), resourceID), "resource", resourceID)
}

// ReadAuthorization fetches an authorization by ID.
// Returns (nil, nil) when the authorization no longer exists.
func (c *Client) ReadAuthorization(ctx context.Context, authID string) (*AuthorizationReadResponse, error) {
	return readTyped[AuthorizationReadResponse](c, ctx, fmt.Sprintf("%s/read/%s", c.authorizationAPI(), authID), "authorization", authID)
}

// ReadTeam fetches a team (group) by ID.
// Returns (nil, nil) when the team no longer exists.
func (c *Client) ReadTeam(ctx context.Context, teamID string) (*TeamReadResponse, error) {
	return readTyped[TeamReadResponse](c, ctx, fmt.Sprintf("%s/read/%s", c.teamAPI(), teamID), "team", teamID)
}

// ReadScript fetches a script by ID. The script body (command) is write-only
// and never returned; CommandOmitted is set instead.
// Returns (nil, nil) when the script no longer exists.
func (c *Client) ReadScript(ctx context.Context, scriptID string) (*ScriptReadResponse, error) {
	return readTyped[ScriptReadResponse](c, ctx, fmt.Sprintf("%s/read/%s", c.scriptAPI(), scriptID), "script", scriptID)
}

// ReadSchedule fetches a schedule by ID.
// Returns (nil, nil) when the schedule no longer exists.
func (c *Client) ReadSchedule(ctx context.Context, scheduleID string) (*ScheduleReadResponse, error) {
	return readTyped[ScheduleReadResponse](c, ctx, fmt.Sprintf("%s/read/%s", c.scheduleAPI(), scheduleID), "schedule", scheduleID)
}

func (c *Client) waitForSession(ctx context.Context, sessionID string) (*SessionReadResponse, error) {
	return Do(func() (*SessionReadResponse, error) {
		return c.ReadSession(ctx, sessionID)
	}, RetryLimit(30), Sleep(10*time.Second), RetryResultChecker(func(r any) bool {
		res, ok := r.(*SessionReadResponse)
		if !ok || res == nil {
			// nil means 404: right after create the row may not be readable yet.
			return true
		}
		return strings.ToLower(res.Status) == "creating"
	}))
}

// ---------------------------------------------------------------------------
// Authorizations
// ---------------------------------------------------------------------------

func (c *Client) CreateAuthorization(ctx context.Context, name, description, permissions, resourceType string) (*IDResponse, error) {
	body, err := encode(CreateAuthorizationRequest{
		AuthorizationName: name,
		Resource:          resourceType,
		Description:       description,
		Permissions:       permissions,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.authorizationAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("duplicate authorization with name %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating authorization %s, reason %s", name, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	// Wait for the authorization to become functional before reporting success
	// (parity with the Terraform provider).
	if _, err := c.waitForAuthorization(ctx, out.ID); err != nil {
		return nil, fmt.Errorf("failed to create authorization %s: %w", name, err)
	}
	return &out, nil
}

func (c *Client) waitForAuthorization(ctx context.Context, authID string) (*AuthorizationReadResponse, error) {
	return Do(func() (*AuthorizationReadResponse, error) {
		return c.ReadAuthorization(ctx, authID)
	}, RetryLimit(20), Sleep(10*time.Second), RetryResultChecker(func(r any) bool {
		res, ok := r.(*AuthorizationReadResponse)
		if !ok || res == nil {
			// nil means 404: right after create the row may not be readable yet.
			return true
		}
		return strings.ToLower(res.Status) == "creating"
	}))
}

func (c *Client) UpdateAuthorization(ctx context.Context, authID, name, description, permissions, resourceType string) (*IDResponse, error) {
	body, err := encode(UpdateAuthorizationRequest{
		AuthorizationName:        name,
		AuthorizationDescription: description,
		Permissions:              permissions,
		ResourceType:             resourceType,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.authorizationAPI(), authID), body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error updating authorization %s, reason %s", name, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteAuthorization(ctx context.Context, authID string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.authorizationAPI(), authID), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting authorization %s: %s", authID, readBody(resp))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Teams (groups)
// ---------------------------------------------------------------------------

func (c *Client) CreateTeam(ctx context.Context, name string, members, endpoints []string) (*IDResponse, error) {
	body, err := encode(map[string]any{"Name": name, "Members": members, "Endpoints": endpoints})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.teamAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("duplicate group with name %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating group %s: %s", name, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateTeam(ctx context.Context, id, name string, members, endpoints []string, slackChannelID string) error {
	body, err := encode(map[string]any{"Name": name, "Members": members, "Endpoints": endpoints, "SlackChannelID": slackChannelID})
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.teamAPI(), id), body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error updating group %s: %s", name, readBody(resp))
	}
	return nil
}

func (c *Client) DeleteTeam(ctx context.Context, id, name string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.teamAPI(), id), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting group %s: %s", name, readBody(resp))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Scripts
// ---------------------------------------------------------------------------

func (c *Client) CreateScript(ctx context.Context, req ScriptRequest) (*IDResponse, error) {
	body, err := encode(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.scriptAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("duplicate script with name %s", req.Name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating script %s: %s", req.Name, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateScript(ctx context.Context, id string, req ScriptRequest) error {
	body, err := encode(req)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.scriptAPI(), id), body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("duplicate script with name %s", req.Name)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error updating script %s", req.Name)
	}
	return nil
}

func (c *Client) DeleteScript(ctx context.Context, id, name string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.scriptAPI(), id), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting script %s: %s", name, readBody(resp))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Schedules
// ---------------------------------------------------------------------------

// CreateSchedule creates a schedule. Note: the server upserts by name, so
// creating a schedule whose name already exists adopts the existing row.
// The response carries the full schedule object including its ID.
func (c *Client) CreateSchedule(ctx context.Context, req ScheduleRequest) (*ScheduleReadResponse, error) {
	body, err := encode(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.scheduleAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating schedule %s: %s", req.Name, readBody(resp))
	}
	var out ScheduleReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSchedule(ctx context.Context, id string, req ScheduleRequest) (*ScheduleReadResponse, error) {
	body, err := encode(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.scheduleAPI(), id), body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error updating schedule %s: %s", req.Name, readBody(resp))
	}
	var out ScheduleReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteSchedule deletes a schedule. The server treats deleting an
// already-deleted schedule as success.
func (c *Client) DeleteSchedule(ctx context.Context, id string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.scheduleAPI(), id), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting schedule %s: %s", id, readBody(resp))
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func encode(v any) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, fmt.Errorf("failed to json encode request body: %w", err)
	}
	return buf, nil
}

func mustReq(method, url string, body *bytes.Buffer) *http.Request {
	var r *http.Request
	if body == nil {
		r, _ = http.NewRequest(method, url, nil)
	} else {
		r, _ = http.NewRequest(method, url, body)
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(b))
}

// resolveToken mirrors the Terraform provider's token resolution: an explicit
// token (raw, or JSON in either the deployments-config or simple shape), with a
// fallback to ~/.adaptive/token.
func resolveToken(serviceToken, workspaceURL string) (string, string, error) {
	if serviceToken == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("serviceToken not provided and failed to read ~/.adaptive/token: %w", err)
		}
		b, err := os.ReadFile(path.Join(home, ".adaptive", "token"))
		if err != nil {
			return "", "", fmt.Errorf("serviceToken not provided and failed to read ~/.adaptive/token: %w", err)
		}
		serviceToken = string(b)
	}

	// deployments config shape
	var deployments struct {
		Deployments map[string]struct {
			URL     string `json:"url"`
			Token   string `json:"token"`
			Default bool   `json:"default,omitempty"`
		} `json:"deployments"`
	}
	if err := json.Unmarshal([]byte(serviceToken), &deployments); err == nil && len(deployments.Deployments) > 0 {
		for _, d := range deployments.Deployments {
			if d.Default {
				return d.Token, d.URL, nil
			}
		}
		for _, d := range deployments.Deployments {
			return d.Token, d.URL, nil
		}
	}

	// simple {token,url} shape
	var simple struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	if err := json.Unmarshal([]byte(serviceToken), &simple); err == nil && simple.Token != "" {
		return simple.Token, simple.URL, nil
	}

	// raw string
	return strings.TrimSpace(serviceToken), workspaceURL, nil
}
