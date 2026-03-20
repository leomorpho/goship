#!/usr/bin/env bash

set -euo pipefail

resolve_compose() {
  if [[ -n "${1:-}" ]]; then
    # shellcheck disable=SC2206
    COMPOSE_CMD=($1)
    return 0
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD=(docker-compose)
    return 0
  fi

  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD=(docker compose)
    return 0
  fi

  echo "No docker compose command found (docker-compose or docker compose)." >&2
  exit 1
}

resolve_compose "${1:-}"

COMPOSE_FILE="infra/docker/docker-compose.yml"

"${COMPOSE_CMD[@]}" -f "${COMPOSE_FILE}" up -d cache

mailpit_err_file="$(mktemp)"
if ! "${COMPOSE_CMD[@]}" -f "${COMPOSE_FILE}" up -d mailpit >/dev/null 2>"${mailpit_err_file}"; then
  if grep -Eiq "1025.*already allocated|Bind for .*:1025 failed|port is already allocated" "${mailpit_err_file}"; then
    echo "Warning: Mailpit SMTP port 1025 is already in use; continuing without starting goship_mailpit."
    echo "Set HOST_MAILPIT_SMTP_PORT to another port if you want goship_mailpit managed by this project."
  else
    echo "Failed to start mailpit:" >&2
    cat "${mailpit_err_file}" >&2
    rm -f "${mailpit_err_file}"
    exit 1
  fi
fi

rm -f "${mailpit_err_file}"
sleep 2
