#!/bin/bash
CONTAINER=l2geth

until docker-compose logs l2geth | grep -q "Starting Sequencer Loop";
do
    sleep 3
    echo 'Waiting for sequencer...'
done
