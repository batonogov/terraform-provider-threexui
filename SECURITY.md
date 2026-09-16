# Security Policy

## Supported Versions (provider)

| Version | Supported |
| ------- | --------- |
| 3.x     | Yes       |
| 2.x     | Yes       |
| < 2.0   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability in this project, please report it responsibly.

**Email:** [fekinos@me.com](mailto:fekinos@me.com)

Please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact

**Do not** open a public GitHub issue for security vulnerabilities.

## Response

- You can expect an initial response within **48 hours**.
- We will work with you to understand and address the issue before any public disclosure.

## Sensitive Data Handled by the Provider

The 3x-ui panel issues and stores secrets that this provider reads and writes. The following fields are marked `Sensitive` in the schema so Terraform redacts their values from ordinary CLI output:

| Surface | Sensitive fields |
| --- | --- |
| Provider config | `password`, `bootstrap_password`, `two_factor_code` |
| `threexui_inbound` (`stream_settings.reality_settings`) | `private_key`, `short_ids`, `mldsa65_seed` |
| `threexui_inbound` (`wireguard_settings`) | `secret_key`, peer `private_key`/`pre_shared_key`, multi-client `clients[].private_key`/`clients[].pre_shared_key` (3x-ui v3.4.2+) |
| `threexui_inbound` (`amneziawg_settings`) | `server.private_key`, `server.header_protection_key`, `clients[].private_key`/`clients[].pre_shared_key` (3x-ui v3.7.0+) |
| `threexui_inbound` (`tls_settings`) | `certificates[].key` (inline private key PEM) |
| `threexui_inbound_client` | `id` (contains the client UUID), `client_id`, `password` (trojan/ss), `auth` (hysteria), `secret` (MTProto) |
| `threexui_panel_general` | `ldap_password` |
| `threexui_panel_telegram` | `tg_bot_token` |
| `threexui_panel_discord` | `discord_bot_token` |
| `threexui_panel_email` | `smtp_password` |
| `threexui_panel_security` | `two_factor_token` |
| `threexui_panel_user` | `password` |
| `threexui_node` | `api_token`, `pinned_cert_sha256` |
| `threexui_xray_outbounds` | per-protocol credentials (`vless_settings.id`, `vmess_settings.id`, password/pass fields, WireGuard keys) |
| Data sources | `threexui_inbounds.inbounds`, `threexui_nodes.nodes` (`apiToken`, `pinnedCertSha256`), `threexui_settings.json`, `threexui_xray_config.json` (full payloads) |

## Protecting Terraform State

Terraform state stores **all** sensitive values in plaintext, regardless of the `Sensitive` flag in the schema. Always:

- **Use a remote backend** with encryption at rest (S3 with SSE, GCS with CMEK, Terraform Cloud, etc.).
- **Restrict state-file access** to the smallest set of operators who already need panel access.
- **Avoid committing `terraform.tfstate`** or `*.tfstate.backup` to git — they are in the default `.gitignore`, but worth verifying.
- **Rotate the panel password** if a state file is exposed; the provider stores the credentials it was given.

## Write-Only Arguments (Terraform 1.11+ / OpenTofu 1.11+)

Starting with provider v3.13.0, resources that manage secrets offer write-only (`_wo`) attribute alternatives. Write-only values are sent to the 3x-ui panel but **never stored** in Terraform plan or state artifacts. They require Terraform ≥ 1.11 or OpenTofu ≥ 1.11. Older runtimes reject a configured `_wo` value because they do not advertise write-only-attribute support; use the corresponding plain `Sensitive` attribute when an upgrade is not possible.

| Resource | Write-only attribute | Version trigger |
| --- | --- | --- |
| `threexui_panel_user` | `password_wo` | `password_wo_version` |
| `threexui_panel_security` | `two_factor_token_wo` | `two_factor_token_wo_version` |
| `threexui_panel_telegram` | `tg_bot_token_wo` | `tg_bot_token_wo_version` |
| `threexui_panel_discord` | `discord_bot_token_wo` | `discord_bot_token_wo_version` |
| `threexui_panel_general` | `ldap_password_wo` | `ldap_password_wo_version` |
| `threexui_panel_email` | `smtp_password_wo` | `smtp_password_wo_version` |
| `threexui_node` | `api_token_wo`, `pinned_cert_sha256_wo` | `api_token_wo_version`, `pinned_cert_sha256_wo_version` |
| `threexui_inbound_client` | `password_wo`, `secret_wo` | `password_wo_version`, `secret_wo_version` |

### How it works

- Set `<field>_wo` to the secret value and `<field>_wo_version` to an integer.
- To update the secret, increment `<field>_wo_version` — Terraform detects the change and re-sends the `_wo` value.
- The plain `<field>` attribute remains for backward compatibility. Using it on Terraform 1.11+ / OpenTofu 1.11+ produces a warning suggesting migration to `_wo`.

### Trade-offs

- Write-only values are not available in data sources or during `import`. After import, provide the secret again via `_wo` or the plain attribute.
- Both attributes can coexist in a configuration. The `_wo` value takes priority.

## Network and Authentication Notes

- The provider authenticates to 3x-ui over HTTP/HTTPS with form-based session cookies.
- Use HTTPS in production. The `insecure_skip_verify` flag exists for self-signed lab environments only.
- The provider performs automatic re-login on HTTP 401/404. 2FA support is partial — see the provider docs for the trade-offs.
- Auto-generated Reality keys, short IDs, and WireGuard keys are returned by 3x-ui on create. The provider preserves them across `Read` operations via `UseStateForUnknown` plan modifiers, so a `terraform plan` after `apply` will not surface them as drift.

## Responsible Disclosure Window

We aim to publish a fix and advisory within **30 days** of a confirmed report. If a vulnerability is being actively exploited, the timeline tightens — please flag this in your initial email.
