devnet-up: build spawn

devnet-down:
    kurtosis enclave rm -f op-rollup-boost
    kurtosis clean

stress-test:
    chmod +x ./scripts/ci/stress.sh \
        && ./scripts/ci/stress.sh run

build:
    docker buildx build -t flashbots/rollup-boost:develop .

spawn:
    kurtosis run github.com/ethpandaops/optimism-package --args-file ./scripts/ci/kurtosis-params.yaml --enclave op-rollup-boost