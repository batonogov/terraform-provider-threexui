# Repository guidance

## Entry points and sources of truth

- This directory name differs from the Go module: use
  `github.com/batonogov/terraform-provider-threexui` and registry address
  `registry.terraform.io/batonogov/threexui`. `main.go` serves the provider;
  `provider/provider.go` registers resources/data sources and configures the shared client.
- Use the Go version in `go.mod`, golangci-lint **v2**, and Task v3.
  `Taskfile.yml` and `.github/workflows/ci.yml` define the actual verification commands.
- Before changing API behavior, inspect the matching `3x-ui-<version>/` snapshot:
  `internal/web/{controller,service,entity}/`, `internal/database/model/`,
  `frontend/src/schemas/`, and `internal/xray/`. Look under `internal/` first,
  then the legacy root `web/`, `database/`, and `xray/` paths.
  These are upstream reference sources, not provider code.
- `provider/drift_test.go` checks upstream contracts against snapshots, falling back to
  committed `provider/testdata/upstream_contract.json` when snapshots are absent.
  Feature limitations belong in `docs/guides/version-compatibility.md`, not a release log here.

## Verification

| Command | Scope / prerequisite |
| --- | --- |
| `task pre-commit` | fmt → vet → lint → build; **does not run tests** |
| `task test:unit` | Unit tests only; no Docker or Terraform |
| `task test:unit:coverage` | CI's unit coverage, written to `coverage.out` |
| `go test ./provider -run '^TestName$' -count=1 -v` | One unit test |
| `task test:acc` | Full SQLite acceptance suite; Docker Compose and Terraform on PATH |
| `THREEXUI_VERSION=v3.11.0 task test:acc:compat` | Full suite against a selected panel version |
| `task test:acc:postgres` | Separate PostgreSQL acceptance suite |
| `pre-commit run --all-files` | Go checks plus Markdown/YAML/JSON and other hooks |

- `task test` runs both unit and acceptance suites. Acceptance tasks start with
  `docker compose down -v`: use only the disposable test stack, never a live panel.
  They set admin/admin credentials (fresh v3.11.0 panels generate random credentials),
  wait for Xray, warm the version cache, and tear down containers.
- For one acceptance test against an **already prepared disposable panel**:

  ```bash
  TF_ACC=1 TF_ACC_TERRAFORM_PATH="$(command -v terraform)" \
    THREEXUI_ENDPOINT=http://localhost:2053 \
    THREEXUI_USERNAME=admin THREEXUI_PASSWORD=admin \
    THREEXUI_VERSION=v3.11.0 \
    go test ./provider -run '^TestAccInboundReality$' -count=1 -timeout 600s -v
  ```

  Match `THREEXUI_VERSION` to the running panel so version guards are accurate.
  Docker's provider `base_path` is `/`, **not** `/panel/`.
- Acceptance tests use `ProtoV6ProviderFactories`. Gate version-specific features with
  `requireMinVersion` / `requireBelowVersion`; `resource_inbound_matrix_test.go`
  covers protocol create/update/import round-trips.
- Xray config tests restart the core; version may temporarily be `"Unknown"`.
  Use the test helper `waitForXrayVersion(t, ctx, client)`, not a direct version read.
  Tests fetching release lists must use `testAccSkipOnXrayVersionsRateLimit` or
  `skipOnUpstreamRateLimit`: the panel can hit GitHub's anonymous rate limit.
  Keep the bounded skip/deadline behavior in `TestAccXrayVersionDrift`; widening its
  polling can exhaust the entire test binary's timeout.
- Preserve read-after-write/destroy visibility polling for SQLite. PostgreSQL has
  different timing; green SQLite tests do not substitute for its acceptance job.
- `testAccStoreInboundStream` must await both masks and `tlsSettings` after API edits;
  mask-only comparisons accept stale fingerprint reads. Keep
  `TestStoreInboundStreamWaitsForTLS` guarding that delay.
- Failure logs land in `artifacts/3xui-container.log`. Task defers run in reverse
  order: log capture must stay after the teardown defer so it executes first.
- New provider behavior needs direct unit tests, not just `TestAcc*` tests:
  CI uploads **unit-only** coverage. Require green CI **and** clean Codecov patch
  coverage before merge.

## Data ownership and API traps

- Inbound conversion has three layers: typed Terraform models ↔ `map[string]any`
  (`expand*`/`flatten*`) ↔ API JSON strings (`build*`). Change both directions.
  `Inbound.UnmarshalJSON` in `types.go` accepts both object and string API forms.
