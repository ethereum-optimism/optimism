// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { OVM_L1ERC20Gateway } from "@eth-optimism/contracts/build/contracts/OVM/bridge/tokens/OVM_L1ERC20Gateway.sol";
import { iOVM_ERC20 } from "@eth-optimism/contracts/build/contracts/iOVM/precompiles/iOVM_ERC20.sol";

/**
 * @title OVM_L1ERC20Gateway
 * @dev The L1 ERC20 Gateway is a contract which stores deposited L1 funds that are in use on L2.
 * It synchronizes a corresponding L2 ERC20 Gateway, informing it of deposits, and listening to it
 * for newly finalized withdrawals.
 *
 * This contract extends OVM_L1ERC20Gateway, which is where we
 * takes care of most of the initialization and the cross-chain logic.
 * If you are looking to implement your own deposit/withdrawal contracts, you
 * may also want to extend this contract in a similar manner.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract L1ERC20Gateway is OVM_L1ERC20Gateway {


    constructor(
        iOVM_ERC20 _l1ERC20,
        address _l2DepositedERC20,
        address _l1messenger
    )
        OVM_L1ERC20Gateway(
            _l1ERC20,
            _l2DepositedERC20,
            _l1messenger
        )
    {
    }

}
