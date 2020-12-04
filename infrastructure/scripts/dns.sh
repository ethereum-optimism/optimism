#!/bin/bash
# Put the DNS service account credentials into k8s.
# TODO: Replace with SealedSecret.

K8S_SECRET_NAME="dns-creds"

while getopts "hc:r:" opt; do
  case "$opt" in
    c)
      CREDENTIAL_PATH="$OPTARG"
      ;;
    h)
      echo "USAGE: ./dns.sh -c <dns_credential_file_path>"
      exit 1
      ;;
  esac
done

if [ "$CREDENTIAL_PATH" == "" ]; then
  echo "[Err] no service account credentials path was provided (-c)"
  exit 1
fi

kubectl create secret generic $K8S_SECRET_NAME --from-file=$CREDENTIAL_PATH
