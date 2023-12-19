set -euo pipefail

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

# replace mustGetAddress by getAddress in Deploy.s.sol
sed -i 's/mustGetAddress/getAddress/g' ./scripts/Deploy.s.sol

FOUNDRY_PROFILE=kdeploy forge script -vvv test/kontrol/KontrolDeployment.sol:KontrolDeployment --sig 'runKontrolDeployment()'
echo "Created state diff json"

JSON_SCRIPTS=test/kontrol/scripts/json
STATEDIFF=Deploy.json
python3 ${JSON_SCRIPTS}/clean_json.py snapshots/state-diff/${STATEDIFF}
echo "Cleaned state diff json"

CONTRACT_NAMES=deployments/hardhat/.deploy
python3 ${JSON_SCRIPTS}/reverse_key_values.py ${CONTRACT_NAMES} ${CONTRACT_NAMES}Reversed
CONTRACT_NAMES=${CONTRACT_NAMES}Reversed

PROOFS_DIR=test/kontrol/proofs
SUMMARY_NAME=DeploymentSummary
kontrol summary ${SUMMARY_NAME} snapshots/state-diff/${STATEDIFF} --contract-names ${CONTRACT_NAMES} --output-dir ${PROOFS_DIR}
echo "Added state updates to ${PROOFS_DIR}/${SUMMARY_NAME}.sol"
