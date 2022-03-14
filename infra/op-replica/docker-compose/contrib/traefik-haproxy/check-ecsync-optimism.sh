#!/bin/sh
MAX_LAG=2
# Assumes that servers are labeled -a, -b, -c etc and contain their domain name, and that corresponding
# sequencer health hosts exist
HEALTH_HOST=optimismhealth

__domain=$(echo -n $HAPROXY_SERVER_NAME | cut -d '.' -f2-)
__host=$(echo -n $HAPROXY_SERVER_NAME | cut -d '.' -f1)
__dashcount=$(echo -n $__host | grep -o "-" | wc -w)
__suffix=$(echo -n $__host | cut -d '-' -f$(($__dashcount+1)))

__sequencerheight=$(curl -s -m2 -N -L "$HEALTH_HOST-$__suffix.$__domain/metrics" | grep  ^replica_health_sequencer_height | cut -d' ' -f2)
__replicaheight=$(curl -s -m2 -N -L "$HEALTH_HOST-$__suffix.$__domain/metrics" | grep ^replica_health_height | cut -d' ' -f2)
if [ -z "$__sequencerheight" -o -z "$__replicaheight" ]; then
  exit 1
fi
__distance=$(expr $__sequencerheight - $__replicaheight)
if [ $__distance -le $MAX_LAG ]; then
  exit 0
else
  exit 1
fi

