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

for image in "${images[@]}"
do
    docker save ethereumoptimism/$image > /images/$image.tar &
done

wait
