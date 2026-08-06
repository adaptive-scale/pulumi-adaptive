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

	DisableOutputCapture bool `json:"disable_output_capture"`
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

func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.sessionAPI(), sessionID), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting session %s", sessionID)
	}
	return nil
}

func (c *Client) readSession(ctx context.Context, sessionID string) (map[string]any, error) {
	resp, err := c.do(ctx, mustReq("GET", fmt.Sprintf("%s/read/%s", c.sessionAPI(), sessionID), nil))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error reading session %s", sessionID)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) waitForSession(ctx context.Context, sessionID string) (map[string]any, error) {
	return Do(func() (map[string]any, error) {
		return c.readSession(ctx, sessionID)
	}, RetryLimit(30), Sleep(10*time.Second), RetryResultChecker(func(r any) bool {
		res, ok := r.(map[string]any)
		if !ok || res == nil {
			return true
		}
		status, ok := res["Status"].(string)
		if !ok {
			return true
		}
		return strings.ToLower(status) == "creating"
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

func (c *Client) readAuthorization(ctx context.Context, authID string) (map[string]any, error) {
	resp, err := c.do(ctx, mustReq("GET", fmt.Sprintf("%s/read/%s", c.authorizationAPI(), authID), nil))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error reading authorization %s", authID)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) waitForAuthorization(ctx context.Context, authID string) (map[string]any, error) {
	return Do(func() (map[string]any, error) {
		return c.readAuthorization(ctx, authID)
	}, RetryLimit(20), Sleep(10*time.Second), RetryResultChecker(func(r any) bool {
		res, ok := r.(map[string]any)
		if !ok || res == nil {
			return true
		}
		status, ok := res["Status"].(string)
		if !ok {
			return true
		}
		return strings.ToLower(status) == "creating"
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

func (c *Client) UpdateTeam(ctx context.Context, id, name string, members, endpoints []string) error {
	body, err := encode(map[string]any{"Name": name, "Members": members, "Endpoints": endpoints})
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

func (c *Client) CreateScript(ctx context.Context, name, command, endpoint string) (*IDResponse, error) {
	body, err := encode(map[string]any{"Name": name, "Command": command, "Endpoint": endpoint})
	if err != nil {
		return nil, err
	}
	resp, err := c.do(ctx, mustReq("POST", c.scriptAPI()+"/create", body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("duplicate script with name %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error creating script %s: %s", name, readBody(resp))
	}
	var out IDResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateScript(ctx context.Context, id, name, command, endpoint string) error {
	body, err := encode(map[string]any{"Name": name, "Command": command, "Endpoint": endpoint})
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/update/%s", c.scriptAPI(), id), body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("duplicate script with name %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error updating script %s", name)
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
