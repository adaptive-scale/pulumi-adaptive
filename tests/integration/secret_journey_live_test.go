//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// These are stored-configuration lifecycle tests, not database-login tests.
// Every stack owns one uniquely named integration on the explicitly selected
// backend. No existing integrations or endpoints are adopted.
func TestSecretJourneyLive(t *testing.T) {
	cfg := harness.RequireProviderConfig(t)
	for _, typ := range []string{"postgres", "snowflake"} {
		t.Run(typ, func(t *testing.T) {
			for _, journey := range []struct {
				name string
				run  func(*testing.T, harness.Config, string)
			}{
				{"program_transitions", journeyProgramTransitions},
				{"external_managed", journeyExternalManaged},
				{"external_empty_and_omitted", journeyExternalEmpty},
				{"multiple_secrets", journeyMultipleSecrets},
				{"out_of_band_deletion", journeyDeletion},
				{"sampling_boundary", journeySamplingBoundary},
			} {
				t.Run(journey.name, func(t *testing.T) { journey.run(t, cfg, typ) })
			}
		})
	}
}

const journeyPasswordA = "synthetic-journey-password-A"
const journeyPasswordB = "synthetic-journey-password-B"
const journeyCertificate = "-----BEGIN CERTIFICATE-----\nsynthetic-journey-certificate\n-----END CERTIFICATE-----"

type secretJourney struct {
	cfg                   harness.Config
	typ, name, id         string
	password, certificate *string
	tags                  []string
	stack                 *auto.Stack
}

func newSecretJourney(t *testing.T, cfg harness.Config, typ string, password, certificate *string) *secretJourney {
	t.Helper()
	j := &secretJourney{cfg: cfg, typ: typ, name: uniqueName("pulumi-it-journey-" + typ), password: password, certificate: certificate}
	// Register before DeployStack so its destroy cleanup executes first even
	// when an assertion aborts a journey before the explicit destroy step.
	t.Cleanup(func() {
		if j.id != "" {
			assertTerraformGone(t, cfg, "resource", j.id)
		}
	})
	outputs, stack := harness.DeployStack(t, cfg, stackName("journey-"+typ), func(ctx *pulumi.Context) error {
		args := &adaptive.ResourceArgs{Name: pulumi.String(j.name), Type: pulumi.String(j.typ), Host: pulumi.String("db.example.invalid"), Username: pulumi.String("admin")}
		if j.typ == "postgres" {
			args.Port = pulumi.String("5432")
			args.SslMode = pulumi.String("require")
		}
		if j.password != nil {
			args.Password = pulumi.String(*j.password)
		}
		if j.certificate != nil {
			if j.typ == "postgres" {
				args.TlsRootCert = pulumi.String(*j.certificate)
			} else {
				args.Clientcert = pulumi.String(*j.certificate)
			}
		}
		if j.tags != nil {
			args.Tags = pulumi.ToStringArray(j.tags)
		}
		db, err := adaptive.NewResource(ctx, "db", args)
		if err != nil {
			return err
		}
		ctx.Export("id", db.ID())
		return nil
	})
	j.stack, j.id = stack, harness.StringOutput(t, outputs, "id")
	j.clean(t)
	return j
}

func (j *secretJourney) passwordKey() string {
	if j.typ == "snowflake" {
		return "databasePassword"
	}
	return "password"
}
func (j *secretJourney) certificateKey() string {
	if j.typ == "snowflake" {
		return "clientcert"
	}
	return "rootCert"
}

func journeyCase(t *testing.T, name string, run func(*testing.T)) {
	t.Helper()
	if !t.Run(name, run) {
		t.FailNow()
	}
}

