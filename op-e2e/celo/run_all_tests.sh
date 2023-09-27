#!/bin/bash
set -eo pipefail

SCRIPT_DIR=$(readlink -f "$(dirname "$0")")
TEST_GLOB=$1
cd "$SCRIPT_DIR" || exit 1
source "$SCRIPT_DIR/shared.sh"

## Start geth
cd "$SCRIPT_DIR/../.." || exit 1
trap 'cd "$SCRIPT_DIR/../.." && make devnet-down' EXIT  # kill bg job at exit
make devnet-up

# Wait for geth to be ready
for _ in {1..10}
do
	if cast block &> /dev/null
	then
		break
	fi
	sleep 0.2
done

## Run tests
echo Geth ready, start tests
failures=0
tests=0
cd "$SCRIPT_DIR" || exit 1
for f in test_*"$TEST_GLOB"*
do
	echo -e "\nRun $f"
	if "./$f"
	then
		tput setaf 2 || true
		echo "PASS $f"
	else
		tput setaf 1 || true
		echo "FAIL $f ❌"
		((failures++)) || true
	fi
	tput sgr0 || true
	((tests++)) || true
done

## Final summary
echo
if [[ $failures -eq 0 ]]
then
	tput setaf 2 || true
	echo All tests succeeded!
else
	tput setaf 1 || true
	echo $failures/$tests failed.
fi
tput sgr0 || true
exit $failures
