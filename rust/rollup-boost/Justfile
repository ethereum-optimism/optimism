# Build docker image (release)
build:
    docker buildx build --build-arg RELEASE=true -t flashbots/rollup-boost:develop .

# Build docker image (debug)
build-debug:
    docker buildx build --build-arg RELEASE=false -t flashbots/rollup-boost:develop .

clippy:
    cargo clippy --workspace -- -D warnings

fmt:
    cargo fmt --all

test:
    cargo nextest run --workspace
