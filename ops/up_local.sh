#!/bin/bash

if ! [ -x "$(command -v yq)" ]; then
  echo 'Error: yq is not installed. brew install yq' >&2
  exit 1
fi

#Set conservative defaults
#BUILD=${BUILD:-1}     #build unless override
#DAEMON=${DAEMON:-0}   #run win foreground unless overrride

if [[ $BUILD == 1 ]]; then
  echo 'You set BUILD to 1, which means that all your dockers will be (re)built'
fi

if [[ $BUILD == 0 ]]; then
  echo 'You set BUILD to 0, which means that you want to use existing Docker images'
fi

if [[ $DAEMON == 1 ]]; then
  echo 'You set DAEMON to 1, which means that your local L1/L2 will run in the background'
fi

if [[ $DAEMON == 0 ]]; then
  echo 'You set DAEMON to 0, which means that your local L1/L2 will run in the front and you will see all the debug log information'
fi

# # Build dependencies, if needed
# if [[ $BUILD == 1 ]]; then
yarn
yarn build
# fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
ORIGINAL_DOCKERFILE="docker-compose.yml"
DOCKERFILE="docker-compose-omgx.yml"
#DOCKERFILE="docker-compose.yml"
OMGX_DOCKERFILE=docker-compose-omgx-services.yml
#replace all occurances of image: ethereumoptimism/ with image: omgx/
#append :latest tag to all apps
yq eval '(.services.[].image | select(. == "ethereumoptimism*")) |= sub("ethereumoptimism", "omgx")' ${ORIGINAL_DOCKERFILE} | \
yq eval '(.services.[].image) += ":latest"' - \
> ${DOCKERFILE}

if [[ $BUILD == 1 ]]; then
    docker-compose build --parallel -- builder l2geth l1_chain
    docker-compose build --parallel -- deployer dtl batch_submitter relayer integration_tests
    docker image tag ethereumoptimism/builder omgx/builder:latest
    docker image tag ethereumoptimism/hardhat omgx/hardhat:latest
    # comented out because this command does not seem to wait for completion of the parallel build in GH actions?
    # see error in https://github.com/omgnetwork/optimism/pull/154/checks?check_run_id=2984030034
    docker image tag ethereumoptimism/deployer omgx/deployer:latest
    docker image tag ethereumoptimism/data-transport-layer omgx/data-transport-layer:latest
    docker image tag ethereumoptimism/l2geth omgx/l2geth:latest
    docker image tag ethereumoptimism/message-relayer omgx/message-relayer:latest
    docker image tag ethereumoptimism/batch-submitter omgx/batch-submitter:latest
    docker image tag ethereumoptimism/integration-tests omgx/integration-tests:latest
    docker build ../ --file $DIR/docker/Dockerfile.omgx_monorepo --tag omgx/omgx_builder:latest
    docker build ../ --file $DIR/docker/Dockerfile.omgx_deployer --tag omgx/omgx_deployer:latest
    docker build ../ --file $DIR/docker/Dockerfile.omgx_message-relayer-fast --tag omgx/omgx_message-relayer-fast:latest
else
    docker-compose -f $DIR/$DOCKERFILE -f $DIR/$OMGX_DOCKERFILE pull
fi

if [[ $DAEMON == 1 ]]; then
    docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/$OMGX_DOCKERFILE \
    up --no-build --detach -V
else
    docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/$OMGX_DOCKERFILE \
    up --no-build -V
fi
