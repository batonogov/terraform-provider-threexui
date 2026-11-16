#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FILTER="$ROOT/scripts/compat-matrix.jq"

# Unsorted input, numeric patch/minor ordering, and unsupported patches/lines.
ACTUAL=$(jq -nc '{
  versions: [
    {version: "v3.10.2", supported: true},
    {version: "v3.4.9", supported: true},
    {version: "v3.4.11", supported: false},
    {version: "v3.4.10", supported: true},
    {version: "v3.9.0", supported: true},
    {version: "v3.10.0", supported: true},
    {version: "v4.0.0", supported: true},
    {version: "v5.0.0", supported: false}
  ]
}' | jq -c -f "$FILTER")
test "$ACTUAL" = '["v3.4.10","v3.9.0","v3.10.2","v4.0.0"]'
test "$(echo '{"versions":[]}' | jq -c -f "$FILTER")" = '[]'
echo "PASS: compat matrix selects the latest supported patch per minor line"
