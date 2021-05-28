#!/bin/bash
# Script for loading all the docker images in CI
declare -a images=(
    "deployer"
    "data-transport-layer"
    "batch-submitter"
    "message-relayer"
    "integration-tests"
    "l2geth"
    "hardhat"
)

## now loop through the above array
for image in "${images[@]}"
do
    docker load --input /tmp/images/$image.tar &
done

wait
