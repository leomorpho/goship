#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_DIR="$ROOT/app/web/controllers"

if [ ! -d "$TARGET_DIR" ]; then
  echo "controllers directory not found: $TARGET_DIR" >&2
  exit 1
fi

if rg -n "QueryProfile\(" "$TARGET_DIR" -g '*.go' >/tmp/goship-controller-queryprofile.out; then
  echo "controller auth boundary violation: direct QueryProfile() usage is not allowed in app/web/controllers." >&2
  cat /tmp/goship-controller-queryprofile.out >&2
  echo "Use middleware identity context (auth_profile_id/auth_user_id) + service/ORM lookup by id instead." >&2
  rm -f /tmp/goship-controller-queryprofile.out
  exit 1
fi

rm -f /tmp/goship-controller-queryprofile.out
echo "controller auth boundary check passed."

