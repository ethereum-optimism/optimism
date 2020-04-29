#!/bin/sh
# wait-for-ovm.sh <ovm url with port>
# NOTE: set the CLEAR_DATA_KEY environment variable to clear the $POSTGRES_DIR and $IPFS_DIR on startup.
# Directory will only be cleared if CLEAR_DATA_KEY is set AND different from last start.

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

clear_data_if_necessary()
{
  POSTGRES_DIR=${POSTGRES_DIR:-/data/postgres}
  IPFS_DIR=${IPFS_DIR:-/data/ipfs}
  CLEAR_DATA_FILE_PATH="${IPFS_DIR}/.clear_data_key_${CLEAR_DATA_KEY}"

  if [ -n "$CLEAR_DATA_KEY" -a ! -f "$CLEAR_DATA_FILE_PATH" ]; then
    echo "Detected change in CLEAR_DATA_KEY. Purging data."
    rm -rf ${IPFS_DIR}/*
    rm -rf ${IPFS_DIR}/.clear_data_key_*
    echo "Local data cleared from '${IPFS_DIR}/*'"
    echo "Contents of ipfs dir: $(ls -alh $IPFS_DIR)"

    rm -rf ${POSTGRES_DIR}/*
    echo "Local data cleared from '${POSTGRES_DIR}/*'"
    echo "Contents of postgres dir: $(ls -alh $POSTGRES_DIR)"
    touch $CLEAR_DATA_FILE_PATH
  else
    echo "No change detected in CLEAR_DATA_KEY not deleting data."
  fi
}

clear_data_if_necessary

wait_for_server_to_be_reachable $OVM_URL_WITH_PORT

>&2 echo "OVM is up!"