func (j *secretJourney) clean(t *testing.T) {
	t.Helper()
	harness.Refresh(t, j.stack)
	plan := harness.Preview(t, j.stack)
	require.Zero(t, plan["update"])
	require.Zero(t, plan["create"])
	require.Zero(t, plan["delete"])
	// Automation API Export deliberately adds --show-secrets. Use a normal
	// CLI export for the encrypted-checkpoint assertion, never print its data.
	command := exec.CommandContext(context.Background(), "pulumi", "stack", "export", "--stack", j.stack.Name(), "--non-interactive")
	command.Dir = j.stack.Workspace().WorkDir()
	command.Env = os.Environ()
	for key, value := range j.stack.Workspace().GetEnvVars() {
		command.Env = append(command.Env, key+"="+value)
	}
	deployment, err := command.Output()
	require.NoError(t, err)
	for _, canary := range []string{journeyPasswordA, journeyPasswordB, journeyCertificate} {
		encoded, err := json.Marshal(canary)
		require.NoError(t, err)
		require.False(t, strings.Contains(string(deployment), string(encoded)), "plaintext test credential appeared in encrypted checkpoint export")
	}
}

func (j *secretJourney) up(t *testing.T) {
	t.Helper()
	harness.Up(t, j.stack)
	j.clean(t)
}

func (j *secretJourney) remote(t *testing.T) map[string]any {
	t.Helper()
	remote := terraformRead(t, j.cfg, "resource", j.id)
	configuration, ok := remote["configuration"].(map[string]any)
	require.True(t, ok)
	return configuration
}

func (j *secretJourney) remotePassword(t *testing.T, password string) {
	t.Helper()
	configuration := j.remote(t)
	configuration["name"], configuration["type"] = j.name, j.typ
	configuration[j.passwordKey()] = password
	if j.certificate != nil {
		configuration[j.certificateKey()] = *j.certificate
	}
	encoded, err := yaml.Marshal(configuration)
	require.NoError(t, err)
	before := serverDigests(t, j.cfg, j.id)[j.passwordKey()]
	response, _ := rawTerraformAPI(t, j.cfg, "POST", "/resource/update/"+j.id, map[string]any{"integrationType": j.typ, "config": string(encoded), "userTags": j.tags})
	require.Equal(t, 200, response.StatusCode)
	after := serverDigests(t, j.cfg, j.id)[j.passwordKey()]
	if password == "" {
		value, present := j.remote(t)[j.passwordKey()]
		require.True(t, present)
		require.Equal(t, "", value)
		require.Empty(t, after)
	} else {
		require.NotEmpty(t, after)
	}
	require.NotEqual(t, before, after, "external mutation must actually change the fingerprint")
}

func (j *secretJourney) dirty(t *testing.T) {
	t.Helper()
	for cycle := 1; cycle <= 3; cycle++ {
		harness.Refresh(t, j.stack)
		require.Equal(t, 1, harness.Preview(t, j.stack)["update"], "cycle %d", cycle)
	}
}

func (j *secretJourney) rejectOmission(t *testing.T, argument string) {
	t.Helper()
	before := terraformRead(t, j.cfg, "resource", j.id)
	_, err := j.stack.Up(context.Background())
	require.ErrorContains(t, err, "server-held values for "+argument)
	after := terraformRead(t, j.cfg, "resource", j.id)
	for _, key := range []string{"configuration", "redactedDigests", "userTags"} {
		left, err := json.Marshal(before[key])
		require.NoError(t, err)
		right, err := json.Marshal(after[key])
		require.NoError(t, err)
		require.Equal(t, string(left), string(right), "rejected update changed remote "+key)
	}
}

func (j *secretJourney) destroy(t *testing.T) {
	t.Helper()
	harness.Destroy(t, j.stack)
	assertTerraformGone(t, j.cfg, "resource", j.id)
}

