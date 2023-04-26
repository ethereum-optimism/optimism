#!/bin/bash
set -euo pipefail
SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
cd "${SCRIPTS_DIR}"

OUTPUT_ORACLE="0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0"
L1_API=${1:?Must specify L1 endpoint as first arg}
L2_API=${2:?Must specify L2 endpoint as second arg}
DATADIR=${3:-oracledata}
L1_HEAD_NUM=${4:-latest}

# Get the current L1 head block hash
L1_HEAD=$(cast block --rpc-url "$L1_API" "$L1_HEAD_NUM" hash)
echo "L1 head: $L1_HEAD"

# Get the latest L2 output index from the Output Oracle
L2_OUTPUT_INDEX=$(cast call --rpc-url "$L1_API" --block "$L1_HEAD" "${OUTPUT_ORACLE}" 'latestOutputIndex()')
echo "L2 Output Index: $L2_OUTPUT_INDEX"

# Load the actual output proposal, this returns the outputRoot, timestamp and l2BlockNumber fields
L2_OUTPUT=$(cast call --rpc-url "$L1_API" --block "$L1_HEAD" "${OUTPUT_ORACLE}" 'getL2Output(uint256) (bytes32,uint128,uint128)' "$L2_OUTPUT_INDEX")
echo "L2 Output: $L2_OUTPUT"
IFS=$'\n' OUTPUT_COMPONENTS=(${L2_OUTPUT})

# Extract the claimed outputRoot which is the first return value
L2_CLAIM=${OUTPUT_COMPONENTS[0]}
echo "L2 Claim: $L2_CLAIM"

# Extract the L2 block number which is the third return value
L2_BLOCK_NUM="${OUTPUT_COMPONENTS[2]}"
echo "L2 Block Number: $L2_BLOCK_NUM"

# Select an agreed L2 block. In this case, use a block 100 blocks before the output claim's block
# In a real challenge you'd search back through the prior commitments until you reached one that you agreed with and
# use the block that commitment is from as the agreed L2 head
L2_HEAD_NUM=$(( L2_BLOCK_NUM - 100 ))
L2_HEAD=$(cast block --rpc-url "$L2_API" "$L2_HEAD_NUM" hash)
echo "L2 Head $L2_HEAD ($L2_HEAD_NUM)"

echo "Building op-program"
make op-program-host

CMD=("$(pwd)/bin/op-program" "--datadir" "$DATADIR" "--network=goerli" "--l1.head=$L1_HEAD" "--l2.head=$L2_HEAD" "--l2.claim=$L2_CLAIM" "--l2.blocknumber=$L2_BLOCK_NUM")
ONLINE_CMD=("${CMD[@]}")
ONLINE_CMD+=("--l1" "$L1_API" "--l2" "$L2_API")

echo "Running in online mode"
echo "${ONLINE_CMD[@]}"
"${ONLINE_CMD[@]}" || true

echo "To rerun in offline mode use:"
echo "${CMD[@]}"
