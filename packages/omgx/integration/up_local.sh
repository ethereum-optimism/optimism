#!/bin/bash
# Run everything except the integration tests

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"
DOCKERFILE="docker-compose-local.yml"

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