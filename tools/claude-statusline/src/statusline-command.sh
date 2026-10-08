#!/usr/bin/env bash
# Claude Code status line script.
# Reads Claude's stdin JSON, enriches it with ccusage data, and prints a compact
# single-line status summary.

set -u

CCUSAGE_PACKAGE_DEFAULT="ccusage@16.2.3"
DAILY_CACHE_TTL_DEFAULT=60
MONTHLY_CACHE_TTL_DEFAULT=900

DEBUG_ENABLED="${CLAUDE_STATUSLINE_DEBUG:-0}"
CCUSAGE_PACKAGE="${CLAUDE_STATUSLINE_CCUSAGE_PACKAGE:-$CCUSAGE_PACKAGE_DEFAULT}"
DAILY_CACHE_TTL_SECONDS="${CLAUDE_STATUSLINE_DAILY_CACHE_TTL_SECONDS:-$DAILY_CACHE_TTL_DEFAULT}"
MONTHLY_CACHE_TTL_SECONDS="${CLAUDE_STATUSLINE_MONTHLY_CACHE_TTL_SECONDS:-$MONTHLY_CACHE_TTL_DEFAULT}"
CACHE_DIR="${CLAUDE_STATUSLINE_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/claude-statusline}"
ALLOW_UNPINNED_CCUSAGE="${CLAUDE_STATUSLINE_ALLOW_UNPINNED_CCUSAGE:-0}"
CCUSAGE_EXPECTED_VERSION="${CLAUDE_STATUSLINE_CCUSAGE_VERSION:-}"

if [ -z "$CCUSAGE_EXPECTED_VERSION" ]; then
  case "$CCUSAGE_PACKAGE" in
    ccusage@*)
      CCUSAGE_EXPECTED_VERSION="${CCUSAGE_PACKAGE#ccusage@}"
      ;;
  esac
fi

input=$(cat)

log_debug() {
  if [ "$DEBUG_ENABLED" != "0" ]; then
    printf 'claude-statusline: %s\n' "$*" >&2
  fi
}

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

get_mtime_epoch() {
  local file="$1"

  if stat -f %m "$file" >/dev/null 2>&1; then
    stat -f %m "$file"
    return 0
  fi

  stat -c %Y "$file" 2>/dev/null
}

format_k() {
  local value="$1"
  local decimals="$2"
  awk -v value="$value" -v decimals="$decimals" \
    'BEGIN { printf "%.*fk", decimals, value / 1000 }'
}

get_ccusage_version() {
  local version_output=""

  version_output="$(ccusage --version 2>/dev/null)" || return 1
  version_output="${version_output%%$'\n'*}"
  version_output="$(trim "$version_output")"
  version_output="${version_output#ccusage }"
  version_output="${version_output#v}"
  [ -n "$version_output" ] || return 1

  printf '%s' "$version_output"
}

resolve_ccusage() {
  local local_ccusage_version=""

  if command -v ccusage >/dev/null 2>&1; then
    local_ccusage_version="$(get_ccusage_version || true)"

    if [ -n "$CCUSAGE_EXPECTED_VERSION" ] && \
      [ "$local_ccusage_version" = "$CCUSAGE_EXPECTED_VERSION" ]; then
      CCUSAGE_CMD=(ccusage)
      log_debug "using ccusage $local_ccusage_version from PATH"
      return 0
    fi

    if [ "$ALLOW_UNPINNED_CCUSAGE" = "1" ]; then
      CCUSAGE_CMD=(ccusage)
      log_debug "using unpinned ccusage from PATH (${local_ccusage_version:-unknown version})"
      return 0
    fi

    if [ -n "$CCUSAGE_EXPECTED_VERSION" ]; then
      log_debug "skipping PATH ccusage (${local_ccusage_version:-unknown version}); expected $CCUSAGE_EXPECTED_VERSION"
    else
      log_debug "skipping PATH ccusage (${local_ccusage_version:-unknown version}); no expected version configured"
    fi
  fi

  if command -v npx >/dev/null 2>&1; then
    CCUSAGE_CMD=(npx -y "$CCUSAGE_PACKAGE")
    log_debug "using $CCUSAGE_PACKAGE via npx"
    return 0
  fi

  return 1
}

