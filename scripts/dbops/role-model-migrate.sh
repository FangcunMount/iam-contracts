#!/usr/bin/env bash

set -euo pipefail

MODE="${IAM_ROLE_MODEL_MODE:-}"
CONFIRM="${IAM_ROLE_MODEL_CONFIRM:-}"
WRITES_STOPPED="${IAM_ROLE_MODEL_WRITES_STOPPED:-}"
BIN="${IAM_ROLE_MODEL_BIN:-}"
CATALOG="${IAM_ROLE_MODEL_CATALOG:-}"
REPORT_DIR="${IAM_ROLE_MODEL_REPORT_DIR:-/opt/backups/iam/role-model}"
TIMEOUT="${IAM_ROLE_MODEL_TIMEOUT:-10m}"
TIMESTAMP="$(date -u +%Y%m%d_%H%M%S)"

fail() {
  echo "role-model-migrate: result=failed reason=$1" >&2
  exit 1
}

require_value() {
  local name="$1"
  local value="$2"
  if [ -z "$value" ]; then
    fail "missing $name"
  fi
  case "$value" in
    *$'\n'*|*$'\r'*)
      fail "invalid $name"
      ;;
  esac
}

normalize_mysql_aliases() {
  if [ -z "${MYSQL_USER:-}" ] && [ -n "${MYSQL_USERNAME:-}" ]; then
    export MYSQL_USER="$MYSQL_USERNAME"
  fi
  if [ -z "${MYSQL_DATABASE:-}" ] && [ -n "${MYSQL_DBNAME:-}" ]; then
    export MYSQL_DATABASE="$MYSQL_DBNAME"
  fi
  if [ -z "${QS_MYSQL_USER:-}" ] && [ -n "${QS_MYSQL_USERNAME:-}" ]; then
    export QS_MYSQL_USER="$QS_MYSQL_USERNAME"
  fi
  if [ -z "${QS_MYSQL_DATABASE:-}" ] && [ -n "${QS_MYSQL_DBNAME:-}" ]; then
    export QS_MYSQL_DATABASE="$QS_MYSQL_DBNAME"
  fi
  if [ -z "${QS_MYSQL_HOST:-}" ] && [ -n "${MYSQL_HOST:-}" ]; then
    export QS_MYSQL_HOST="$MYSQL_HOST"
  fi
  if [ -z "${QS_MYSQL_PORT:-}" ]; then
    export QS_MYSQL_PORT="${MYSQL_PORT:-3306}"
  fi
  if [ -z "${QS_MYSQL_USERNAME:-}" ] && [ -n "${MYSQL_USERNAME:-}" ]; then
    export QS_MYSQL_USERNAME="$MYSQL_USERNAME"
    export QS_MYSQL_USER="${QS_MYSQL_USER:-$MYSQL_USERNAME}"
  fi
  if [ -z "${QS_MYSQL_PASSWORD:-}" ] && [ -n "${MYSQL_PASSWORD:-}" ]; then
    export QS_MYSQL_PASSWORD="$MYSQL_PASSWORD"
  fi
}

extract_fingerprint() {
  local path="$1"
  local fingerprint
  fingerprint="$(grep -oE '"fingerprint"[[:space:]]*:[[:space:]]*"[0-9a-f]{64}"' "$path" | head -n 1 | grep -oE '[0-9a-f]{64}' || true)"
  if ! [[ "$fingerprint" =~ ^[0-9a-f]{64}$ ]]; then
    fail "fingerprint missing from report"
  fi
  printf '%s' "$fingerprint"
}

extract_checksum() {
  local path="$1"
  local checksum
  checksum="$(grep -oE '"checksum"[[:space:]]*:[[:space:]]*"[0-9a-f]{64}"' "$path" | head -n 1 | grep -oE '[0-9a-f]{64}' || true)"
  if ! [[ "$checksum" =~ ^[0-9a-f]{64}$ ]]; then
    printf ''
    return 0
  fi
  printf '%s' "$checksum"
}

run_tool() {
  local op="$1"
  shift
  "$BIN" role-model-migrate "$op" --timeout="$TIMEOUT" "$@"
}