func journeyProgramTransitions(t *testing.T, cfg harness.Config, typ string) {
	j := newSecretJourney(t, cfg, typ, nil, nil)
	journeyCase(t, "1A_create_omitted", func(t *testing.T) { require.NotNil(t, digests(t, j.stack)); require.Empty(t, digests(t, j.stack)) })
	journeyCase(t, "2A_omitted_to_managed", func(t *testing.T) {
		j.password = strPtr(journeyPasswordA)
		j.up(t)
		require.NotEmpty(t, serverDigests(t, cfg, j.id)[j.passwordKey()])
	})
	first := serverDigests(t, cfg, j.id)[j.passwordKey()]
	journeyCase(t, "2C_managed_A_to_B", func(t *testing.T) {
		j.password = strPtr(journeyPasswordB)
		j.up(t)
		require.NotEqual(t, first, serverDigests(t, cfg, j.id)[j.passwordKey()])
	})
	journeyCase(t, "2D_managed_to_clear", func(t *testing.T) {
		j.password = strPtr("")
		j.up(t)
		require.Equal(t, "", j.remote(t)[j.passwordKey()])
		require.Empty(t, serverDigests(t, cfg, j.id))
	})
	journeyCase(t, "2F_clear_then_omit", func(t *testing.T) { j.password = nil; j.up(t) })
	journeyCase(t, "2B_clear_to_managed", func(t *testing.T) { j.password = strPtr(""); j.up(t); j.password = strPtr(journeyPasswordA); j.up(t) })
	journeyCase(t, "2E_managed_to_omitted_rejected", func(t *testing.T) {
		j.password = nil
		j.tags = []string{"journey=omission-guard"}
		j.rejectOmission(t, "password")
		j.password = strPtr(journeyPasswordA)
		j.up(t)
	})
	j.destroy(t)
}

func journeyExternalManaged(t *testing.T, cfg harness.Config, typ string) {
	j := newSecretJourney(t, cfg, typ, strPtr(journeyPasswordA), nil)
	journeyCase(t, "1C_create_managed", func(t *testing.T) { require.NotEmpty(t, digests(t, j.stack)[j.passwordKey()]) })
	original := serverDigests(t, cfg, j.id)[j.passwordKey()]
	j.remotePassword(t, journeyPasswordB)
	journeyCase(t, "8_no_refresh_no_remote_observation", func(t *testing.T) { require.Zero(t, harness.Preview(t, j.stack)["update"]) })
	journeyCase(t, "3A_4A_rotate_and_repeat_refresh", func(t *testing.T) { j.dirty(t); require.Equal(t, original, digests(t, j.stack)[j.passwordKey()]) })
	journeyCase(t, "4B_restore_managed_A", func(t *testing.T) { j.up(t); require.Equal(t, original, serverDigests(t, cfg, j.id)[j.passwordKey()]) })
	journeyCase(t, "3B_remote_clear_restore_A", func(t *testing.T) {
		j.remotePassword(t, "")
		j.dirty(t)
		j.up(t)
		require.Equal(t, original, serverDigests(t, cfg, j.id)[j.passwordKey()])
	})
	journeyCase(t, "4D_adopt_rotation_by_explicit_write", func(t *testing.T) {
		j.remotePassword(t, journeyPasswordB)
		observed := serverDigests(t, cfg, j.id)[j.passwordKey()]
		j.dirty(t)
		j.password = strPtr(journeyPasswordB)
		j.up(t)
		require.Equal(t, observed, digests(t, j.stack)[j.passwordKey()])
	})
	journeyCase(t, "3E_4E_omit_after_rotation_rejected", func(t *testing.T) {
		j.remotePassword(t, journeyPasswordA)
		j.dirty(t)
		j.password = nil
		j.tags = []string{"journey=rotated-omission"}
		j.rejectOmission(t, "password")
		j.password = strPtr(journeyPasswordB)
		j.up(t)
	})
	j.destroy(t)
}

