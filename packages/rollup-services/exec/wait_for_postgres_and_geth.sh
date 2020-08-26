#!/bin/bash
# wait_for_postgres.sh -- accepts a command to run after postgres connection succeeds
set -e

cmd="$@"

RETRIES=30

echo "Connection info: ${POSTGRES_HOST}:${POSTGRES_PORT}, user=${POSTGRES_USER}, postgres db=${POSTGRES_DATABASE}"

until PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DATABASE -c "select 1" > /dev/null 2>&1 || [ $RETRIES -eq 0 ]; do
  echo "Waiting for Postgres server, $((RETRIES--)) remaining attempts..."
  sleep 1
done

if [ $RETRIES -eq 0 ]; then
  echo "Timeout reached waiting for Postgres!"
  exit 1
fi

echo "Connected to Postgres"


if [[ -n "$RUN_L2_CHAIN_DATA_PERSISTER" || -n "$RUN_QUEUED_GETH_SUBMITTER" ]]; then
  COUNT=0
  until $(curl --output /dev/null --silent --fail -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "id": 9999999, "method": "net_version"}' $L2_NODE_WEB3_URL); do
    sleep 1
    echo "Slept $COUNT times for $L2_NODE_WEB3_URL to be up..."

    if [ "$COUNT" -ge 30 ]; then
      echo "Timeout waiting for l2 geth node at $L2_NODE_WEB3_URL"
      exit 1
    fi
    COUNT=$(($COUNT+1))
  done
  echo "Connected to geth"
fi

>&2 echo "Continuing with startup..."

exec $cmd
