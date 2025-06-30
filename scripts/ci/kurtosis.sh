#!/bin/bash

set -exu

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

run() {
    # Note we use `rollup-boost` in combination with `op-geth-builder` as the JSON RPC servers to assert transaction relaying functionality
    # as well as inclusion of transactions that have only been sent to the builder (verifying the builder's payloads are being included in the canonical chain)

    # Execute 100 transfer transactions sent directly to the builder
    # Only if the builder is running and connected to the network with rollup-boost
    # the transactions will be included in the canonical blocks and finalized.

    # Figure out first the builder's JSON-RPC URL
    ROLLUP_BOOST_SOCKET=$(kurtosis port print op-rollup-boost op-mev-rollup-boost-2151908-op-kurtosis-0 rpc)
    OP_RETH_BUILDER_SOCKET=$(kurtosis port print op-rollup-boost op-el-builder-2151908-1-op-reth-op-node-op-kurtosis rpc)

    # Private key with prefunded balance
    PREFUNDED_PRIV_KEY=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
    PREFUNDED_ADDRESS=$(cast wallet address --private-key $PREFUNDED_PRIV_KEY)

    # Safe check to ensure there is balance in the prefunded address
    BALANCE=$(cast balance --rpc-url $ROLLUP_BOOST_SOCKET $PREFUNDED_ADDRESS)

    # We have to check for "0" string because it is returned in wei and the value is too big to compare as a number
    if [ "$BALANCE" = "0" ]; then
        echo "❌ Prefunded address has no balance"
        exit 1
    else
        echo "✅ Prefunded address has balance"
    fi

    # Deploy the contract with contender, this should be enough to check that the
    # builder is working as expected
    docker run --rm --network host -v /tmp/.contender:/root/.contender flashbots/contender:latest setup -p $PREFUNDED_PRIV_KEY scenario:stress.toml -r $ROLLUP_BOOST_SOCKET --optimism

    # Run the fill-block scenario on the builder
    docker run --rm --network host -v /tmp/.contender:/root/.contender flashbots/contender:latest spam --tps 50 --min-balance 0.2eth -p $PREFUNDED_PRIV_KEY -r $OP_RETH_BUILDER_SOCKET --optimism fill-block
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
    "run")
        run
        ;;
    "clean")
        clean
        ;;
    *)
        echo "Usage: $0 {install|deploy|run|clean}"
        echo "Commands:"
        echo "  install - Install Kurtosis CLI"
        echo "  deploy  - Deploy the Optimism package"
        echo "  run     - Run the Optimism package"
        echo "  clean   - Clean up the Kurtosis environment"
        exit 1
        ;;
esac
