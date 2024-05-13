#!/bin/bash
set -e

main() {
    echo ""
    # 0xd7e14964d3D4A7F23442f09a02dAdF6BfDf5B3FC + 0x1111000000000000000000000000000000001111 = 0xe8f2...5C50D
    # https://sepolia.etherscan.io/address/0xe8f24964D3D4A7f23442f09A02dAdf6BFdf5C50D
    local aliased_address="0xe8f24964D3D4A7f23442f09A02dAdf6BFdf5C50D"

    echo "##########"
    echo "Deploy L2ProxyAdmin To op-sepolia L2"
    echo "##########"
    echo ""

    forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'run()' --rpc-url "$OP_SEPOLIA_RPC_URL" --broadcast --private-key "$DEPLOY_PRIVATE_KEY"

}

main "$@"