#!/bin/bash
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

DEPLOY_SCRIPT="./scripts/Deploy.s.sol"

# Create a backup
cp ${DEPLOY_SCRIPT} ${DEPLOY_SCRIPT}.bak

# Replace mustGetAddress by getAddress in Deploy.s.sol
# This is needed because the Kontrol deployment is only a partial
# version of the full Optimism deployment. Since not all the components
# of the system are deployed, we'd get some reverts on the `mustGetAddress` functions
awk '{gsub(/mustGetAddress/, "getAddress")}1' ${DEPLOY_SCRIPT} > temp && mv temp ${DEPLOY_SCRIPT}

FOUNDRY_PROFILE=kdeploy forge script -vvv test/kontrol/deployment/KontrolDeployment.sol:KontrolDeployment --sig 'runKontrolDeployment()'
echo "Created state diff json"

# Restore the file from the backup
cp ${DEPLOY_SCRIPT}.bak ${DEPLOY_SCRIPT}
rm ${DEPLOY_SCRIPT}.bak

# Clean and store the state diff json in snapshots/state-diff/Kontrol-Deploy.json
JSON_SCRIPTS=test/kontrol/scripts/json
GENERATED_STATEDIFF=Deploy.json # Name of the statediff json produced by the deployment script
STATEDIFF=Kontrol-${GENERATED_STATEDIFF} # Name of the Kontrol statediff
mv snapshots/state-diff/${GENERATED_STATEDIFF} snapshots/state-diff/${STATEDIFF}
python3 ${JSON_SCRIPTS}/clean_json.py snapshots/state-diff/${STATEDIFF}
jq . snapshots/state-diff/${STATEDIFF} > temp && mv temp snapshots/state-diff/${STATEDIFF} # Prettify json
echo "Cleaned state diff json"

CONTRACT_NAMES=deployments/hardhat/.deploy
python3 ${JSON_SCRIPTS}/reverse_key_values.py ${CONTRACT_NAMES} ${CONTRACT_NAMES}Reversed
CONTRACT_NAMES=${CONTRACT_NAMES}Reversed

SUMMARY_DIR=test/kontrol/proofs/utils
SUMMARY_NAME=DeploymentSummary
LICENSE=MIT
kontrol summary ${SUMMARY_NAME} snapshots/state-diff/${STATEDIFF} --contract-names ${CONTRACT_NAMES} --output-dir ${SUMMARY_DIR} --license ${LICENSE}
forge fmt ${SUMMARY_DIR}/${SUMMARY_NAME}.sol
forge fmt ${SUMMARY_DIR}/${SUMMARY_NAME}Code.sol
echo "Added state updates to ${SUMMARY_DIR}/${SUMMARY_NAME}.sol"
