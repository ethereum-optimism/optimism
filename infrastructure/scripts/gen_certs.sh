#!/bin/bash

set -u
set -o pipefail

###
### gen_certs.sh - generate certificates for a given DNS Domain
###
### Usage:
###   gen_certs.sh [options]
###
### Options:
###   -d | --domain-name <name>  The DNS Domain Name of the nodes in the Vault Cluster
###   -s | --service-name <name> The name of the service (default: vault)
###   -h | --help                Show help / usage
### 

DOMAIN=""
CERT_DIR=".certs"
SERVICE_NAME="vault"
LOG=""

# usage displays some helpful information about the script and any errors that need
# to be emitted
usage() {
	MESSAGE=${1:-}

	awk -F'### ' '/^###/ { print $2 }' $0 >&2

	if [[ "${MESSAGE}" != "" ]]; then
		echo "" >&2
		echo "${MESSAGE}" >&2
		echo "" >&2
	fi

	exit 255
}

# validate_config ensures that required variables are set
validate_config() {
	if [[ $(basename ${PWD}) != "infrastructure" ]]; then
		usage "Please execute this script from the \"infrastructure\" directory"
	fi

	if [[ "${DOMAIN}" == "" ]]; then
		usage "The Domain Name (-d) was not specified"
	fi

	if [[ "${SERVICE_NAME}" == "" ]]; then
		usage "The Service Name (-s) was not specified"
	fi
}

# genconfig creates the OpenSSL Configuration File needed to produce the TLS Material
genconfig() {
	echo "> Create Config" >&2

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
O  = OMG Network
CN = ${DOMAIN}

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
DNS.1 = localhost
DNS.2 = ${DOMAIN}
DNS.3 = *.${DOMAIN}
EOF
}

# gencerts produces the TLS material
function gencerts {
	echo "> Generating Certificates" >&2

	openssl req \
		-new \
		-sha256 \
		-newkey rsa:2048 \
		-days 7300 \
		-nodes \
		-x509 \
		-subj "/C=TH/ST=Bankok/L=Vault/O=OMG Network Vault CA" \
		-keyout "$CA_KEY" \
		-out "$CA_CERT" > ${LOG} 2>&1

	openssl genrsa -out "$TLS_KEY" 2048 >> ${LOG} 2>&1

	# Generate a CSR using the configuration and the key just generated. We will
	# give this CSR to our CA to sign.
	openssl req \
		-new -key "$TLS_KEY" \
		-out "$CSR" \
		-config "$CONFIG" >> ${LOG} 2>&1

	# Sign the CSR with our CA. This will generate a new certificate that is signed
	# by our CA.
	openssl x509 \
		-req \
		-days 190 \
		-in "$CSR" \
		-CA "$CA_CERT" \
		-CAkey "$CA_KEY" \
		-CAcreateserial \
		-sha256 \
		-extensions v3_req \
		-extfile "$CONFIG" \
		-out "$TLS_CERT" >> ${LOG} 2>&1

	openssl x509 -in "$TLS_CERT" -noout -text >> ${LOG} 2>&1

	rm -f openssl.cnf
}

# copycerts takes the generated TLS Material and copies it over to the certs 
# directory where it can be installed into your kubernetes cluster
copycerts() {
	echo "> Copying TLS Material to k8s/certs" >&2

	cp -f ${CA_CERT} k8s/certs
	cp -f ${TLS_KEY} k8s/certs
	cp -f ${TLS_CERT} k8s/certs
}

# gensecret creates the kubenernetes secret containing the TLS Material that will
# be consumed by vault in the pods.
gensecret() {
	cd ./k8s/certs
	OUTPUT=$(kubectl apply -k .)
	CERT_SECRET=$(echo $OUTPUT | cut -d' ' -f1 | cut -d'-' -f3)

	cd ..
	yq w -i vault-overrides.yaml global.certSecretName omgnetwork-certs-$CERT_SECRET
	yq w -i vault-overrides.yaml server.extraVolumes[1].name omgnetwork-certs-$CERT_SECRET

	cd ..
}

##
## main
##

while [[ $# -gt 0 ]]; do
	case $1 in 
	-d | --domain-name) 
		DOMAIN=$2
		shift
	;;
	-s | --service-name) 
		SERVICE_NAME=$2
		shift
	;;
	-h | --help) 
		usage
	;;
	--)
		shift 
		break
		;;
	-*) usage "Invalid argument: $1" 1>&2 ;;
	*) usage "Invalid argument: $1" 1>&2 ;;
	esac
	shift
done

validate_config

if [[ ! -d ${CERT_DIR} ]]; then
	mkdir -p ${CERT_DIR}
fi

rm -f ${CERT_DIR}/*

CONFIG="${CERT_DIR}/openssl.cnf"
CA_CERT="${CERT_DIR}/ca.crt"
CA_KEY="${CERT_DIR}/ca.key"
TLS_KEY="${CERT_DIR}/${SERVICE_NAME}.key"
TLS_CERT="${CERT_DIR}/${SERVICE_NAME}.crt"
CSR="${CERT_DIR}/${SERVICE_NAME}.csr"
LOG="${CERT_DIR}/process.log"

genconfig
gencerts
copycerts
gensecret

echo ""
echo "> In your shell, execute this command:"
echo "export VAULT_CACERT=$(PWD)/k8s/certs/ca.crt"
echo ""
