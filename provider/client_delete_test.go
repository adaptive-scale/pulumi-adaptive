package adaptive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeleteSessionWaitsForTerminationBeforeForceDelete(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	var reads atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.URL.Path)
		mu.Unlock()

		switch r.URL.Path {
		case "/api/v1/terraform/session/delete/session-1":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/terraform/session/read/session-1":
			readCount := reads.Add(1)
			status := "terminating"
			if readCount >= 2 {
				status = "terminated"
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprintf(w, `{"id":"session-1","status":%q}`, status)
		case "/api/v1/terraform/session/forcedelete/session-1":
			if reads.Load() < 2 {
				t.Error("force-delete was called before the endpoint terminated")
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient("token", server.URL)
	c.deletePollInterval = time.Millisecond
	c.deleteTimeout = 100 * time.Millisecond
	if err := c.DeleteSession(context.Background(), "session-1"); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	wantLast := "/api/v1/terraform/session/forcedelete/session-1"
	if len(calls) < 4 || calls[len(calls)-1] != wantLast {
		t.Fatalf("calls = %v, want final call %s", calls, wantLast)
	}
}

func TestDeleteSessionForceDeletesAStuckTerminatingEndpoint(t *testing.T) {
	var forceDeleteCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/terraform/session/delete/session-2":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/terraform/session/read/session-2":
			if forceDeleteCalled.Load() {
				http.Error(w, "session not found", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = fmt.Fprint(w, `{"id":"session-2","status":"terminating"}`)
		case "/api/v1/sessions/force-delete":
			forceDeleteCalled.Store(true)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient("token", server.URL)
	c.deletePollInterval = time.Millisecond
	c.deleteTimeout = 5 * time.Millisecond
	if err := c.DeleteSession(context.Background(), "session-2"); err != nil {
		t.Fatal(err)
	}
	if !forceDeleteCalled.Load() {
		t.Fatal("console force-delete fallback was not called")
	}
}

func TestDeleteSessionIsIdempotentWhenEndpointIsGone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("token", server.URL)
	if err := c.DeleteSession(context.Background(), "missing"); err != nil {
		t.Fatal(err)
	}
}
