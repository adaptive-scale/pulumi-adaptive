package adaptive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func optionalGroupList(groups []string) *[]string {
	if groups == nil {
		return nil
	}
	return &groups
}

func (c *Client) requireJITApproverGroups(ctx context.Context, requireEnabled bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.workspaceURL+"/cli/admin/jit.approver-groups", nil)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return fmt.Errorf("cannot verify JIT approver group support; no changes submitted: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JIT approver groups are unsupported (HTTP %d); no changes submitted", resp.StatusCode)
	}
	var capability struct {
		Supported bool `json:"supported"`
		Enabled   bool `json:"enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&capability); err != nil {
		return err
	}
	if !capability.Supported || (requireEnabled && !capability.Enabled) {
		return fmt.Errorf("JIT approver group writes are unavailable; no changes submitted")
	}
	return nil
}
