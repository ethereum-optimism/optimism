// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// TODO(maurelian): remove this when the contract is complete
import { console } from "forge-std/console.sol";

contract DelayedVetoable {
    /// @notice An event that is emitted each time a call is forwarded.
    /// @param data The address of the implementation contract
    event Forwarded(bytes data);

    /// @notice The address that all calls are forwarded to after the delay.
    address internal _target;

    /// @notice Sets the target admin during contract deployment.
    /// @param target Address of the target.
    constructor(address target) {
        _target = target;
    }

    /// @notice Used when no data is passed to the contract.
    receive() external payable {
        _handleCall();
    }

    /// @notice Used for all calls that pass data to the contract.
    fallback() external payable {
        _handleCall();
    }

    /// @notice Handles forwards the call to the target.
    function _handleCall() internal {
        require(_target != address(0), "DelayedVetoable: target not initialized");

        emit Forwarded(msg.data);
        (bool success,) = _target.call(msg.data);
        assembly {
            // Success == 0 means a revert. We'll revert too and pass the data up.
            if iszero(success) { revert(0x0, returndatasize()) }

            // Otherwise we'll just return and pass the data up.
            return(0x0, returndatasize())
        }
    }
}
