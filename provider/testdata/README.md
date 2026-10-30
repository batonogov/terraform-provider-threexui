# Test Corpus

Fixture files used by `corpus_test.go` for round-trip testing of
flatten/expand conversion layers.

## Structure

| File | Description |
| --- | --- |
| `settings_*.json` | Inbound settings payloads (per-protocol) |
| `stream_settings_*.json` | Stream settings payloads (per-transport) |
| `sniffing_*.json` | Sniffing config payloads |
| `xray_template.json` | Full Xray config template (all sections) |
| `malformed_*` | Edge cases: truncated (.txt), wrong types, nulls, empty |

## Refreshing the Corpus

When the upstream 3x-ui panel changes its API payloads:

1. Start a local instance: `docker compose up -d && sleep 15`
2. Set the test credentials with `docker compose exec -T 3xui /app/x-ui setting -username admin -password admin` (fresh v3.11.0+ panels otherwise generate random first-user credentials)
3. Log in and create representative inbounds via the web UI
4. Fetch raw payloads:

   ```bash
   # Login (3x-ui v3 requires CSRF)
   COOKIE=$(mktemp)
   CSRF=$(curl -fsS -c "$COOKIE" -b "$COOKIE" http://localhost:2053/csrf-token 2>/dev/null \
     | sed -n 's/.*"obj":"\([^"]*\)".*/\1/p' || true)
   if [ -n "$CSRF" ]; then
     curl -fsS -c "$COOKIE" -b "$COOKIE" -H "X-CSRF-Token: $CSRF" \
       -d "username=admin&password=admin&_csrf=$CSRF" \
       http://localhost:2053/login >/dev/null
   else
     curl -fsS -c "$COOKIE" -b "$COOKIE" -d 'username=admin&password=admin' \
       http://localhost:2053/login >/dev/null
   fi

   # Inbound list (extract settings/streamSettings/sniffing from each)
   curl -s -b "$COOKIE" http://localhost:2053/panel/api/inbounds/list | jq .

   # Xray template
   curl -s -b "$COOKIE" -H "X-CSRF-Token: $CSRF" \
     -X POST http://localhost:2053/panel/xray | jq .obj

   # Panel settings
   curl -s -b "$COOKIE" -H "X-CSRF-Token: $CSRF" \
     -X POST http://localhost:2053/panel/setting/all | jq .obj
   ```

5. Replace the fixture files with the new payloads
6. Remove zero-value / empty-string fields that the provider's build
   functions intentionally skip (these are not round-trip stable)
7. Run `go test ./provider/ -run TestCorpus -v` and fix any failures
