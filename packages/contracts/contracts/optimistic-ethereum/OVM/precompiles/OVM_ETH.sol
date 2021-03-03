// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";

/* Contract Imports */
import { OVM_L2DepositedERC20 } from "../bridge/tokens/OVM_L2DepositedERC20.sol";

/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that 
 * unlike on Layer 1, Layer 2 accounts do not have a balance field.
 * 
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ETH is OVM_L2DepositedERC20 {
    constructor(
        address _l2CrossDomainMessenger,
        address _l1ETHGateway
    ) 
        OVM_L2DepositedERC20(
            _l2CrossDomainMessenger,
            "ovmWETH",
            "oWETH"
        )
        public 
    {
        init(iOVM_L1TokenGateway(_l1ETHGateway));
    }
}
