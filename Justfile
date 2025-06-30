kurtosis-devnet-up: build kurtosis-spawn

kurtosis-devnet-down:
    kurtosis enclave rm -f op-rollup-boost
    kurtosis clean

kurtosis-stress-test:
    chmod +x ./scripts/ci/kurtosis.sh \
        && ./scripts/ci/kurtosis.sh run

# Build docker image (release)
build:
    docker buildx build --build-arg RELEASE=true -t flashbots/rollup-boost:develop .

# Build docker image (debug)
build-debug:
    docker buildx build --build-arg RELEASE=false -t flashbots/rollup-boost:develop .

kurtosis-spawn:
    # Pinning to this commit until https://github.com/ethpandaops/optimism-package/issues/349 is fixed
    kurtosis run github.com/ethpandaops/optimism-package@452133367b693e3ba22214a6615c86c60a1efd5e --args-file ./scripts/ci/kurtosis-params.yaml --enclave op-rollup-boost
