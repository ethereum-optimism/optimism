#!/usr/bin/env bash
# Splits the test contracts in TEST_FILES across the CircleCI nodes of a job, and prints this
# node's share as tab-separated "<file> <contract>" lines. Prints nothing when this node gets no
# contracts.
#
# For jobs that run too few test files to split by file: heavy fuzz runs only the changed files,
# and one file can hold most of its fuzz tests.
#
# Forge lists the contracts, so the split covers exactly the contracts `forge test` would run.
# Fuzz and invariant tests weigh 100 times a unit test, since a campaign can run until its timeout.
#
# Each file is split on its own, so a file's split depends only on that file. In the merge
# queue each node diffs against develop at its own checkout, and the entry ahead can merge in
# between. The nodes then list different files, but still split every file they share the same
# way, so no contract of a shared file is lost.
#
# --match-contract matches names only. A name in two listed files can run on both nodes, never
# on neither.
#
# Usage: TEST_FILES="<files>" split-test-contracts.sh
set -euo pipefail

TEST_FILES="${TEST_FILES:?TEST_FILES must list the test files to split}"
NODE_INDEX="${CIRCLE_NODE_INDEX:-0}"
NODE_TOTAL="${CIRCLE_NODE_TOTAL:-1}"
TAB=$(printf '\t')

LIST=$(forge test --list --json --match-path "./{$(echo "$TEST_FILES" | paste -sd, -)}")

# Forge lists abstract contracts too, but only runs their tests in the contracts that inherit
# them. One "<file>\t<contract>" line for each, to leave out of the split. A name the file also
# declares as a plain contract is kept, since one of the two is in a comment. At worst that
# leaves a node with no tests to run, which fails loudly, and never skips a test.
declared() { # <file> <prefix regex>
  { grep -oE "^[[:space:]]*$2contract[[:space:]]+[A-Za-z0-9_\$]+" "$1" || true; } \
    | awk '{ print $NF }' | LC_ALL=C sort -u
}
ABSTRACT=$(echo "$TEST_FILES" | while IFS= read -r f; do
  LC_ALL=C comm -23 <(declared "$f" 'abstract[[:space:]]+') <(declared "$f" '') \
    | F="$f" awk '{ print ENVIRON["F"] "\t" $0 }'
done)

# Test functions declared with parameters are fuzz tests. forge fmt can put the parameters on
# the lines after the "(". Search the whole test tree, not just TEST_FILES, so that a contract
# weighs the same on every node, inherited tests included.
FUZZ_TESTS=$(find test -name '*.sol' | LC_ALL=C sort | while IFS= read -r f; do
  grep -ohE 'function (test|invariant)[A-Za-z0-9_$]*\(([^)[:space:]]|$)' "$f" || true
done | sed -E 's/^function ([A-Za-z0-9_$]+).*/\1/' | LC_ALL=C sort -u)

# One "<weight>\t<contract>\t<file>" line per listed contract.
WEIGHTS=$(echo "$LIST" | jq -r --arg fuzz "$FUZZ_TESTS" --arg abstract "$ABSTRACT" '
  ($fuzz | split("\n") | map(select(. != "") | {(.): true}) | add // {}) as $isfuzz
  | ($abstract | split("\n") | map(select(. != "") | {(.): true}) | add // {}) as $isabstract
  | to_entries[] | .key as $file | .value | to_entries[]
  | select($isabstract[$file + "\t" + .key] | not)
  | "\([.value[] | if $isfuzz[.] or startswith("invariant") or startswith("statefulFuzz") then 100 else 1 end]
      | add)\t\(.key)\t\($file)"')

if [ -z "$WEIGHTS" ]; then
  echo "error: forge listed no test contracts in TEST_FILES" >&2
  exit 1
fi

# Each file starts at its own node, from a checksum of its path, so that files with a single
# contract spread out. One "<file>\t<checksum>" line each.
OFFSETS=$(echo "$TEST_FILES" | while IFS= read -r f; do
  printf '%s\t%s\n' "$f" "$(printf '%s' "$f" | cksum | cut -d' ' -f1)"
done)

# Greedy split within each file: heaviest contract first, each to the least loaded node.
# LC_ALL=C keeps the order the same on every node.
echo "$WEIGHTS" | LC_ALL=C sort -t "$TAB" -k3,3 -k1,1nr -k2,2 \
  | LC_ALL=C awk -F '\t' -v node="$NODE_INDEX" -v total="$NODE_TOTAL" '
  NR == FNR { offset[$1] = $2 % total; next }
  $3 != file {
    file = $3
    for (i = 0; i < total; i++) load[i] = 0
  }
  {
    best = -1
    for (j = 0; j < total; j++) {
      i = (offset[file] + j) % total
      if (best < 0 || load[i] < load[best]) best = i
    }
    load[best] += $1
    if (best == node) print $3 "\t" $2
  }' <(echo "$OFFSETS") -
