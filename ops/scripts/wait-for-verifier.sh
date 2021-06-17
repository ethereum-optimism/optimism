#!/bin/bash
CONTAINER=l2geth

RETRIES=30
i=0
until docker-compose logs verifier | grep -q "Starting Verifier Loop";
do
    sleep 3
    if [ $i -eq $RETRIES ]; then
        echo 'Timed out waiting for verifier'
        break
    fi
    echo 'Waiting for verifier...'
    ((i=i+1))
done
