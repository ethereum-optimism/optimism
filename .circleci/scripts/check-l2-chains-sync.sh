#!/usr/bin/env bash
# Checks that the chain names in the scheduled-weekly-tests matrix in
# .circleci/continue/main.yml match the keys in ./circleci/l2-rpcs.json.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
L2_RPCS_FILE=".circleci/l2-rpcs.json"
MAIN_YML_FILE=".circleci/continue/main.yml"
L2_RPCS_PATH="$REPO_ROOT/$L2_RPCS_FILE"
MAIN_YML_PATH="$REPO_ROOT/$MAIN_YML_FILE"

json_chains=$(jq -r 'keys | sort | .[]' "$L2_RPCS_PATH")

# Extract the fork_op_chain list from the scheduled-weekly-tests matrix block.
# Relies on the block structure: the list appears after `fork_op_chain:` and
# ends at the next non-list line (e.g. `test_profile:`).
yaml_chains=$(awk '
  /scheduled-weekly-tests/    { in_workflow=1 }
  in_workflow && /fork_op_chain:/ { in_list=1; next }
  in_workflow && in_list && /^[[:space:]]*- / {
    sub(/^[[:space:]]*- /, ""); print; next
  }
  in_workflow && in_list      { exit }
' "$MAIN_YML_PATH" | sort)

if [ "$json_chains" = "$yaml_chains" ]; then
  echo "OK: $L2_RPCS_FILE and scheduled-weekly-tests matrix are in sync."
else
  echo "ERROR: $L2_RPCS_FILE and the scheduled-weekly-tests matrix are out of sync."
  echo ""
  echo "  In l2-rpcs.json only:"
  comm -23 <(echo "$json_chains") <(echo "$yaml_chains") | sed 's/^/    /'
  echo "  In matrix only:"
  comm -13 <(echo "$json_chains") <(echo "$yaml_chains") | sed 's/^/    /'
  echo ""
  echo "Update $L2_RPCS_FILE or the matrix in $MAIN_YML_FILE to match."
  exit 1
fi
