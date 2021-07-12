#!/bin/bash

set -x

export VAULT_CACERT=$DEVDIR/docker/ca/certs/ca.crt
export VAULT_TOKEN="totally-secure"
export AUTHORITY_ADDRESS="0x4BC91c7fA64017a94007B7452B75888cD82185F7"

# Create a policy that only allows access to submit blocks

tee submit-blocks.hcl <<EOF
path "immutability-eth-plugin/wallets/plasma-deployer/accounts/$AUTHORITY_ADDRESS/plasma/submitBlock" {
    capabilities = ["update", "create"]
}
EOF

vault policy write submit-blocks submit-blocks.hcl

# Set VAULT_SA_NAME to the service account you created earlier
export VAULT_SA_NAME=$(kubectl get sa omisego-service -o jsonpath="{.secrets[*]['name']}")

# Set SA_JWT_TOKEN value to the service account JWT used to access the TokenReview API
export SA_JWT_TOKEN=$(kubectl get secret $VAULT_SA_NAME -o jsonpath="{.data.token}" | base64 --decode; echo)

# Set SA_CA_CRT to the PEM encoded CA cert used to talk to Kubernetes API
export SA_CA_CRT=$(kubectl get secret $VAULT_SA_NAME -o jsonpath="{.data['ca\.crt']}" | base64 --decode; echo)

# Set K8S_HOST to minikube IP address
export K8S_HOST=$(minikube ip)

# Enable the Kubernetes auth method at the default path ("auth/kubernetes")
vault auth enable kubernetes

# Tell Vault how to communicate with the Kubernetes (Minikube) cluster
vault write auth/kubernetes/config token_reviewer_jwt="$SA_JWT_TOKEN" kubernetes_host="https://$K8S_HOST:8443" kubernetes_ca_cert="$SA_CA_CRT"

# Create a role named, 'omisego-service' to map Kubernetes Service Account to authority permissions and 24h TTL
vault write auth/kubernetes/role/authority bound_service_account_names=omisego-service bound_service_account_namespaces=default policies=submit-blocks ttl=24h
