//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/stretchr/testify/require"
)

// uniqueName returns a per-run unique object name, so repeated test runs don't
// collide on the backend's name-uniqueness constraints.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// stackName returns a unique, valid Pulumi stack name.
func stackName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// find returns the first item matching pred, or nil.
func find[T any](items []T, pred func(T) bool) *T {
	for i := range items {
		if pred(items[i]) {
			return &items[i]
		}
	}
	return nil
}

// retryFind polls a list function until an item matches (tolerating brief
// read-after-write lag on the Client API), failing the test if none appears.
func retryFind[T any](t *testing.T, desc string, list func() ([]T, error), pred func(T) bool) T {
	t.Helper()
	var lastErr error
	for i := 0; i < 6; i++ {
		items, err := list()
		if err != nil {
			lastErr = err
		} else if f := find(items, pred); f != nil {
			return *f
		}
		time.Sleep(3 * time.Second)
	}
	if lastErr != nil {
		t.Fatalf("%s: client API error: %v", desc, lastErr)
	}
	t.Fatalf("%s: not found via Client API after retries", desc)
	return *new(T)
}

// rawTerraformAPI issues a request against the /api/v1/terraform surface with
// the service token, for out-of-band mutations the tests need.
func rawTerraformAPI(t *testing.T, cfg harness.Config, method, path string, body any) (*http.Response, []byte) {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	url := strings.TrimRight(cfg.URL, "/") + "/api/v1/terraform" + path
	req, err := http.NewRequest(method, url, buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", cfg.ServiceToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	return resp, out.Bytes()
}
