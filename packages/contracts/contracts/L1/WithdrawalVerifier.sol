//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle } from "./L2OutputOracle.sol";

/**
 * @title WithdrawalVerifier
 */
contract WithdrawalVerifier {
    L2OutputOracle public immutable l2oracle;
    address public immutable withdrawalsPredeploy;

    constructor(L2OutputOracle _l2Oracle, address _withdrawalsPredeploy) {
        l2oracle = _l2Oracle;
        withdrawalsPredeploy = _withdrawalsPredeploy;
    }

    // function verifyWithdrawal(// WithdrawalMessage message,
    // L2OutputTimestamp timestamp,
    // WithdrawalsRootInclusionProof storageRootProof,
    // WithdrawalMessageInclusionProof messageProof
    // )
    // {

    // }

}
