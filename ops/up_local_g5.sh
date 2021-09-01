#!/bin/bash

#if ! [ -x "$(command -v yq)" ]; then
#  echo 'Error: yq is not installed. brew install yq' >&2
#  exit 1
#fi

if [[ $BUILD == 2 ]]; then
  echo 'You set BUILD to 2, which means that we will use existing docker images on your computer'
fi

if [[ $BUILD == 1 ]]; then
  echo 'You set BUILD to 1, which means that all your dockers will be (re)built'
fi

if [[ $BUILD == 0 ]]; then
  echo 'You set BUILD to 0, which means that you want to pull Docker images from Dockerhub'
fi

if [[ $DAEMON == 1 ]]; then
  echo 'You set DAEMON to 1, which means that your local L1/L2 will run in the background'
fi

if [[ $DAEMON == 0 ]]; then
  echo 'You set DAEMON to 0, which means that your local L1/L2 will run in the front and you will see all the debug log information'
fi

if [[ $GETH == 1 ]]; then
  echo 'You set GETH to 1, which means that you will run a Geth L1 with a configurable blocktime'
fi

if [[ $GETH == 0 ]]; then
  echo 'You set GETH to 0, which means that you will run the standard hardhat L1'
fi

#Build dependencies, if needed
if [[ $BUILD == 1 ]]; then
  yarn
  yarn build
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
DOCKERFILE="docker-compose-geth.yml"

if [[ $BUILD == 1 ]]; then
    docker-compose -f $DIR/$DOCKERFILE build --parallel -- builder l2geth l1_chain
    docker-compose -f $DIR/$DOCKERFILE build --parallel -- deployer dtl batch_submitter relayer integration_tests
    docker-compose -f $DIR/$DOCKERFILE build -- omgx_message-relayer-fast
    docker-compose -f $DIR/$DOCKERFILE build -- gas_oracle
    docker-compose -f $DIR/$DOCKERFILE build -- omgx_deployer
elif [[ $BUILD == 0 ]]; then
    docker-compose -f $DIR/$DOCKERFILE pull
fi

if [[ $DAEMON == 1 ]]; then
    docker-compose \
    -f $DIR/$DOCKERFILE \
    up --no-build --detach -V
else
    docker-compose \
    -f $DIR/$DOCKERFILE \
    up --no-build -V
fi
