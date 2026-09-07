# AGENTS.md

This file provides guidance to AI coding agents (Claude Code, Cursor, Codex, Aider, etc.) when working with code in this repository. It follows the [AGENTS.md](https://agents.md) convention.

## Project

Terraform provider for the [3x-ui](https://github.com/MHSanaei/3x-ui) panel.
Go (version pinned in [`go.mod`](go.mod)) with `terraform-plugin-framework`. Module: `github.com/batonogov/terraform-provider-threexui`.
Registry: `batonogov/threexui`. All provider code lives in `provider/`.
Releases are automated: conventional commits on `main` → Release Please PR → GoReleaser (GPG-signed) → Terraform Registry.

Provider config attributes: `endpoint`, `username`, `password`, `base_path`,
`bootstrap_username`/`bootstrap_password` (first-run setup), `two_factor_code` (TOTP),
`insecure_skip_verify`, `request_timeout`, `max_retries`.
Env vars: `THREEXUI_ENDPOINT`, `THREEXUI_USERNAME`, `THREEXUI_PASSWORD`,
`THREEXUI_BASE_PATH`, `THREEXUI_INSECURE_SKIP_VERIFY`, `THREEXUI_REQUEST_TIMEOUT`,
`THREEXUI_MAX_RETRIES`. Bootstrap and 2FA attributes have **no** env-var fallback.

---

## Commands

| Command | Description |
| --- | --- |
| `task build` | Build provider binary |
| `task fmt` | `gofmt -w provider/*.go` |
| `task vet` | `go vet ./...` |
| `task lint` | `golangci-lint run` |
| `task test` | Unit + acceptance tests |
| `task test:unit` | Unit tests (no Docker/Terraform needed) |
| `task test:unit:coverage` | Unit tests with coverage (`coverage.out`) |
| `task test:acc` | Acceptance tests (Docker lifecycle included) |
| `task test:acc:compat` | Acceptance tests vs a pinned `THREEXUI_VERSION` (compat matrix) |
| `task test:acc:postgres` | Acceptance tests with a PostgreSQL backend (docker compose `--profile postgres`) |
| `task pre-commit` | fmt + vet + lint + build |

**Watch CI status for a PR:**

```bash
gh pr checks <PR>          # one-shot
gh pr checks <PR> --watch  # live
```

Every PR must end with a **green CI + a clean Codecov report**. The
`ci.yml` test job uploads coverage to Codecov (`codecov/codecov-action`),
which then posts a **Patch coverage** comment. Review it as part of the PR:
see [Testing → Codecov patch coverage](#testing) below for the rule.

**Version-specific acc tests:**

```bash
THREEXUI_VERSION=v3.2.0 task test:acc:compat
```

**Single acceptance test:**

```bash
TF_ACC=1 THREEXUI_ENDPOINT=http://localhost:2053 \
  THREEXUI_USERNAME=admin THREEXUI_PASSWORD=admin \
  THREEXUI_VERSION=v3.7.0 \
  go test ./provider -run TestAccInboundReality -count=1 -timeout 600s -v
```

---

## Architecture

### Three-layer conversion (inbound)

`threexui_inbound` and `threexui_inbound_client` are the core resources.
`settings`, `stream_settings`, `sniffing` are JSON strings in the 3x-ui API but typed
blocks in the Terraform schema. Conversion flows through three layers:

```text
typed model  (structs with types.String / types.Int64 / types.Bool)
  ↔  untyped map  (map[string]any — expand* / flatten* functions)
  ↔  JSON string  (build* / flatten* functions)
```

Schema helpers: `inbound_*_schema.go`, `settings.go`, `stream_settings.go`, `sniffing.go`.
`inbound_client` does read-modify-write on the inbound's `settings.clients` array
under its own `inboundClientMu` mutex (a third RMW lock alongside `settingsMu` and
`xrayTemplateMu`). Both core resources accept `restart_xray = true` to restart
xray-core after create/update/delete (`POST /panel/api/server/restartXrayService`).

### Panel settings (singletons)

Five resources — `panel_general`, `panel_security`, `panel_telegram`,
`panel_email`, `panel_subscription` — share `resource_settings_tabs.go`.

- Each is a singleton with ID `"settings"`.
- Shared CRUD via `settingsApplyTyped` / `settingsReadTyped`.
- Delete only clears TF state — does not reset panel settings.
- `settingsMu` serializes read-modify-write (separate from `xrayTemplateMu`).
- `settingsSecrets` on Client remembers the last configured `tgBotToken` /
  `twoFactorToken` / `ldapPassword` / `smtpPassword` and replays them on partial updates when a GET
  returns an empty/redacted sentinel. Note: 3x-ui v3.0.2–v3.3.1 actually returns
  these secrets **raw** on `/setting/all`, so the replay path is defensive and
  mainly fires when the panel genuinely has no secret stored.
- Changing a restart-triggering key triggers a provider-initiated restart
  (`POST /setting/restartPanel`, SIGHUP; 3x-ui does **not** auto-restart) in two
  resources, each gated by the shared `panelSettingsNeedRestart` check followed by
  `SendRestart` + `WaitForReady`: `panel_general` and `panel_subscription`. The rule
  behind the list is **"does the panel read this once at startup?"**, not "is it a
  binding setting":
  - `panel_general`: the web-server binding keys (`webListen`, `webDomain`, `webPort`,
    `webBasePath`, `webCertFile`, `webKeyFile`, `sessionMaxAge`) plus the cron wiring
    read once in `web.Server.Start()` — `timeLocation` (every job's timezone,
    `internal/web/web.go:503`) and `ldapEnable`/`ldapSyncCron` (whether the LDAP sync
    job is registered, and on what schedule, `web.go:376-383`).
  - `panel_subscription`: the sub-server binding keys (`subEnable`, `subListen`,
    `subDomain`, `subPort`, `subPath`, `subCertFile`, `subKeyFile`) **and every setting
    `(*sub.Server).initRouter()` reads** — route registration (`subJsonEnable`,
    `subJsonPath`, `subClashEnable`, `subClashPath`), JSON/Clash body content
    (`subJsonMux`, `subJsonRules`, `subJsonFinalMask`, `subJsonObservatory`,
    `subClashRules`, `subClashEnableRouting`), detection/shape switches
    (`subJsonAutoDetect`, `subJsonAlwaysArray`, `subJsonUserAgentRegex`,
    `subClashAutoDetect`, `subClashUserAgentRegex`, `subEncrypt`, `subUpdates`,
    `remarkTemplate` — which is read per request by the panel's own link provider too, but frozen at startup for the sub server) and page presentation (`subTitle`, `subSupportUrl`,
    `subProfileUrl`, `subAnnounce`, `subHideSettings`, `subEnableRouting`,
    `subRoutingRules`, `subIncyEnableRouting`, `subIncyRoutingRules`). `initRouter`
    freezes all of them into the `SUBController` it builds
    (`3x-ui-3.7.0/internal/sub/sub.go:50-301`) and runs only from `Start()` (#443).
  - `panel_telegram` / `panel_email`: the notifier cron registration, read once in
    `web.Server.Start()` — `tgRunTime` (stats-notify schedule, `internal/web/web.go:389`),
    `tgBotEnable` (whether the bot's cron jobs exist at all, `web.go:387-403`), and
    `tgEnabledEvents`/`tgCpu`/`tgMemory` + `smtpEnable`/`smtpEnabledEvents`/`smtpCpu`/
    `smtpMemory`, which feed `cpuAlarmWanted()`/`memoryAlarmWanted()` (`web.go:408-412`,
    `:439-448`, `:469-478`) and decide whether the alarm jobs are registered for either
    notifier (#449). These two resources share `settingsApplyTyped` with `panel_security`,
    so the restart lives there; `panel_security` owns no startup-read key and stays quiet.
    The bot **process** is hot-reloaded (`controller/setting.go:165-172` → `web.go:650`),
    so `tgBotToken`/`tgBotChatId`/`tgBotAPIServer` do NOT restart — but toggling
    `tgBotEnable` without one leaves the bot running with no periodic report.
    Two of these keys are gated on *what changed*, not on the value changing at all
    (`restartKeyRules`): the alarm thresholds only decide whether the sampler job is
    registered (`threshold <= 0`), and the event lists only matter for `cpu.high`/
    `memory.high` membership — so `tgCpu` 80 → 90, or adding `backup` to
    `tgEnabledEvents`, takes effect live and must not bounce the panel.
  A key **absent** from the panel's `/setting/all` response never restarts: AllSetting
  is serialised whole, so an absent key means that version predates the setting and
  `/setting/update` will drop it too. Restarting for a value the panel cannot store
  would bounce it on every apply — 7 of the 13 versions in `compat-versions.json`
  predate the notifier keys.
  The 3x-ui sub server only (re)binds at startup, so without this restart a changed
  `sub_path`/`sub_port`/… does not take effect and the subscription URL 404s (#291).
  Only the three link-generation URIs (`subURI`, `subJsonURI`, `subClashURI`) are read
  per request (`internal/sub/service.go:2704-2706`) and do **not** restart — `subTitle`
  looks like one of them but is not. For `webBasePath` the provider additionally calls
  `SetBasePath` + `WaitForReady` so subsequent requests target the new path.

### Write-only secret attributes (Terraform 1.11+ / OpenTofu 1.11+)

Managed secrets have write-only (`_wo`) alternatives following the AWS/Azure pattern.
Each secret gets three attributes: old `Sensitive` attr + `WriteOnly` attr + `_wo_version` trigger.

| Resource | Old attr | Write-only | Version trigger |
| --- | --- | --- | --- |
| `panel_user` | `password` | `password_wo` | `password_wo_version` |
| `panel_security` | `two_factor_token` | `two_factor_token_wo` | `two_factor_token_wo_version` |
| `panel_telegram` | `tg_bot_token` | `tg_bot_token_wo` | `tg_bot_token_wo_version` |
| `panel_email` | `smtp_password` | `smtp_password_wo` | `smtp_password_wo_version` |
| `panel_general` | `ldap_password` | `ldap_password_wo` | `ldap_password_wo_version` |
| `threexui_node` | `api_token`, `pinned_cert_sha256` | `api_token_wo`, `pinned_cert_sha256_wo` | `api_token_wo_version`, `pinned_cert_sha256_wo_version` |
| `threexui_inbound_client` | `password`, `secret` | `password_wo`, `secret_wo` | `password_wo_version`, `secret_wo_version` |

- `_wo` values require Terraform/OpenTofu 1.11+. Older runtimes reject a configured
  write-only value; users on those versions must keep using the plain `Sensitive` attr.
- `PreferWriteOnlyAttribute` validator warns on TF >= 1.11 when using old attr.
- `int64validator.AlsoRequires` enforces that `*_wo_version` can only be set with `*_wo`.
- Write-only values read from `req.Config` (not plan/state — framework nulls them).
- Two resolution strategies:
  - **panel_user**: `password` is nulled in state when `password_wo` is used. Update
    falls back to provider credentials when state password is empty. No ModifyPlan
    needed because state has no prior password to conflict with.
  - **settings (security/telegram/email/general)**: `resolveXxxWO` copies `_wo` value into
    the old model field before `expand*()`. ModifyPlan marks the plain attr as Unknown
    on `woVersionTriggered` so Terraform accepts a new sensitive value from Apply.
    Needed because flatten reads the masked value back, which would otherwise be
    rejected as "inconsistent values for sensitive".
  - **inbound_client**: `resolveInboundClientSecretsWO` copies `password_wo` and
    `secret_wo` from config on Create; `resolveInboundClientSecretsWOUpdate` only
    copies them when their version trigger changes. Its inlined ModifyPlan marks
    each corresponding plain attr Unknown independently, following the two-secret
    node pattern.
- Version trigger: `resolveXxxWOUpdate` only sends the `_wo` value when
  `woVersionTriggered(plan, state)` (version changes, or first use when state has none).

### Xray settings (two modes)

Seven xray resources share `resource_xray_settings.go`: `xray_basics`, `xray_dns`,
`xray_routing`, `xray_balancers`, `xray_reverse`, `xray_outbounds`,
`xray_observatory`. Each section has its own `*_schema.go` file.

List-backed nested blocks that model one JSON object use
`singletonListNestedBlock` (`listvalidator.SizeAtMost(1)`). Keep collection blocks
such as outbounds, balancers, policy levels, costs, Freedom noises/final rules,
WireGuard peers, and observatory entries repeatable. This prevents accepted extra
blocks from being silently dropped by expand functions that read element zero.

| Mode | Resource | Behavior |
| --- | --- | --- |
| Merge root | `xray_basics` | Reads full template, merges changed fields, writes back |
| Set path | all others | Reads/writes a section by JSON path |

`xrayTemplateMu` serializes read-modify-write to prevent race conditions.

### Xray version (`resource_xray_version.go`)

Separate resource — singleton with ID `"xray_version"`. Calls `InstallXray` + polls
`waitForXrayVersion` (180×1s) until the version matches; if the panel still reports a
stale version after 30 attempts it re-issues `InstallXray` once (3x-ui v3.2.6–v3.2.7
sometimes silently drop the first install, #262). If the panel still reports
`"Unknown"` after `nudgeAfter` (60s) of polling, the provider issues a single
`RestartXrayService` (bounded by `nudgeTimeout` = 30s context) to prod the core
into reporting its version — this is a cheap local call (no GitHub download) and
is non-fatal: the nudge error is silently swallowed and polling continues. Read
treats `"Unknown"` as a soft-fail (Warning + preserved state). Delete is a no-op
with a warning (removing from state does NOT revert the installed version).

### Panel user (`resource_panel_user.go`)

`threexui_panel_user` — no read API exists. Read is a no-op, state is preserved
from plan. Create uses provider credentials as old credentials; Update uses
prior state credentials (falls back to provider credentials when state
password is empty, e.g. after using `password_wo`). See Write-only section
above for the `password` / `password_wo` lifecycle.

### Cluster node (`resource_node.go`)

`threexui_node` — manages a remote 3x-ui panel registered as a cluster node
(multi-node surface, `/panel/api/nodes/*`; available since v3.0.2, no legacy
fallback). Routes are gin subpaths: list `GET /list`, read `GET /get/:id`, create
`POST /add`, update `POST /update/:id`, delete `POST /del/:id` (POST, not DELETE).
Create does a form-POST `/add`, then re-reads `/get/:id` for observed state. The
**central panel probes the node for reachability (`ensureReachable`) before
persisting it**, so the node's web API must be reachable from the central panel
at apply time — there is no provider-side flag to bypass this (decided in #315).
Read does `GET /get/:id` and treats a missing node (the panel signals this as
HTTP 200 + `success:false` with a gorm "record not found" message, NOT HTTP 404
— see `util.go jsonMsgObj`; handled via `isAPIRecordNotFound`) as remove-from-
state. Update form-POSTs `/update/:id` then re-reads; the **3x-ui server
restarts the Xray core itself** when `outbound_tag` changes
(`controller/node.go:180-181`), so the provider does NOT call
`RestartXrayService` (unlike `threexui_inbound`'s `restart_xray = true`). Delete
POSTs `/del/:id`; 3x-ui refuses to delete a node that still owns inbounds
(DB-002, #314 R3) — surfaced as a clear error so the operator detaches the
inbounds first. Import is by numeric id (`ImportStatePassthroughID`). `api_token`
and `pinned_cert_sha256` are `Sensitive` (panel returns them raw, no redaction —
issue #314 R1) with write-only `_wo` variants following the settings-style
strategy (`api_token_wo`+`_wo_version`, `pinned_cert_sha256_wo`+`_wo_version`;
`resolveNodeSecretsWO`/`resolveNodeSecretsWOUpdate` + an **inlined** ModifyPlan
(not the shared `modifyPlanWOVersion` generic — the node has two write-only secrets
and the generic re-reads/overwrites `resp.Plan` on each call, which would erase the
first `Unknown` mark on the second). Schema lives in `node_schema.go`;
the typed `Node` model is shared with the `threexui_nodes` data source (`types.go`).

### Host group (`resource_host_group.go`)

`threexui_host_group` — manages a 3x-ui v3.5.0+ host group (bulk host management,
`/panel/api/hosts/*`; available since v3.5.0, no legacy fallback — older panels use
`/get/:id` with numeric id). One group = one `groupId`; routes are gin subpaths:
list aside, create `POST /add`, read `GET /get/:groupId`, update `POST /update/:groupId`,
delete `POST /del/:groupId` — all with a **JSON body** (`entity.HostGroup`), not
form-POST like inbounds/nodes. Create does `POST /add` (server generates
`groupId` via `random.NumLower(16)` when omitted), then re-reads `/get/:groupId`
for canonical state. Read treats a missing group (HTTP 200 + `success:false` with an
explicit `"host group not found"` message — **not** a gorm record-not-found like
nodes; detected via the separate `isHostGroupNotFound`, not `isAPIRecordNotFound`) as
remove-from-state. Update `POST /update/:groupId` (delete-then-recreate under transaction) then
re-read; delete `POST /del/:groupId`. Import is by `groupId`
(`ImportStatePassthroughID`). `inbound_ids` is Required (`min=1`), `remark` Required,
`port` 0–65535, `security`/`mihomo_ip_version` enum-validated. No sensitive fields →
no write-only variants. v3.5.0+ only.

### HTTP client (`client.go`)

Cookie auth, auto re-login on 401/404, CSRF token handling (enforced on all
`/panel/api/*` + login routes since ≥ v3.0.2; Bearer API-token auth bypasses it).
Supports bootstrap credentials for first-run panel setup (the panel auto-creates an
`admin`/`admin` user on an empty DB; bootstrap ordering differs by generation —
v2.9.x tries bootstrap first, v3.x tries primary first).
Provider attributes: `two_factor_code` (TOTP for 2FA login), `max_retries`
(default 1, max 10), `request_timeout`.
Three retry layers: 5xx on idempotent writes, read-after-write for SQLite
visibility lag, rate-limit retry for `GetXrayVersions`.
Two independent API-surface auto-detections (each probed once and cached):
(a) client API — probes `/panel/api/clients/list` to pick v3.1.0+
`/panel/api/clients/*` over `/panel/api/inbounds/*`; (b) settings/xray API — probes
`/panel/api/setting/all` to pick v3.3.0+ `/panel/api/setting/*` + `/panel/api/xray/*`
over `/panel/setting/*` + `/panel/xray/*`. Both fall back to legacy endpoints
mid-run via `markLegacy*` on 404.

### JSON format compatibility (`types.go`)

3x-ui v3.1.0 returns `settings`, `streamSettings`, `sniffing` as nested JSON
objects instead of escaped strings. Custom `UnmarshalJSON` on `Inbound`
normalises both formats to plain strings — rest of the code is unaffected.

### Data sources

Eight data sources: `inbounds`, `nodes`, `server_status`, `xray_versions`, `xray_config`,
`settings`, `online_clients`, `client_traffics`. All are read-only GET wrappers
that return the raw panel payload — none accept filter arguments. JSON attrs that
contain secrets (UUIDs, private keys) must be marked `Sensitive: true`.

`threexui_nodes` wraps `GET /panel/api/nodes/list` (3x-ui multi-node/cluster surface,
available since v3.0.2; no legacy fallback). The response is a **node tree**
including transitive sub-nodes (read-only projections with `Id == 0`,
`transitive == true`). The payload carries each node's `apiToken` and
`pinnedCertSha256` **raw** (no redaction layer), so the `nodes` attr is
`Sensitive: true`.

---

## Critical Gotchas

These are non-obvious constraints that have caused real bugs.

| Gotcha | Why it matters |
| --- | --- |
| `email` in `threexui_inbound_client` is **Required** | The `clients`/`client_traffics` tables key on it (`uniqueIndex; not null` / `gorm:"unique"`); the panel rejects an empty email with `"client email is required"` before the SQL layer |
| Docker `base_path` is `/` | Do **not** set `THREEXUI_BASE_PATH=/panel/` |
| Data-source JSON attrs need `Sensitive: true` | Payloads contain UUIDs, private keys, tokens |
| `Optional+Computed` attrs need `UseStateForUnknown` | Prevents false drift (`known after apply`) after import |
| `xray_routing` / `xray_outbounds` repeated lists need `ModifyPlan` reconciliation | Their nested attributes are `Optional+Computed` + `UseStateForUnknown`, and `ListNestedBlock` elements are matched **by index**. On reorder/removal the modifiers can therefore copy the prior element's unset fields into a different semantic object and persist the polluted payload. `XrayRoutingResource.ModifyPlan` makes the configured `rule` list authoritative; `XrayOutboundsResource.ModifyPlan` does the same for the complete `outbound` list, including mux and protocol-specific descendants. Outbounds planning stays in framework `types.List` values rather than decoding to `[]XrayOutboundEntry`, so a whole unknown collection, an unknown object element, and partial unknown leaves remain representable. Regression coverage: `TestAccXrayRoutingRulesReorderNoFieldBleed` and `TestAccXrayOutboundsReorderNoFieldBleed`. The same schema pattern exists on other repeated Xray blocks (`xray_dns`, `xray_basics`) and needs its own ownership/default analysis before applying this pattern. |
| Outbound `stream_settings` must be re-injected from state in `ModifyPlan` | `XrayOutboundsResource.ModifyPlan` resets the planned `outbound` list to the configured one after the authoritative-list reconciliation, so a `stream_settings` block that is absent from the configuration (all its fields are `Optional+Computed` + `UseStateForUnknown`, which can only restore siblings already present in the plan) would be stripped from every outbound on any unrelated update — worse than drift, since `xray_outbounds` is set-path and **wholesale replaces** the `outbounds` array (via `setJSONPath`), permanently deleting server-side `streamSettings`. `mergeOutboundStreamSettingsFromState` therefore copies prior-state `stream_settings` objects (matched by `tag`) back onto configured elements that omit the block, before the configured list is committed to the plan. Note the same class of bug existed pre-feature for a different reason: outbound `streamSettings` was entirely *unmodeled*, so **any** provider write dropped it silently — the model/block itself (`stream_settings` + `tls_settings`) is what makes this feature safe to use at all. |
| `alignBlocksWithPlan` | Prevents "was absent, but now present" for Optional blocks; skipped during Import |
| `reality_settings.settings` is `SingleNestedAttribute` | Not a block. Uses `objectplanmodifier.UseStateForUnknown()` |
| REALITY `min_client_ver` unset ≠ gate disabled (xray-core ≥ v26.7.11) | An empty `minClientVer` is **not** "no lower bound": `infra/conf/transport_security.go` substitutes `[]byte{26, 3, 27}` for it, so the server rejects every client that reports an older or absent Xray version (sing-box, older clients). `0.0.0` is the only value that removes the bound. The provider therefore rejects `""` on `reality_settings.min_client_ver`/`max_client_ver` (`realityClientVerValidators`) — omit the attribute rather than writing an empty string, since `expandRealitySettings` drops empty strings while `flattenRealitySettingsToModel` reads them back as null, which cannot round-trip. Xray parses the value as up to three dot-separated bytes, each 0-255 |
| REALITY `maxTimediff` is read under two spellings | The panel persists the field as `maxTimediff` (`frontend/src/schemas/protocols/security/reality.ts`) while xray-core declares `json:"maxTimeDiff"` (`infra/conf/transport_security.go`); the panel spelling binds only because `encoding/json` falls back to a case-insensitive match. Since 3x-ui stores `streamSettings` as opaque JSON text, an inbound authored by the API or external tooling keeps whichever spelling it was written with, so `flattenRealitySettings` accepts **both**, canonical `maxTimeDiff` first — reading only one would import the other as null and drop it on the next unrelated update, silently disabling the gate. The provider always **writes** the panel spelling. Same shape as the panel's own `dest` → `target` aliasing. The field is a `uint64` in milliseconds (`transport/internet/reality/config.go` multiplies by `time.Millisecond`), hence `int64validator.AtLeast(0)`; `0` disables the time check |
| `intValue` narrows on 32-bit release targets | `intValue` returns `int`, which is 32 bits on the `386`/`arm` builds GoReleaser ships, so any field the upstream schema models as 64-bit wraps there. Use `int64Value` on those paths — `reality_settings.max_timediff` does. Note the residual ceiling: a value decoded from JSON arrives as `float64`, so it is only exact up to 2^53 |
| Omitting an `Optional+Computed` attribute is not a clear | With `UseStateForUnknown`, an attribute deleted from the configuration plans as unknown and the modifier copies the prior state — Update keeps sending the old value. This is load-bearing (it is what keeps `TestAccInboundImportNoDrift_Reality` driftless when the config omits server-populated fields), so a "clear on omission" plan modifier is **not** the fix: the two cases are indistinguishable in config. Attributes needing a reversible bound must therefore document an in-band sentinel instead — REALITY uses `min_client_ver = "0.0.0"`, `max_client_ver = "255.255.255"`, `max_timediff = 0`. Covered by the final step of `TestAccInboundReality` |
| Subscription resource does **double apply** | Workaround: `sub_json_enable` not saved on first apply with `sub_enable` |
| Policy levels: `{"0": {...}}` ↔ `[{id=0}]` | Xray JSON uses a map, TF uses a list |
| DNS servers: string vs object | Address-only → string; with extra fields → object |
| Protocol names are version-specific | `hysteria2` is a distinct protocol only on v3.0.2/v3.1.0; from v3.2.0 use `protocol = "hysteria"` with `streamSettings.version = 2`. `tunnel` requires ≥ v3.2.0, `tun` requires ≥ v3.2.7, and `mtproto` requires ≥ v3.3.0 |
| `xray_version` delete is a no-op | Removing from state does NOT revert the installed xray version |
| `web_base_path` change triggers panel restart | Must also update provider `base_path`; code auto-updates client |
| `panel_outbound` vs `panel_proxy` is version-specific | 3x-ui v3.3.1 **replaced** `panelProxy` (HTTP/SOCKS5 URL, present only in v3.2.0–v3.3.0; absent in v3.0.2/3.1.0) with `panelOutbound` (an Xray outbound/balancer tag). The provider's `panel_proxy` attr is `Deprecated` (provider-side annotation) and maps to the old field; gin silently ignores the unknown `panelProxy` form value on v3.3.1 |
| v3.4.0 AllSetting delta | 3x-ui v3.4.0 **added** 14 fields now managed by the provider: SMTP notifications (`smtp*`, 10 fields, `panel_email` resource, `smtpPassword` is sensitive + write-only), `tgEnabledEvents`/`tgMemory` (`panel_telegram`), `remarkTemplate`/`subHideSettings` (`panel_subscription`). v3.4.0 also **dropped** `remarkModel`/`subEmailInRemark`/`subShowInfo`/`tgBotLoginNotify` from `AllSetting`; the provider still sends them (backward compat for v3.1.x–v3.3.x) and they sit in `intentionallySkipped` in `drift_test.go` |
| v3.4.1 AllSetting delta | 3x-ui v3.4.1 **added** 2 subscription fields now managed by `panel_subscription`: `subIncyEnableRouting`/`subIncyRoutingRules` (Incy client routing injection). No fields were removed |
| v3.4.2 AllSetting + model delta | 3x-ui v3.4.2 **added** `ldapInsecureSkipVerify` to `AllSetting` (managed by `panel_general`). `Client`/`ClientModel` gained WireGuard multi-client fields (`privateKey`/`publicKey`/`allowedIPs`/`preSharedKey`/`keepAlive`); WG inbounds now have native `dns` + `clients[]` (single-peer inbounds migrate automatically). `ClientModel` also gained `resetUp`/`resetDown`. `updateUser` (password change) now requires a `twoFactorCode` form field when 2FA is enabled (`internal/web/controller/setting.go`); `updateSetting` requires it only when disabling 2FA — the provider auto-attaches `twoFactorCode` on both (see gotcha). xray-core bumped to v26.6.27. **Observatory/BurstObservatory** (`frontend/src/schemas/observatory.ts`) are new top-level xray template keys (`xray.ts:33-34`, opaque `z.unknown()`), auto-synced by the frontend with balancers (`balancer-helpers.ts:syncObservatories`) and absent from the default config; they are now exposed as the `threexui_xray_observatory` typed resource (`xray_observatory_schema.go`), which co-owns both the `observatory` and `burstObservatory` top-level keys — the panel's frontend `syncObservatories` only runs in the browser on UI-save, not on API-driven changes, so the two do not conflict at the API level (see gotcha) |
| v3.5.0 AllSetting + model delta | 3x-ui v3.5.0 **AllSetting is identical to v3.4.2** (0 field delta — `entity.go` `AllSetting` struct unchanged at 94 fields). `Client`/`ClientRecord` gained MTProto multi-client fields `secret` (FakeTLS, `json:"secret,omitempty"`) and `adTag` (`json:"adTag,omitempty"`) for the new `mtg-multi` sidecar; inbound-level MTProto `secret`/`adTag` are now vestigial (`StripMtprotoInboundSecret`/`StripMtprotoInboundAdTag`). xray-core bumped to **v26.7.11**. New top-level xray template keys: `env` (`map[string]string`, managed by `xray_basics`) and `outbounds[].targetStrategy` (domain-strategy enum, managed by `xray_outbounds`). **Shadowsocks** `method` `none`/`plain` and **vmess** `security` `none`/`zero` were removed in xray-core v26.7.11 — the panel auto-migrates them on startup (`migrateShadowsocksRemovedCiphers`/`migrateVmessRemovedSecurities` in `internal/database/db.go`), causing silent drift if a provider-managed inbound still carries them (see cipher gotcha). **ValidateOutboundConfig** now pre-rejects unencrypted public vless/trojan outbounds at apply (was a silent xray crash). **finalmask.tcp + REALITY** is rejected at save (XTLS/Xray-core#6453). **EnsureDnsServerRouting** (`internal/web/service/xray_setting_dns_routing.go`) auto-inserts managed `direct` allow-rules for private DNS servers before the `geoip:private` block on every xray-template save — the provider filters these (`isManagedDnsAllowRule`) to avoid permanent drift. New `entity.HostGroup` struct + `/panel/api/hosts/*` routes (bulk host management) are managed by the `threexui_host_group` resource. Balancer-to-balancer fallback reserves the `_bl_` tag prefix; `balancer.fallback_tag` + `balancer.strategy.settings` (expected/max_rtt/tolerance/baselines/costs) are now exposed on `xray_balancers`. WireGuard live updates no longer drop clients (`WireguardClientsToPeers` now runs in `GenXrayInboundConfig`) |
| MTProto multi-client on v3.5.0 (`mtg-multi`) | 3x-ui v3.5.0 moved MTProto to the `mtg-multi` engine: each client carries its own FakeTLS `secret` and optional `adTag` on `Client` (per-client), surfaced on `threexui_inbound_client` as `secret` (Sensitive) and `ad_tag` (RegexMatches `^[0-9a-fA-F]{32}$`). The panel heals per-client secrets on save via `HealMtprotoClientSecrets`, rebuilding the domain suffix from the inbound's `fakeTlsDomain` (default `www.cloudflare.com`) — so only the random 32-hex middle of `secret` must stay stable across applies. Inbound-level MTProto `secret`/`adTag` are vestigial. The typed `mtproto_settings` block on `threexui_inbound` manages `fakeTlsDomain`, domain-fronting listeners, IP preference, Xray routing, and public addresses; clients are added separately through `threexui_inbound_client`. Per-client `secret`/`ad_tag` fields are v3.5.0+ only and absent from `Client` on v3.4.x |
| v3.6.0 AllSetting + model delta | 3x-ui v3.6.0 **added** 9 `AllSetting` fields now managed by the provider: `smtpFrom`/`smtpFromName` (`panel_email`), `subJsonAutoDetect`/`subJsonAlwaysArray`/`subJsonUserAgentRegex`/`subClashAutoDetect`/`subClashUserAgentRegex` (`panel_subscription`), `subShowIdentityOnAllLinks`/`outboundDownThreshold` (`panel_general`). `Inbound` gained `trafficResetDay` (`threexui_inbound.traffic_reset_day`; pre-v3.6.0 panels report 0, so the attr is `Optional + Computed` with a `Between(0, 31)` validator rather than a hard default). Node `apiToken` became write-only upstream ([3x-ui #5613](https://github.com/MHSanaei/3x-ui/pull/5613)); xray-core v26.7.28 |
| v3.7.0 AllSetting + model delta | 3x-ui v3.7.0 **added** 2 `AllSetting` fields: `ipLimitAllowlist` (`panel_general.ip_limit_allowlist` — addresses exempt from the per-client IP limit) and `subJsonObservatory` (`panel_subscription.sub_json_observatory` — client-side balancer observatory blob). `Inbound` gained `disableFlow` (`threexui_inbound.disable_flow` — opts the inbound out of auto XTLS Vision). `Client` gained 6 fields: `resetDay`/`resetMax` (calendar-day renewals, `0` keeps the old rolling interval), `trafficReset`/`trafficResetDay` (per-client reset cycle, independent of the inbound's own) — all four surfaced on `threexui_inbound_client` — plus `forwardedPorts` and `allowedIPsByInbound`, which belong to the AmneziaWG surface and sit in `clientIntentionallySkipped` in `drift_test.go`. A new `amneziawg` protocol was added upstream and is implemented by the provider as the `amneziawg_settings` block (#441). `ApiToken` gained `scope`/`expiresAt` — the provider does not manage API tokens, so no gate covers them. Go toolchain upstream is 1.27; first start runs automatic schema migrations, so a panel upgrade needs a DB backup |
| AmneziaWG settings are nested, unlike every other protocol | The `amneziawg` settings blob is `{"server": {...}, "clients": [...]}` (`3x-ui-3.7.0/internal/amneziawg/types.go:242-245`), while `buildSettingsJSON`/`flattenSettings` are one flat snake_case→camelCase key table. The server object is therefore forwarded verbatim, protocol-gated exactly like WireGuard's `clients[]`, and `expandAmneziawgServerFromModel` emits camelCase directly — folding it into the flat table would collide on `mtu` (WireGuard), `privateKey`/`publicKey` and `routeThroughXray` (MTProto). Peers live in `amneziawg_settings.clients[]` and are owned by `threexui_inbound`, NOT by `threexui_inbound_client` — the same rule as WireGuard. Two panel behaviours make this surface unforgiving. First, an inbound saved with settings that carry no `server` object makes the panel **regenerate the whole server block, keypair included** (`normalizeAmneziaWGSettings`, `internal/web/service/inbound_amneziawg.go:171-200`) — on update as well as create, verified live — so `amneziawg_settings.server` is enforced by a `ConfigValidator` and its absence is a plan-time error; `TestAccInboundAmneziawgKeysSurviveUnrelatedUpdate` guards the other half, that a declared-but-empty block survives an unrelated edit. Second, a **partial** `server` block is worse than none: the panel generates its randomised obfuscation set only when the settings carry no `server` object, so a block that sets just a subnet stores `jc = 0`, blank `h1`-`h4` and no header protection — plain WireGuard under an AmneziaWG name, with nothing reporting it. Create therefore runs in two phases (`splitAmneziawgServer` / `applyAmneziawgServerOverrides`): post the inbound without the block, let the panel fill it, then apply the configured fields on top. Update needs none of that, since state already holds the generated set. Third, `preserveInboundSettings` must NOT re-inject `clients[]` for wireguard/amneziawg (`protocolOwnsClients`) — those peers belong to the inbound, so copying the existing array back makes removing the last peer impossible: the apply fails with "block count changed from 0 to 1" while the peer keeps connecting. Fourth, `clients[].public_key` is **Required**, unlike its WireGuard counterpart: the panel rejects a keyless peer on inbound create and update (`inbound.go:1042-1044`, `client_inbound_apply.go:441`) and derives keys only on the `/panel/api/clients` endpoints, which do not own these peers. The panel generates the keypair, subnet and a randomised obfuscation set when the inbound is created without settings, so every attribute is `Optional+Computed`. Fields upstream marks `omitempty` (`mtu`, `i1`-`i5`, the timings, `externalInterface`, …) are stripped on save, so the schema rejects `""`/`0` for them — omit the attribute instead; `primaryDns`/`secondaryDns`/`h1`-`h4` are the opposite, where `""` is a meaningful cleared value, and the `omitempty` booleans read back as `false` rather than null. `allowedIPsByInbound` is deliberately unimplemented: the panel accepts it only on the client CRUD endpoints and never echoes it back |
| `disable_flow` blanks client `flow` | On a `threexui_inbound` with `disable_flow = true`, 3x-ui clears `flow` on every attached client: `clientWithInboundFlow` (`internal/web/service/client_crud.go`) runs on the add-client and update-client paths, and `stripClientFlows` re-runs over the inbound settings on every inbound add/update. A `threexui_inbound_client` with `flow` set on such an inbound fails the apply (`.flow: was cty.StringVal("xtls-rprx-vision"), but now cty.StringVal("")`), and flipping `disable_flow` on later silently strips flows off already-managed clients. Leave client `flow` unset on `disable_flow` inbounds |
| `traffic_reset_day` cannot be 0 | Upstream `normalizeTrafficResetDay` clamps any value below 1 up to 1, on the inbound path (`AddInbound`/`UpdateInbound`) and, since v3.7.0, on the client path too (`normalizeClientTrafficReset`, which additionally normalizes an empty `trafficReset` to `never`). Omitting the form key and posting an explicit `0` are therefore indistinguishable to the panel. Both `traffic_reset_day` attributes use `Between(1, 31)` so a non-round-trippable `0` is rejected at plan time instead of failing mid-apply with an inconsistent result |
| Sub-server settings are frozen at startup | Every setting `(*sub.Server).initRouter()` reads — not just the binding keys — is captured into the `SUBController` it builds (`3x-ui-3.7.0/internal/sub/sub.go:50-301`), and `initRouter` runs only from `Start()`, which `main.go` calls at boot and on SIGHUP. That covers `subJsonMux`/`subJsonRules`/`subJsonFinalMask`/`subJsonObservatory`, the `subJsonPath`/`subClashPath`/`subJsonEnable`/`subClashEnable` route switches, and presentation fields like `subTitle`. All of them are in `restartKeys` since #443; before that a change applied to the panel DB and to state while every served subscription kept the old value — the same silent no-op as #291. When adding a `panel_subscription` attribute, check whether `initRouter` reads it and add it to `restartKeys` if so. The only per-request exceptions are `subURI`/`subJsonURI`/`subClashURI` |
| Removed ciphers cause silent drift on v3.5.0 | xray-core v26.7.11 (3x-ui v3.5.0) **removed** shadowsocks `method` `none`/`plain` and vmess `security` `none`/`zero`. The panel runs `migrateShadowsocksRemovedCiphers` (→ `chacha20-ietf-poly1305`) and `migrateVmessRemovedSecurities` (→ `auto`) on **every startup** (`internal/database/db.go`, not seeder-gated, idempotent). A provider-managed inbound carrying one of these values applies cleanly, then drifts after the next panel restart: `terraform plan` shows state=`none`, panel=`chacha20-ietf-poly1305` (or `auto`). Avoid these values in `threexui_inbound` settings / `threexui_inbound_client.security` |
| `threexui_inbound.all_time` is a constant 0 | Upstream carried `allTime` on `model.Inbound` and `xray.ClientTraffic` from v2.6.7 ([PR #3387](https://github.com/MHSanaei/3x-ui/pull/3387)) and removed it in v3.1.0 ([PR #4469](https://github.com/MHSanaei/3x-ui/pull/4469)), which also drops the `all_time` columns at startup — the drop migration is still in every snapshot (`3x-ui-3.7.0/internal/web/service/inbound_migration.go:55-63`). No supported panel (v3.2.x+) sends the field, so `all_time` always reads 0. The attribute now carries a `DeprecationMessage` and is excluded from `trafficCounterPaths` (a constant needs no unknown-marking); removing it is breaking and waits for the next major (#442). Note Terraform >= 1.15 surfaces the deprecation at the *reference* site, so anyone reading the attribute sees a warning from `validate` onward |
| WireGuard `workers` deprecated | xray-core v26.6.22 (3x-ui v3.4.0) removed the WireGuard `workers` field; `xray_outbound_wireguard` now marks it `DeprecationMessage` — accepted for backward compat, ignored by xray |
| Write-only attrs quirks | See "Write-only secret attributes" section above — read `_wo` from `req.Config`; ModifyPlan marks plain `Unknown` on version change; `panel_user` nulls state password instead |
| 2FA re-confirmation on v3.4.2 (provider auto-sends) | 3x-ui v3.4.2 made `/setting/updateUser` require a `twoFactorCode` whenever 2FA is enabled, and `/setting/update` require it when disabling 2FA. The provider's `Client.UpdateUser` and `Client.UpdateSettings` automatically attach `twoFactorCode` (from the provider `two_factor_code` config) to the payload when set — older panels (≤ v3.4.1) bind into `AllSetting` and ignore the extra key, so this is backward-compatible. `UpdateSettings` copies the caller's map first so the code never leaks into plan/state. No resource-level change is needed; users only need `two_factor_code` set when 2FA is on |
| WireGuard `peer` vs `clients` on v3.4.2 | On v3.4.2, single-peer WG inbounds auto-migrate upstream from `settings.peers[]` into `wireguard_settings.clients[]`. A provider-managed WG inbound declared with only `peer[]` may therefore surface `clients[]` populated and `peer[]` empty after the panel upgrades — the next `terraform plan` shows an unexpected diff (it self-heals on apply, since Read records the new state). Also: the legacy `peer` block is serialised with snake_case JSON keys while the v3.4.2 `clients[]` block uses camelCase (matching upstream `client_wireguard.go`); both arrays are Optional, so populate EITHER `peer` OR `clients` per inbound, never both (the panel treats them as separate models). The `clients[].email` attr is Optional in the schema but the panel requires a non-empty unique email (it keys traffic counters on it), mirroring the generic `threexui_inbound_client` `email` gotcha |
| Deleting a WG/AmneziaWG inbound would orphan its peers' emails | `DelInbound` → `ClientService.DetachInbound` (`3x-ui-3.7.0/internal/web/service/client_link.go:291-296`) deletes only the `client_inbounds` link rows; the `clients` rows themselves — which carry the `uniqueIndex` on `email` — survive. Peers owned by `threexui_inbound` (WireGuard `clients[]` since v3.4.2, AmneziaWG since v3.7.0) would be left unreachable but still holding their email, so recreating the inbound fails with `Duplicate email: <addr>`. The same is true of a peer merely dropped from the block: `UpdateInbound` rewrites `settings.clients[]` without it and keeps the row (measured on v3.7.0). `InboundResource.Delete` therefore calls `releaseInboundOwnedPeers` after `DeleteInbound`, and `Update` calls `releasePeerEmailsRemovedByUpdate` for the peers that left the plan, both deleting through `/panel/api/clients/del/<email>` — verified on v3.7.0, where create+delete+create fails without it and succeeds with it (#452). Failures there are warnings, not errors: the destroy still has to complete. Gated on `protocolOwnsClients`, so protocols whose clients belong to `threexui_inbound_client` are untouched — that resource deletes its own rows. Rows still leak in two cases: the inbound removed **outside Terraform** (panel UI, direct API), and a create that fails after `AddInbound` succeeded — no state is written, so `Delete` never runs. Both need manual cleanup |
| WireGuard `clients[]` is managed by `threexui_inbound`, NOT `threexui_inbound_client` | Unlike vmess/vless/trojan/shadowsocks/hysteria — whose clients live in `settings.clients[]` and are owned by the separate `threexui_inbound_client` resource (the inbound settings layer deliberately **strips** `clients[]` for those protocols) — WireGuard multi-client peers live in `wireguard_settings.clients[]` and are managed by `threexui_inbound` itself. `buildSettingsJSON`/`flattenSettings` forward `clients[]` ONLY when `protocol == "wireguard"` (protocol-aware pass-through, #342). Adding a WireGuard client therefore means a `clients { ... }` block under `wireguard_settings`, not a `threexui_inbound_client` resource. Mixing the two surfaces on one WG inbound is unsupported. |
| xray-settings acc-tests restart xray-core | `TestAccXrayBasics`/`DNS`/`Routing`/`Balancers`/`Reverse`/`Outbounds` each apply a new config = xray restart; `version` becomes `"Unknown"` for ~30-90s after. Use `waitForXrayVersion(t, ctx, client)` poll-loop in any acc-test probing version after them (#280) |
| `TestAccXrayVersionDrift` drift sim skips on pickup failure | Step 2 simulates drift via `InstallXray` + bounded poll (3 installs × 6 polls). If the panel accepts the install but never swaps the binary (intermittent on GitHub-hosted runners, esp. downgrades), it `t.Skipf`s — same environment-issue category as #279/#280, NOT a provider regression. The poll budget is deliberately tight: the old 20-poll loop ran ~7.5 min and ate the whole test binary's `-timeout`, panic-killing unrelated tests (#306). Do NOT widen the budget back to 20. A `t.Deadline()` guard at the top of the test also skips cleanly when the preceding `TestAccXrayVersion`/`TestAccXrayVersionResource` tests already burned the binary budget via their 180s `waitForXrayVersion` stalls — otherwise the panic fires mid-Step-2 before drift's own skip path |
| `GetXrayVersions` hits GitHub anonymously | 3x-ui's `getXrayVersion` handler has no `Authorization`; 60 req/h/IP on shared runners. Use `skipOnUpstreamRateLimit(t, err)` / `testAccSkipOnXrayVersionsRateLimit(t)` to skip, not fail (#279) |
| PostgreSQL runner is slower than SQLite | Timing-sensitive flakes (xray restart windows, InstallXray pickup) surface on the `test:acc:postgres` job but not on `test:acc`. A green SQLite run does NOT prove PostgreSQL is green |
| Acceptance failure logs are captured before teardown | `task test:acc*` conditionally writes Docker logs to `.pi/artifacts/3xui-container.log` from a deferred command. Deferred commands run in reverse order, so keep the log-capture defer after the compose-down defer; CI uploads that file after the task has cleaned up containers. |

**Always check 3x-ui source snapshots** (`3x-ui-<version>/`) before assuming API behavior.
Key paths: `internal/database/model/`, `internal/web/service/`, `internal/web/controller/`, `internal/web/entity/`, `frontend/src/schemas/`, `xray/`.

> Note: 3x-ui reorganised its source tree under `internal/` (in v3.4.x snapshots the model/service/controller/entity files live under `internal/...`, not `database/...`/`web/...`). Older `3x-ui-<version>/` snapshots (≤ v3.3.x) still use the pre-`internal/` layout.

---

## Conventions

### Keeping AGENTS.md current

AGENTS.md is the only place that documents non-obvious behavior an agent cannot
derive from reading one file. Update it **in the same PR** when you:

- add or remove a resource, data source, or schema attribute;
- add a new mutex, ModifyPlan behavior, or read-modify-write path;
- change API endpoint usage, retry budgets, or version-gated behavior;
- hit a new non-obvious constraint that caused (or would cause) a real bug —
  add it to "Critical Gotchas".

Prefer citing the 3x-ui source snapshot (`3x-ui-<version>/`) as evidence for
API-behavior claims rather than restating them from memory.

### Working artifacts

Save scratch artifacts (subagent audit output, draft analysis, large intermediate
results) under `.pi/artifacts/`, not in the repo root. `.pi/` is git-ignored and
reserved for agent tooling; nothing under it is committed. This keeps audit
output reusable within a session without polluting the tree.

### Commits

Conventional Commits: `feat:`, `fix:`, `docs:`, `ci:`, `test:`, `chore:`.
Imperative mood, concise subjects.

### File naming

| Pattern | Example |
| --- | --- |
| Resources | `provider/resource_<name>.go` |
| Data sources | `provider/data_source_<name>.go` |
| Schema helpers | `provider/<area>_schema.go` |

### Documentation

- **README** is localized in **7 languages** (en, ru, fa, ar, zh, es, tr). When changing
  `README.md`, update all `README.<locale>.md` files in the same PR. Persian/Arabic
  wrap body in `<div dir="rtl">`.
- **SECURITY.md** tracks sensitive fields — add a row when adding resources that
  handle secrets.
- **`docs/guides/`** for operational walkthroughs needing more than an `examples/` folder.
- **`context7.json`** (repo root) controls how [Context7](https://context7.com) indexes
  this project. It excludes the `3x-ui-<version>/` source snapshots (~1900 upstream
  files that would otherwise drown the provider's own API in search results), `CHANGELOG.md`,
  and build/cache artefacts, and exposes `rules` for coding agents. Update `excludeFolders`
  whenever a new snapshot is added (see "Adding a new 3x-ui version to CI").
- `scripts/check-readme-parity.sh` compares machine-identifiable resource/data-source
  names and example links across all seven READMEs; translated prose and row order are
  intentionally ignored. `TestDocumentedSchemaSections` in `provider/docs_schema_test.go`
  provides an opt-in semantic guard for complex hand-written nested schema sections.
  When adding a guarded section, add one `assertDocumentedBlock` call for the parent and
  each nested block; the test checks field names, nesting, and block shape but deliberately
  does not couple documentation to prose wording.

### Testing

- Unit tests: `TestXxx` naming, table-driven where practical.
- Acceptance tests: `TestAccXxx`, `terraform-plugin-testing`,
  `ProtoV6ProviderFactories` (not `ProviderFactories`).
- Version-aware skipping: `requireMinVersion(t, "vX.Y.Z")` for features added in a
  specific 3x-ui version; `requireBelowVersion(t, "vX.Y.Z")` for features removed
  upstream (e.g. `hysteria2`, dropped in v3.2.0). Currently supported: **v3.2.x**,
  **v3.3.x**, **v3.4.x**, **v3.5.x**, **v3.6.x**, **v3.7.x** (up to v3.7.0).
  `requireBelowVersion(t, "v3.2.0")` gates (legacy `dokodemo-door`, `hysteria2`)
  no longer fire in CI now that v3.1.x is out of the matrix — they only run when
  a panel older than v3.2.0 is pointed at manually.
- Flaky test quarantine: `skipOnFlakyVersions(t, ...)` / `skipIfFlaky(t)` with
  `THREEXUI_SKIP_FLAKY` env var to skip known-broken upstream versions.
- Xray-version acc-tests use `waitForXrayVersion(t, ctx, client)` (180×1s retry on
  `ErrXrayVersionUnknown`) — covers both the cold-start race (handled by the
  `_wait-for-xray` + `_warm-xray-version-cache` Taskfile gates, the latter pre-warms
  3x-ui's 15-min GitHub-API cache) and mid-test restarts after `TestAccXray*`
  settings tests. **Do not** call `client.GetCurrentXrayVersion()` directly in
  acc-tests (#280).
- Any acc-test exercising `GetXrayVersions` (directly or via `threexui_xray_versions`
  data source / `xray_version` resource) must pre-flight with
  `testAccSkipOnXrayVersionsRateLimit(t)`, or wrap the error with
  `skipOnUpstreamRateLimit(t, err)`. Rate limit is an environment issue, skip —
  don't fail (#279).
- Protocol matrix test (`resource_inbound_matrix_test.go`): comprehensive
  create/update/import round-trip for every protocol.
- Destroy checks use `destroyVisibilityAttempts` (60 × 500 ms) to handle
  SQLite visibility lag after delete.
- **Codecov patch coverage is a PR gate.** The `ci.yml` test job uploads
  `coverage.out` to Codecov, which posts a *Patch coverage* comment on the PR
  (e.g. `⚠️ Patch coverage is 39% with 28 lines missing`). A red Codecov
  report must be resolved before merge — do not ignore it. Concretely:
  - New `provider/*.go` code is **unit-tested**, not only acc-tested. A data
    source's `Schema`/`Configure`/`Read`, a resource's CRUD methods, and new
    `Client` methods must each have a direct unit test. Acc tests don't count
    toward patch coverage and run only with `TF_ACC=1`.
  - If Codecov flags a line, either cover it with a unit test or, if the line
    is genuinely unreachable/defensive (e.g. a can't-happen error branch),
    document why inline. Do **not** merge with a red patch-coverage report
    just because CI is green — `task test:unit:coverage` reproduces the same
    numbers locally (`coverage.out`).
  - Install the Codecov GitHub app on the repo so uploads/comments are
    processed reliably (the bot warns: *"Please install the 'codecov app svg
    image'…"* when the app is missing).

### Pre-commit

`pre-commit install` sets up hooks: `task fmt`, `task vet`, `task lint`, `task build`,
`markdownlint-cli2` (markdown files), plus standard checks (trailing whitespace,
end-of-file fixer, YAML/JSON validation, large files, merge conflicts).
Run manually: `pre-commit run --all-files`.

### Supply chain

CI lint job also runs `govulncheck ./...` before fmt/vet/lint.
All CI actions pinned to commit SHA (`@<sha> # vN`).
Pre-commit hooks use `--freeze` format (`rev: <sha>  # frozen: <tag>`).

### Adding a new 3x-ui version to CI

`compat-versions.json` is the single source of truth for tested versions.
CI and flake-tracking workflows read the matrix dynamically from this file.
`scripts/sync-versions.sh check` validates that README tables, docker-compose,
and Taskfile defaults are in sync.

When a new 3x-ui version is released:

1. **`compat-versions.json`** — add the new version to the `versions` array, update `default_version` if it's the latest patch
2. **`scripts/sync-versions.sh fix`** — auto-updates README tables, docker-compose default, and Taskfile defaults
3. **`scripts/sync-versions.sh check`** — verify all surfaces are in sync
4. **Source snapshot** — copy the 3x-ui source to `3x-ui-<version>/` (drift tests use the latest snapshot)
5. **`provider/drift_test.go`** — if fields were added/removed from upstream structs, update `intentionallySkipped` and known-field maps
6. **`provider/testdata/upstream_contract.json`** — update `all_setting_fields`, `protocols_go_model`, `version` to match the latest 3x-ui release (used by drift tests when no local snapshot is present)
7. **`docs/`** — if provider schema changed, update the corresponding resource/data-source doc
8. **`AGENTS.md`** — update the "up to vX.Y.Z" version reference if the minor line changed
9. **`context7.json`** — add the new `3x-ui-<version>/` snapshot folder to `excludeFolders` so Context7 does not index the upstream source copy
