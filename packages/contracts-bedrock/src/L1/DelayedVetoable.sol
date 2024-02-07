// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";

/// @title DelayedVetoable
/// @notice This contract enables a delay before a call is forwarded to a target contract, and during the delay period
///         the call can be vetoed by the authorized vetoer.
///         This contract does not support value transfers, only data is forwarded.
///         Additionally, this contract cannot be used to forward calls with data beginning with the function selector
///         of the queuedAt(bytes32) function. This is because of input validation checks which solidity performs at
///         runtime on functions which take an argument.
contract DelayedVetoable is ISemver {
    /// @notice Error for when attempting to forward too early.
    error ForwardingEarly();

    /// @notice Error for unauthorized calls.
    error Unauthorized(address expected, address actual);

    /// @notice An event that is emitted when the delay is activated.
    /// @param delay The delay that was activated.
    event DelayActivated(uint256 delay);

    /// @notice An event that is emitted when a call is initiated.
    /// @param callHash The hash of the call data.
    /// @param data The data of the initiated call.
    event Initiated(bytes32 indexed callHash, bytes data);

    /// @notice An event that is emitted each time a call is forwarded.
    /// @param callHash The hash of the call data.
    /// @param data The data forwarded to the target.
    event Forwarded(bytes32 indexed callHash, bytes data);

    /// @notice An event that is emitted each time a call is vetoed.
    /// @param callHash The hash of the call data.
    /// @param data The data forwarded to the target.
    event Vetoed(bytes32 indexed callHash, bytes data);

    /// @notice The address that all calls are forwarded to after the delay.
    address internal immutable TARGET;

    /// @notice The address that can veto a call.
    address internal immutable VETOER;

    /// @notice The address that can initiate a call.
    address internal immutable INITIATOR;

    /// @notice The delay which will be set after the initial system deployment is completed.
    uint256 internal immutable OPERATING_DELAY;

    /// @notice The current amount of time to wait before forwarding a call.
    uint256 internal _delay;

    /// @notice The time that a call was initiated.
    mapping(bytes32 => uint256) internal _queuedAt;

    /// @notice A modifier that reverts if not called by the vetoer or by address(0) to allow
    ///         eth_call to interact with this proxy without needing to use low-level storage
    ///         inspection. We assume that nobody is able to trigger calls from address(0) during
    ///         normal EVM execution.
    modifier readOrHandle() {
        if (msg.sender == address(0)) {
            _;
        } else {
            // This WILL halt the call frame on completion.
            _handleCall();
        }
    }

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Sets the target admin during contract deployment.
    /// @param vetoer_ Address of the vetoer.
    /// @param initiator_ Address of the initiator.
    /// @param target_ Address of the target.
    /// @param operatingDelay_ Time to delay when the system is operational.
    constructor(address vetoer_, address initiator_, address target_, uint256 operatingDelay_) {
        // Note that the _delay value is not set here. Having an initial delay of 0 is helpful
        // during the deployment of a new system.
        VETOER = vetoer_;
        INITIATOR = initiator_;
        TARGET = target_;
        OPERATING_DELAY = operatingDelay_;
    }

    /// @notice Gets the initiator
    /// @return initiator_ Initiator address.
    function initiator() external virtual readOrHandle returns (address initiator_) {
        initiator_ = INITIATOR;
    }

    //// @notice Queries the vetoer address.
    /// @return vetoer_ Vetoer address.
    function vetoer() external virtual readOrHandle returns (address vetoer_) {
        vetoer_ = VETOER;
    }

    //// @notice Queries the target address.
    /// @return target_ Target address.
    function target() external readOrHandle returns (address target_) {
        target_ = TARGET;
    }

    /// @notice Gets the delay
    /// @return delay_ Delay address.
    function delay() external readOrHandle returns (uint256 delay_) {
        delay_ = _delay;
    }

    /// @notice Gets entries in the _queuedAt mapping.
    /// @param callHash The hash of the call data.
    /// @return queuedAt_ The time the callHash was recorded.
    function queuedAt(bytes32 callHash) external readOrHandle returns (uint256 queuedAt_) {
        queuedAt_ = _queuedAt[callHash];
    }

    /// @notice Used for all calls that pass data to the contract.
    fallback() external {
        _handleCall();
    }

    /// @notice Receives all calls other than those made by the vetoer.
    ///         This enables transparent initiation and forwarding of calls to the target and avoids
    ///         the need for additional layers of abi encoding.
    function _handleCall() internal {
        // The initiator and vetoer activate the delay by passing in null data.
        if (msg.data.length == 0 && _delay == 0) {
            if (msg.sender != INITIATOR && msg.sender != VETOER) {
                revert Unauthorized(INITIATOR, msg.sender);
            }
            _delay = OPERATING_DELAY;
            emit DelayActivated(_delay);
            return;
        }

        bytes32 callHash = keccak256(msg.data);

        // Case 1: The initiator is calling the contract to initiate a call.
        if (msg.sender == INITIATOR && _queuedAt[callHash] == 0) {
            if (_delay == 0) {
                // This forward function will halt the call frame on completion.
                _forwardAndHalt(callHash);
            }
            _queuedAt[callHash] = block.timestamp;
            emit Initiated(callHash, msg.data);
            return;
        }

        // Case 2: The vetoer is calling the contract to veto a call.
        // Note: The vetoer retains the ability to veto even after the delay has passed. This makes censoring the vetoer
        //       more costly, as there is no time limit after which their transaction can be included.
        if (msg.sender == VETOER && _queuedAt[callHash] != 0) {
            delete _queuedAt[callHash];
            emit Vetoed(callHash, msg.data);
            return;
        }

        // Case 3: The call is from an unpermissioned actor. We'll forward the call if the delay has
        // passed.
        if (_queuedAt[callHash] == 0) {
            // The call has not been initiated, so we'll treat this is an unauthorized initiation attempt.
            revert Unauthorized(INITIATOR, msg.sender);
        }

        if (_queuedAt[callHash] + _delay > block.timestamp) {
            // Not enough time has passed, so we'll revert.
            revert ForwardingEarly();
        }

        // Delete the call to prevent replays
        delete _queuedAt[callHash];
        _forwardAndHalt(callHash);
    }

    /// @notice Forwards the call to the target and halts the call frame.
    function _forwardAndHalt(bytes32 callHash) internal {
        // Forward the call
        emit Forwarded(callHash, msg.data);
        (bool success, bytes memory returndata) = TARGET.call(msg.data);
        if (success == true) {
            assembly {
                return(add(returndata, 0x20), mload(returndata))
            }
        } else {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }
}
