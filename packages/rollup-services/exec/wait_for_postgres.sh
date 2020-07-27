#!/bin/bash
# wait_for_postgres.sh -- accepts a command to run after postgres connection succeeds
set -e

cmd="$@"

RETRIES=30

echo "Connection info: ${POSTGRES_HOST}:${POSTGRES_PORT}, user=${POSTGRES_USER}, postgres db=${POSTGRES_DATABASE}"

until PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DATABASE -c "select 1" > /dev/null 2>&1 || [ $RETRIES -eq 0 ]; do
  echo "Waiting for postgres server, $((RETRIES--)) remaining attempts..."
  sleep 1
done

if [ $RETRIES -eq 0 ]; then
  echo "Timeout reached waiting for postgres!"
  exit 1
fi

>&2 echo "Connected to postgres. Continuing with startup..."

exec $cmd
