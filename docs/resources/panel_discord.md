---
page_title: "threexui_panel_discord Resource - 3x-ui"
subcategory: "Panel Settings"
description: |-
  Manages Discord bot notification settings in the 3x-ui panel (v3.8.0+).
---

# threexui_panel_discord (Resource)

Manages the Discord notification bot settings of the 3x-ui panel — the Discord counterpart of the Telegram bot, with its own settings tab.

~> **Note:** Requires 3x-ui **v3.8.0+**. On older panels reads return `null` and writes have no effect.

This is a singleton resource -- only one instance should exist per provider. Deleting this resource only removes it from Terraform state; it does not reset the settings.

~> **Note:** Changing `discord_bot_enable`, `discord_enabled_events`, `discord_cpu` or `discord_memory` can trigger a **panel restart** and therefore brief panel downtime. The CPU/memory alarm samplers are registered once, at panel startup, and read the Discord thresholds/event list there; the periodic-report cron and the bot gateway themselves are hot-reloaded by the panel, so `discord_run_time`, `discord_bot_token` and `discord_channel_id` take effect immediately and do **not** restart the panel. A restart fires only on a change that actually alters what the panel wired up at startup: flipping `discord_bot_enable` (the alarm registration reads it at boot and the hot-reload never re-checks it), switching `discord_cpu`/`discord_memory` between 0 and a threshold, or adding or removing `cpu.high`/`memory.high` in `discord_enabled_events`. Re-tuning an already-active threshold (80 → 90) or adding an unrelated event takes effect immediately and does not restart.

## Example Usage

```hcl
resource "threexui_panel_discord" "settings" {
  discord_bot_enable     = true
  discord_bot_token_wo   = "your-discord-bot-token"
  discord_bot_token_wo_version = 1
  discord_channel_id     = "1234567890123456789"
  discord_admin_ids      = "111111111111111111"
  discord_lang           = "en-US"
  discord_bot_backup     = true
  discord_run_time       = "@daily"
  discord_cpu            = 80
  discord_memory         = 90
  discord_enabled_events = "login,backup,cpu.high,memory.high"
}
```

## Argument Reference

- `discord_bot_enable` (Optional, Boolean) - Enable the Discord notification bot.
- `discord_bot_token` (Optional, String, Sensitive) - Discord bot token. The panel never returns the stored token (read back as empty on 3x-ui v3.8.x); keep it in configuration or use the write-only variant.
- `discord_bot_token_wo` (Optional, String, WriteOnly) - Write-only version of `discord_bot_token`. Not persisted in state. Terraform 1.11+ / OpenTofu 1.11+.
- `discord_bot_token_wo_version` (Optional, Number) - Increment to trigger re-send of `discord_bot_token_wo`. Must be set together with `discord_bot_token_wo`.
- `discord_channel_id` (Optional, String) - Discord channel ID the bot posts to.
- `discord_admin_ids` (Optional, String) - Comma-separated Discord admin/user IDs allowed to use bot commands.
- `discord_run_time` (Optional, String) - Cron expression for the periodic stats report (e.g. `@daily`).
- `discord_bot_backup` (Optional, Boolean) - Send database backups through the Discord bot.
- `discord_cpu` (Optional, Number) - CPU usage threshold (%) for Discord alerts (0-100).
- `discord_memory` (Optional, Number) - Memory usage threshold (%) for Discord alerts (0-100).
- `discord_lang` (Optional, String) - Discord bot language.
- `discord_enabled_events` (Optional, String) - Comma-separated event types to send via Discord (e.g. `login`, `backup`, `cpu.high`, `memory.high`).

All fields require 3x-ui v3.8.0+.

## Attribute Reference

All arguments are also exported as attributes.

## Import

```shell
terraform import threexui_panel_discord.settings settings
```
