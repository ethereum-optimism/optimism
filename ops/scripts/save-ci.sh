#!/bin/bash
# Script for saving all the docker images in CI
declare -a images=(
    "deployer"
    "data-trasport-layer"
    "batch-submitter"
    "message-relayer"
    "integration-tests"
    "l2geth"
    "hardhat"
)

docker images

mkdir -p ./images

for image in "${images[@]}"
do
    docker save docker.io/ethereumoptimism/$image > ./images/$image.tar &
done

wait
