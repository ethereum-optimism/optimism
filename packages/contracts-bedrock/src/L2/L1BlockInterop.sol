// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L1Block } from "src/L2/L1Block.sol";

/// @notice Thrown when a non-depositor account attempts to set L1 block values.
error NotDepositor();

/// @notice Thrown when dependencySetSize does not match the length of the dependency set.
error DependencySetSizeMismatch();

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1BlockInterop
/// @notice Interop extenstions of L1Block.
contract L1BlockInterop is L1Block {
    /// @notice The chain IDs of the interop dependency set.
    uint256[] public dependencySet;

    /// @custom:semver 1.3.0+interop
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+interop");
    }

    /// @notice Updates the L1 block values for an Interop upgraded chain.
    /// Params are packed and passed in as raw msg.data instead of ABI to reduce calldata size.
    /// Params are expected to be in the following order:
    ///   1. _baseFeeScalar      L1 base fee scalar
    ///   2. _blobBaseFeeScalar  L1 blob base fee scalar
    ///   3. _sequenceNumber     Number of L2 blocks since epoch start.
    ///   4. _timestamp          L1 timestamp.
    ///   5. _number             L1 blocknumber.
    ///   6. _basefee            L1 base fee.
    ///   7. _blobBaseFee        L1 blob base fee.
    ///   8. _hash               L1 blockhash.
    ///   9. _batcherHash        Versioned hash to authenticate batcher by.
    ///  10. _dependencySetSize  Size of the interop dependency set.
    ///  11. _dependencySet      Array of chain IDs for the interop dependency set.
    function setL1BlockValuesInterop() external {
        assembly {
            // Revert if the caller is not the depositor account.
            if xor(caller(), DEPOSITOR_ACCOUNT) {
                mstore(0x00, 0x3cc50b45) // 0x3cc50b45 is the 4-byte selector of "NotDepositor()"
                revert(0x1C, 0x04) // returns the stored 4-byte selector from above
            }
            // sequencenum (uint64), blobBaseFeeScalar (uint32), baseFeeScalar (uint32)
            sstore(sequenceNumber.slot, shr(128, calldataload(4)))
            // number (uint64) and timestamp (uint64)
            sstore(number.slot, shr(128, calldataload(20)))
            sstore(basefee.slot, calldataload(36)) // uint256
            sstore(blobBaseFee.slot, calldataload(68)) // uint256
            sstore(hash.slot, calldataload(100)) // bytes32
            sstore(batcherHash.slot, calldataload(132)) // bytes32

            // Load dependencySetSize from calldata (at offset 164 after calldata for setL1BlockValuesEcotone ends)
            let dependencySetSize_ := shr(248, calldataload(164))

            // Revert if dependencySetSize_ doesn't match the length of dependencySet in calldata
            if xor(add(165, mul(dependencySetSize_, 0x20)), calldatasize()) {
                mstore(0x00, 0x44165b6a) // 0x44165b6a is the 4-byte selector of "DependencySetSizeMismatch()"
                revert(0x1C, 0x04) // returns the stored 4-byte selector from above
            }

            // Use memory to hash and get the start index of dependencySet
            mstore(0x00, dependencySet.slot)
            let dependencySetStartIndex := keccak256(0x00, 0x20)

            // Iterate over calldata dependencySet and write to store dependencySet
            for { let i := 0 } lt(i, dependencySetSize_) { i := add(i, 1) } {
                // Load value from calldata and write to storage (dependencySet) at index
                let val := calldataload(add(165, mul(i, 0x20)))
                sstore(add(dependencySetStartIndex, i), val)
            }

            // Update length of dependencySet array
            sstore(dependencySet.slot, dependencySetSize_)
        }
    }

    /// @notice Returns true if a chain ID is in the interop dependency set and false otherwise.
    ///         Every chain ID is in the interop dependency set of itself.
    /// @param _chainId The chain ID to check.
    /// @return True if the chain ID to check is in the interop dependency set. False otherwise.
    function isInDependencySet(uint256 _chainId) public view returns (bool) {
        // Every chain ID is in the interop dependency set of itself.
        if (_chainId == block.chainid) {
            return true;
        }

        uint256 length = dependencySet.length;
        for (uint256 i = 0; i < length;) {
            if (dependencySet[i] == _chainId) {
                return true;
            }
            unchecked {
                i++;
            }
        }

        return false;
    }

    /// @notice Returns the size of the interop dependency set.
    /// @return The size of the interop dependency set.
    function dependencySetSize() external view returns (uint8) {
        return uint8(dependencySet.length);
    }
}
