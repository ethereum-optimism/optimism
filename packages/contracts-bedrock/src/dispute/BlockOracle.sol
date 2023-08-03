// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

/// @title BlockOracle
/// @notice Stores a map of block numbers => block hashes for use in dispute resolution
contract BlockOracle {
    /// @notice The BlockInfo struct contains a block's hash and estimated timestamp.
    struct BlockInfo {
        Hash hash;
        Timestamp timestamp;
    }

    /// @notice Maps block numbers to block hashes and timestamps
    mapping(uint256 => BlockInfo) internal blocks;

    /// @notice Loads a block hash for a given block number, assuming that the block number
    ///         has been stored in the oracle.
    /// @param _blockNumber The block number to load the block hash and timestamp for.
    /// @return blockInfo_ The block hash and timestamp for the given block number.
    function load(uint256 _blockNumber) external view returns (BlockInfo memory blockInfo_) {
        blockInfo_ = blocks[_blockNumber];
        if (Hash.unwrap(blockInfo_.hash) == 0) revert BlockHashNotPresent();
    }

    /// @notice Stores a block hash for the previous block number, assuming that the block number
    ///         is within the acceptable range of [tip - 256, tip].
    function checkpoint() external returns (uint256 blockNumber_) {
        // Fetch the block hash for the given block number and revert if it is out of
        // the `BLOCKHASH` opcode's range.
        // SAFETY: This block hash will always be accessible by the `BLOCKHASH` opcode,
        //         and in the case of `block.number = 0`, we'll underflow.
        bytes32 blockHash = blockhash(blockNumber_ = block.number - 1);

        // Estimate the timestamp of the block assuming an average block time of 13 seconds.
        Timestamp estimatedTimestamp = Timestamp.wrap(uint64(block.timestamp - 13));

        // Persist the block information.
        blocks[blockNumber_] = BlockInfo({
            hash: Hash.wrap(blockHash),
            timestamp: estimatedTimestamp
        });
    }
}
