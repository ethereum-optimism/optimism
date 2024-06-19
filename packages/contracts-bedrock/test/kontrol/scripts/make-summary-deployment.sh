#!/bin/bash
set -euo pipefail

export FOUNDRY_PROFILE=kdeploy

SCRIPT_HOME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# shellcheck source=/dev/null
source "$SCRIPT_HOME/common.sh"
# Sanity check on arguments
if [ $# -gt 1 ]; then
  echo "At most one argument can be provided. Instead $# were provided" 1>&2
  exit 1
elif [ $# -eq 1 ]; then
  if [ "$1" != "-h" ] && [ "$1" != "--help" ] && [ "$1" != "container" ] && [ "$1" != "local" ] && [ "$1" != "dev" ]; then
    notif "Invalid argument. Must be \`container\`, \`local\`, \`dev\`, \`-h\` or \`--help\`"
    exit 1
  else
    parse_first_arg "$@"
  fi
fi

cleanup() {
  trap
  # Restore the original script from the backup
  if [ -f "$DEPLOY_SCRIPT.bak" ]; then
    cp "$DEPLOY_SCRIPT.bak" "$DEPLOY_SCRIPT"
    rm "$DEPLOY_SCRIPT.bak"
  fi

  if [ -f "snapshots/state-diff/Deploy.json" ]; then
    rm "snapshots/state-diff/Deploy.json"
  fi

  if [ "$LOCAL" = false ]; then
    clean_docker
  fi
}

# Set trap to call cleanup function on exit
trap cleanup EXIT ERR

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

DEPLOY_SCRIPT="./scripts/deploy/Deploy.s.sol"
conditionally_start_docker

# Create a backup
cp $DEPLOY_SCRIPT $DEPLOY_SCRIPT.bak

# Replace mustGetAddress by getAddress in Deploy.s.sol
# This is needed because the Kontrol deployment is only a partial
# version of the full Optimism deployment. Since not all the components
# of the system are deployed, we'd get some reverts on the `mustGetAddress` functions
awk '{gsub(/mustGetAddress/, "getAddress")}1' $DEPLOY_SCRIPT > temp && mv temp $DEPLOY_SCRIPT

CONTRACT_NAMES=deployments/kontrol.json
SCRIPT_SIG="runKontrolDeployment()"
if [ "$KONTROL_FP_DEPLOYMENT" = true ]; then
  CONTRACT_NAMES=deployments/kontrol-fp.json
  SCRIPT_SIG="runKontrolDeploymentFaultProofs()"
fi

DEPLOY_CONFIG_PATH=deploy-config/hardhat.json \
DEPLOYMENT_OUTFILE="$CONTRACT_NAMES" \
  forge script -vvv test/kontrol/deployment/KontrolDeployment.sol:KontrolDeployment --sig $SCRIPT_SIG
echo "Created state diff json"

# Clean and store the state diff json in snapshots/state-diff/Kontrol-Deploy.json
JSON_SCRIPTS=test/kontrol/scripts/json
GENERATED_STATEDIFF=31337.json # Name of the statediff json produced by the deployment script
STATEDIFF=Kontrol-$GENERATED_STATEDIFF # Name of the Kontrol statediff
mv snapshots/state-diff/$GENERATED_STATEDIFF snapshots/state-diff/$STATEDIFF
python3 $JSON_SCRIPTS/clean_json.py snapshots/state-diff/$STATEDIFF
jq . snapshots/state-diff/$STATEDIFF > temp && mv temp snapshots/state-diff/$STATEDIFF # Prettify json
echo "Cleaned state diff json"

python3 $JSON_SCRIPTS/reverse_key_values.py $CONTRACT_NAMES ${CONTRACT_NAMES}Reversed
CONTRACT_NAMES=${CONTRACT_NAMES}Reversed

SUMMARY_DIR=test/kontrol/proofs/utils
SUMMARY_NAME=DeploymentSummary
LICENSE=MIT

if [ "$KONTROL_FP_DEPLOYMENT" = true ]; then
  SUMMARY_NAME=DeploymentSummaryFaultProofs
fi

copy_to_docker # Copy the newly generated files to the docker container
run kontrol load-state-diff $SUMMARY_NAME snapshots/state-diff/$STATEDIFF --contract-names $CONTRACT_NAMES --output-dir $SUMMARY_DIR --license $LICENSE
if [ "$LOCAL" = false ]; then
    # Sync Snapshot updates to the host
    docker cp "$CONTAINER_NAME:/home/user/workspace/$SUMMARY_DIR" "$WORKSPACE_DIR/$SUMMARY_DIR/.."
fi
forge fmt $SUMMARY_DIR/$SUMMARY_NAME.sol
forge fmt $SUMMARY_DIR/${SUMMARY_NAME}Code.sol
echo "Added state updates to $SUMMARY_DIR/$SUMMARY_NAME.sol"
