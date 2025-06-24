devnet-up: build-debug spawn

devnet-down:
    kurtosis enclave rm -f op-rollup-boost
    kurtosis clean

stress-test:
    chmod +x ./scripts/ci/stress.sh \
        && ./scripts/ci/stress.sh run

# Build docker image (release)
build:
    docker buildx build --build-arg RELEASE=true -t flashbots/rollup-boost:develop .

# Build docker image (debug)
build-debug:
    docker buildx build --build-arg RELEASE=false -t flashbots/rollup-boost:develop .

spawn:
    kurtosis run github.com/ethpandaops/optimism-package --args-file ./scripts/ci/kurtosis-params.yaml --enclave op-rollup-boost