package adaptive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// State-refresh reads.
//
// These are distinct from readSession and readAuthorization, which are
// create-time waiters: they poll until the object stops reporting "creating".
// A refresh needs a single request, and an object deleted out of band has to
// come back as not-found rather than as an error.
//
// Every function here returns (nil, nil) when the object no longer exists, so
// a Read can return an empty ID and let Pulumi plan a recreate.

// readInto performs a single GET and decodes the body into out. found is false
// on 404. 202 is accepted alongside 200 because the session read answers 202.
func (c *Client) readInto(ctx context.Context, url, kind, id string, out interface{}) (bool, error) {
	resp, err := c.do(ctx, mustReq("GET", url, nil))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body) // drain to allow connection reuse
		return false, fmt.Errorf("error reading %s %s (status %d): %s", kind, id, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return false, fmt.Errorf("failed to decode %s response: %w", kind, err)
	}
	return true, nil
}

// ResourceResponse mirrors the backend's TerraformResourceResponse.
//
// Configuration carries the stored settings with credentials removed, and
// RedactedKeys names what was removed. Attributes named there are write-only
// and must keep whatever the program last declared.
type ResourceResponse struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	IntegrationType string                 `json:"integrationType"`
	UserTags        []string               `json:"userTags"`
	DefaultCluster  string                 `json:"defaultCluster,omitempty"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
	RedactedKeys    []string               `json:"redactedKeys,omitempty"`
}

func (c *Client) GetResource(ctx context.Context, id string) (*ResourceResponse, error) {
	var resp ResourceResponse
	found, err := c.readInto(ctx, fmt.Sprintf("%s/read/%s", c.resourceAPI(), id), "resource", id, &resp)
	if err != nil || !found {
		return nil, err
	}
	return &resp, nil
}

// SessionResponse mirrors the backend's TerraformSessionResponse. Resource,
// Cluster, Authorization and Groups come back as names, matching what the write
// path accepts.
type SessionResponse struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Resource         string   `json:"resource"`
	Cluster          string   `json:"cluster,omitempty"`
	Authorization    string   `json:"authorization,omitempty"`
	Status           string   `json:"status"`
	SessionType      string   `json:"sessionType"`
	TTL              string   `json:"ttl,omitempty"`
	IdleTimeout      string   `json:"idleTimeout,omitempty"`
	PauseTimeout     string   `json:"pauseTimeout,omitempty"`
	Memory           string   `json:"memory,omitempty"`
	CPU              string   `json:"cpu,omitempty"`
	IsJITEnabled     bool     `json:"isJitEnabled"`
	SessionUsers     []string `json:"sessionUsers"`
	AccessApprovers  []string `json:"accessApprovers"`
	Groups           []string `json:"groups"`
	UserTags         []string `json:"userTags"`
	ScriptOnlyAccess bool     `json:"scriptOnlyAccess"`
}

func (c *Client) GetSession(ctx context.Context, id string) (*SessionResponse, error) {
	var resp SessionResponse
	found, err := c.readInto(ctx, fmt.Sprintf("%s/read/%s", c.sessionAPI(), id), "endpoint", id, &resp)
	if err != nil || !found {
		return nil, err
	}
	return &resp, nil
}

// AuthorizationResponse mirrors the backend's TerraformAuthorizationResponse.
type AuthorizationResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ResourceType string `json:"resourceType"`
	Permissions  string `json:"permissions"`
	Status       string `json:"status"`
}

func (c *Client) GetAuthorization(ctx context.Context, id string) (*AuthorizationResponse, error) {
	var resp AuthorizationResponse
	found, err := c.readInto(ctx, fmt.Sprintf("%s/read/%s", c.authorizationAPI(), id), "authorization", id, &resp)
	if err != nil || !found {
		return nil, err
	}
	return &resp, nil
}

// TeamResponse mirrors the backend's TerraformTeamResponse. Members are emails
// and endpoints are endpoint names, matching the write path.
type TeamResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Members   []string `json:"members"`
	Endpoints []string `json:"endpoints"`
}

func (c *Client) GetTeam(ctx context.Context, id string) (*TeamResponse, error) {
	var resp TeamResponse
	found, err := c.readInto(ctx, fmt.Sprintf("%s/read/%s", c.teamAPI(), id), "group", id, &resp)
	if err != nil || !found {
		return nil, err
	}
	return &resp, nil
}

// ScriptResponse mirrors the backend's TerraformScriptResponse.
//
// There is no Command field: a script body routinely embeds credentials, so the
// backend withholds it and reports CommandOmitted instead.
type ScriptResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Endpoint       string `json:"endpoint"`
	Description    string `json:"description,omitempty"`
	CommandOmitted bool   `json:"commandOmitted"`
}

func (c *Client) GetScript(ctx context.Context, id string) (*ScriptResponse, error) {
	var resp ScriptResponse
	found, err := c.readInto(ctx, fmt.Sprintf("%s/read/%s", c.scriptAPI(), id), "script", id, &resp)
	if err != nil || !found {
		return nil, err
	}
	return &resp, nil
}
