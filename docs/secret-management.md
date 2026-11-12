---
title: Secret management
meta_desc: Manage credentials, explicit clears, imports, and secret drift with the Adaptive Pulumi provider.
layout: package
---

# Secret management and operational workflow

This guide covers credentials stored in an `adaptive.Resource` integration: passwords,
keys, certificates, tokens, and other fields the selected integration treats as
secrets. It explains the behavior implemented by this provider revision; use a
provider build containing these changes before relying on the updated empty-value,
drift, update-guard, and migration behavior. A source change or merged PR alone
does not publish a new provider release.

These workflows manage **stored Adaptive configuration**, not the actual password
on a database, external secret-manager rotation, or connection health. Script
bodies use a separate algorithm described at the end of this guide.

## The three rules

1. **Supply the intended value to retain or rotate a held credential.** An update
   replaces the integration's configuration; it is not a secret-preserving patch.
2. **Use an explicit empty string to authorize a supported clear.** Omitting a
   recognized server-held credential is rejected rather than silently deleting it.
3. **Refresh, preview, then apply.** Refresh observes remote changes and updates
   recorded state. It does not write your desired credentials back to Adaptive.

## Provider login is not a managed resource secret

The provider's Adaptive service token authenticates management API operations.
A resource password is a credential stored in the integration being managed.
They have different lifecycles:

| Credential | Source | Purpose |
| --- | --- | --- |
| Provider service token | `ADAPTIVE_SVC_TOKEN`, or the CLI login fallback | Authorize provider API calls. Not a resource password or a declared Pulumi provider configuration field. |
| Managed resource secret | A resource argument, preferably from secret Pulumi config | Configure the selected integration. Stored as a secret property in Pulumi state when schema-marked secret. |

See [installation and configuration](./installation-configuration.md) for login
and workspace selection. Confirm the selected Adaptive workspace and Pulumi stack
before operating. A URL embedded in supported JSON login credentials can determine
the target; do not assume setting `ADAPTIVE_URL` alone overrides that identity.

Do not put the service token in a resource password argument or configure it as
`adaptive:serviceToken`. Do not print login/token files to diagnose authentication.

## Store values securely

Set a secret interactively so its value is not embedded in a shell command:

```sh
pulumi config set --secret dbPassword
```

Read it as a secret in the program. For example, in Python:

```python
import pulumi
import pulumi_adaptive as adaptive

config = pulumi.Config()
db = adaptive.Resource(
    "db",
    name="example-postgres",
    type="postgres",
    host="db.example.internal",
    port="5432",
    username="app",
    ssl_mode="require",
    password=config.require_secret("dbPassword"),
)
```

In TypeScript, use `new pulumi.Config().requireSecret("dbPassword")` for the
`password` argument. Other language SDKs expose the same resource contract using
their generated field names.

Schema-marked secret fields are masked in ordinary Pulumi previews and encrypted
in state, but they are **not absent from state**. Protect the state backend,
secrets-provider keys, configuration files, history, and CI access. Secret input
marking does not prevent your own program from printing a value or sending it to
an unrelated service. Prefer secret-aware config reads even when the destination
argument is schema-marked secret.

Avoid plaintext literals, decrypted exports, debug logs containing credentials,
and shell arguments containing real secret values. `pulumi stack export
--show-secrets` intentionally decrypts exported secrets; the Go Automation API's
`Stack.Export` also requests decrypted output in the SDK used by this repository.
A normal CLI export without that flag is the appropriate encrypted-export check.
Clearing a credential today does not remove it from historical checkpoints or
previously created plaintext exports.

## Omission, explicit clear, and managed values

| Program input | Meaning for this field |
| --- | --- |
| Argument omitted | No value is supplied by the program. It is **not** an instruction to preserve a held credential during replacement updates. |
| Explicit `""` | Intentionally clear the value, where that integration supports an empty setting. |
| Nonempty value | Supply the intended credential, whether retaining it or rotating it. |

Python omission means leaving out the argument, not using an empty string.
TypeScript omission means leaving it out or leaving it undefined.

The public argument and backend key need not match:

| Integration / field | Python argument | TypeScript argument | Backend configuration key |
| --- | --- | --- | --- |
| PostgreSQL password | `password` | `password` | `password` |
| Snowflake password | `password` | `password` | `databasePassword` |
| PostgreSQL root certificate | `tls_root_cert` | `tlsRootCert` | `rootCert` |
| Snowflake client certificate | `clientcert` | `clientcert` | `clientcert` |

