// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

import { IDripCheck } from "../IDripCheck.sol";

/**
 * @title  FeeVault
 * @notice Minimal interface for the FeeVault
 */
interface FeeVault {
    /**
     * @notice Returns the minimal balance for a withdrawal to be possible.
     */
    function MIN_WITHDRAWAL_AMOUNT() external pure returns (uint256);

    /**
     * @notice Triggers a withdrawal.
     */
    function withdraw() external;
}

/**
 * @title  CheckFeeVault
 * @notice DripCheck for checking if the FeeVault has enough ether in it to be
 *         poked.
 */
contract CheckFeeVault is IDripCheck {
    struct Params {
        address target;
    }

    /**
     * @notice External event used to help client-side tooling encode parameters.
     *
     * @param params Parameters to encode.
     */
    event _EventToExposeStructInABI__Params(Params params);

    /**
     * @inheritdoc IDripCheck
     */
    function check(bytes memory _params) external view returns (bool) {
        Params memory params = abi.decode(_params, (Params));

        uint256 min = FeeVault(params.target).MIN_WITHDRAWAL_AMOUNT();

        // Check target balance is above threshold.
        return params.target.balance >= min;
    }
}
