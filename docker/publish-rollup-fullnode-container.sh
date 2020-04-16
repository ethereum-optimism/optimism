#!/bin/sh
# Publish Rollup Fullnode Container
# Optional 1st argument is tag name. Default is 'latest'.

set -e

if [ $# -eq 1 ]; then
 FULLNODE_TAG=$1
 echo "Found tag '$FULLNODE_TAG'. Using this as the container tag."
fi

SCRIPT_DIR=$(dirname $0)
ROOT_DIR=$SCRIPT_DIR/..

TAG=${FULLNODE_TAG:-latest}

if [ -z "$AWS_ACCOUNT_NUMBER" ]; then
  echo "No AWS_ACCOUNT_NUMBER env variable is set. Please set it to use this script."
  exit 1
fi

# Make sure we build so the container is using the current source
yarn --cwd $ROOT_DIR clean && yarn --cwd $ROOT_DIR build

echo "\nAuthenticating within ECR...\n"
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node"

echo "\nBuilding fullnode container...\n"
docker build -t "optimism/rollup-full-node:$TAG" "$ROOT_DIR"

echo "\nTagging fullnode container as $TAG in ECR...\n"
docker tag "optimism/rollup-full-node:$TAG" "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node:$TAG"

echo "\nPushing fullnode container to ECR...\n"
docker push "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node:$TAG"

echo "\nPublish complete!"
