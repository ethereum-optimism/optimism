#!/bin/bash

set -e

CONTAINER_IMAGES=("omgnetwork/vault:latest")
MODE="create"
GCR_HOST="gcr.io"
GCP_PROJECT=""

find_tag() {
  echo -n $(docker images --filter=reference="$1" --format="{{ .ID }}")
}

tag_and_push() {
  echo ""
  destination_tag="$GCR_HOST/$GCP_PROJECT/$1"
  docker tag $1 $destination_tag
  docker push $destination_tag
  echo ""
}

remove_tag() {
  echo "Deleting tag "$1"..."
  docker rmi $1
  gcloud container images delete $1
  echo ""
}

while getopts "hdp:r:" opt; do
  case "$opt" in
    d)
      MODE="delete"
      ;;
    p)
      GCP_PROJECT="$OPTARG"
      ;;
    r)
      GCR_HOST="$OPTARG"
      ;;
    h)
      echo "USAGE: ./gcr_docker_tags.sh -p <gcp_project> -r <registry_host>[default:gcr.io, us.gcr.io, ...]"
      exit 1
      ;;
  esac
done
shift "$(($OPTIND -2))"

if [ "$MODE" == "create" ]; then
  echo "EXISTING IMAGES IN GCR:"
  gcloud container images list
  echo ""

  for img in "${CONTAINER_IMAGES[@]}"; do
    gcr_tag="$GCR_HOST/$GCP_PROJECT/$img"
    id=$(find_tag $gcr_tag)

    if [ "$id" != "" ]; then
      echo "ALREADY TAGGED: $gcr_tag -> $id"
    else
      tag_and_push $img $gcr_tag
    fi
    
    echo "Done."
  done

  echo ""
  echo "WARNING:"
  echo "Images already tagged as GCR hosted are not overridden!"
  echo "To replace any existing container images in GCR, first run:"
  echo ""
  echo "    $ gcloud container images delete <image>"
  echo ""
  echo "...then rerun this script."
else
  for img in "${CONTAINER_IMAGES[@]}"; do
    gcr_tag="$GCR_HOST/$GCP_PROJECT/$img"
    remove_tag $gcr_tag
  done
  
  echo ""
  echo "Done."
fi
