#!/bin/bash
set -eo pipefail

source .envrc

# Create JWT if it does not exist, yet.
[ -f generated/jwt-secret.txt ] || openssl rand -hex 32 > generated/jwt-secret.txt

PROJECT=cel2-testnet
cd docker
docker-compose -p $PROJECT up -d --no-build op-node
sleep 4
docker-compose -p $PROJECT up -d --no-build op-proposer op-batcher
