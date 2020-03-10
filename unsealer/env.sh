#!/usr/bin/env sh
set -e

#
# Sets the Unsealer Vault environment variables
#
# Usage:
#   source env.sh name-of-keybase-identity
#
# Requirements:
#   - Keybase

KEYBASE="$1"
if [ -z "$KEYBASE" ]; then
  echo "Missing KEYBASE"
  exit 1
fi

export VAULT_CACERT=$HOME/etc/vault.unsealer/root.crt
export VAULT_TOKEN=$(cat keybase.$KEYBASE.root.b64 | base64 --decode | keybase pgp decrypt)
export VAULT_ADDR=https://localhost:8200