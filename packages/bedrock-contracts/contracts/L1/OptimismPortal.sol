//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Inherited Imports */
import { DepositFeed } from "./abstracts/DepositFeed.sol";
import { WithdrawalsRelay } from "./abstracts/WithdrawalsRelay.sol";

/* Interactions Imports */
import { L2OutputOracle } from "./L2OutputOracle.sol";

/**
 * @title OptimismPortal
 * @notice The OptimismPortal is a contract on L1 used to deposit and withdraw between L2 and L1.
 * The OptimismPortal must inherit from both the DepositFeed and WithdrawalsRelay as it holds the
 * pool of ETH which is deposited to and withdrawn from L2. Aside from affecting the ETH balance,
 * the deposit and withdrawal codepaths should be independent from one another.
 */
contract OptimismPortal is DepositFeed, WithdrawalsRelay {
    /***************
     * Constructor *
     ***************/

    constructor(L2OutputOracle _l2Oracle, uint256 _finalizationPeriod)
        WithdrawalsRelay(_l2Oracle, _finalizationPeriod)
    {}

    /**********************
     * External Functions *
     **********************/

    /**
     * @notice Accepts value so that users can send ETH directly to this contract and
     * have the funds be deposited to their address on L2.
     * @dev This is intended as a convenience function for EOAs. Contracts should call the
     * depositTransaction() function directly.
     */
    receive() external payable {
        depositTransaction(msg.sender, msg.value, 30000, false, bytes(""));
    }
}
