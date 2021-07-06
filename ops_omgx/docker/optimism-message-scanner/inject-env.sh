#!/bin/bash
if [ -f "/opt/secret2env" ]; then
      BRANCH_NAME=`cat /branch_name`
      ENVIRON=`/opt/secret2env -name $BRANCH_NAME`
      sed -i "2s#^#$ENVIRON#" /opt/wait-for-l1-and-l2.sh
fi
cmd="$@"
$cmd
