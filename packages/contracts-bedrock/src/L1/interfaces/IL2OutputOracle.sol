// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { Types } from "src/libraries/Types.sol";

/// @title IL2OutputOracle
/// @notice Interface for the L2OutputOracle contract.
interface IL2OutputOracle is ISemver {
    /// @notice Emitted when an output is proposed.
    /// @param outputRoot    The output root.
    /// @param l2OutputIndex The index of the output in the l2Outputs array.
    /// @param l2BlockNumber The L2 block number of the output root.
    /// @param l1Timestamp   The L1 timestamp when proposed.
    event OutputProposed(
        bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp
    );

    /// @notice Emitted when outputs are deleted.
    /// @param prevNextOutputIndex Next L2 output index before the deletion.
    /// @param newNextOutputIndex  Next L2 output index after the deletion.
    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    function SUBMISSION_INTERVAL() external view returns (uint256);
    function L2_BLOCK_TIME() external view returns (uint256);
    function CHALLENGER() external view returns (address);
    function PROPOSER() external view returns (address);
    function FINALIZATION_PERIOD_SECONDS() external view returns (uint256);
    function deleteL2Outputs(uint256 _l2OutputIndex) external;
    function proposeL2Output(
        bytes32 _outputRoot,
        uint256 _l2BlockNumber,
        bytes32 _l1BlockHash,
        uint256 _l1BlockNumber
    )
        external
        payable;
    function getL2Output(uint256 _l2OutputIndex) external view returns (Types.OutputProposal memory);
    function getL2OutputAfter(uint256 _l2BlockNumber) external view returns (Types.OutputProposal memory);
    function latestOutputIndex() external view returns (uint256);
}
