# Build docker image (release)
build:
    docker buildx build --build-arg RELEASE=true -t flashbots/rollup-boost:develop .

# Build docker image (debug)
build-debug:
    docker buildx build --build-arg RELEASE=false -t flashbots/rollup-boost:develop .