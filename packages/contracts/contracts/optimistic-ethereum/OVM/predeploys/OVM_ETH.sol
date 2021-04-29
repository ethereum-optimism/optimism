// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Interface Imports */
import { iOVM_TokenGateway } from "../../iOVM/bridge/tokens/iOVM_TokenGateway.sol";

/* Contract Imports */
// import { OVM_L2DepositedERC20 } from "../bridge/tokens/OVM_L2TokenGateway.sol";
import { OVM_L2ERC20 } from "../../libraries/standards/OVM_L2ERC20.sol";

/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that
 * unlike on Layer 1, Layer 2 accounts do not have a balance field.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ETH is OVM_L2ERC20 {
    constructor(
        iOVM_TokenGateway _l2EthGateway
    )
        OVM_L2ERC20(
            "Ether",
            "ETH"
        )
    {
        // We immediately transfer ownership to the L2 Token Gateway for OVM_ETH
        // @todo: it's not so simple given this is already deployed
        transferOwnership(address(_l2EthGateway));
    }
}
