// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

import { IDripCheck } from "../IDripCheck.sol";

/**
 * @title CheckBalanceHigh
 * @notice DripCheck for checking if an account's balance is above a given threshold.
 */
contract CheckBalanceHigh is IDripCheck {
    struct Params {
        address target;
        uint256 threshold;
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

        // Check target balance is above threshold.
        return params.target.balance > params.threshold;
    }
}
