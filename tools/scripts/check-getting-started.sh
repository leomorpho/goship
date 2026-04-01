#!/usr/bin/env bash

set -euo pipefail

echo "== getting started proof =="

WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

echo "-- build ship from fresh clone path"
go build -o "$WORKDIR/ship" ./tools/cli/ship/cmd/ship
"$WORKDIR/ship" --help >/dev/null

echo "-- generate starter app"
cd "$WORKDIR"
"$WORKDIR/ship" new myapp --module example.com/myapp --no-i18n >/dev/null
cd myapp

echo "-- onboarding loop"
"$WORKDIR/ship" db:migrate >/dev/null
"$WORKDIR/ship" test >/dev/null
"$WORKDIR/ship" verify --profile fast >/dev/null

echo "getting started proof passed"
