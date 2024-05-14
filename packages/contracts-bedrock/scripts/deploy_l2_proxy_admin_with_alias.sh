#!/bin/bash
set -e

main() {
    echo ""
    echo "##########"
    echo "Deploy L2ProxyAdmin To op-sepolia L2"
    echo "##########"
    echo ""

    forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'run()' --rpc-url "$OP_SEPOLIA_RPC_URL" --broadcast --private-key "$DEPLOY_PRIVATE_KEY"
    # forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'run()' --rpc-url "$OP_SEPOLIA_RPC_URL" --private-key "$DEPLOY_PRIVATE_KEY"

}

main "$@"