#!/bin/bash
# Start Rinkeby services

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
DOCKERFILE="docker-compose.yml"

SERVICES=$(docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/docker-compose-local.env.yml \
    config --services \
    | tr '\n' ' ')

docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/docker-compose-local.env.yml \
    down -v --remove-orphans

docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/docker-compose-local.env.yml \
    up $SERVICES
