#Env vars
export ETH_RPC_URL=http://localhost:8545 #127.0.0.1:8545
export PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 # First anvil private key
export ETH_RPC_URL=
export PRIVATE_KEY=

# create deployments/hardhat/.deploy and snapshots/state-diff/Deploy.json if necessary
if [ ! -d "deployments/hardhat" ]; then
  mkdir deployments/hardhat;
fi
if [ ! -f "deployments/hardhat/.deploy" ]; then
  touch deployments/hardhat/.deploy;
fi
if [ ! -d "snapshots/state-diff" ]; then
  mkdir snapshots/state-diff;
fi
if [ ! -f "snapshots/state-diff/Deploy.json" ]; then
  touch snapshots/state-diff/Deploy.json;
fi

FOUNDRY_PROFILE=kontrol forge script -vvvvv test/kontrol/StateDiff.sol:MakeStateDiff --sig 'testStateDiff()' # --broadcast # --rpc-url $ETH_RPC_URL --private-key $PRIVATE_KEY
# forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'runWithStateDiff()' # --rpc-url $ETH_RPC_URL --private-key $PRIVATE_KEY
echo "Created state diff json"

STATEDIFF=TestDiff.json
STATEDIFF=Deploy.json
python3 clean_json.py snapshots/state-diff/${STATEDIFF}
echo "Cleaned state diff json"

CONTRACT_NAMES='CounterNames.json'

# Clean json produced by Deployer.s.sol::runWithStateDiff()
CONTRACT_NAMES=deployments/hardhat/.deploy
python3 reverse_key_values.py ${CONTRACT_NAMES} ${CONTRACT_NAMES}Reversed
CONTRACT_NAMES=${CONTRACT_NAMES}Reversed

STATEDIFF_CONTRACT=test/kontrol/state-change
#/StateDiffCheatcode.sol
kontrol summary StateDiffCheatcode snapshots/state-diff/${STATEDIFF} --contract-names ${CONTRACT_NAMES} --output-dir ${STATEDIFF_CONTRACT}
echo "Added State Updates to ${STATEDIFF_CONTRACT}"
