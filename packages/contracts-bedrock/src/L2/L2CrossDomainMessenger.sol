// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000007
/// @title L2CrossDomainMessenger
/// @notice The L2CrossDomainMessenger is a high-level interface for message passing between L1 and
///         L2 on the L2 side. Users are generally encouraged to use this contract instead of lower
///         level message passing contracts.
contract L2CrossDomainMessenger is ProxyAdminOwnedBase, CrossDomainMessenger, ISemver {
    /// @custom:semver 2.4.0
    string public constant version = "2.4.0";

    /// @notice Gas reserved on L2 for finalizing `relayMessage` after the safe call.
    uint64 public constant LOCAL_RELAY_RESERVED_GAS = 40_000;

    /// @notice Gas reserved on L2 for the execution between the `hasMinGas` check and the
    ///         external call in `relayMessage`.
    uint64 public constant LOCAL_RELAY_GAS_CHECK_BUFFER = 5_000;

    /// @notice Constant costs of `relayMessage` on L1, added to the base gas for a message.
    uint64 public constant REMOTE_RELAY_CONSTANT_OVERHEAD = 200_000;

    /// @notice Gas L1 needs for the dynamic parts of the CALL opcode in `relayMessage`.
    uint64 public constant REMOTE_RELAY_CALL_OVERHEAD = 40_000;

    /// @notice Gas reserved on L1 for finalizing `relayMessage` after the safe call. Covers
    ///         recording the outcome in a fresh storage slot, including EIP-8037 state gas.
    uint64 public constant REMOTE_RELAY_RESERVED_GAS = 125_000;

    /// @notice Gas reserved on L1 for the execution between the `hasMinGas` check and the
    ///         external call in `relayMessage`.
    uint64 public constant REMOTE_RELAY_GAS_CHECK_BUFFER = 5_000;

    /// @notice Legacy alias for `REMOTE_RELAY_CONSTANT_OVERHEAD`.
    /// @custom:legacy
    uint64 public constant RELAY_CONSTANT_OVERHEAD = REMOTE_RELAY_CONSTANT_OVERHEAD;

    /// @notice Legacy alias for `REMOTE_RELAY_CALL_OVERHEAD`.
    /// @custom:legacy
    uint64 public constant RELAY_CALL_OVERHEAD = REMOTE_RELAY_CALL_OVERHEAD;

    /// @notice Legacy alias for `REMOTE_RELAY_RESERVED_GAS`.
    /// @custom:legacy
    uint64 public constant RELAY_RESERVED_GAS = REMOTE_RELAY_RESERVED_GAS;

    /// @notice Legacy alias for `REMOTE_RELAY_GAS_CHECK_BUFFER`.
    /// @custom:legacy
    uint64 public constant RELAY_GAS_CHECK_BUFFER = REMOTE_RELAY_GAS_CHECK_BUFFER;

    /// @notice Constructs the L2CrossDomainMessenger contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _l1CrossDomainMessenger L1CrossDomainMessenger contract on the other network.
    function initialize(CrossDomainMessenger _l1CrossDomainMessenger) external initializer {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        __CrossDomainMessenger_init({ _otherMessenger: _l1CrossDomainMessenger });
    }

    /// @notice Getter for the remote messenger.
    ///         Public getter is legacy and will be removed in the future. Use `otherMessenger()` instead.
    /// @return L1CrossDomainMessenger contract.
    /// @custom:legacy
    function l1CrossDomainMessenger() public view returns (CrossDomainMessenger) {
        return otherMessenger;
    }

    /// @inheritdoc CrossDomainMessenger
    function baseGas(bytes memory _message, uint32 _minGasLimit) public pure override returns (uint64) {
        return _baseGas({
            _message: _message,
            _minGasLimit: _minGasLimit,
            _relayConstantOverhead: REMOTE_RELAY_CONSTANT_OVERHEAD,
            _relayCallOverhead: REMOTE_RELAY_CALL_OVERHEAD,
            _relayReservedGas: REMOTE_RELAY_RESERVED_GAS,
            _relayGasCheckBuffer: REMOTE_RELAY_GAS_CHECK_BUFFER
        });
    }

    /// @inheritdoc CrossDomainMessenger
    function _sendMessage(address _to, uint64 _gasLimit, uint256 _value, bytes memory _data) internal override {
        IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).initiateWithdrawal{ value: _value }(
            _to, _gasLimit, _data
        );
    }

    /// @inheritdoc CrossDomainMessenger
    function _isOtherMessenger() internal view override returns (bool) {
        return AddressAliasHelper.undoL1ToL2Alias(msg.sender) == address(otherMessenger);
    }

    /// @inheritdoc CrossDomainMessenger
    function _isUnsafeTarget(address _target) internal view override returns (bool) {
        return _target == address(this) || _target == address(Predeploys.L2_TO_L1_MESSAGE_PASSER);
    }

    /// @inheritdoc CrossDomainMessenger
    function _localRelayReservedGas() internal pure override returns (uint64) {
        return LOCAL_RELAY_RESERVED_GAS;
    }

    /// @inheritdoc CrossDomainMessenger
    function _localRelayGasCheckBuffer() internal pure override returns (uint64) {
        return LOCAL_RELAY_GAS_CHECK_BUFFER;
    }
}
