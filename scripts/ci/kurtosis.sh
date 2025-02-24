#!/bin/bash

install() {
    # https://docs.kurtosis.com/install/
    echo "Installing Kurtosis..."
    
    echo "deb [trusted=yes] https://apt.fury.io/kurtosis-tech/ /" | sudo tee /etc/apt/sources.list.d/kurtosis.list
    sudo apt update
    sudo apt install -y kurtosis-cli
    
    # Validate installation
    if which kurtosis > /dev/null; then
        echo "✅ Kurtosis installation completed and verified!"
    else
        echo "❌ Kurtosis installation failed. 'kurtosis' command not found in PATH"
        return 1
    fi
}

install_contender() {
    cargo install --git https://github.com/flashbots/contender --bin contender --force
}

deploy() {
    # Run the kurtosis optimism package
    echo "Running Kurtosis..."
    kurtosis analytics disable
    kurtosis run github.com/ethpandaops/optimism-package --args-file ./scripts/ci/kurtosis-params.yaml
}

run() {
    # Execute 100 transfer transactions sent directly to the builder
    # Only if the builder is running and connected to the network with rollup-boost
    # the transactions will be included in the canonical blocks and finalized.

    # Figure out first the builder's JSON-RPC URL
    L2_PORT=$(docker inspect --format='{{(index .NetworkSettings.Ports "8545/tcp" 0).HostPort}}' $(docker ps --filter "name=op-el-builder-" -q))
    
    # Private key with prefunded balance
    PREFUNDED_PRIV_KEY=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
    PREFUNDED_ADDRESS=$(cast wallet address --private-key $PREFUNDED_PRIV_KEY)

    # Safe check to ensure there is balance in the prefunded address
    BALANCE=$(cast balance --rpc-url http://localhost:$L2_PORT $PREFUNDED_ADDRESS)

    # We have to check for "0" string because it is returned in wei and the value is too big to compare as a number
    if [ "$BALANCE" = "0" ]; then
        echo "❌ Prefunded address has no balance"
        exit 1
    else
        echo "✅ Prefunded address has balance"
    fi
    
    # Download the scenario for contender
    wget https://raw.githubusercontent.com/flashbots/contender/refs/heads/main/scenarios/stress.toml -O /tmp/scenario.toml

    # Deploy the contract with contender, this should be enough to check that the
    # builder is working as expected
    # I have not managed to send a working transaction with cast yet. That should replace
    # this setup eventually since we are only testing if a single transaction is included
    # in the canonical chain.
    contender setup -p $PREFUNDED_PRIV_KEY /tmp/scenario.toml http://localhost:$L2_PORT
}

clean() {
    # Clean up the kurtosis environment
    echo "Cleaning up..."
    kurtosis clean -a
}

# Main execution block
case "$1" in
    "install")
        install
        ;;
    "install-contender")
        install_contender
        ;;
    "deploy")
        deploy
        ;;
    "run")
        run
        ;;
    "clean")
        clean
        ;;
    *)
        echo "Usage: $0 {install|install-contender|deploy|run|clean}"
        echo "Commands:"
        echo "  install - Install Kurtosis CLI"
        echo "  install-contender - Install Contender"
        echo "  deploy  - Deploy the Optimism package"
        echo "  run     - Run the Optimism package"
        echo "  clean   - Clean up the Kurtosis environment"
        exit 1
        ;;
esac
