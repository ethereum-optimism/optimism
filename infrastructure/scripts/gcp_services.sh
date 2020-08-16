#! /bin/bash

set -e

SERVICES=("compute.googleapis.com")
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

echo "Enabling GCP services..."

for svc in "${SERVICES[@]}"; do
  gcloud services $MODE $svc
done

echo "Done."
