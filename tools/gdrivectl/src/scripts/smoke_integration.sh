#!/usr/bin/env bash
set -euo pipefail

# Optional local integration smoke harness.
# Requires live Google API access and valid document/file ids.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

GCLOUD_BIN="${GDRIVECTL_GCLOUD_BIN:-gcloud}"
TIMEOUT="${GDRIVECTL_TIMEOUT:-15s}"
SEARCH_QUERY="${GDRIVECTL_SEARCH_QUERY:-name contains 'RFC'}"
FILE_ID="${GDRIVECTL_FILE_ID:-}"
DOC_ID="${GDRIVECTL_DOC_ID:-}"
EXPORT_MIME="${GDRIVECTL_EXPORT_MIME:-text/plain}"
EXPORT_OUT="${GDRIVECTL_EXPORT_OUT:-/tmp/gdrivectl-export.txt}"

if [[ -z "${FILE_ID}" || -z "${DOC_ID}" ]]; then
  cat <<EOF
Missing required env vars:
  GDRIVECTL_FILE_ID
  GDRIVECTL_DOC_ID

Example:
  GDRIVECTL_FILE_ID=<file_id> GDRIVECTL_DOC_ID=<doc_id> scripts/smoke_integration.sh
EOF
  exit 2
fi

run_expect_ok() {
  local name="$1"
  shift
  echo "[smoke] ${name}"
  set +e
  "$@"
  local rc=$?
  set -e
  if [[ $rc -ne 0 ]]; then
    echo "[smoke] FAIL (${name}) exit=${rc}"
    exit "${rc}"
  fi
  echo "[smoke] OK (${name})"
}

echo "[smoke] root=${ROOT_DIR}"
echo "[smoke] using gcloud=${GCLOUD_BIN}"
echo "[smoke] timeout=${TIMEOUT}"

run_expect_ok "build" go -C "${ROOT_DIR}" build ./...
run_expect_ok "help" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --help
run_expect_ok "doctor" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --gcloud-bin "${GCLOUD_BIN}" --timeout "${TIMEOUT}" doctor
run_expect_ok "doctor-json" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --gcloud-bin "${GCLOUD_BIN}" --timeout "${TIMEOUT}" --json doctor
run_expect_ok "search" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --gcloud-bin "${GCLOUD_BIN}" --timeout "${TIMEOUT}" search --query "${SEARCH_QUERY}" --page-size 5 --json
run_expect_ok "file-meta" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --gcloud-bin "${GCLOUD_BIN}" --timeout "${TIMEOUT}" file-meta --id "${FILE_ID}" --json
run_expect_ok "doc-tabs" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --gcloud-bin "${GCLOUD_BIN}" --timeout "${TIMEOUT}" doc-tabs --id "${DOC_ID}" --json
run_expect_ok "doc-export" go -C "${ROOT_DIR}" run ./cmd/gdrivectl --gcloud-bin "${GCLOUD_BIN}" --timeout "${TIMEOUT}" doc-export --id "${DOC_ID}" --mime "${EXPORT_MIME}" --out "${EXPORT_OUT}"

if [[ ! -s "${EXPORT_OUT}" ]]; then
  echo "[smoke] FAIL (doc-export) output is empty: ${EXPORT_OUT}"
  exit 5
fi

echo "[smoke] all checks passed"
