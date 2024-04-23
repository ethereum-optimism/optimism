// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title Transient
/// @notice Transient handles transient storage for reentrancy.
///         Forked from https://github.com/jtriley-eth/tcontext
library Transient {
    /// @notice Slot for transient context.
    uint256 internal constant CONTEXT_SLOT = 0;

    /// @notice Get the transient context.
    /// @return _ctx Transient context.
    function getTransientContext() internal view returns (uint256 _ctx) {
        assembly {
            mstore(0x00, tload(CONTEXT_SLOT))
            _ctx := keccak256(0x00, 0x20)
        }
    }

    /// @notice Set a transient value.
    /// @param _value   Value to set.
    /// @param _target  Target contract to call.
    /// @param _payload Payload to call target with.
    function setTransientValue(uint256 _value, address _target, bytes memory _payload) public {
        assembly {
            tstore(CONTEXT_SLOT, add(tload(CONTEXT_SLOT), 1))
        }

        uint256 ctx = getTransientContext();

        assembly {
            tstore(ctx, _value)
        }

        if (_target == address(0)) return;

        (bool success,) = _target.call(_payload);

        require(success, "setTransientValue::call");
    }

    /// @notice Get value in transient context.
    /// @return _value Transient value.
    function getTransientValue() public view returns (uint256 _value) {
        uint256 ctx = getTransientContext();

        assembly {
            _value := tload(ctx)
        }
    }
}
