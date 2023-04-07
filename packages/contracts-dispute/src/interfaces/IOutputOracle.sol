// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title L2OutputOracle Interface
/// @notice An interface for the L2OutputOracle contract.
interface IOutputOracle {
    /// @notice Deletes the L2 output for the given parameter.
    function deleteL2Outputs(uint256) external;

    /// @notice OutputProposal represents a commitment to the L2 state. The timestamp is the L1
    ///         timestamp that the output root is posted. This timestamp is used to verify that the
    ///         finalization period has passed since the output root was submitted.
    /// @custom:field outputRoot    Hash of the L2 output.
    /// @custom:field timestamp     Timestamp of the L1 block that the output root was submitted in.
    /// @custom:field l2BlockNumber L2 block number that the output corresponds to.
    struct OutputProposal {
        bytes32 outputRoot;
        uint128 timestamp;
        uint128 l2BlockNumber;
    }

    /// @notice Returns the L2 output for the given parameter.
    function getL2Output(uint256 _l2OutputIndex) external view returns (OutputProposal memory);
}
