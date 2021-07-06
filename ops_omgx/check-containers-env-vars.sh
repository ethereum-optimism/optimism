#!/usr/bin/env bash
SERVICE_IMAGES=$(docker ps|grep omgx|awk '{print $2}'|tr '\n' ' ')
#COMMAND=$(docker ps --no-trunc|grep $SERVICE|awk '{print $3}'|sed 's#"##g')

for srv in $SERVICE_IMAGES; do

  CONTAINER_ID=$(docker ps|grep $SERVICE|awk '{print $1}')
  PIDS=$(docker exec -it $CONTAINER_ID ps xua|egrep -v 'ps|PID'|awk '{print $2}'|tr '\n' ' ')
  echo "\n IMAGE: $SERVICE \n ---------"
  for PID in $PIDS; do
    ENVIRON=$(docker exec -it $CONTAINER_ID cat /proc/$PID/environ|tr '\0' '\n' )
    echo "ENVIRONMENT-VARIABLES: \n $ENVIRON \n -------- \n"
  done
done
