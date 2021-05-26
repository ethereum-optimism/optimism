#!/bin/bash
# Start Rinkeby services

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
DOCKERFILE="docker-compose.yml"

SERVICES=$(docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/docker-compose-rinkeby.env.yml \
    config --services \
    | tr '\n' ' ')

docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/docker-compose-rinkeby.env.yml \
    down -v --remove-orphans

docker-compose \
    -f $DIR/$DOCKERFILE \
    -f $DIR/docker-compose-rinkeby.env.yml \
    up $SERVICES
