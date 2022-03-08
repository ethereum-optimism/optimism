# Builds an image using Buildx. Usage:
# build <name> <tag> <dockerfile> <context>
function build() {
    echo "Building $1."
    echo "Tag: $2"
    echo "Dockerfile: $3"
    echo "Context: $4"
    docker buildx build \
        --tag "$2" \
        --cache-from "type=local,src=/tmp/.buildx-cache/$1" \
        --cache-to="type=local,dest=/tmp/.buildx-cache-new/$1" \
        --file "$3" \
        --load "$4" \
        &
}

mkdir -p /tmp/.buildx-cache-new
build l2geth "ethereumoptimism/l2geth:latest" "./ops/docker/Dockerfile.geth" .
build l1chain "ethereumoptimism/hardhat:latest" "./ops/docker/hardhat/Dockerfile" ./ops/docker/hardhat

wait

build deployer "ethereumoptimism/deployer:latest" "./ops/docker/Dockerfile.deployer" .
build dtl "ethereumoptimism/data-transport-layer:latest" "./ops/docker/Dockerfile.data-transport-layer" .
build relayer "ethereumoptimism/message-relayer:latest" "./ops/docker/Dockerfile.message-relayer" .
build integration-tests "ethereumoptimism/integration-tests:latest" "./ops/docker/Dockerfile.integration-tests" .

wait