Use public SDK arguments in the program. Do not substitute `database_password`
for the Snowflake `password` argument merely because the backend uses
`databasePassword`.

An empty password might be valid when another authentication method is used, but
successful configuration persistence is not proof that the target connection will
work. Confirm the integration's authentication requirements separately.

## Normal deployment workflow

From the intended project and selected stack:

```sh
pulumi refresh
pulumi preview
pulumi up
pulumi refresh
pulumi preview
```

Review the prompts and planned operations; these commands can affect the whole
stack. The first refresh records observations without rewriting remote
credentials. The final preview should have no remaining secret-field changes
once the intended write and its read-back have converged.

A normal preview does not automatically discover every external change. An
ordinary `up` without a planned update also does not necessarily rewrite a
credential. A clean preview is not a database-login test.

### Keep or rotate a managed secret

1. Keep every still-held credential available from its real secret source.
2. To rotate, update secret config with `pulumi config set --secret dbPassword`.
3. Keep the program reading `require_secret("dbPassword")` / `requireSecret`.
4. Run the workflow above, approving only the intended changes.
5. Verify connection/runtime behavior separately when credential validity matters.

A metadata-only change, such as a tag edit, still replaces the configuration.
It therefore requires explicit values or clears for every recognized held secret,
not just the field being changed.

### Clear a secret deliberately

For Snowflake, retain the resource's other settings and explicitly set:

```python
snowflake = adaptive.Resource(
    "warehouse",
    name="example-snowflake",
    type="snowflake",
    hostname="example-account",
    username="app",
    database_name="EXAMPLE_DB",
    warehouse="EXAMPLE_WH",
    schema="PUBLIC",
    role="EXAMPLE_ROLE",
    password="",  # Explicit clear, not omission.
)
```

1. Supply any other credentials that must remain held, such as `clientcert`.
2. Refresh and preview, then apply the explicit clear.
3. Refresh again and check that the next preview is clean.
4. Keep `password=""` if the desired state is explicitly empty. Matching empty
   values no longer produce recurring absent-versus-empty diffs.
5. Alternatively, remove the argument **after** the backend acknowledges clearing
   it. Review any one-time input-removal update, then refresh/preview again.

Do not jump directly from a managed password to omission while Adaptive still
holds it. Clearing password does not authorize deleting a separate certificate.

## External changes and drift

Adaptive's resource read API withholds recognized credential values and can
return opaque, server-generated fingerprints. The provider records applied
fingerprints after writes and compares subsequent observations to that baseline.
It may put a secret-typed fingerprint marker in refreshed state to make a field's
drift visible. A marker is not a credential: never copy it into config or source.

| Desired state / external event | After refresh | Resolution |
| --- | --- | --- |
| Managed A, remote still A | No secret drift when the recorded fingerprint matches | No corrective write needed. |
| Managed A, external rotation to B | Password change appears; repeated refreshes retain the drift | Apply to restore A, or deliberately supply B and apply to adopt the rotation. |
| Managed A, external clear | Missing desired credential appears | Apply to restore A, or change desired state to an explicit clear. |
| Explicit empty, someone adds a password | Conflict appears, including an initially empty baseline | Apply the explicit clear to remove the added password. |
| Omitted, someone adds a password after a known-empty baseline | Removal intent appears | Supply an intended value or explicitly clear; applying continued omission is rejected. |
| Last secret cleared, then added again | Addition is compared with the retained known-empty baseline | Choose retention/rotation or explicit clear. |

Refresh alone does not acknowledge or adopt a detected rotation as a successful
provider write. Repeated refreshes must not make an unresolved change disappear.
Applying an explicitly supplied rotated value is the adoption workflow.

## Importing an existing integration

Import reads identity and nonsecret settings; it cannot recover withheld
credentials. Acquire required values from an authorized secret source before an
update. Privileged secret export, if used, is a separate sensitive operation—not
a routine provider read or a prerequisite for simply importing identity.

```sh
pulumi import adaptive:index:Resource db <adaptive-resource-id>
```

Then:

1. Review generated/adopted settings, including the selected cluster.
2. Add each credential you intend to retain using secret config, or explicitly
   clear it if that is supported and intentional.
3. Preview. Supplying a previously withheld value can produce a one-time addition.
4. Apply, then refresh/preview for convergence.

