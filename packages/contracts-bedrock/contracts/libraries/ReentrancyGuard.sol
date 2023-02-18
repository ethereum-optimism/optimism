// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/**
 * @title ReentrancyGuard
 * @notice A contract that provides custom reentrancy guard modifiers.
 */
contract ReentrancyGuard {
    /**
     * @notice Modifier for a per-message reentrancy guard.
     */
    modifier perMessageNonReentrant(bytes32 _msgHash) {
        bytes32 _hashMsgHash;
        assembly {
            // Re-hash the `_msgHash` with the `0xcafebabe` salt to reduce the possibility
            // of collisions with existing storage slots.
            mstore(0x00, _msgHash)
            mstore(0x20, 0xcafebabe)
            _hashMsgHash := keccak256(0x00, 0x40)

            // Check if the reentrancy lock for the `_msgHash` is set. If so, revert.
            if sload(_hashMsgHash) {
                // MEMORY SAFETY: We're reverting, so it's fine that we're clobbering the free
                // memory pointer.

                // Store selector for "Error(string)" in scratch space
                mstore(0x00, 0x08c379a0)
                // Store pointer to the string in scratch space
                mstore(0x20, 0x20)
                // Add the length of the "ReentrancyGuard: reentrant call" string (31 bytes)
                mstore(0x40, 0x1f)
                // Store "ReentrancyGuard: reentrant call" in the zero slot
                // (plus a 0 byte for padding)
                mstore(0x60, 0x5265656e7472616e637947756172643a207265656e7472616e742063616c6c00)
                // Revert with 'Error("ReentrancyGuard: reentrant call")'
                revert(0x1c, 0x64)
            }
            // Trigger the reentrancy lock for `_msgHash`.
            sstore(_hashMsgHash, 0x01)
        }
        _;
        assembly {
            // Clear the reentrancy lock for `_msgHash`
            sstore(_hashMsgHash, 0x00)
        }
    }
}
