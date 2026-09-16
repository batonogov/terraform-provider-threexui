#!/usr/bin/env bash
# sync-versions.sh — validate that CI matrices, README tables, and other
# surfaces stay in sync with compat-versions.json.
#
# Usage:
#   scripts/sync-versions.sh check    # fail if any surface is out of sync
#   scripts/sync-versions.sh fix      # update all surfaces in-place
#
# Exit codes: 0 = OK / fixed, 1 = drift detected (check mode).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPAT="$ROOT/compat-versions.json"
MODE="${1:-check}"

if ! command -v jq &>/dev/null; then
  echo "error: jq is required" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Extract versions from the single source of truth
# ---------------------------------------------------------------------------
VERSIONS_JSON=$(jq -c '[.versions[] | select(.supported == true) | .version]' "$COMPAT")
IFS=$'\n' read -r -d '' -a VERSIONS < <(echo "$VERSIONS_JSON" | jq -r '.[]' && printf '\0')
DEFAULT_VERSION=$(jq -r '.default_version' "$COMPAT")

# Build the sorted (newest-first) array for README tables.
IFS=$'\n' read -r -d '' -a VERSIONS_DESC < <(echo "$VERSIONS_JSON" | jq -r 'reverse | .[]' && printf '\0')

echo "Source: $COMPAT"
echo "Versions (${#VERSIONS[@]}): ${VERSIONS[*]}"
echo "Default: $DEFAULT_VERSION"
echo ""

DRIFT=0

# ---------------------------------------------------------------------------
# 1. CI (ci.yml) — matrix is now dynamic from compat-versions.json,
#    so we only verify the prepare-matrix job references the file correctly.
# ---------------------------------------------------------------------------
check_ci() {
  local file="$ROOT/.github/workflows/ci.yml"
  if grep -q 'compat-versions.json' "$file"; then
    echo "ci.yml: OK (dynamic matrix from compat-versions.json)"
  else
    echo "ci.yml: DRIFT — no reference to compat-versions.json in dynamic matrix"
    DRIFT=1
  fi
}

# ---------------------------------------------------------------------------
# 2. Flake-tracking (flake-tracking.yml) — same dynamic approach.
# ---------------------------------------------------------------------------
check_flake() {
  local file="$ROOT/.github/workflows/flake-tracking.yml"
  if grep -q 'compat-versions.json' "$file"; then
    echo "flake-tracking.yml: OK (dynamic matrix from compat-versions.json)"
  else
    echo "flake-tracking.yml: DRIFT — no reference to compat-versions.json in dynamic matrix"
    DRIFT=1
  fi
}

# ---------------------------------------------------------------------------
# 3. README compatibility tables (7 locales)
# ---------------------------------------------------------------------------
get_status_label() {
  case "$1" in
    README.md) echo "Tested" ;;
    README.ru_RU.md) echo "Тестируется" ;;
    README.es_ES.md) echo "Probado" ;;
    README.fa_IR.md) echo "تست‌شده" ;;
    README.ar_EG.md) echo "تم اختباره" ;;
    README.zh_CN.md) echo "已测试" ;;
    README.tr_TR.md) echo "Test edildi" ;;
    *) echo "" ;;
  esac
}

get_header_line() {
  case "$1" in
    README.md) echo '| 3x-ui version | Status |' ;;
    README.ru_RU.md) echo '| Версия 3x-ui | Статус |' ;;
    README.es_ES.md) echo '| Versión de 3x-ui | Estado |' ;;
    README.fa_IR.md) echo '| نسخهٔ 3x-ui | وضعیت |' ;;
    README.ar_EG.md) echo '| إصدار 3x-ui | الحالة |' ;;
    README.zh_CN.md) echo '| 3x-ui 版本 | 状态 |' ;;
    README.tr_TR.md) echo '| 3x-ui sürümü | Durum |' ;;
    *) echo "" ;;
  esac
}

