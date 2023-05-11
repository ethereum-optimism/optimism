// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @title IL2OutputOracle
 * @notice A minimal interface for the L2OutputOracle contract.
 */
interface IL2OutputOracle {
    /**
     * @notice Deletes all output proposals after and including the proposal that corresponds to
     *         the given output index. Only the challenger address can delete outputs.
     *
     * @param _l2OutputIndex Index of the first L2 output to be deleted. All outputs after this
     *                       output will also be deleted.
     */
    function deleteL2Outputs(uint256 _l2OutputIndex) external;
}
