#!/bin/bash

# Vault running in the container must listen on a different port.

VAULT_CREDENTIALS="/vault/config/unseal.json"

CONFIG_DIR="/vault/config"

CA_CERT="$CONFIG_DIR/ca.crt"
CA_KEY="$CONFIG_DIR/ca.key"
TLS_KEY="$CONFIG_DIR/my-service.key"
TLS_CERT="$CONFIG_DIR/my-service.crt"
CONFIG="$CONFIG_DIR/openssl.cnf"
CSR="$CONFIG_DIR/my-service.csr"

export VAULT_ADDR="https://127.0.0.1:8900"
export VAULT_CACERT="$CA_CERT"

function create_config {

	cat > "$CONFIG" << EOF

[req]
default_bits = 2048
encrypt_key  = no
default_md   = sha256
prompt       = no
utf8         = yes

# Speify the DN here so we aren't prompted (along with prompt = no above).
distinguished_name = req_distinguished_name

# Extensions for SAN IP and SAN DNS
req_extensions = v3_req

# Be sure to update the subject to match your organization.
[req_distinguished_name]
C  = TH
ST = Bangkok
L  = Vault
O  = omiseGO
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
IP.2  = 192.168.64.1
IP.3  = 192.168.122.1
DNS.1 = localhost
EOF
}

function gencerts {

    create_config
	openssl req \
	-new \
	-sha256 \
	-newkey rsa:2048 \
	-days 120 \
	-nodes \
	-x509 \
	-subj "/C=US/ST=Maryland/L=Vault/O=My Company CA" \
	-keyout "$CA_KEY" \
	-out "$CA_CERT"

	# Generate the private key for the service. Again, you may want to increase
	# the bits to 2048.
	openssl genrsa -out "$TLS_KEY" 2048

	# Generate a CSR using the configuration and the key just generated. We will
	# give this CSR to our CA to sign.
	openssl req \
	-new -key "$TLS_KEY" \
	-out "$CSR" \
	-config "$CONFIG"

	# Sign the CSR with our CA. This will generate a new certificate that is signed
	# by our CA.
	openssl x509 \
	-req \
	-days 120 \
	-in "$CSR" \
	-CA "$CA_CERT" \
	-CAkey "$CA_KEY" \
	-CAcreateserial \
	-sha256 \
	-extensions v3_req \
	-extfile "$CONFIG" \
	-out "$TLS_CERT"

	openssl x509 -in "$TLS_CERT" -noout -text

	rm openssl.cnf

  chown -R nobody:nobody $CONFIG_DIR && chmod -R 777 $CONFIG_DIR
}

gencerts

nohup vault server -dev -dev-root-token-id=test-root-token -log-level=debug -config /vault/config/vault.hcl &
VAULT_PID=$!

function unseal() {
    VAULT_INIT=$(cat $VAULT_CREDENTIALS)
    UNSEAL_KEY=$(echo $VAULT_INIT | jq -r '.unseal_keys_hex[0]')
    ROOT_TOKEN=$(echo $VAULT_INIT | jq -r .root_token)
    vault operator unseal $UNSEAL_KEY
    export VAULT_TOKEN=$ROOT_TOKEN
}

function configure_plugin {
	plugin_file="immutability-eth-plugin"

	echo "ADDING TO CATALOG: sys/plugins/catalog/secret/${plugin_file}"

	# just testing for now
	plugin_file="${plugin_file}"
	ls -latr /vault/plugins
	sha256sum=`cat /vault/plugins/SHA256SUMS | awk '{print $1}'`
	vault write sys/plugins/catalog/secret/${plugin_file} \
		  sha_256="$sha256sum" \
		  command="$plugin_file --ca-cert=$CA_CERT --client-cert=$TLS_CERT --client-key=$TLS_KEY"

	if [[ $? -eq 2 ]] ; then
	  echo "Vault Catalog update failed!"
	  exit 2
	fi

	echo "MOUNTING: ${plugin_file}"
	vault secrets enable -path=${plugin_file} -plugin-name=${plugin_file} plugin
	if [[ $? -eq 2 ]] ; then
	  echo "Failed to mount ${plugin_file} plugin for test!"
	  exit 2
	fi
}

function test_banner {
    echo "************************************************************************************************************************************"
}

function test_plugin {
	# test_banner
	# echo "SMOKE TEST BASIC WALLET FUNCTIONALITY"
	# test_banner
	# /vault/scripts/smoke.wallet.sh
	# test_banner
	# echo "SMOKE TEST WHITELIST FUNCTIONALITY"
	# test_banner
	# /vault/scripts/smoke.whitelist.sh
	# test_banner
	# echo "SMOKE TEST BLACKLIST FUNCTIONALITY"
	# test_banner
	# /vault/scripts/smoke.blacklist.sh
	# test_banner
	# echo "SMOKE TEST PLASMA FUNCTIONALITY"
	# test_banner
	# /vault/scripts/smoke.plasma.sh
	# echo "SMOKE TEST OVM SUBMIT BATCH"
	#test_banner
	# /vault/scripts/smoke.ovm.sh
	echo "SMOKE TEST OVM CUSTOM ENCODING"
	test_banner
	/vault/scripts/smoke.encode_asb.sh
}

if [ -f "$VAULT_CREDENTIALS" ]; then
    sleep 10
    unseal
    vault status
    vault secrets list
    test_banner
    test_plugin
else
    sleep 10
    VAULT_INIT=$(vault operator init -key-shares=1 -key-threshold=1 -format=json | jq .)
    echo $VAULT_INIT > $VAULT_CREDENTIALS
    unseal
    configure_plugin
    vault audit enable file file_path=stdout
    vault status
    vault secrets list
    test_banner
    test_plugin
fi


if [ -n "$TEST" ]; then 
    echo "Dying."
else
    echo "Don't exit until vault dies."
    wait $VAULT_PID
fi

