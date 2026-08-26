# Integration tests

These drive the real provider against a real Adaptive backend: every test creates
objects, verifies them, and destroys them. Nothing is mocked, which is the point
— secret fingerprints only move server-side, and state-shape regressions only
appear when Pulumi decodes state written by an earlier run.

## Two credential tiers

Most tests need only a **service token**. Only those that read objects back
through the Client App API — as an independent check that the provider wrote what
it claimed — need Client App credentials as well.

| Tier | Needs | Used by |
|---|---|---|
| `harness.RequireProviderConfig` | `ADAPTIVE_SVC_TOKEN` (or `~/.adaptive/token`) | drift, lifecycle, credentials, schedule, data protection |
| `harness.RequireConfig` | the above **plus** `ADAPTIVE_CLIENT_ID` / `ADAPTIVE_CLIENT_SECRET` | everything that verifies through the Client App API |

Tests skip cleanly when their tier is unavailable, so a service token alone gets
you a useful run.

## Running against a local backend

```sh
cd ../..                # repo root
make install            # the plugin must be on PATH; tests fail fast without it

export ADAPTIVE_URL=http://localhost:8080
export ADAPTIVE_SVC_TOKEN=...      # omit to use ~/.adaptive/token

cd tests/integration
go test -tags=integration -run 'Drift|Lifecycle|Credentials' -v -timeout 45m .
```

Or put the variables in `tests/.env.local` (gitignored, loaded automatically):

```sh
ADAPTIVE_URL=http://localhost:8080
ADAPTIVE_SVC_TOKEN=...
ADAPTIVE_CLIENT_ID=...
ADAPTIVE_CLIENT_SECRET=...
```

`make test-integration` from the repo root runs everything, installing the plugin
first.

## Things worth knowing before you debug a failure

**The `integration` build tag is mandatory.** Without it `go test` reports "no
test files" rather than failing, so a run that looks instantaneous did nothing.

**The token must be a workspace admin.** The `/terraform` read endpoints are
admin-gated, so a non-admin token makes refresh and import fail everywhere.
`preflight` checks this once and says so, rather than letting thirty tests fail
separately.

**Service tokens expire (~3h).** `preflight` catches that too, with the same
reasoning.

**Endpoint creation polls** — up to 5 minutes each, since the server reports
`creating` until the session is up. Hence `-timeout 45m` and `t.Parallel()` on
the slow tests. Each test has its own stack and its own file backend under
`t.TempDir()`, and names are nanosecond-unique, so parallelism is safe.

**Cleanup is best-effort on failure.** `DeployStack` destroys on `t.Cleanup` but
only logs if that fails, because failing there would mask the real error. After a
run that failed badly, check for leftover `pulumi-it-*` objects — they do not
collide with later runs, but they accumulate.

## What the suite covers

- **Per type** — create and verify through the Client App API.
- **Lifecycle** (`lifecycle_test.go`) — create → update → refresh → import →
  destroy, the paths a create-only test cannot reach.
- **Secret drift** (`secret_drift_test.go`) — a secret changed out-of-band, for a
  field the program manages and one it does not, plus `rootCert`, plus the quiet
  case. Also asserts the recorded fingerprints survive a refresh and move on an
  update, which is the property the mechanism rests on and which no diff shows.
- **Credentials** (`credentials_test.go`) — nothing credential-shaped reaches
  state, and leftover `adaptive:*` stack config is ignored rather than honoured.
- **Refresh/import** (`refresh_import_test.go`) — clean refresh, out-of-band
  delete, drift, import, and import of an id that does not exist.
