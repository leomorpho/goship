#!/usr/bin/env bash

set -euo pipefail

root="$(pwd)"
queries_dir="$root/db/queries"
config_file="$root/db/bobgen.yaml"
gen_dir="$root/db/gen"

file_mtime() {
  local file="$1"
  if stat -c '%Y' "$file" >/dev/null 2>&1; then
    stat -c '%Y' "$file"
    return 0
  fi
  if stat -f '%m' "$file" >/dev/null 2>&1; then
    stat -f '%m' "$file"
    return 0
  fi
  echo "unable to determine file mtime for: $file" >&2
  return 1
}

if [ ! -d "$queries_dir" ] || [ ! -f "$config_file" ]; then
  exit 0
fi

has_sql=0
while IFS= read -r -d '' _f; do
  has_sql=1
  break
done < <(find "$queries_dir" -type f -name '*.sql' -print0)

if [ "$has_sql" -eq 0 ]; then
  exit 0
fi

if [ ! -d "$gen_dir" ]; then
  echo "bob generated code is missing. run: ship db:generate" >&2
  exit 1
fi

# GoShip is currently in a hybrid DB-access state:
# - some query files are consumed directly via db/queries.Get(...)
# - some query families still have maintained wrappers under db/gen/<name>.go
#
# Until Bob generation is the only path, drift should only be enforced for
# query files that have a checked-in wrapper sibling in db/gen.
latest_input_ts=0
tracked_inputs=0
while IFS= read -r -d '' f; do
  base="$(basename "$f" .sql)"
  if [ ! -f "$gen_dir/$base.go" ]; then
    continue
  fi
  tracked_inputs=1
  ts="$(file_mtime "$f")"
  if [ "$ts" -gt "$latest_input_ts" ]; then
    latest_input_ts="$ts"
  fi
done < <(find "$queries_dir" -type f -name '*.sql' -print0)

if [ "$tracked_inputs" -eq 0 ]; then
  exit 0
fi

latest_generated_ts=0
while IFS= read -r -d '' f; do
  ts="$(file_mtime "$f")"
  if [ "$ts" -gt "$latest_generated_ts" ]; then
    latest_generated_ts="$ts"
  fi
done < <(find "$gen_dir" -type f -name '*.go' -print0)

if [ "$latest_generated_ts" -lt "$latest_input_ts" ]; then
  echo "bob generated code appears stale. run: ship db:generate" >&2
  exit 1
fi
