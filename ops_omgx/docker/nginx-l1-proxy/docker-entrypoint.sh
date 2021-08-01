#!/bin/bash

#set -eo pipefail
export L1_ENDPOINT=`/opt/secret2env -name $SECRETNAME|grep -w L1_ENDPOINT|sed 's/L1_ENDPOINT=//g'`
cp -fRv /nginx/tmp/* /usr/local/openresty/nginx/conf/
gomplate -f /docker-entrypoint.d/nginx.template.conf > /usr/local/openresty/nginx/conf/nginx.conf
cat /usr/local/openresty/nginx/conf/nginx.conf
sh -c "/nginx-reloader.sh &"
exec openresty "$@"
