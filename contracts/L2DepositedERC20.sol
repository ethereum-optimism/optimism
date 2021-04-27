// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { OVM_L2DepositedERC20 } from "@eth-optimism/contracts/build/contracts/OVM/bridge/tokens/OVM_L2DepositedERC20.sol";

/**
 * @title L2DepositedERC20
 * @dev The L2 Deposited ERC20 is an ERC20 implementation which represents L1 assets deposited into L2.
 * This contract mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract L2DepositedERC20 is OVM_L2DepositedERC20 {

    constructor(
        address _l2CrossDomainMessenger,
        string memory _name,
        string memory _symbol
    )
        OVM_L2DepositedERC20(_l2CrossDomainMessenger, _name, _symbol)
    {}

}
