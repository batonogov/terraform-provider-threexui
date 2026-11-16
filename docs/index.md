---
page_title: "3x-ui Provider"
description: |-
  Terraform provider for managing 3x-ui panel resources.
---

# 3x-ui Provider

The 3x-ui provider allows you to manage [3x-ui](https://github.com/MHSanaei/3x-ui) panel resources using Terraform. It supports managing inbounds, clients, panel settings, and Xray configuration.

## Example Usage

```hcl
provider "threexui" {
  endpoint = "http://localhost:2053"
  username = "admin"
  password = "admin"
}

resource "threexui_inbound" "vless" {
  port     = 443
  protocol = "vless"
  enable   = true
  remark   = "VLESS Reality"

  vless_settings {
    decryption = "none"
  }

  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "www.amazon.com:443"
      server_names = ["www.amazon.com"]
    }
  }

  sniffing {
    enabled       = true
    dest_override = ["http", "tls", "quic", "fakedns"]
  }
}
```

## Authentication

The provider authenticates to the 3x-ui panel using username and password. These can be provided directly in the provider configuration, through environment variables, or indirectly through Terraform input variables such as `TF_VAR_threexui_username` and `TF_VAR_threexui_password`.

### Environment Variables

The following provider attributes can be set via `THREEXUI_*` environment variables. When both HCL configuration and an environment variable are present, the HCL value takes precedence.

| Attribute | Environment Variable | Default |
| --- | --- | --- |
| `endpoint` | `THREEXUI_ENDPOINT` | _(none — required)_ |
| `base_path` | `THREEXUI_BASE_PATH` | `/` |
| `username` | `THREEXUI_USERNAME` | `admin` |
| `password` | `THREEXUI_PASSWORD` | `admin` |
| `insecure_skip_verify` | `THREEXUI_INSECURE_SKIP_VERIFY` | `false` |
| `request_timeout` | `THREEXUI_REQUEST_TIMEOUT` | `30s` |
| `max_retries` | `THREEXUI_MAX_RETRIES` | `1` |

`bootstrap_username`, `bootstrap_password`, and `two_factor_code` intentionally have no environment-variable fallback. Configure them explicitly in the provider block (usually through sensitive Terraform input variables) when bootstrap authentication or 2FA is required.

Minimal configuration using environment variables:

```hcl
provider "threexui" {}
```

```shell
export THREEXUI_ENDPOINT="http://localhost:2053"
export THREEXUI_USERNAME="admin"
export THREEXUI_PASSWORD="secret"
terraform apply
```

For the attributes in the table, precedence is: explicit HCL > `THREEXUI_*` environment variable > built-in default.

For first-run bootstrap of a fresh panel, you can configure `bootstrap_username` and `bootstrap_password` in addition to the steady-state `username` and `password`. Use this together with `threexui_panel_user` to rotate the panel to the steady-state credentials during the same apply. On 3x-ui v2.9.x, failed logins can expose the submitted password in panel logs or Telegram login notifications, so the provider tries bootstrap credentials before the steady-state credentials. On 3x-ui v3.x, the provider tries steady-state credentials first and falls back to bootstrap credentials only if the panel rejects them. The provider does not silently try `admin`/`admin`; bootstrap credentials must be configured explicitly.

```hcl
provider "threexui" {
  endpoint = "http://localhost:2053"

  username = var.threexui_username
  password = var.threexui_password

  bootstrap_username = var.bootstrap_username
  bootstrap_password = var.bootstrap_password
}

resource "threexui_panel_user" "admin" {
  username = var.threexui_username
  password = var.threexui_password
}
```

Fresh 3x-ui v3.11.0+ panels generate random first-user credentials, not `admin`/`admin`. Supply those initial credentials from the first-start log, or set explicit ones with `x-ui setting -username <user> -password <pass>`. Declare password input variables as `sensitive = true`. See the [panel-user example](https://github.com/batonogov/terraform-provider-threexui/tree/main/examples/panel-user) for the complete configuration.

3x-ui v3 protects login and unsafe API requests with CSRF tokens. The provider fetches and refreshes those tokens automatically; no additional configuration is required.

-> **Note:** The provider has **partial** 2FA support. You can supply a TOTP code via the `two_factor_code` attribute, and it will be sent with the initial login request. However, TOTP codes expire every 30 seconds. Because the provider performs automatic re-login when the session expires (on HTTP 401/404), subsequent logins will fail once the original code is no longer valid. For short-lived operations (a single `terraform apply`) this may work, but long-running or repeated runs will require a fresh code each time.

## Argument Reference

- `endpoint` (Optional, String) - Base URL of the 3x-ui panel, e.g. `http://localhost:2053`. Can also be set via `THREEXUI_ENDPOINT` environment variable.
- `base_path` (Optional, String) - Base path configured in 3x-ui (`webBasePath`). Default is `/`. Can also be set via `THREEXUI_BASE_PATH` environment variable.
- `username` (Optional, String) - 3x-ui username. Default is `admin`. Can also be set via `THREEXUI_USERNAME` environment variable.
- `password` (Optional, String, Sensitive) - 3x-ui password. Default is `admin`. Can also be set via `THREEXUI_PASSWORD` environment variable.
- `bootstrap_username` (Optional, String) - Bootstrap username for explicit first-run credential rotation. On 3x-ui v2.9.x it is tried before the primary `username`/`password` to avoid exposing the desired password in failed-login logs; on 3x-ui v3.x it is tried only after the primary credentials are rejected. Must be set together with `bootstrap_password`.
- `bootstrap_password` (Optional, String, Sensitive) - Bootstrap password for explicit first-run credential rotation. On 3x-ui v2.9.x it is tried before the primary `username`/`password` to avoid exposing the desired password in failed-login logs; on 3x-ui v3.x it is tried only after the primary credentials are rejected. Must be set together with `bootstrap_username`.
- `two_factor_code` (Optional, String, Sensitive) - TOTP code for 2FA login. Used for the initial authentication request. See the note above about re-login limitations.
- `insecure_skip_verify` (Optional, Boolean) - Skip TLS certificate verification (useful for self-signed certs). Default is `false`. Can also be set via `THREEXUI_INSECURE_SKIP_VERIFY` environment variable.
- `request_timeout` (Optional, String) - HTTP request timeout (e.g. `30s`, `1m`). Default is `30s`. Can also be set via `THREEXUI_REQUEST_TIMEOUT` environment variable.
- `max_retries` (Optional, Number) - Maximum number of additional attempts on transient HTTP 5xx responses for **idempotent write endpoints only** (`UpdateInbound`, `UpdateInboundClient`, `UpdateSettings`, `UpdateXrayTemplate`, `SetXrayOutboundTestURL`). Each retry waits 500ms and emits a `Warn`-level log entry, so upstream flakiness is observable rather than silently absorbed. Set to `0` to disable retries entirely. Allowed range: `0..10`. Default is `1`. Can also be set via `THREEXUI_MAX_RETRIES` environment variable.

-> **Why this exists:** 3x-ui's pre-v2.9.0 inbound update path was not transactional, and a panic in the panel's handler goroutine surfaced as a 5xx via `gin.Recovery`. v2.9.0 reworked the path to run inside a SQLite transaction (`buildRuntimeInboundForAPI(tx, ...)`) and is more robust, but transient upstream 5xx responses can still happen under write pressure. The retry is scoped to write endpoints whose service-level handlers replace the entire row by id (truly idempotent at the SQL level), so repeating the same request after a transient failure is safe.