A clean import preview does not prove that supplied plaintext matches the remote
credential, or that a future tag update can preserve omitted credentials. Initial
fingerprints are observations, not verification of historical ownership.

## Safety checks and recovery

The held-value guard runs during actual update, not every preview. Preview can
therefore succeed while `up` refuses the write.

| Symptom | Meaning / action |
| --- | --- |
| `updating resource would clear server-held values for ...` | Supply every named public argument with its intended value, or explicitly `""` to authorize clearing. Do not copy a fingerprint marker. |
| `cannot verify server-held values ... no update was sent` | The pre-update read failed or was detectably incomplete. Restore authentication/read permission, connectivity, and a usable backend response; refresh and retry. Do not bypass the guard. |
| CLI login/service token rejected | Refresh the CLI login with `adaptive login`, or refresh the authorized service-token injection. Confirm the target again. |
| Successful write followed by more drift | Post-write fingerprint capture is best-effort. Failed/partial capture retains prior evidence; verify actual remote state before retrying. |
| Recurring empty-password additions | Check that the active provider binary contains this revision's fix. Keep explicit empty distinct from omission; do not hide the symptom with `ignoreChanges`. |

Failed, not-found, or detectably incomplete pre-update reads stop the update
before its replacement request. The provider does not fall back to an empty
baseline to authorize clearing. Post-write captures are different: they cannot
undo an already completed write and retain previous evidence on failure.

Do not use `ignoreChanges` as a substitute for a credential source or explicit
clear intent. This provider does not offer a general "preserve omitted secrets"
mode for full-configuration updates.

## Baselines and upgrades

- **Unknown baseline:** no fingerprints were captured, for example after a failed
  create read or in legacy state. A complete later observation may initialize it;
  this does not prove whether a historical rotation occurred before that read.
- **Known-empty baseline:** an observation established no fingerprinted secrets.
  This remains an initialized baseline, so a later first addition is detectable.
- **Recorded baseline:** retain applied fingerprints across refreshes, except
  acknowledged clears. Missing metadata is not proof of absence.
- **Explicit empty intent:** a recognized held secret contradicts `""` even when
  the fingerprint baseline is unknown or was incorrectly seeded previously.
- **Legacy field rename:** internal `redactedDigests` state migrates to
  `appliedDigests`, including an explicitly empty legacy map. Normal upgrades
  should not require users to edit checkpoint JSON. Back up state through your
  established procedure before recovery work; do not manually delete credentials
  or baselines as an ordinary troubleshooting step.

## Limits to plan for

- Detection requires backend fingerprint coverage and an attributable mapping.
  Schema secret marking alone does not establish backend redaction or fingerprint
  support for every integration/value representation. Older servers can report
  withheld keys without fingerprints: omission protection can work even when
  rotation detection cannot.
- Nested/unmapped fingerprint paths may produce warnings rather than an
  actionable field diff. Do not assume every warning has automatic remediation.
- Fingerprints compare observations; they are not secret versions or an audit
  history. A→B→A entirely between reads is not detectable from the final value.
- The API has no completeness token. A previously unknown secret silently omitted
  from an otherwise plausible response cannot be distinguished from absence.
- Post-write reads are separate from mutations. A concurrent external write
  before capture can affect the observed baseline; there is no atomic version
  precondition in this workflow.
- Changing a database password directly, or rotating a resolved external secret
  while the stored reference stays unchanged, is not necessarily stored-config
  drift. Verify target access and use audit/rotation controls separately.
- Server fingerprint-key changes can look like widespread drift. Investigate the
  backend change before approving unexpected credential rewrites; fingerprints
  are scoped comparison metadata, not plaintext recovery or a security guarantee.

## Script bodies are different

`adaptive.Script.command` is a required secret body. Import cannot retrieve it;
supply the command before applying. Its `commandDigest` read behavior is not the
Resource `appliedDigests`/held-value algorithm above: do not assume the same
stable-baseline, omission, or automatic-reconciliation guarantees. This guide's
resource update guard must not be generalized to all resource families.

## References

- [Provider installation and authentication](./installation-configuration.md)
- [Pulumi secrets and encryption](https://www.pulumi.com/docs/iac/concepts/secrets/)
- [Pulumi refresh](https://www.pulumi.com/docs/iac/cli/commands/pulumi_refresh/)
- [Pulumi state and backends](https://www.pulumi.com/docs/iac/concepts/state-and-backends/)
