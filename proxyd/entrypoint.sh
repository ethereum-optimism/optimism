#!/bin/sh

ulimit -Sn 65000
echo "Updating CA certificates."
update-ca-certificates
echo "Running CMD."
exec "$@"
