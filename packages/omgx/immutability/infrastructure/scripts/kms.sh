#! /bin/bash

set -e

KEY_RING_NAME="omgnetwork-vault-keyring"
UNSEAL_KEY_NAME="omgnetwork-vault-unseal-key"
K8S_SECRET_NAME="kms-creds"

CREDENTIAL_PATH=""
GCP_REGION=""

check_ring_exists() {
  local ring_name=$1
  local location=$2
  local rings=( $(gcloud kms keyrings list --location $location --format=json | jq -r '.[].name' | cut -d '/' -f 6) )
  if [ "$rings" != "" ]; then
    for r in "${rings[@]}"; do
      if [ "$r" == "$ring_name" ]; then
        echo "    [INFO] KMS key ring already exists, skipping" >&2
        echo "1"
        return
      fi
    done
  fi
  echo "0"
}

check_key_exists() {
  local ring_name=$1
  local key_name=$2
  local location=$3
  local keys=( $(gcloud kms keys list --keyring $ring_name --location $location --format=json | jq -r '.[].name' | cut -d '/' -f 8) )
  if [ "$keys" != "" ]; then
    for k in "${keys[@]}"; do
      if [ "$k" == "$key_name" ]; then
        echo "    [INFO] KMS unseal key already exists in ring, skipping" >&2
        echo "1"
        return
      fi
    done
  fi
  echo "0"
}

check_k8s_secret_exists() {
  local secret_name=$1
  local secrets=( $(kubectl get secrets -o json | jq -r '.items[].metadata.name' ) )
  for s in "${secrets[@]}"; do
    if [ "$s" == "$secret_name" ]; then
      echo "    [INFO] KMS credentials already exist in K8S, skipping" >&2
      echo "1"
      return
    fi
  done
  echo "0"
}

while getopts "hc:r:" opt; do
  case "$opt" in
    c)
      CREDENTIAL_PATH="$OPTARG"
      ;;
    r)
      GCP_REGION="$OPTARG"
      ;;
    h)
      echo "USAGE: ./kms.sh -c <kms_credential_file_path> -r <gcp_region>"
      exit 1
      ;;
  esac
done
shift "$(($OPTIND -2))"

if [ "$CREDENTIAL_PATH" == "" ]; then
  echo "[Err] no service account credentials path was provided (-c)"
  exit 1
elif [ "$GCP_REGION" == "" ]; then
  echo "[Err] no GCP location/region provided (-r)"
  exit 1
fi

echo "+ Activating GCP KMS Service Account Credentials"
gcloud auth activate-service-account --key-file $CREDENTIAL_PATH

echo "+ Creating KMS Key Ring"
ring_exists=$(check_ring_exists $KEY_RING_NAME $GCP_REGION)
if [ "$ring_exists" == "0" ]; then
  gcloud kms keyrings create $KEY_RING_NAME --location $GCP_REGION
fi

echo "+ Creating Symmetric Unseal Key"
key_exists=$(check_key_exists $KEY_RING_NAME $UNSEAL_KEY_NAME $GCP_REGION)
if [ "$key_exists" == "0" ]; then
  gcloud kms keys create $UNSEAL_KEY_NAME \
    --keyring $KEY_RING_NAME \
    --location $GCP_REGION \
    --purpose "encryption"
fi

echo "+ Injecting KMS Service Account Credentials into GKE"
secret_exists=$(check_k8s_secret_exists $K8S_SECRET_NAME)
if [ "$secret_exists" == "0" ]; then
  kubectl create secret generic $K8S_SECRET_NAME --from-file=$CREDENTIAL_PATH
fi

echo "+ Revoking GCloud Authentication"
gcloud auth revoke
