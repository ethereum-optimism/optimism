// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title Transient
/// @notice Library for transient storage.
library TransientContext {
    /// @notice Slot for call depth.
    ///         Equal to bytes32(uint256(keccak256("transient.calldepth")) - 1).
    bytes32 internal constant CALL_DEPTH_SLOT = 0x7a74fd168763fd280eaec3bcd2fd62d0e795027adc8183a693c497a7c2b10b5c;

    /// @notice Gets the call depth.
    /// @return _callDepth Current call depth.
    function callDepth() internal view returns (uint256 _callDepth) {
        assembly {
            _callDepth := tload(CALL_DEPTH_SLOT)
        }
    }

    /// @notice Gets value in transient storage for a slot at the current call depth.
    /// @param _slot Slot to get.
    /// @return _value Transient value.
    function get(bytes32 _slot) internal view returns (uint256 _value) {
        assembly {
            mstore(0, tload(CALL_DEPTH_SLOT))
            mstore(32, _slot)
            _value := tload(keccak256(0, 64))
        }
    }

    /// @notice Sets a value in transient storage for a slot at the current call depth.
    /// @param _slot    Slot to set.
    /// @param _value   Value to set.
    function set(bytes32 _slot, uint256 _value) internal {
        assembly {
            mstore(0, tload(CALL_DEPTH_SLOT))
            mstore(32, _slot)
            tstore(keccak256(0, 64), _value)
        }
    }

    /// @notice Increments call depth.
    function increment() internal {
        assembly {
            tstore(CALL_DEPTH_SLOT, add(tload(CALL_DEPTH_SLOT), 1))
        }
    }

    /// @notice Decrements call depth.
    ///         Reverts if call depth is zero.
    function decrement() internal {
        assembly {
            if iszero(tload(CALL_DEPTH_SLOT)) { revert(0, 0) }
            tstore(CALL_DEPTH_SLOT, sub(tload(CALL_DEPTH_SLOT), 1))
        }
    }
}

/// @title TransientReentrancyAware
/// @notice Reentrancy-aware modifier for transient storage, which increments and
///         decrements the call depth when entering and exiting a function.
contract TransientReentrancyAware {
    /// @notice Modifier to make a function reentrancy-aware.
    modifier reentrantAware() {
        TransientContext.increment();
        _;
        TransientContext.decrement();
    }
}
