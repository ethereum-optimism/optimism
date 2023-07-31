// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

/// @title BlockHashOracle
/// @notice Stores a map of block numbers => block hashes for use in dispute resolution
contract BlockHashOracle {
    /// @notice Maps block numbers to block hashes
    mapping(uint256 => Hash) internal blockHashes;

    /// @notice Loads a block hash for a given block number, assuming that the block number
    ///         has been stored in the oracle.
    /// @param _blockNumber The block number to load the block hash for.
    /// @return blockHash_ The block hash for the given block number.
    function load(uint256 _blockNumber) external view returns (Hash blockHash_) {
        blockHash_ = blockHashes[_blockNumber];
        if (Hash.unwrap(blockHash_) == 0) revert BlockHashNotPresent();
    }

    /// @notice Stores a block hash for a given block number, assuming that the block number
    ///         is within the acceptable range of [tip - 256, tip].
    /// @param _blockNumber The block number to persist the block hash for.
    function store(uint256 _blockNumber) external {
        bytes32 blockHash = blockhash(_blockNumber);
        if (blockHash == 0) revert BlockNumberOOB();
        blockHashes[_blockNumber] = Hash.wrap(blockHash);
    }
}
