package adaptive

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestServicesAuthorizationClientCompatibility(t *testing.T) {
	for _, resourceType := range []string{"servicelist", "postgres", "ssh"} {
		t.Run(resourceType, func(t *testing.T) {
			const initial = "allow:\n  - operation_type: GET\n    route: /api/*\n"
			const updated = "deny:\n  - operation_type: DELETE\n    route: /api/*\n"
			policy := initial
			deleted := false
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "test-token" {
					t.Error("service token missing")
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/v1/terraform/authorization/create", "/api/v1/terraform/authorization/update/auth-1":
					if r.Method != http.MethodPost {
						t.Errorf("write method = %s", r.Method)
					}
					key, wantPolicy := "resource", initial
					if r.URL.Path != "/api/v1/terraform/authorization/create" {
						key, wantPolicy = "resourceType", updated
					}
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					want := map[string]string{"name": "services-read", "description": "policy", key: resourceType, "permissions": wantPolicy}
					if !reflect.DeepEqual(body, want) {
						t.Errorf("wire payload = %#v, want %#v", body, want)
					}
					policy = body["permissions"]
					if key == "resource" {
						_, _ = w.Write([]byte(`{"id":"auth-1"}`))
					} else {
						_, _ = w.Write([]byte(`{"ok":true}`))
					}
				case "/api/v1/terraform/authorization/read/auth-1":
					_ = json.NewEncoder(w).Encode(AuthorizationReadResponse{
						ID: "auth-1", Name: "services-read", ResourceType: resourceType, Permissions: policy, Status: "created",
					})
				case "/api/v1/terraform/authorization/delete/auth-1":
					deleted = true
					_, _ = w.Write([]byte(`{"ok":true}`))
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			})
			ctx := context.Background()
			created, err := c.CreateAuthorization(ctx, "services-read", "policy", initial, resourceType)
			if err != nil || created.ID != "auth-1" {
				t.Fatalf("create: (%+v, %v)", created, err)
			}
			if _, err := c.UpdateAuthorization(ctx, created.ID, "services-read", "policy", updated, resourceType); err != nil {
				t.Fatal(err)
			}
			read, err := c.ReadAuthorization(ctx, created.ID)
			if err != nil || read == nil {
				t.Fatalf("read: (%+v, %v)", read, err)
			}
			// Exercise the import/refresh mapping used by the native resource.
			args := applyAuthorizationRead(AuthorizationArgs{}, read, true)
			if args.ResourceType != resourceType || args.Permissions != updated {
				t.Fatalf("import mapping: %+v", args)
			}
			if err := c.DeleteAuthorization(ctx, created.ID); err != nil || !deleted {
				t.Fatalf("delete: deleted=%v, error=%v", deleted, err)
			}
		})
	}
}

func TestServicesAuthorizationUpdateReportsValidationFailure(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`"Invalid Services authorization rules"`))
	})
	if _, err := c.UpdateAuthorization(context.Background(), "auth", "services", "", "deny: []", "servicelist"); err == nil {
		t.Fatal("a rejected policy update must not report success")
	}
}