run_ccusage() {
  "${CCUSAGE_CMD[@]}" "$@" 2>/dev/null
}

read_cache_value() {
  local cache_file="$1"
  local ttl_seconds="$2"
  local cache_mtime=""
  local now_epoch=""
  local age=0

  [ -f "$cache_file" ] || return 1

  cache_mtime="$(get_mtime_epoch "$cache_file")"
  [ -n "$cache_mtime" ] || return 1

  now_epoch="$(date +%s)"
  age=$((now_epoch - cache_mtime))
  [ "$age" -lt "$ttl_seconds" ] || return 1

  cat "$cache_file"
}

write_cache_value() {
  local cache_file="$1"
  local cached_value="$2"
  local tmp_file=""

  mkdir -p "$CACHE_DIR" 2>/dev/null || return 1
  tmp_file="$(mktemp "$CACHE_DIR/monthly-cost.XXXXXX")" || return 1

  printf '%s\n' "$cached_value" > "$tmp_file" || return 1
  mv "$tmp_file" "$cache_file" || return 1
}

load_total_cost_display() {
  local cache_prefix="$1"
  local period_key="$2"
  local ttl_seconds="$3"
  shift 3

  local cache_file="$CACHE_DIR/$cache_prefix-$period_key.txt"
  local cached_value=""
  local cost_json=""
  local total_cost=""
  local cost_display=""

  if cached_value="$(read_cache_value "$cache_file" "$ttl_seconds")"; then
    log_debug "using cached $cache_prefix total from $cache_file"
    printf '%s' "$cached_value"
    return 0
  fi

  cost_json="$(run_ccusage "$@")"
  [ -n "$cost_json" ] || return 1

  total_cost="$(printf '%s' "$cost_json" | jq -r '.totals.totalCost // empty' 2>/dev/null)"
  [ -n "$total_cost" ] || return 1
  [ "$total_cost" != "null" ] || return 1

  cost_display="$(printf '$%.2f' "$total_cost" 2>/dev/null || printf '$%s' "$total_cost")"
  write_cache_value "$cache_file" "$cost_display" >/dev/null 2>&1 || \
    log_debug "unable to update $cache_prefix cache at $cache_file"

  printf '%s' "$cost_display"
}

load_daily_display() {
  local day_start="$1"

  load_total_cost_display "daily-cost" "$day_start" "$DAILY_CACHE_TTL_SECONDS" \
    daily --since "$day_start" --until "$day_start" --json
}

load_monthly_display() {
  local month_start="$1"

  load_total_cost_display "monthly-cost" "$month_start" "$MONTHLY_CACHE_TTL_SECONDS" \
    --since "$month_start" --json
}

validate_ttl() {
  local variable_name="$1"
  local variable_value="$2"
  local default_value="$3"

  case "$variable_value" in
    ''|*[!0-9]*)
      log_debug "invalid $variable_name '$variable_value'; using default"
      printf '%s' "$default_value"
      ;;
    *)
      printf '%s' "$variable_value"
      ;;
  esac
}

DAILY_CACHE_TTL_SECONDS="$(
  validate_ttl "daily cache TTL" "$DAILY_CACHE_TTL_SECONDS" "$DAILY_CACHE_TTL_DEFAULT"
)"
MONTHLY_CACHE_TTL_SECONDS="$(
  validate_ttl "monthly cache TTL" "$MONTHLY_CACHE_TTL_SECONDS" "$MONTHLY_CACHE_TTL_DEFAULT"
)"