func journeyExternalEmpty(t *testing.T, cfg harness.Config, typ string) {
	j := newSecretJourney(t, cfg, typ, strPtr(""), nil)
	journeyCase(t, "1B_create_explicit_clear", func(t *testing.T) { require.NotNil(t, digests(t, j.stack)); require.Empty(t, digests(t, j.stack)) })
	journeyCase(t, "3C_4C_empty_external_add_clear", func(t *testing.T) {
		j.remotePassword(t, journeyPasswordA)
		j.dirty(t)
		require.Empty(t, digests(t, j.stack))
		j.up(t)
		require.Empty(t, serverDigests(t, cfg, j.id))
	})
	journeyCase(t, "3D_omitted_external_add", func(t *testing.T) {
		j.password = nil
		j.up(t)
		j.remotePassword(t, journeyPasswordB)
		j.dirty(t)
		j.tags = []string{"journey=added-omission"}
		j.rejectOmission(t, "password")
		j.password = strPtr(journeyPasswordA)
		j.up(t)
	})
	journeyCase(t, "3F_prune_last_then_readd", func(t *testing.T) {
		j.remotePassword(t, "")
		harness.Refresh(t, j.stack)
		require.NotNil(t, digests(t, j.stack))
		require.Empty(t, digests(t, j.stack))
		j.password = nil
		j.up(t)
		j.remotePassword(t, journeyPasswordA)
		j.dirty(t)
		j.password = strPtr("")
		j.up(t)
	})
	j.destroy(t)
}

func journeyMultipleSecrets(t *testing.T, cfg harness.Config, typ string) {
	j := newSecretJourney(t, cfg, typ, strPtr(journeyPasswordA), strPtr(journeyCertificate))
	certificate := serverDigests(t, cfg, j.id)[j.certificateKey()]
	require.NotEmpty(t, certificate)
	journeyCase(t, "7C_nonsecret_update_resupplies_held", func(t *testing.T) {
		j.tags = []string{"journey=metadata"}
		j.up(t)
		require.Equal(t, certificate, serverDigests(t, cfg, j.id)[j.certificateKey()])
	})
	journeyCase(t, "7A_clear_password_retain_certificate", func(t *testing.T) {
		j.password = strPtr("")
		j.up(t)
		remote := serverDigests(t, cfg, j.id)
		require.Empty(t, remote[j.passwordKey()])
		require.Equal(t, certificate, remote[j.certificateKey()])
	})
	journeyCase(t, "7B_omit_certificate_while_clearing_password_rejected", func(t *testing.T) {
		j.certificate = nil
		j.tags = []string{"journey=certificate-omission"}
		argument := "tlsRootCert"
		if typ == "snowflake" {
			argument = "clientcert"
		}
		j.rejectOmission(t, argument)
		j.certificate = strPtr(journeyCertificate)
		j.up(t)
	})
	journeyCase(t, "7A_clear_both_then_omit", func(t *testing.T) {
		j.certificate = strPtr("")
		j.up(t)
		require.Empty(t, serverDigests(t, cfg, j.id))
		j.password = nil
		j.certificate = nil
		j.up(t)
	})
	j.destroy(t)
}

func journeyDeletion(t *testing.T, cfg harness.Config, typ string) {
	j := newSecretJourney(t, cfg, typ, strPtr(""), nil)
	oldID := j.id
	journeyCase(t, "6C_deleted_resource_refresh_recreate", func(t *testing.T) {
		rawTerraformAPI(t, cfg, "POST", "/resource/delete/"+oldID, nil)
		assertTerraformGone(t, cfg, "resource", oldID)
		harness.Refresh(t, j.stack)
		require.Equal(t, 1, harness.Preview(t, j.stack)["create"])
		outputs := harness.Up(t, j.stack)
		j.id = harness.StringOutput(t, outputs, "id")
		require.NotEqual(t, oldID, j.id)
		j.clean(t)
	})
	j.destroy(t)
	assertTerraformGone(t, cfg, "resource", oldID)
}

func journeySamplingBoundary(t *testing.T, cfg harness.Config, typ string) {
	j := newSecretJourney(t, cfg, typ, strPtr(journeyPasswordA), nil)
	journeyCase(t, "8_A_B_A_between_refreshes_is_unobservable", func(t *testing.T) {
		original := serverDigests(t, cfg, j.id)[j.passwordKey()]
		j.remotePassword(t, journeyPasswordB)
		j.remotePassword(t, journeyPasswordA)
		require.Equal(t, original, serverDigests(t, cfg, j.id)[j.passwordKey()])
		j.clean(t)
	})
	j.destroy(t)
}
