# Builds an image using Buildx. Usage:
# build <name> <tag> <dockerfile> <context>
function build() {
    echo "Building $1."
    echo "Tag: $2"
    echo "Dockerfile: $3"
    echo "Context: $4"
    docker buildx build \
        --tag "$2" \
        --build-arg LOCAL_REGISTRY=localhost:5000 \
        --cache-from "type=local,src=/tmp/.buildx-cache/$1" \
        --cache-to="type=local,dest=/tmp/.buildx-cache-new/$1" \
        --file "$3" \
        --load "$4" \
        &
}

mkdir -p /tmp/.buildx-cache-new
docker buildx build --tag "localhost:5000/ethereumoptimism/builder:latest" --cache-from "type=local,src=/tmp/.buildx-cache/builder" --cache-to="type=local,mode=max,dest=/tmp/.buildx-cache-new/builder" --file "./ops/docker/Dockerfile.monorepo" --push . &
build l2geth "ethereumoptimism/l2geth:latest" "./ops/docker/Dockerfile.geth" .
build l1chain "ethereumoptimism/hardhat:latest" "./ops/docker/hardhat/Dockerfile" ./ops/docker/hardhat

wait

# BuildX builds everything in a container when docker-container is selected as
# the backend. Unfortunately, this means that the built image must be pushed
# then re-pulled in order to make the container accessible to the Docker daemon.
# We have to use the docker-container backend since the the docker backend does
# not support cache-from and cache-to.
docker pull localhost:5000/ethereumoptimism/builder:latest

# Re-tag the local registry version of the builder so that docker-compose and
# friends can see it.
docker tag localhost:5000/ethereumoptimism/builder:latest ethereumoptimism/builder:latest

build deployer "ethereumoptimism/deployer:latest" "./ops/docker/Dockerfile.deployer" .
build dtl "ethereumoptimism/data-transport-layer:latest" "./ops/docker/Dockerfile.data-transport-layer" .
build batch_submitter "ethereumoptimism/batch-submitter:latest" "./ops/docker/Dockerfile.batch-submitter" .
build relayer "ethereumoptimism/message-relayer:latest" "./ops/docker/Dockerfile.message-relayer" .
build integration-tests "ethereumoptimism/integration-tests:latest" "./ops/docker/Dockerfile.integration-tests" .

wait
