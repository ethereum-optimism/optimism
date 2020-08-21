#! /bin/bash

set -e

SERVICES=("compute.googleapis.com" "cloudkms.googleapis.com" "containerregistry.googleapis.com", "iap.googleapis.com",
	"iam.googleapis.com")
SECONDARY=("container.googleapis.com")

MODE="enable"
GCP_PROJECT=""

while getopts "hdp:" opt; do
  case "$opt" in
    d)
      MODE="disable"
      ;;
    p)
      GCP_PROJECT="$OPTARG"
      ;;
    h)
      echo "USAGE: ./gcp_services.sh -p <gcp_project> [-d:disable]"
      exit 1
      ;;
  esac
done
shift "$(($OPTIND -2))"

if [ "$MODE" == "enable" ]; then
  echo "Enabling GCP services. This may take some time..."

  for svc in "${SERVICES[@]}"; do
    gcloud services $MODE $svc
  done

  for svc in "${SECONDARY[@]}"; do
    gcloud services $MODE $svc
  done
else
  echo "Disabling GCP services. This may take some time..."

  for svc in "${SECONDARY[@]}"; do
    gcloud services $MODE $svc
  done

  for svc in "${SERVICES[@]}"; do
    gcloud services $MODE $svc
  done
fi

echo "Done."
