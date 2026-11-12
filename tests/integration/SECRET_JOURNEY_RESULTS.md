# Secret journey verification results

Date: 2026-11-12 (IST).
Checkout: `fix/preserve-explicit-empty-resource-fields`, HEAD `0ac1d54f8aee29de6975dd13a89a6b2106ce384d`, plus the local uncommitted provider fix and regression tests.
Live target: explicitly authorized `http://localhost:8080`, using CLI authentication resolved internally by the harness.

## Verdict

The synthetic V1 empty-map migration failure is fixed locally. A dedicated empty-object legacy migration now removes the obsolete property while preserving `appliedDigests: {}`. Absent/null and nonempty legacy maps retain their existing behavior. The full provider race suite passes, and the actual Pulumi engine successfully persists the injected checkpoint through repeated refresh/preview and subsequent secret drift/reconciliation.

The original shape's normal historical reachability remains unproven: the backend omitted empty maps and v0.2.0 did not explicitly allocate them. This is a compatibility-hardening fix, not a demonstrated historical customer incident.

The complete 14-case live import/legacy suite now passes against localhost for PostgreSQL and Snowflake, including both new injected empty-V1 checkpoint cases. There are no failures or skips, and all 14 exact-ID deletions and stack removals were verified. The earlier normal transition and existing-secret checks below remain historical pre-workaround evidence; the import/legacy suite was freshly rerun with the reviewed workaround.

Counts below are terminal Go test leaves, not parent/group pass events. Each leaf can contain multiple assertions and operations.

| Evidence tier | Result |
| --- | --- |
| Earlier live transition matrix, PostgreSQL and Snowflake (pre-workaround) | 48 PASS, 0 FAIL, 0 SKIP |
| Latest live import/legacy checkpoint matrix, both integrations (with workaround) | 14 PASS, 0 FAIL, 0 SKIP |
| Earlier existing managed/unmanaged/root-cert/quiet secret regressions (pre-workaround) | 4 PASS, 0 FAIL, 0 SKIP |
| Full provider suite under race detector, including absent/null/empty/nonempty migrations | 358 PASS, 0 FAIL, 0 SKIP |
| Actual Pulumi engine with controlled fake API, including guard failures | PASS |
| Provider and integration-module vet; whitespace check | PASS |

The two earlier live A→B→A sampling checks intentionally establish that limitation; passing those assertions does NOT mean intermediate rotations are detected.

## Live transition checklist

Every row below was run independently for PostgreSQL and Snowflake in `TestSecretJourneyLive/<integration>/...`.

| Sequence IDs | Sequence | Result |
| --- | --- | --- |
| 1A | Create with password omitted; refresh/preview clean and initialized empty baseline | PASS |
| 1B | Create with explicit empty; repeated matching refresh remains clean | PASS |
| 1C | Create with managed credential; capture nonempty fingerprint | PASS |
| 2A | Omitted → managed A | PASS |
| 2B | Explicit empty → managed A | PASS |
| 2C | Managed A → managed B; fingerprint changes and converges | PASS |
| 2D | Managed value → explicit clear; backend reports present empty value | PASS |
| 2E | Managed value → omitted plus metadata change; reject and verify remote config/tags/fingerprint unchanged | PASS |
| 2F | Explicit clear acknowledged → omit; clean convergence | PASS |
| 3A, 4A | External rotation; three refresh/preview rounds remain dirty and preserve applied baseline | PASS |
| 4B | Reapply original A; original server fingerprint restored | PASS |
| 3B | External clear of managed A; refresh notices; apply restores A | PASS |
| 4D | External rotation → explicitly supply rotated value → apply; capture new baseline | PASS |
| 3E, 4E | External rotation → omit desired password; reject without remote changes | PASS |
| 3C, 4C | Explicit empty → external addition → repeated drift → explicit clear → clean | PASS |
| 3D | Omitted with known-empty baseline → external addition → drift; omission blocked; explicit supplied value succeeds | PASS |
| 3F | Prune final cleared fingerprint → later external addition; detect again | PASS |
| 7C | Metadata-only update with all held secrets supplied | PASS |
| 7A | Clear password while retaining certificate; certificate fingerprint unchanged | PASS |
| 7B | Omit still-held certificate while clearing password; reject with correct public argument name | PASS |
| 7A | Clear both secrets, then omit both | PASS |
| 6C | Delete integration out-of-band → refresh drops state → actual up recreates different ID | PASS |
| 8 | Without refresh, preview does not observe an external rotation | EXPECTED LIMITATION VERIFIED |
| 8 | A→B→A between observations; final fingerprint equals original and preview stays clean | EXPECTED LIMITATION VERIFIED |

Normal CLI checkpoint exports (without `--show-secrets`) were checked for absence of JSON-encoded synthetic password and multiline-certificate canaries. Automation API `Stack.Export` deliberately requests decrypted exports and must not be used for that assertion. No checkpoint contents or real credentials were printed.

## Live import and legacy checkpoint checklist

`TestSecretJourneyImport` ran these leaves for both integrations:

