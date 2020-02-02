#!/bin/bash

set -x

export VAULT_CACERT=$DEVDIR/docker/ca/certs/ca.crt

# We used a self-signed cert for Vault so the agent needs it
kubectl create configmap cacerts --from-file=vault-cacert=$VAULT_CACERT

kubectl create configmap vault-agent-config --from-file=./vault-agent-config.hcl

