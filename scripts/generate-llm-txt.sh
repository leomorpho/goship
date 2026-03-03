#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_file="${repo_root}/LLM.txt"
tmp_file="${out_file}.tmp"

# Build a deterministic ordered list of documentation sources.
declare -a files=()

if [[ -f "${repo_root}/README.md" ]]; then
  files+=("README.md")
fi

if [[ -f "${repo_root}/docs/00-index.md" ]]; then
  files+=("docs/00-index.md")
fi

while IFS= read -r f; do
  # 00-index is already included first when present.
  if [[ "$f" == "docs/00-index.md" ]]; then
    continue
  fi
  files+=("$f")
done < <(cd "${repo_root}" && find docs -type f -name '*.md' | sort)

{
  echo "# LLM.txt"
  echo
  echo "Generated file. Do not edit manually."
  echo "Source of truth is the markdown docs in this repository."
  echo

  for rel in "${files[@]}"; do
    abs="${repo_root}/${rel}"
    if [[ ! -f "$abs" ]]; then
      continue
    fi

    echo "---"
    echo "FILE: ${rel}"
    echo "---"
    cat "$abs"
    echo
  done
} > "${tmp_file}"

if [[ -f "${out_file}" ]] && cmp -s "${tmp_file}" "${out_file}"; then
  rm -f "${tmp_file}"
  exit 0
fi

mv "${tmp_file}" "${out_file}"