- `import-zero`: actual engine import, no secret recovered, quiet imported baseline — PASS.
- `import-held`: no plaintext recovered; omitted-held metadata update rejected and independently read back; resupply then clear succeeds — PASS.
- `legacy-managed-no-baseline`: modify only the disposable checkpoint; first complete observation seeds quietly; persisted refresh/preview cycles — PASS.
- `legacy-empty-no-baseline`: explicit empty contradicts a held credential; repeated drift and correction — PASS.
- `legacy-empty-buggy-seeded`: prior buggy seed does not suppress explicit-empty contradiction; correction converges — PASS.
- `legacy-v1-migration`: recorded NONEMPTY `redactedDigests` migrates and preserves baseline through actual engine refresh — PASS.
- `legacy-empty-v1-migration`: inject the historical empty-map shape; repeated actual-engine refresh/preview removes the old field, preserves known emptiness and actionable drift, then explicit reconciliation converges — PASS.

Both nonempty and explicitly injected empty V1 checkpoint shapes now pass live, independently of whether a normal old release produced the empty shape.

## Offline failure and compatibility checklist

Repository-owned `provider/secret_baseline_test.go` and `provider/secret_journey_test.go` exercise the real inferred provider against a loopback fake API.

- Failed/missing/old-server/partial post-create fingerprint captures leave the baseline unknown; later complete observation seeds, subsequent rotations remain detectable — PASS, with historical-observation limitation.
- Successful pre-update guard observation followed by failed or partial post-write capture retains only prior applied evidence, not the guard observation — PASS.
- Current nil versus initialized-empty baseline; legacy absent baseline; import absent baseline; V1 absent baseline; V1 recorded nonempty baseline — PASS.
- V1 initialized-empty `redactedDigests: {}` — PASS after the dedicated empty-object migration; the prior failure is documented below.
- Missing fingerprint support with and without recorded evidence, then capability recovery — PASS for documented behavior; unavailable history is not invented.
- Already-seeded explicit clear → persistent drift → explicit corrective write → empty state → later addition — PASS.
- Unmapped/nested or mapped-nonsecret drift keys produce warnings rather than unsafe public fingerprint values — PASS for warning behavior, not automatic remediation.
- 403/500/404 and detectably incomplete pre-update responses reject before POST, including prior markers with an empty saved map — PASS in provider and actual-engine fake-API tests.
- Redacted keys without fingerprint support still protect omitted credentials and permit explicit clear — PASS.
- Controlled missing-metadata, equality/sampling, and post-write acknowledgment-race boundaries remain limitations, not successful detection guarantees.

## Resolved regression (historical failure)

Test: `TestSecretJourneyBaselineOrigin/v1_known_empty`.

Before the workaround:

1. Start with a legacy checkpoint containing `redactedDigests: {}`.
2. Refresh migrates to `appliedDigests: {}`, but the obsolete empty property is restored in the encoded checkpoint.
3. A subsequent refresh fails decoding: `redactedDigests: Unrecognized field 'redactedDigests' on 'adaptive.ResourceState'`.

Regression command from repository root (now passes):

```sh
go -C provider test -run '^TestSecretJourneyBaselineOrigin/v1_known_empty$' -count=1 -v .
```

Source anchors: `provider/secret_journey_test.go:214-264`; `provider/resource_resource.go:resourceStateV1`, `migrateResourceStateV1`, `StateMigrations`.
Pinned SDK interaction: `pulumi-go-provider@v1.3.2/infer/resource.go:1580-1623` retains the old decoder/encoder during typed migration; `infer/internal/ende/ende.go:360-377` restores empty-map paths. The provider now registers `resourceStateV1Empty` / `migrateResourceStateV1Empty` before the normal map migration, decoding only a present empty object as `*struct{}` and producing a non-nil empty applied map. Absent/null values and nonempty maps continue through the existing path. No public schema or dependency change is required.

The additional workaround was applied after the expanded verification round. Before/after actual-engine tests demonstrate removal of the obsolete field through persisted checkpoint round-trips; `TestSecretJourneyNullV1Baseline` covers null fallback. The new live `legacy-empty-v1-migration` cases now pass for both integrations after refreshing CLI authentication.

## Cleanup

- Transition matrix: explicit destroys and exact-ID 404 assertions; independent follow-up read-back verified all 14 IDs, including replacements.
- Existing secret regressions: independent follow-up read-back verified all 4 IDs.
- Latest import matrix: all 14 IDs independently read back as 404; all 14 disposable stacks removed according to recorded evidence.
- Initial instrumentation-aborted transition matrix: independent follow-up verified all 14 IDs as 404.
- Only uniquely owned disposable integrations were created/mutated/deleted; no access endpoints or real database login tests were run.

## Evidence files

Raw JSON event logs and machine evidence reside in the Hermes scratch directory and are prunable; this report and test source are repository-owned.

- `pr24-journey-live-v2.jsonl`
- `pr24-journey-import-testlog-final.jsonl`
- `pr24-journey-import-results.json`
- `pr24-existing-secret-tests-final.jsonl`
- `pr24-journey-provider-final.jsonl`
- `pr24-journey-engine-final.jsonl`
- `pr24-journey-offline-results.json` (historical pre-fix evidence)
- `pr24-empty-migration-provider-green.jsonl` (358 provider leaf passes)
- `pr24-empty-migration-engine-red.jsonl` / `pr24-empty-migration-engine-green.jsonl`
- `pr24-empty-migration-live-red.jsonl` (historical HTTP401 preflight blocker; not a migration failure)
- `pr24-empty-migration-live-final.jsonl` / `pr24-empty-migration-live-results.json` (latest 14 live passes with deletion and stack-removal evidence)

No commits, pushes, backend code changes, schema generation, or SDK generation were performed for this verification round.
