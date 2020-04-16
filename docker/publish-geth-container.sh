#!/bin/sh
# Publish Geth Container
# Optional 1st argument is tag name. Default is 'latest'.

set -e

if [ $# -eq 1 ]; then
  GETH_TAG=$1
  echo "Found tag '$GETH_TAG'. Using this as the container tag."
fi

BASE_DIR=$(dirname $0)
TAG=${GETH_TAG:-latest}

if [ -z "$AWS_ACCOUNT_NUMBER" ]; then
  echo "No AWS_ACCOUNT_NUMBER env variable is set. Please set it to use this script."
  exit 1
fi

echo "\nAuthenticating within ECR...\n"
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/geth"

echo "\nBuilding Geth container...\n"
docker build -t "optimism/geth:$TAG" "$BASE_DIR/geth/."

echo "\nTagging Geth container as $TAG...\n"
docker tag "optimism/geth:$TAG" "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/geth:$TAG"

echo "\nPushing Geth container to ECR...\n"
docker push "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/geth:$TAG"

echo "\nPublish complete!"
