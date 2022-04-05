//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { DepositFeed } from "./DepositFeed.sol";
import { WithdrawalVerifier } from "./WithdrawalVerifier.sol";
import { L2OutputOracle } from "./L2OutputOracle.sol";


contract OptimismPortal is DepositFeed, WithdrawalVerifier {
    constructor(L2OutputOracle _l2Oracle, uint256 _finalizationWindow)
        WithdrawalVerifier(_l2Oracle, _finalizationWindow)
    {}


    /**
     * Accepts value so that users can send ETH directly to this contract and
     * have the funds be deposited to their address on L2.
     * Note: this is intended as a convenience function for EOAs. Contracts should call the
     * depositTransaction() function directly.
     */
    receive() external payable {
        depositTransaction(msg.sender, msg.value, 30000, false, bytes(""));
    }
}