check_readme() {
  local file="$1"
  local basename
  basename=$(basename "$file")
  local status
  status=$(get_status_label "$basename")
  local header
  header=$(get_header_line "$basename")

  if [ -z "$status" ]; then
    echo "$basename: SKIP (unknown locale)"
    return
  fi

  # Build the expected table rows (newest first)
  local expected=""
  for v in "${VERSIONS_DESC[@]}"; do
    expected+="| $v | $status |"$'\n'
  done

  # Extract the current table rows after the matching header line
  local actual
  actual=$(awk -v hdr="$header" '
    $0 == hdr {found=1; next}
    found && /^\| ---/ {next}
    found && /^\| v/ {print}
    found && /^[^|]/ {exit}
  ' "$file" | sort)

  local expected_sorted
  expected_sorted=$(echo "$expected" | grep '^| v' | sort)

  if [ "$actual" = "$expected_sorted" ]; then
    echo "$basename: OK"
  else
    echo "$basename: DRIFT DETECTED"
    diff <(echo "$actual") <(echo "$expected_sorted") || true
    DRIFT=1

    if [ "$MODE" = "fix" ]; then
      # Build the full table block: header + separator + rows
      local table_block
      table_block="${header}"$'\n'"| --- | --- |"$'\n'
      for v in "${VERSIONS_DESC[@]}"; do
        table_block+="| $v | $status |"$'\n'
      done
      # Remove trailing newline for sed insertion
      table_block="${table_block%$'\n'}"

      # Replace from the header line through the last version row
      # Use a temp file approach for BSD sed compatibility
      local tmp
      tmp=$(mktemp)
      # Pass the multi-line block via the environment (ENVIRON) rather than
      # -v: BWK/one-true-awk rejects literal newlines in -v values with
      # "newline in string". ENVIRON handles embedded newlines portably.
      BLOCK="$table_block" awk -v header="$header" '
        BEGIN { block = ENVIRON["BLOCK"] }
        $0 == header { print block; found=1; next }
        found && /^\| ---/ { next }
        found && /^\| v/ { next }
        found && !/^\|/ { found=0 }
        !found { print }
      ' "$file" > "$tmp" && mv "$tmp" "$file"
      echo "  -> Fixed $basename"
    fi
  fi
}

# ---------------------------------------------------------------------------
# 4. Version compatibility guide
# ---------------------------------------------------------------------------
get_version_guide_note() {
  case "$1" in
    v3.8.5) echo 'Rebuilt subscription page; Xray survives a bad config (port-collision saves and enables are refused); large-fleet node sync; empty REALITY `min_client_ver` now means no minimum (xray-core v26.9.9).' ;;
    v3.8.0) echo 'TUIC v5 inbounds, AmneziaWG outbounds, a Discord notification bot (10 `discord*` settings), Happ client customization (`subHapp*`), `sub_profile_mode`, xray-core v26.9.9 with one-time template migrations, and hardened installs.' ;;
    v3.7.0) echo 'Native AmneziaWG inbounds, calendar-day client renewals with a per-client traffic reset cycle, inbound `disable_flow`, an IP-limit allowlist, and scoped API tokens.' ;;
    v3.6.0) echo 'Node `apiToken` becomes write-only ([3x-ui #5613](https://github.com/MHSanaei/3x-ui/pull/5613)); xray-core v26.7.28.' ;;
    v3.5.0) echo 'Host groups, MTProto multi-client support, Xray `env`, outbound `target_strategy`, and expanded balancer settings.' ;;
    v3.4.2) echo 'WireGuard multi-client support, `ldap_insecure_skip_verify`, and Xray Observatory/BurstObservatory.' ;;
    v3.4.1) echo 'Incy subscription routing injection settings.' ;;
    v3.4.0) echo 'SMTP notifications and expanded Telegram/subscription settings.' ;;
    v3.3.1) echo 'Live config apply; `panelProxy` replaced by the `panelOutbound` egress bridge.' ;;
    v3.3.0) echo '`subThemeDir`, `warpUpdateInterval`, MTProto, and the node-sync surface.' ;;
    *) echo '' ;;
  esac
}