- Inbound TLS fingerprints belong in `tlsSettings.settings.fingerprint` (panel editor
  and `internal/sub/json_service.go:844-857` in 3x-ui v3.11.0); outbounds need
  `tlsSettings.fingerprint`. Keep the outbound lift and nested-first legacy read.
  Xray v26.10.21's Hysteria2 transport ignores this fingerprint selector, but its
  separate ChromeParrot mode does use QUIC/uTLS; do not claim Hysteria2 never uses uTLS
  (`apernet/quic-go@184d081eef3e/internal/handshake/tls_conn_utls.go`, #503).
- Three separate read-modify-write locks protect shared documents:
  `settingsMu` (panel settings), `xrayTemplateMu` (Xray template), and
  `inboundClientMu` (clients). Inbound writes that mutate/reconcile clients share
  the last lock with `inbound_client`; release it before
  `releasePeerEmailsRemovedByUpdate`, which locks it itself.
- WireGuard/AmneziaWG/TUIC peers belong to `threexui_inbound`, **not**
  `threexui_inbound_client` (`protocolOwnsClients`). Other protocols' clients belong
  to the separate client resource. Do not re-inject removed inbound-owned peers
  from server state; that prevents removing the last peer.
- Since panel v3.9.0, inbound Update silently retains stored `enable` and clients.
  Preserve `ensureInboundEnable` and per-peer reconciliation in `resource_inbound.go`.
  The two-read reconciliation grace avoids duplicate adds on stale older-panel reads.
  Peer removal/deletion also needs email cleanup: deleting an inbound alone leaves
  globally unique client rows behind.
- AmneziaWG/TUIC settings nest `server` and `clients`; do not route server fields
  through the flat protocol key table. AmneziaWG creation deliberately lets the
  panel generate its server/key/obfuscation defaults before applying overrides;
  a partial server POST would silently lose randomized protection.
- Panel settings resources share `resource_settings_tabs.go`; they update one
  shared settings document and Delete only removes Terraform state. Preserve
  cached/state secret replay when panel reads redact secrets.
- For a new settings key, check **where upstream reads it**. Startup-frozen values
  (including subscription content/presentation and notifier cron registration)
  need `panelSettingsNeedRestart`, not just binding changes. Preserve
  `restartKeyRules`' derived comparisons and absent-key guard; unsupported keys
  must not cause perpetual restarts. `webBasePath` changes also update the client path.
- `xray_basics` merges the template root; other Xray resources replace owned JSON
  paths via `resource_xray_settings.go`. Unmodeled fields in a replaced section can
  be destroyed. `xray_observatory` owns both observatory keys. Shared
  `xrayApplyTyped` skips empty values; Delete does not reset panel configuration.
- Keep endpoint selection, legacy fallback, CSRF, re-login, and 2FA attachment
  centralized in `client.go`. HTTP 200 with `success:false` can mean missing data:
  nodes and host groups have different not-found helpers. Do not treat every 404
  as deletion; it can mean an unsupported API.

## Terraform planning and secrets

- Do not decode the entire inbound config/plan into Go slice-backed models during
  validation or `ModifyPlan`: dynamic blocks can be unknown. Read needed paths
  with `GetAttribute` into framework `types.*` values and defer unknown branches.
- `UseStateForUnknown` is appropriate only for values stable through the write.
  Node heartbeats, GUIDs, traffic counters, and server-materialized peer fields can
  change during apply; copying prior values into the plan causes inconsistent results.
- Omission of `Optional+Computed` is generally **preserve**, not clear. Keep documented
  clear sentinels rather than globally changing omission semantics. Blocks such as
  `xmux`, `masquerade`, and `final_mask` deliberately differ; preserve their removal rules.
- Nested lists match by index, so copied state can bleed between reordered objects.
  Preserve the identity/config reconciliation in Xray `ModifyPlan` methods
  (basics, DNS, routing, outbounds). Outbounds restore omitted `stream_settings`
  by **tag** and omitted fields of declared `sockopt`; do not restore intentionally
  removed blocks. Keep unknown lists/elements representable as framework values.
- Preserve `alignBlocksWithPlan` and its import exception: flattening must not
  introduce optional blocks absent from the plan. Use `singletonListNestedBlock`
  only for singleton JSON objects, not genuinely repeatable collections.
- Shared TLS certificates are a nested **attribute**, not a block. Keep
  `tlsCertificatesNullOnCreate` so omitted Go-slice-backed certificates do not
  remain undecodable unknowns. TLS acceptance fixtures use real inline self-signed
  certificates; nonexistent server file paths can break the panel's config probe.
- Write-only secrets are read from `req.Config`, never plan/state. Updates send them
  only when the `_wo_version` trigger changes; settings-style plans must mark the
  paired plain attribute unknown. Node/client resources handle two secrets together:
  calling the single-secret `modifyPlanWOVersion` helper twice overwrites the first change.
  `panel_user` instead nulls the state password and falls back to provider credentials.
- Custom string types must be wired through schema, model, flatten, and
  `ValueFromTerraform`. Reuse `urlSchemeNormalizedType` for panel-rewritten subscription
  URLs; a plan modifier cannot rewrite explicitly configured strings. Keep
  `jsonObjectType`/canonical JSON handling for `final_mask`.
- Mark secret-bearing JSON data sources `Sensitive: true`, not just individual
  resource secrets. Use `int64Value`, not narrowing `intValue`, for 64-bit wire fields:
  release builds include 32-bit targets.

## Changes that must travel together

- Schema changes need matching models, expand/flatten logic, unit/round-trip tests,
  resource docs, and relevant examples. Update `SECURITY.md` for new secret surfaces.
  Maintain concise guidance here when changing ownership, planning, retries, or API quirks.
- README changes must reach all seven `README*.md` translations; Persian/Arabic
  bodies keep their RTL wrapper. Check with `scripts/check-readme-parity.sh`.
- Docs are partly hand-written: use `tfplugindocs validate --provider-name threexui`
  and `go test ./provider -run 'TestDocumentedSchemaSections|TestInboundProtocolDescriptionListsVersionedProtocols' -count=1`.
  Example checks are `terraform fmt -check -recursive examples/` and
  `scripts/validate-examples.sh`.
- Panel versions come from `compat-versions.json`: edit it, run
  `scripts/sync-versions.sh fix`, then `scripts/sync-versions.sh check`.
  New snapshots also require `context7.json` exclusions and refreshed drift mappings
  plus `provider/testdata/upstream_contract.json` (including applicable field lists).
- Terraform support comes from `terraform-versions.json`: latest patches of the
  two newest stable minor lines, newest as default. CI tests both against the default
  panel; panel compatibility and PostgreSQL jobs use only the default Terraform.
- Conventional commits feed Release Please/GoReleaser. Keep workflow actions pinned
  to commit SHAs and update pre-commit hooks with `pre-commit autoupdate --freeze`.
- Put scratch output in git-ignored `artifacts/`, not the repository root.
