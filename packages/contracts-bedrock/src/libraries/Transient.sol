// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title Transient
/// @notice Library for transient storage.
library Transient {
    /// @notice Slot for call depth.
    uint256 internal constant CALL_DEPTH_SLOT = 0;

    /// @notice Gets the current call depth.
    /// @return _callDepth Current call depth.
    function getCallDepth() internal view returns (uint256 _callDepth) {
        assembly {
            _callDepth := tload(CALL_DEPTH_SLOT)
        }
    }

    /// @notice Increments call depth.
    function incrementCallDepth() internal {
        // Get the current call depth
        uint256 currentCallDepth = getCallDepth();

        // Increment the call depth
        assembly {
            tstore(CALL_DEPTH_SLOT, add(currentCallDepth, 1))
        }
    }

    /// @notice Decrements call depth.
    function decrementCallDepth() internal {
        // Get the current call depth
        uint256 currentCallDepth = getCallDepth();

        // Decrement the call depth
        assembly {
            tstore(CALL_DEPTH_SLOT, sub(currentCallDepth, 1))
        }
    }

    /// @notice Gets corresponding transient slot for an arbitrary slot at a given call depth.
    /// @param _callDepth Call depth to get transient slot for.
    /// @param _slot      Slot to get transient slot for.
    /// @return Corresponding transient slot at the given call depth.
    function getTransientSlot(uint256 _callDepth, bytes32 _slot) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(_callDepth, _slot));
    }

    /// @notice Gets corresponding transient slot for an arbitrary slot at current call depth.
    /// @param _slot Slot to get transient slot for.
    /// @return Corresponding transient slot at the current call depth.
    function getTransientSlotCurrentDepth(bytes32 _slot) internal view returns (bytes32) {
        return keccak256(abi.encodePacked(getCallDepth(), _slot));
    }

    /// @notice Sets a value in transient storage at slot at a given call depth.
    /// @param _callDepth Call depth to get transient slot for.
    /// @param _slot    Slot to set.
    /// @param _value   Value to set.
    function setTransientValue(uint256 _callDepth, bytes32 _slot, uint256 _value) internal {
        // Get the slot at the given call depth
        bytes32 slot = getTransientSlot(_callDepth, _slot);

        // Set the value at the slot
        assembly {
            tstore(slot, _value)
        }
    }

    /// @notice Sets a value in transient storage at slot at current call depth.
    /// @param _slot    Slot to set.
    /// @param _value   Value to set.
    function setTransientValueCurrentDepth(bytes32 _slot, uint256 _value) internal {
        // Get the slot at the current call depth
        bytes32 slot = getTransientSlotCurrentDepth(_slot);

        // Set the value at the slot
        assembly {
            tstore(slot, _value)
        }
    }

    /// @notice Gets value in transient storage at slot at a given call depth.
    /// @param _callDepth Call depth to get transient slot for.
    /// @param _slot Slot to get.
    /// @return _value Transient value.
    function getTransientValue(uint256 _callDepth, bytes32 _slot) internal view returns (uint256 _value) {
        // Get the slot at the current call depth
        bytes32 slot = getTransientSlot(_callDepth, _slot);

        // Get the value at the slot
        assembly {
            _value := tload(slot)
        }
    }

    /// @notice Gets value in transient storage at slot at current call depth.
    /// @param _slot Slot to get.
    /// @return _value Transient value.
    function getTransientValueCurrentDepth(bytes32 _slot) internal view returns (uint256 _value) {
        // Get the slot at the current call depth
        bytes32 slot = getTransientSlotCurrentDepth(_slot);

        // Get the value at the slot
        assembly {
            _value := tload(slot)
        }
    }
}

/// @title TransientReentrancyAware
/// @notice Reentrancy-aware modifier for transient storage, which increments and
///         decrements the call depth when entering and exiting a function.
contract TransientReentrancyAware {
    modifier reentrantAware() {
        Transient.incrementCallDepth();
        _;
        Transient.decrementCallDepth();
    }
}
