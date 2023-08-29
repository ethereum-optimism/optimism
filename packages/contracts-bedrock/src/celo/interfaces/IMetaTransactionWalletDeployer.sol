// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IMetaTransactionWalletDeployer {
    function deploy(address, address, bytes calldata) external;
}
