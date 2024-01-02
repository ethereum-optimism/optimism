// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @title BlockOracle
/// @notice Stores a map of block numbers => block hashes for use in dispute resolution
contract BlockOracle is ISemver {
    /// @notice The BlockInfo struct contains a block's hash and child timestamp.
    struct BlockInfo {
        Hash hash;
        Timestamp childTimestamp;
    }

    /// @notice Emitted when a block is checkpointed.
    event Checkpoint(uint256 indexed blockNumber, Hash indexed blockHash, Timestamp indexed childTimestamp);

    /// @notice Maps block numbers to block hashes and timestamps
    mapping(uint256 => BlockInfo) internal blocks;

    /// @notice Semantic version.
    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    /// @notice Loads a block hash for a given block number, assuming that the block number
    ///         has been stored in the oracle.
    /// @param _blockNumber The block number to load the block hash and timestamp for.
    /// @return blockInfo_ The block hash and timestamp for the given block number.
    function load(uint256 _blockNumber) external view returns (BlockInfo memory blockInfo_) {
        blockInfo_ = blocks[_blockNumber];
        if (Hash.unwrap(blockInfo_.hash) == 0) revert BlockHashNotPresent();
    }

    /// @notice Stores a block hash for the previous block number.
    /// @return blockNumber_ The block number that was checkpointed, which is always
    ///                      `block.number - 1`.
    function checkpoint() external returns (uint256 blockNumber_) {
        // SAFETY: This block hash will always be accessible by the `BLOCKHASH` opcode,
        //         and in the case of `block.number = 0`, we'll underflow.
        // Persist the block information.
        blockNumber_ = block.number - 1;
        Hash blockHash = Hash.wrap(blockhash(blockNumber_));
        Timestamp childTimestamp = Timestamp.wrap(uint64(block.timestamp));

        blocks[blockNumber_] = BlockInfo({ hash: blockHash, childTimestamp: childTimestamp });

        emit Checkpoint(blockNumber_, blockHash, childTimestamp);
    }
}