parts=()
model_seg=""
cost_seg=""
burn_seg=""
block_time=""
daily_display=""
monthly_display=""
remaining_int=""
model_display_name=""
total_input=0
total_output=0
context_window_size=0
remaining=""
have_context_window=0
have_jq=0

if command -v jq >/dev/null 2>&1; then
  context_fields="$(printf '%s' "$input" | jq -r '
    [
      .model.display_name // "",
      .context_window.total_input_tokens // 0,
      .context_window.total_output_tokens // 0,
      .context_window.context_window_size // 0,
      (.context_window.remaining_percentage // "")
    ] | @tsv
  ' 2>/dev/null)"

  if [ -n "$context_fields" ]; then
    have_jq=1
    IFS=$'\t' read -r model_display_name total_input total_output context_window_size remaining <<< "$context_fields"
    if [ "${context_window_size:-0}" -gt 0 ]; then
      have_context_window=1
    fi
    if [ -n "$remaining" ]; then
      remaining_int="$(printf '%.0f' "$remaining" 2>/dev/null || printf '%s' "$remaining")"
    fi
  fi
else
  log_debug "jq not found; context and cost totals will be omitted"
fi

if resolve_ccusage; then
  ccusage_out="$(printf '%s' "$input" | run_ccusage statusline --visual-burn-rate emoji --cost-source cc)"

  if [[ "$ccusage_out" == *"|"*"|"* ]] && [[ "$ccusage_out" != ❌* ]]; then
    IFS='|' read -r raw_model_seg raw_cost_seg raw_burn_seg _ <<< "$ccusage_out"
    model_seg="$(trim "${raw_model_seg:-}")"
    cost_seg="$(trim "${raw_cost_seg:-}")"
    burn_seg="$(trim "${raw_burn_seg:-}")"

    if [[ "$cost_seg" =~ (\([0-9]+h\ [0-9]+m\ left\)|\([0-9]+m\ left\)) ]]; then
      block_time="${BASH_REMATCH[1]}"
    fi
  elif [ -n "$ccusage_out" ]; then
    log_debug "ignoring invalid ccusage statusline output: $ccusage_out"
  else
    log_debug "ccusage statusline returned no output"
  fi

  if [ "$have_jq" -eq 1 ]; then
    day_start="$(date +%Y%m%d)"
    month_start="$(date +%Y%m01)"
    daily_display="$(load_daily_display "$day_start")"
    monthly_display="$(load_monthly_display "$month_start")"
  fi
else
  log_debug "ccusage and npx are both unavailable"
fi

if [ -z "$model_seg" ] && [ -n "$model_display_name" ]; then
  model_seg="🤖 $model_display_name"
fi

[ -n "$model_seg" ] && parts+=("$model_seg")

if [ -n "$daily_display" ] && [ -n "$monthly_display" ]; then
  parts+=("💰 $daily_display today / $monthly_display mo")
elif [ -n "$daily_display" ]; then
  parts+=("💰 $daily_display today")
elif [ -n "$monthly_display" ]; then
  parts+=("💰 $monthly_display mo")
fi

if [ -n "$burn_seg" ]; then
  if [ -n "$block_time" ]; then
    parts+=("$burn_seg $block_time")
  else
    parts+=("$burn_seg")
  fi
fi

if [ "$have_context_window" -eq 1 ]; then
  used_total=$((total_input + total_output))
  used_k="$(format_k "$used_total" 1)"
  window_k="$(format_k "$context_window_size" 0)"

  if [ -n "$remaining_int" ]; then
    parts+=("🧠 $used_k / $window_k ($remaining_int% left)")
  else
    parts+=("🧠 $used_k / $window_k")
  fi
fi

if [ "${#parts[@]}" -eq 0 ]; then
  printf '⚠️ claude-statusline unavailable\n'
  exit 0
fi

output=""
for part in "${parts[@]}"; do
  if [ -z "$output" ]; then
    output="$part"
  else
    output="$output | $part"
  fi
done

printf '%s\n' "$output"
