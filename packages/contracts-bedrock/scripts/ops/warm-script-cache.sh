#!/bin/sh
# Warms forge's script cache after a build, and checks that each script op-deployer can run then
# compiles nothing: op-deployer never compiles. FORGE names the forge binary, as a command name or
# an absolute path (default: forge). `just forge-build` and op-deployer's
# build-embedded-contracts.sh run it.
set -eu

forge="${FORGE:-forge}"
cd "$(dirname "$0")/../.."

# `forge build` compiles only the src/ graph, and `forge script` compiles the scripts/ graph. One
# `forge script` run compiles the script dependency graph into the cache, so later runs of other
# scripts typically compile nothing. The run fails because the function does not exist, but only
# after compiling: compiling happens before argument validation, and --skip-simulation keeps it from
# running anything. --skip keeps tests out of the graph.
out=$("$forge" script scripts/deploy/Deploy.s.sol --deny never --skip "/**/test/**" \
  --sig "idonotexist()" --skip-simulation 2>&1 || true)
case "$out" in
  *"is not implemented in your script"*) ;;
  *)
    printf '%s\n' "$out" >&2
    echo "the forge script cache warm-up did not compile" >&2
    exit 1
    ;;
esac

# `forge script` checks the cache of src/ and of its target's imports only, not of the other scripts,
# so check each script op-deployer can run: the ones with runWithBytes. No --skip, as op-deployer runs
# them: a script can import from test/.
scripts=$(find scripts -name '*.s.sol' -exec grep -l 'function runWithBytes' {} + || true)
if [ -z "$scripts" ]; then
  echo "found no op-deployer scripts" >&2
  exit 1
fi
for script in $scripts; do
  out=$("$forge" script "$script" --deny never --sig "idonotexist()" --skip-simulation 2>&1 || true)
  case "$out" in
    *"No files changed, compilation skipped"*) ;;
    *)
      printf '%s\n' "$out" >&2
      echo "the forge script cache of $script is not warm" >&2
      exit 1
      ;;
  esac
done
