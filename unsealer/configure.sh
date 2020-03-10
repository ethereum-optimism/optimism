#!/bin/sh

set -e

# Configures the Unsealer Vault. This will enable the k/v (version 1) secrets 
# backend as well as the transit backend. A policy will be created for autounseal.
# 
# Requirements:
#   - Vault has been initialized and unsealed
#   - VAULT_TOKEN is set in the environment and has sufficient permissions
# 
# Usage:
#   configure.sh

OVPN_FILE="$(pwd)/../infrastructure/unsealer.ovpn"

if [ -z "$VAULT_TOKEN" ]; then
	echo "Missing VAULT_TOKEN"
	exit 1
fi

function transit {
    vault secrets enable transit
    vault write -f transit/keys/autounseal
    tee autounseal.hcl <<EOF
path "transit/encrypt/autounseal" {
capabilities = [ "update" ]
}

path "transit/decrypt/autounseal" {
capabilities = [ "update" ]
}
EOF

    # Create an 'autounseal' policy
    vault policy write autounseal autounseal.hcl
    vault token create -policy="autounseal"
}

function secrets {
    vault secrets enable -version=1 kv
    vault write kv/consul_gossip_key value=$(consul keygen)
    vault write kv/unseal_token value=$(vault token create -field=token -ttl="8760h" -policy=autounseal)
    vault write kv/vault_cacert value=$(cat $HOME/etc/vault.unsealer/root.crt | base64)
    vault write kv/vault_key value=$(cat $HOME/etc/vault.unsealer/vault.key | base64)
    vault write kv/vault_crt value=$(cat $HOME/etc/vault.unsealer/vault.crt | base64)
    if [ -f "$OVPN_FILE" ]; then
        vault write kv/ovpn_file value=$(cat $OVPN_FILE | base64)
    fi    
}

transit
secrets

echo "--> Done!"