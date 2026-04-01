#!/usr/bin/env bash

set -euo pipefail

echo "== module matrix =="
for module in jobs notifications paidsubscriptions; do
  echo "-- modules/${module}"
  (
    cd "modules/${module}"
    go test . -count=1
  )
done
echo "module matrix passed"
