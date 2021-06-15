// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";

/* Contract Imports */
import { UniswapV2ERC20 } from "../libraries/standards/UniswapV2ERC20.sol";

/* Library Imports */
import { OVM_L2DepositedERC20 } from "../OVM/bridge/tokens/OVM_L2DepositedERC20.sol";

/**
 * @title OVM_L2DepositedERC20
 * @dev The L2 Deposited ERC20 is an ERC20 implementation which represents L1 assets deposited into L2.
 * This contract mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
 * NOTE: This contract implements the Abs_L2DepositedToken contract using Uniswap's ERC20 as the implementation.
 * Alternative implementations can be used in this similar manner.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract MVM_Coinbase is OVM_L2DepositedERC20 {
    constructor(
        address _l2CrossDomainMessenger,
        address _l1ETHGateway
    ) 
        OVM_L2DepositedERC20(
            _l2CrossDomainMessenger,
            "Metis Token",
            "Metis"
        )
    {
        init(iOVM_L1TokenGateway(_l1ETHGateway));
    }
}
