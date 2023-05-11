// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IL2OutputOracle } from "../../dispute/IL2OutputOracle.sol";

/**
 * @title MockL2OutputOracle
 * @notice A mock contract for the L2OutputOracle contract.
 */
contract MockL2OutputOracle is IL2OutputOracle {
    /**
     * @notice Deletes all output proposals after and including the proposal that corresponds to
     *         the given output index. Only the challenger address can delete outputs.
     *
     * @param _l2OutputIndex Index of the first L2 output to be deleted. All outputs after this
     *                       output will also be deleted.
     */
    function deleteL2Outputs(uint256 _l2OutputIndex) external {
        // Do nothing
    }
}
