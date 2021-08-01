#!/bin/bash

export postgres_host=`/opt/secret2env -name $SECRETNAME|grep -w postgres_host|sed 's/postgres_host=//g'`
export postgres_user=`/opt/secret2env -name $SECRETNAME|grep -w postgres_user|sed 's/postgres_user=//g'`
export postgres_pass=`/opt/secret2env -name $SECRETNAME|grep -w postgres_pass|sed 's/postgres_pass=//g'`
export postgres_db=`/opt/secret2env -name $SECRETNAME|grep -w postgres_db|sed 's/postgres_db=//g'`
sleep 10
start
