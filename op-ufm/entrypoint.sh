#!/bin/sh

echo "Updating CA certificates."
update-ca-certificates
echo "Running CMD."
exec "$@"