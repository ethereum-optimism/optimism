#!/bin/sh
# wait-for-ovm.sh <ovm url with port>

set -e

if [ -z "$OVM_URL_WITH_PORT" ]; then
  echo "Must set environment variable OVM_URL_WITH_PORT"
  exit 1
fi

STARTUP_WAIT_TIMEOUT=${STARTUP_WAIT_TIMEOUT:-20}

wait_for_server_to_be_reachable()
{
  if [ -n "$1" ]; then
    COUNT=1
    until $(curl --output /dev/null --silent --fail -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "id": 9999999, "method": "net_version"}' $1); do
      sleep 1
      echo "Slept $COUNT times for $1 to be up..."

      if [ "$COUNT" -ge "$STARTUP_WAIT_TIMEOUT" ]; then
        echo "Timeout waiting for server at $1"
        exit 1
      fi
      COUNT=$(($COUNT+1))
    done
  fi


}

wait_for_server_to_be_reachable $OVM_URL_WITH_PORT

>&2 echo "OVM is up!"
