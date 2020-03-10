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

KEY_SHARES=5
KEY_THRESHOLD=3

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
	export VAULT_CACERT=$VAULT_DIR/root.crt
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
	cat > "$VAULT_DIR/vault.hcl" << EOF
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

  tls_cert_file = "$VAULT_DIR/vault.crt"
  tls_client_ca_file = "$VAULT_DIR/root.crt"
  tls_key_file = "$VAULT_DIR/vault.key"
}

EOF
}

function gencerts {

	cat > "$VAULT_DIR/openssl.cnf" << EOF
[req]
default_bits = 2048
encrypt_key  = no
default_md   = sha256
prompt       = no
utf8         = yes

# Specify the DN here so we aren't prompted (along with prompt = no above).
distinguished_name = req_distinguished_name

# Extensions for SAN IP and SAN DNS
req_extensions = v3_req

# Be sure to update the subject to match your organization.
[req_distinguished_name]
C  = TH
ST = Bangkok
L  = Vault
O  = OmiseGO
CN = localhost

# Allow client and server auth. You may want to only allow server auth.
# Link to SAN names.
[v3_req]
basicConstraints     = CA:FALSE
subjectKeyIdentifier = hash
keyUsage             = digitalSignature, keyEncipherment
extendedKeyUsage     = clientAuth, serverAuth
subjectAltName       = @alt_names

# Alternative names are specified as IP.# and DNS.# for IPs and
# DNS accordingly.
[alt_names]
IP.1  = 127.0.0.1
IP.2  = 10.8.0.2
DNS.1 = localhost
EOF

	openssl req \
	-new \
	-sha256 \
	-newkey rsa:2048 \
	-days 120 \
	-nodes \
	-x509 \
	-subj "/C=TH/ST=Bangkok/L=OmiseGO/O=Unsealer" \
	-keyout "$VAULT_DIR/root.key" \
	-out "$VAULT_DIR/root.crt"

	# Generate the private key for the service. Again, you may want to increase
	# the bits to 2048.
	openssl genrsa -out "$VAULT_DIR/vault.key" 2048

	# Generate a CSR using the configuration and the key just generated. We will
	# give this CSR to our CA to sign.
	openssl req \
	-new -key "$VAULT_DIR/vault.key" \
	-out "$VAULT_DIR/vault.csr" \
	-config "$VAULT_DIR/openssl.cnf"

	# Sign the CSR with our CA. This will generate a new certificate that is signed
	# by our CA.
	openssl x509 \
	-req \
	-days 120 \
	-in "$VAULT_DIR/vault.csr" \
	-CA "$VAULT_DIR/root.crt" \
	-CAkey "$VAULT_DIR/root.key" \
	-CAcreateserial \
	-sha256 \
	-extensions v3_req \
	-extfile "$VAULT_DIR/openssl.cnf" \
	-out "$VAULT_DIR/vault.crt"

	openssl x509 -in "$VAULT_DIR/vault.crt" -noout -text

	rm "$VAULT_DIR/openssl.cnf"
  
}

mkdir -p $VAULT_DIR
genconfig
gencerts

nohup /usr/local/bin/vault server -config $VAULT_DIR/vault.hcl &> /dev/null &
initialize

echo "--> Done!"