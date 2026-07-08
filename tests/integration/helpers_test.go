//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"
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
