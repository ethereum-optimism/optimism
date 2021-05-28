#!/bin/bash
# Script for saving all the docker images in CI
declare -a images=(
    "deployer"
    "data-transport-layer"
    "batch-submitter"
    "message-relayer"
    "integration-tests"
    "l2geth"
    "hardhat"
    "builder"
)

mkdir -p /tmp/images

for image in "${images[@]}"
do
    docker save ethereumoptimism/$image > /tmp/images/$image.tar &
done

wait
