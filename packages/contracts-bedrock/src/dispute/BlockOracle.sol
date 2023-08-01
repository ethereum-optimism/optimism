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
    mapping(uint256 => BlockInfo) internal blockHashes;

    /// @notice Loads a block hash for a given block number, assuming that the block number
    ///         has been stored in the oracle.
    /// @param _blockNumber The block number to load the block hash and timestamp for.
    /// @return blockInfo_ The block hash and timestamp for the given block number.
    function load(uint256 _blockNumber) external view returns (BlockInfo memory blockInfo_) {
        blockInfo_ = blockHashes[_blockNumber];
        if (Hash.unwrap(blockInfo_.hash) == 0) revert BlockHashNotPresent();
    }

    /// @notice Stores a block hash for a given block number, assuming that the block number
    ///         is within the acceptable range of [tip - 256, tip].
    /// @param _blockNumber The block number to persist the block hash for.
    function store(uint256 _blockNumber) external {
        // Fetch the block hash for the given block number and revert if it is out of
        // the `BLOCKHASH` opcode's range.
        bytes32 blockHash = blockhash(_blockNumber);
        if (blockHash == 0) revert BlockNumberOOB();

        // Estimate the timestamp of the block assuming an average block time of 13 seconds.
        Timestamp estimatedTimestamp = Timestamp.wrap(
            uint64(block.timestamp - ((block.number - _blockNumber) * 13))
        );

        // Persist the block information.
        blockHashes[_blockNumber] = BlockInfo({
            hash: Hash.wrap(blockHash),
            timestamp: estimatedTimestamp
        });
    }
}
