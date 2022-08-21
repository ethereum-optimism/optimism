// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IDripCheck } from "../IDripCheck.sol";

/**
 * @title CheckBalanceLow
 * @notice DripCheck for checking if an account's balance is below a given threshold.
 */
contract CheckBalanceLow is IDripCheck {
    struct Params {
        address target;
        uint256 threshold;
    }

    event _EventToExposeStructInABI__Params(Params params);

    function check(bytes memory _params) external view returns (bool) {
        Params memory params = abi.decode(_params, (Params));

        // Check target ETH balance is below threshold.
        return params.target.balance < params.threshold;
    }
}
