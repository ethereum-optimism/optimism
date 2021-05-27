#!/bin/bash
# Script for loading all the docker images in CI
declare -a images=(
    "deployer"
    "data-trasport-layer"
    "batch-submitter"
    "message-relayer"
    "integration-tests"
    "l2geth"
    "hardhat"
)

## now loop through the above array
for image in "${images[@]}"
do
    docker load --input ethereumoptimism/$image.tar &
done

wait
