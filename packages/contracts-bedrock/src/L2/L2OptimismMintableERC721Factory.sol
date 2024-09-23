// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// TODO: implement this
contract L2OptimismMintableERC721Factory {
    function bridge() external view override returns (address) {
        return address(0);
    }

    function BRIDGE() external view override returns (address) {
        return address(0);
    }

    function remoteChainId() external view override returns (uint256) {
        return 0;
    }

    function REMOTE_CHAIN_ID() external view override returns (uint256) {
        return 0;
    }
}
