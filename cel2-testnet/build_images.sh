#!/bin/bash
set -eo pipefail

cd docker
export PATH=$(go1.19 env GOROOT)/bin:$PATH
DOCKER_DEFAULT_PLATFORM=linux/amd64 docker-compose build --progress plain
