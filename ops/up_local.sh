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

#Build dependencies, if needed
if [[ $BUILD == 1 ]]; then
  yarn
  yarn build
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
DOCKERFILE="docker-compose.yml"

if [[ $BUILD == 1 ]]; then
    docker-compose build --parallel -- builder l2geth l1_chain
    docker-compose build --parallel -- deployer dtl batch_submitter relayer integration_tests
    docker-compose build --parallel -- omgx_deployer omgx_message-relayer-fast
    docker-compose build --parallel -- gas_oracle
elif [[ $BUILD == 0 ]]; then
    docker-compose -f $DIR/$DOCKERFILE pull
fi

if [[ $DAEMON == 1 ]]; then
    docker-compose \
    -f $DIR/$DOCKERFILE \
    up --detach -V
    #up --no-build --detach -V
else
    docker-compose \
    -f $DIR/$DOCKERFILE \
    up -V
    #up --no-build -V
fi