main() {
  umask 077

  case "$MODE" in
    preflight|cutover) ;;
    *) fail "unsupported mode" ;;
  esac

  require_value IAM_ROLE_MODEL_BIN "$BIN"
  require_value IAM_ROLE_MODEL_CATALOG "$CATALOG"
  require_value MYSQL_HOST "${MYSQL_HOST:-}"
  require_value MYSQL_USERNAME "${MYSQL_USERNAME:-}"
  require_value MYSQL_PASSWORD "${MYSQL_PASSWORD:-}"
  require_value MYSQL_DBNAME "${MYSQL_DBNAME:-}"
  require_value QS_MYSQL_DBNAME "${QS_MYSQL_DBNAME:-}"

  if [ ! -f "$BIN" ]; then
    fail "maintenance binary is missing"
  fi
  chmod 0700 -- "$BIN" || fail "maintenance binary is not executable"
  if [ ! -x "$BIN" ]; then
    fail "maintenance binary is not executable"
  fi
  if [ ! -f "$CATALOG" ]; then
    fail "event catalog is missing"
  fi
  chmod 0600 -- "$CATALOG" || true
  if [[ "$REPORT_DIR" != /* ]] || [ -L "$REPORT_DIR" ]; then
    fail "report directory is invalid"
  fi

  if [ "$MODE" = "cutover" ]; then
    if [ "$CONFIRM" != "ROLE_MODEL_MIGRATE" ]; then
      fail "confirm phrase mismatch"
    fi
    case "$WRITES_STOPPED" in
      true|1|yes) ;;
      *) fail "writes must be stopped" ;;
    esac
  fi

  normalize_mysql_aliases
  require_value MYSQL_USER "${MYSQL_USER:-}"
  require_value MYSQL_DATABASE "${MYSQL_DATABASE:-}"
  require_value QS_MYSQL_HOST "${QS_MYSQL_HOST:-}"
  require_value QS_MYSQL_USER "${QS_MYSQL_USER:-}"
  require_value QS_MYSQL_PASSWORD "${QS_MYSQL_PASSWORD:-}"
  require_value QS_MYSQL_DATABASE "${QS_MYSQL_DATABASE:-}"

  mkdir -p -- "$REPORT_DIR"
  chmod 0700 -- "$REPORT_DIR"

  local preflight_path apply_path verify_path archive_path fingerprint checksum
  preflight_path="$REPORT_DIR/preflight_${TIMESTAMP}.json"
  apply_path="$REPORT_DIR/apply_${TIMESTAMP}.json"
  verify_path="$REPORT_DIR/verify_${TIMESTAMP}.json"
  archive_path="$REPORT_DIR/archive_${TIMESTAMP}.json"

  if ! run_tool preflight >"$preflight_path"; then
    fail "preflight"
  fi
  chmod 0600 -- "$preflight_path"
  fingerprint="$(extract_fingerprint "$preflight_path")"
  echo "role-model-migrate: mode=$MODE step=preflight result=success fingerprint=$fingerprint report=$(basename "$preflight_path")"

  if [ "$MODE" = "preflight" ]; then
    echo "role-model-migrate: result=success next=cutover_after_writes_stopped"
    exit 0
  fi

  if ! run_tool apply \
    --writes-stopped \
    --fingerprint="$fingerprint" \
    --event-catalog="$CATALOG" >"$apply_path"; then
    fail "apply"
  fi
  chmod 0600 -- "$apply_path"
  echo "role-model-migrate: mode=$MODE step=apply result=success fingerprint=$fingerprint report=$(basename "$apply_path")"

  if ! run_tool verify >"$verify_path"; then
    fail "verify"
  fi
  chmod 0600 -- "$verify_path"
  echo "role-model-migrate: mode=$MODE step=verify result=success fingerprint=$fingerprint report=$(basename "$verify_path")"

  if ! run_tool archive-inheritance \
    --writes-stopped \
    --fingerprint="$fingerprint" >"$archive_path"; then
    fail "archive-inheritance"
  fi
  chmod 0600 -- "$archive_path"
  checksum="$(extract_checksum "$archive_path")"
  if [ -n "$checksum" ]; then
    echo "role-model-migrate: mode=$MODE step=archive result=success fingerprint=$fingerprint checksum=$checksum report=$(basename "$archive_path")"
  else
    echo "role-model-migrate: mode=$MODE step=archive result=success fingerprint=$fingerprint report=$(basename "$archive_path")"
  fi
  echo "role-model-migrate: result=success next=deploy_migration_34"
}

main
