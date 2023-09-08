// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// TODO(maurelian): remove this when the contract is complete
import { console } from "forge-std/console.sol";

contract DelayedVetoable {
    /// @notice Error for when attempting to forward too early.
    error ForwardingEarly();

    /// @notice Error for the target is not set.
    error TargetUnitialized();

    /// @notice An event that is emitted when a call is initiated.
    /// @param callHash The hash of the call data.
    /// @param data The data of the initiated call.
    event Initiated(bytes32 indexed callHash, bytes data);

    /// @notice An event that is emitted each time a call is forwarded.
    /// @param callHash The hash of the call data.
    /// @param data The data forwarded to the target.
    event Forwarded(bytes32 indexed callHash, bytes data);

    /// @notice The address that all calls are forwarded to after the delay.
    address internal _target;

    /// @notice The time that a call was initiated.
    mapping(bytes32 => uint256) internal _queuedAt;

    /// @notice The time to wait before forwarding a call.
    uint256 internal _delay;

    /// @notice Sets the target admin during contract deployment.
    /// @param target Address of the target.
    constructor(address target, uint256 delay) {
        _target = target;
        _delay = delay;
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
        if (_target == address(0)) {
            revert TargetUnitialized();
        }

        bytes32 callHash = keccak256(msg.data);
        if (_queuedAt[callHash] == 0) {
            _queuedAt[callHash] = block.timestamp;
            emit Initiated(callHash, msg.data);
        } else if (_queuedAt[callHash] + _delay < block.timestamp) {
            // Not enough time has passed, so we'll revert.
            revert ForwardingEarly();
        } else {
            // sufficient time has passed.
            // Delete the call to prevent replays
            delete _queuedAt[callHash];

            // Forward the call
            emit Forwarded(callHash, msg.data);
            (bool success,) = _target.call(msg.data);
            assembly {
                // Success == 0 means a revert. We'll revert too and pass the data up.
                if iszero(success) { revert(0x0, returndatasize()) }

                // Otherwise we'll just return and pass the data up.
                return(0x0, returndatasize())
            }
        }
    }
}
