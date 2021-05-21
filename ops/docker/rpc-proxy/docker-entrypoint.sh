#!/bin/bash

set -eo pipefail

if [ -z "$SEQUENCER" ];then
  echo "SEQUENCER env must be set, exiting"
  exit 1
fi

# if [ -z "$REPLICAS" ];then
#   echo "REPLICAS env must be set, exiting"
#   exit 1
# fi

# REPLICAS=$REPLICAS \
gomplate -f /docker-entrypoint.d/nginx.template.conf > /usr/local/openresty/nginx/conf/nginx.conf

cat /usr/local/openresty/nginx/conf/nginx.conf

exec openresty "$@"
