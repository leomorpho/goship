#!/usr/bin/env bash

set -euo pipefail

violations="$(rg -n '^\s*"github\.com/leomorpho/goship/db/ent' app/web/controllers -g '*.go' || true)"
if [[ -n "${violations}" ]]; then
  echo "controller db boundary check failed: controllers must not import db/ent directly"
  echo "${violations}"
  exit 1
fi

echo "controller db import boundary check passed."