check_version_guide() {
  local file="$ROOT/docs/guides/version-compatibility.md"
  local actual_versions
  actual_versions=$(awk '
    /<!-- sync-versions:begin -->/ { found=1; next }
    /<!-- sync-versions:end -->/ { exit }
    found && /^\| v/ { print $2 }
  ' FS='|' "$file" | sed 's/^ *//; s/ *$//' | sort)

  local expected_versions
  expected_versions=$(printf '%s\n' "${VERSIONS[@]}" | sort)

  local guide_drift=0
  if [ "$actual_versions" = "$expected_versions" ]; then
    echo "version-compatibility.md table: OK"
  else
    echo "version-compatibility.md table: DRIFT DETECTED"
    diff <(echo "$actual_versions") <(echo "$expected_versions") || true
    guide_drift=1
    DRIFT=1
  fi

  local documented_defaults
  documented_defaults=$(grep -oE 'THREEXUI_VERSION(=|:-)v[0-9]+\.[0-9]+\.[0-9]+' "$file" \
    | sed -E 's/THREEXUI_VERSION(=|:-)//' | sort -u)
  if [ "$documented_defaults" = "$DEFAULT_VERSION" ]; then
    echo "version-compatibility.md default: OK"
  else
    echo "version-compatibility.md default: DRIFT (expected only $DEFAULT_VERSION, got ${documented_defaults:-none})"
    guide_drift=1
    DRIFT=1
  fi

  if [ "$MODE" = "fix" ] && [ "$guide_drift" -eq 1 ]; then
    local table_block='| 3x-ui version | Status | Notes |'$'\n''| --- | --- | --- |'
    local v
    for v in "${VERSIONS_DESC[@]}"; do
      local note
      note=$(get_version_guide_note "$v")
      if [ -n "$note" ]; then
        table_block+=$'\n'"| $v | Tested | $note |"
      else
        table_block+=$'\n'"| $v | Tested | |"
      fi
    done

    local tmp
    tmp=$(mktemp)
    BLOCK="$table_block" awk '
      BEGIN { block = ENVIRON["BLOCK"] }
      /<!-- sync-versions:begin -->/ { print; print block; skip=1; next }
      /<!-- sync-versions:end -->/ { skip=0; print; next }
      !skip { print }
    ' "$file" > "$tmp" && mv "$tmp" "$file"

    sed -i.bak -E \
      -e "s/(THREEXUI_VERSION=)v[0-9]+\.[0-9]+\.[0-9]+/\\1$DEFAULT_VERSION/g" \
      -e "s/(THREEXUI_VERSION:-)v[0-9]+\.[0-9]+\.[0-9]+/\\1$DEFAULT_VERSION/g" \
      "$file"
    rm -f "$file.bak"
    echo "  -> Fixed version-compatibility.md"
  fi
}

# ---------------------------------------------------------------------------
# 5. docker-compose.yaml default version
# ---------------------------------------------------------------------------
check_docker_compose() {
  local file="$ROOT/docker-compose.yaml"
  local current
  current=$(sed -n 's/.*THREEXUI_VERSION:-\([^}]*\).*/\1/p' "$file")

  if [ "$current" = "$DEFAULT_VERSION" ]; then
    echo "docker-compose.yaml: OK"
  else
    echo "docker-compose.yaml: DRIFT (expected $DEFAULT_VERSION, got $current)"
    DRIFT=1

    if [ "$MODE" = "fix" ]; then
      sed -i.bak "s|THREEXUI_VERSION:-[^}]*|THREEXUI_VERSION:-$DEFAULT_VERSION|" "$file" && rm -f "$file.bak"
      echo "  -> Fixed docker-compose.yaml"
    fi
  fi
}

# ---------------------------------------------------------------------------
# 6. Taskfile.yml defaults
# ---------------------------------------------------------------------------
check_taskfile() {
  local file="$ROOT/Taskfile.yml"
  local current
  current=$(grep 'default "v' "$file" | head -1 | sed 's/.*default "\([^"]*\)".*/\1/')

  if [ "$current" = "$DEFAULT_VERSION" ]; then
    echo "Taskfile.yml: OK"
  else
    echo "Taskfile.yml: DRIFT (expected $DEFAULT_VERSION, got $current)"
    DRIFT=1

    if [ "$MODE" = "fix" ]; then
      sed -i.bak "s|default \"v[^\"]*\"|default \"$DEFAULT_VERSION\"|g" "$file" && rm -f "$file.bak"
      echo "  -> Fixed Taskfile.yml"
    fi
  fi
}

# ---------------------------------------------------------------------------
# 7. README support-policy prose (no hardcoded count or version globs)
# ---------------------------------------------------------------------------
check_support_prose() {
  local file="$1"
  local basename
  basename=$(basename "$file")

  # The prose paragraph appears before the compatibility table header.
  # It must NOT contain individual minor-line globs like "v3.1.x" — the table
  # is the single source of truth.  Match all locales by looking for any
  # table row containing "3x-ui" (the table header in every locale).
  local prose
  prose=$(awk '/^\| .*3x-ui/ {exit} /^$/ {next} {print}' "$file" \
    | grep -iE 'v[0-9]+\.[0-9]+\.x' || true)

  if [ -z "$prose" ]; then
    echo "$basename prose: OK"
  else
    echo "$basename prose: DRIFT — contains version globs before table:"
    echo "$prose"
    DRIFT=1
  fi
}

# ---------------------------------------------------------------------------
# Run all checks
# ---------------------------------------------------------------------------
echo "=== Checking version consistency ==="
echo ""
check_ci
check_flake
check_readme "$ROOT/README.md"
check_readme "$ROOT/README.ru_RU.md"
check_readme "$ROOT/README.es_ES.md"
check_readme "$ROOT/README.fa_IR.md"
check_readme "$ROOT/README.ar_EG.md"
check_readme "$ROOT/README.zh_CN.md"
check_readme "$ROOT/README.tr_TR.md"
check_version_guide
check_docker_compose
check_taskfile
echo ""
echo "=== Checking support-policy prose ==="
check_support_prose "$ROOT/README.md"
check_support_prose "$ROOT/README.ru_RU.md"
check_support_prose "$ROOT/README.es_ES.md"
check_support_prose "$ROOT/README.fa_IR.md"
check_support_prose "$ROOT/README.ar_EG.md"
check_support_prose "$ROOT/README.zh_CN.md"
check_support_prose "$ROOT/README.tr_TR.md"

echo ""
if [ "$DRIFT" -eq 1 ]; then
  if [ "$MODE" = "check" ]; then
    echo "FAIL: drift detected. Run 'scripts/sync-versions.sh fix' to update."
    exit 1
  else
    echo "FIXED: all surfaces updated from compat-versions.json"
  fi
else
  echo "PASS: all surfaces match compat-versions.json"
fi
