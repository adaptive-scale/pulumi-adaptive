//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefreshClean deploys a Resource + Endpoint (with server-defaulted
// memory/cpu/idleTimeout and the default "direct" type) and asserts a refresh
// reports no changes. This is the perpetual-diff regression test.
func TestRefreshClean(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-refresh-res")
	epName := uniqueName("pulumi-it-refresh-ep")

	_, stack := harness.DeployStack(t, cfg, stackName("refresh"), func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(resName),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.com"),
			Port:     pulumi.String("5432"),
			Username: pulumi.String("admin"),
			Password: pulumi.String("not-a-real-password"),
			SslMode:  pulumi.String("require"),
		})
		if err != nil {
			return err
		}
		_, err = adaptive.NewEndpoint(ctx, "ep", &adaptive.EndpointArgs{
			Name:     pulumi.String(epName),
			Resource: db.Name,
			Ttl:      pulumi.String("8h"),
			// type/memory/cpu/idleTimeout deliberately unset: the server fills
			// defaults, and refresh must not adopt them into inputs.
		})
		return err
	})

	harness.AssertRefreshClean(t, stack)
}

// TestRefreshDetectsOutOfBandDelete deploys an Authorization, deletes it
// behind Pulumi's back, and asserts refresh drops it from state and a
// subsequent up recreates it.
func TestRefreshDetectsOutOfBandDelete(t *testing.T) {
	cfg := harness.RequireConfig(t)
	authName := uniqueName("pulumi-it-oob-auth")

	program := func(ctx *pulumi.Context) error {
		auth, err := adaptive.NewAuthorization(ctx, "auth", &adaptive.AuthorizationArgs{
			Name:         pulumi.String(authName),
			ResourceType: pulumi.String("postgres"),
			Permissions:  pulumi.String("allow:\n  - database: postgres\n    privileges: [\"SELECT\"]\n    objects: [\"*.*\"]\n"),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", auth.ID())
		return nil
	}

	outs, stack := harness.DeployStack(t, cfg, stackName("oobdel"), program)
	id := harness.StringOutput(t, outs, "id")

	// Delete it out-of-band.
	resp, body := rawTerraformAPI(t, cfg, "POST", "/authorization/delete/"+id, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "out-of-band delete failed: %s", body)

	changes := harness.Refresh(t, stack)
	assert.GreaterOrEqual(t, changes["delete"], 1, "refresh should drop the deleted authorization (changes: %v)", changes)

	// A subsequent up recreates it.
	preview := harness.Preview(t, stack)
	assert.GreaterOrEqual(t, preview["create"], 1, "preview after refresh should plan a create (changes: %v)", preview)
}

// TestRefreshDetectsDrift changes an endpoint's TTL out-of-band and asserts
// refresh picks up the new value into state.
func TestRefreshDetectsDrift(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-drift-res")
	epName := uniqueName("pulumi-it-drift-ep")

	outs, stack := harness.DeployStack(t, cfg, stackName("drift"), func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(resName),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.com"),
			Port:     pulumi.String("5432"),
			Username: pulumi.String("admin"),
			Password: pulumi.String("not-a-real-password"),
			SslMode:  pulumi.String("require"),
		})
		if err != nil {
			return err
		}
		ep, err := adaptive.NewEndpoint(ctx, "ep", &adaptive.EndpointArgs{
			Name:     pulumi.String(epName),
			Resource: db.Name,
			Ttl:      pulumi.String("8h"),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", ep.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	// Out-of-band TTL change (full update payload, same as the provider sends).
	resp, body := rawTerraformAPI(t, cfg, "POST", "/session/update/"+id, map[string]any{
		"sessionName":  epName,
		"resourceName": resName,
		"sessionType":  "cli",
		"sessionTTL":   "3d",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode, "out-of-band update failed: %s", body)

	changes := harness.Refresh(t, stack)
	assert.GreaterOrEqual(t, changes["update"], 1, "refresh should record the TTL drift (changes: %v)", changes)
}

// TestImportEndpoint creates a Resource + Endpoint out-of-band via the raw
// API, then imports the endpoint into a fresh stack by ID.
func TestImportEndpoint(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-import-res")
	epName := uniqueName("pulumi-it-import-ep")

	// Create the resource out-of-band.
	resResp, resBody := rawTerraformAPI(t, cfg, "POST", "/resource/create", map[string]any{
		"integrationType": "postgres",
		"name":            resName,
		"config": fmt.Sprintf("name: %s\nusername: admin\npassword: not-a-real-password\nhostname: db.example.com\nport: \"5432\"\nsslmode: require\n",
			resName),
		"userTags": []string{},
	})
	require.Equal(t, http.StatusOK, resResp.StatusCode, "resource create failed: %s", resBody)
	var res struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(resBody, &res))
	t.Cleanup(func() {
		_, _ = rawTerraformAPI(t, cfg, "POST", "/resource/delete/"+res.ID, nil)
	})

	// Create the endpoint out-of-band.
	epResp, epBody := rawTerraformAPI(t, cfg, "POST", "/session/create", map[string]any{
		"sessionName":  epName,
		"resourceName": resName,
		"sessionType":  "cli",
		"sessionTTL":   "8h",
	})
	require.Equal(t, http.StatusOK, epResp.StatusCode, "session create failed: %s", epBody)
	var ep struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(epBody, &ep))
	t.Cleanup(func() {
		_, _ = rawTerraformAPI(t, cfg, "POST", "/session/delete/"+ep.ID, nil)
		_, _ = rawTerraformAPI(t, cfg, "POST", "/session/forcedelete/"+ep.ID, nil)
	})

	// Import into a fresh (empty-program) stack.
	_, stack := harness.DeployStack(t, cfg, stackName("import"), func(ctx *pulumi.Context) error {
		return nil
	})
	generated := harness.ImportResource(t, stack, "adaptive:index:Endpoint", "imported-ep", ep.ID)
	assert.NotEmpty(t, generated, "import should generate program code")
	assert.Contains(t, generated, epName, "generated code should carry the endpoint name")
}

// TestImportUnknownIDFails pins that an import naming an id the workspace does
// not have is rejected outright. It used to succeed: the provider signalled
// not-found with an empty ReadResponse, which the engine only honours for
// refresh, so the bogus id landed in state with empty inputs and every later
// preview showed a diff.
func TestImportUnknownIDFails(t *testing.T) {
	cfg := harness.RequireConfig(t)
	_, stack := harness.DeployStack(t, cfg, stackName("import-bad"), func(ctx *pulumi.Context) error {
		return nil
	})

	for _, typ := range []string{"adaptive:index:Endpoint", "adaptive:index:Resource"} {
		t.Run(typ, func(t *testing.T) {
			_, err := harness.ImportResourceErr(t, stack, typ, "nope", uniqueName("not-a-real-id"))
			require.Error(t, err, "import of an unknown id must fail")
			assert.Contains(t, err.Error(), "cannot import",
				"the failure should name the id and say it does not exist")
		})
	}

	// Nothing may have been written to state by the failed imports: the program
	// is empty, so anything that had landed there would show up as a delete.
	assert.Zero(t, harness.Preview(t, stack)["delete"],
		"a rejected import must leave state untouched")
}
