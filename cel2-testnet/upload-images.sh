#!/bin/bash
set -eo pipefail

# Save all images into a tar file
mkdir -p tmp
docker save $(docker image ls | awk '/celo-testnet/ {print $1}') > tmp/celo-testnet-images.tgz
# Upload tar to gcp instance
gcloud compute scp --zone "us-west1-b" --project "blockchaintestsglobaltestnet" tmp/celo-testnet-images.tgz l2-celo-dev:~
gcloud compute ssh --zone "us-west1-b" "l2-celo-dev" --project "blockchaintestsglobaltestnet" --command "docker -H unix:///run/docker.sock load <celo-testnet-images.tgz"
