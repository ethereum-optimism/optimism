#!/bin/bash

set -eo pipefail

if [ -z "$SEQUENCER" ];then
  echo "SEQUENCER env must be set, exiting"
  exit 1
fi

if [ -z "$RPC_METHODS_ALLOWED" ];then
  echo "RPC_METHODS_ALLOWED env must be set, exiting"
  exit 1
fi

gomplate -f /docker-entrypoint.d/nginx.template.conf > /usr/local/openresty/nginx/conf/nginx.conf

cat /usr/local/openresty/nginx/conf/nginx.conf

exec openresty "$@"
