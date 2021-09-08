#!/bin/bash

if [[$STAGE == "all"]]; then
  echo 'You set STAGE to rinkeby. Deploying to Rinkeby...'
  cp env-rinkeby.yml env.yml &&
  serverless -c serverless-rinkeby.yml deploy &&
  rm -rf env.yml &&
  rm -rf .serverless &&
  echo 'You set STAGE to mainnet. Deploying to Mainnet...'
  cp env-mainnet.yml env.yml &&
  serverless -c serverless-mainnet.yml deploy &&
  rm -rf env.yml &&
  rm -rf .serverless
fi

if [[ $STAGE == "rinkeby" ]]; then
  echo 'You set STAGE to rinkeby. Deploying to Rinkeby...'
  cp env-rinkeby.yml env.yml &&
  serverless -c serverless-rinkeby.yml deploy &&
  rm -rf env.yml &&
  rm -rf .serverless
fi

if [[ $STAGE == "mainnet" ]]; then
  echo 'You set STAGE to mainnet. Deploying to Mainnet...'
  cp env-mainnet.yml env.yml &&
  serverless -c serverless-mainnet.yml deploy &&
  rm -rf env.yml &&
  rm -rf .serverless
fi
