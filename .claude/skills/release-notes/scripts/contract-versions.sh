#!/usr/bin/env bash
# Print the contracts whose version changed between two op-contracts refs, as a markdown
# table for the `## Contract versions` section of an op-contracts note.
#
# Each versioned contract under src/ carries exactly one `@custom:semver` annotation, which
# is read straight from git so neither ref needs checking out.
#
# Usage: contract-versions.sh <from-ref> <to-ref>
#        e.g. contract-versions.sh op-contracts/v7.0.0 op-contracts/v8.0.0
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: contract-versions.sh <from-ref> <to-ref>" >&2
  exit 2
fi
from=$1
to=$2

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

# Emits `<contract>\t<version>`, sorted by contract name.
# BSD sed does not expand \t in a replacement, so the tab is a literal.
tab=$'\t'
versions() {
  git grep -e '@custom:semver' "$1" -- 'packages/contracts-bedrock/src/*.sol' \
    | sed -E "s|^$1:||; s|^.*/([^/]+)\.sol:.*@custom:semver[[:space:]]+([^[:space:]]+).*$|\1$tab\2|" \
    | sort -t$'\t' -k1,1
}

versions "$from" > "$workdir/from"
versions "$to" > "$workdir/to"

printf '| Contract | %s | %s |\n| --- | --- | --- |\n' "${from##*/}" "${to##*/}"
join -t$'\t' -a1 -a2 -e '—' -o '0,1.2,2.2' "$workdir/from" "$workdir/to" \
  | awk -F'\t' '$2 != $3 { printf "| `%s` | %s | %s |\n", $1, $2, $3 }'
