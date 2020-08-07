#!/bin/sh
# wait-for-nodes.sh

set -e

cmd="$@"

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

wait_for_server_to_be_reachable $L1_NODE_WEB3_URL
wait_for_server_to_be_reachable $L2_NODE_WEB3_URL

>&2 echo "Dependent servers are up - executing command"
exec $cmd
