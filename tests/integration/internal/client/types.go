//go:build integration

// Package client is a small read-only client for the Adaptive Client App API
// (/api/v3/client/*), used by the integration tests to verify resources created
// by the Pulumi provider. It authenticates with a Client ID + Client Secret.
package client

// Resource mirrors GET /api/v3/client/resources/list items.
type Resource struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Hostname string `json:"hostname"`
}

// Endpoint mirrors GET /api/v3/client/endpoints/list items.
type Endpoint struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	IntegrationType string `json:"integration_type"`
	Status          string `json:"status"`
	IntegrationName string `json:"integration_name"`
}

// Authorization mirrors GET /api/v3/client/authorizations/list items.
// The permission policy body is intentionally not returned by the API.
type Authorization struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ResourceType string `json:"resourceType"`
	Status       string `json:"status"`
	Description  string `json:"description"`
}

// Script mirrors GET /api/v3/client/scripts/list items.
// The script command/body is intentionally not returned by the API.
type Script struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SessionID   string `json:"sessionId"`
	Description string `json:"description"`
}

// Team mirrors GET /api/v3/client/teams/list items.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
