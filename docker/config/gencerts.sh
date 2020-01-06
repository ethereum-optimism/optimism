#!/bin/sh

set -ex

function gencerts {

	mkdir -p /home/root/ca/certs /home/root/ca/private

	cat > "./openssl.cnf" << EOF
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

	openssl req \
	-new \
	-sha256 \
	-newkey rsa:2048 \
	-days 120 \
	-nodes \
	-x509 \
	-subj "/C=US/ST=Maryland/L=Vault/O=My Company CA" \
	-keyout "/home/root/ca/private/ca.key" \
	-out "/home/root/ca/certs/ca.crt"

	# Generate the private key for the service. Again, you may want to increase
	# the bits to 2048.
	openssl genrsa -out "/home/root/ca/private/my-service.key" 2048

	# Generate a CSR using the configuration and the key just generated. We will
	# give this CSR to our CA to sign.
	openssl req \
	-new -key "/home/root/ca/private/my-service.key" \
	-out "/home/root/ca/my-service.csr" \
	-config "openssl.cnf"

	# Sign the CSR with our CA. This will generate a new certificate that is signed
	# by our CA.
	openssl x509 \
	-req \
	-days 120 \
	-in "/home/root/ca/my-service.csr" \
	-CA "/home/root/ca/certs/ca.crt" \
	-CAkey "/home/root/ca/private/ca.key" \
	-CAcreateserial \
	-sha256 \
	-extensions v3_req \
	-extfile "openssl.cnf" \
	-out "/home/root/ca/certs/my-service.crt"

	openssl x509 -in "/home/root/ca/certs/my-service.crt" -noout -text

	rm openssl.cnf

  chown -R nobody:nobody /home/root/ca && chmod -R 777 /home/root/ca
}

gencerts