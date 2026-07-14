//go:build integration

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client calls the Adaptive Client App API with Client ID + Secret auth.
type Client struct {
	baseURL  string
	clientID string
	secret   string
	http     *http.Client
}

// New builds a client. baseURL is the Adaptive base (e.g. http://localhost:8080).
func New(baseURL, clientID, secret string) *Client {
	return &Client{
		baseURL:  baseURL,
		clientID: clientID,
		secret:   secret,
		http:     &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	req.Header.Set("X-Client-ID", c.clientID)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) ListResources() ([]Resource, error) {
	var r []Resource
	return r, c.get("/api/v3/client/resources/list", &r)
}

func (c *Client) ListEndpoints() ([]Endpoint, error) {
	var r []Endpoint
	return r, c.get("/api/v3/client/endpoints/list", &r)
}

func (c *Client) ListAuthorizations() ([]Authorization, error) {
	var r []Authorization
	return r, c.get("/api/v3/client/authorizations/list", &r)
}

func (c *Client) ListScripts() ([]Script, error) {
	var r []Script
	return r, c.get("/api/v3/client/scripts/list", &r)
}

func (c *Client) ListTeams() ([]Team, error) {
	var r []Team
	return r, c.get("/api/v3/client/teams/list", &r)
}

// ListTeamEndpoints returns the endpoints associated with a team. The route has
// a trailing slash, matching the backend route constant.
func (c *Client) ListTeamEndpoints(teamID string) ([]TeamEndpoint, error) {
	var r []TeamEndpoint
	return r, c.get("/api/v3/client/team/"+teamID+"/endpoint/list/", &r)
}
