#!/bin/sh
# Publish The Graph Container
# Optional 1st argument is tag name. Default is 'latest'.

set -e

if [ $# -eq 1 ]; then
 THE_GRAPH_TAG=$1
 echo "Found tag '$THE_GRAPH_TAG'. Using this as the container tag."
fi

BASE_DIR=$(dirname $0)
TAG=${THE_GRAPH_TAG:-latest}

if [ -z "$AWS_ACCOUNT_NUMBER" ]; then
  echo "No AWS_ACCOUNT_NUMBER env variable is set. Please set it to use this script."
  exit 1
fi

echo "\nAuthenticating within ECR...\n"
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/the-graph"

echo "\nBuilding The Graph container...\n"
docker build -t "optimism/the-graph:$TAG" "$BASE_DIR/the-graph/."

echo "\nTagging The Graph container as $TAG in ECR...\n"
docker tag "optimism/the-graph:$TAG" "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/the-graph:$TAG"

echo "\nPushing The Graph container to ECR...\n"
docker push "$AWS_ACCOUNT_NUMBER.dkr.ecr.us-east-2.amazonaws.com/optimism/the-graph:$TAG"

echo "\nPublish complete!"
