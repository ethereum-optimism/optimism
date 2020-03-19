#!/bin/sh

set -e

#
# Initializes and starts the unsealer vault. This script will generate the CA certificate and
# TLS key material for the Unsealer Vault. It will create the vault configuration, start the
# vault process and initialize the vault.
#
# Vault will be configured in ~/etc/vault.unsealer. 
#
# The inputs are:
# 	Keybase Identity of the Unsealer Admin. The Vault Root Token will be encrypted using 
#	this identity's PGP key.
#
#	Keybase Identities (exactly 5) of the Keyshard Holders. The Keyshards will be encrypted 
#	using these PGP keys
#
# Usage:
#   initialize.sh "keybase:kasima" "keybase:kasima,keybase:jake,keybase:bob,keybase:alice,keybase:eve"
#

KEY_SHARES=1
KEY_THRESHOLD=1

VAULT_DIR=$HOME/etc/vault.unsealer

ADMIN="$1"
if [ -z "$ADMIN" ]; then
	echo "Missing ADMIN identity"
	exit 1
fi

KEYBASE="$2"
if [ -z "$KEYBASE" ]; then
	echo "Missing KEYBASE identities"
	exit 1
else
	IFS=',' read -ra KEYBASE_IDS <<< "$KEYBASE"
	for (( COUNTER=0; COUNTER<$KEY_SHARES; COUNTER++ ))
	do
		if [ -z "${KEYBASE_IDS[$COUNTER]}" ]
		then
			echo "$KEY_SHARES Keybase identities are required!"
			exit 1
		fi
	done
fi

function check {
	EXIT_CODE=$1
	MESSAGE=$2
	if [ $EXIT_CODE -ne 0 ]; then
		echo $MESSAGE
		exit $EXIT_CODE
	fi
}

function initialize {
	export VAULT_CACERT=$VAULT_DIR/ca.pem
  	UNSEAL=$(vault operator init -format=json -key-shares=$KEY_SHARES -key-threshold=$KEY_THRESHOLD -root-token-pgp-key=$ADMIN -pgp-keys="$KEYBASE" | jq .)
	check $? "Unable to initialize Vault"
	IFS=',' read -ra KEYBASE_IDS <<< "$KEYBASE"
	FN=$(echo $ADMIN | sed 's/keybase:/keybase./g')
	echo $UNSEAL | jq -r .root_token > $FN.root.b64
	for (( COUNTER=0; COUNTER<$KEY_SHARES; COUNTER++ ))
	do
		FN=$(echo "${KEYBASE_IDS[$COUNTER]}" | sed 's/keybase:/keybase./g')
		echo $UNSEAL | jq .unseal_keys_b64 | jq -r  '.['$COUNTER']' > $FN.b64
	done
	
}

function genconfig {
	cat > "$VAULT_DIR/vault.hcl" <<-EOF
		default_lease_ttl = "24h"
		disable_mlock = "true"
		max_lease_ttl = "43800h"

		backend "file" {
			path = "$VAULT_DIR/data"
		}

		api_addr = "https://localhost:8200"
		ui = "false"

		listener "tcp" {
			address = "0.0.0.0:8200"
			tls_cert_file = "$VAULT_DIR/services.pem"
			tls_client_ca_file = "$VAULT_DIR/ca.pem"
			tls_key_file = "$VAULT_DIR/services-key.pem"
		}
	EOF
}

function gencerts {
	consul tls ca create -domain=consul

	consul tls cert create \
		-server \
		-days=120 \
		-additional-ipaddress="10.8.0.2" \
		-additional-ipaddress="192.168.64.1" \
		-additional-dnsname="omisego-consul-server" \
		-additional-dnsname="*.omisego-consul-server" \
		-additional-dnsname="*.omisego-consul-server.default" \
		-additional-dnsname="*.omisego-consul-server.default.svc" \
		-dc=dc1 \
		-domain=consul
		
	mv consul-agent-ca.pem $VAULT_DIR/ca.pem
	mv consul-agent-ca-key.pem $VAULT_DIR/ca-key.pem
	mv dc1-server-consul-0.pem $VAULT_DIR/services.pem
	mv dc1-server-consul-0-key.pem $VAULT_DIR/services-key.pem
}

mkdir -p $VAULT_DIR
genconfig
gencerts

nohup /usr/local/bin/vault server -config $VAULT_DIR/vault.hcl &> /dev/null &
initialize
kill $(lsof -ti:8200)

echo "--> Done!"