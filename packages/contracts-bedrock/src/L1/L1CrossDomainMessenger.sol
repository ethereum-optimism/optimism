// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";

/// @custom:proxied true
/// @title L1CrossDomainMessenger
/// @notice The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
///         for sending and receiving data on the L1 side. Users are encouraged to use this
///         interface instead of interacting with lower-level contracts directly.
contract L1CrossDomainMessenger is CrossDomainMessenger, ProxyAdminOwnedBase, ReinitializableBase, ISemver {
    /// @custom:legacy
    /// @custom:spacer superchainConfig
    /// @notice Spacer taking up the legacy `superchainConfig` slot.
    address private spacer_251_0_20;

    /// @notice Contract of the OptimismPortal.
    /// @custom:network-specific
    IOptimismPortal public portal;

    /// @custom:legacy
    /// @custom:spacer systemConfig
    /// @notice Spacer taking up the legacy `systemConfig` slot.
    address private spacer_253_0_20;

    /// @notice Semantic version.
    /// @custom:semver 2.13.0
    string public constant version = "2.13.0";

    /// @notice Gas reserved on L1 for finalizing `relayMessage` after the safe call. Covers
    ///         recording the outcome in a fresh storage slot, including EIP-8037 state gas.
    uint64 public constant LOCAL_RELAY_RESERVED_GAS = 125_000;

    /// @notice Gas reserved on L1 for the execution between the `hasMinGas` check and the
    ///         external call in `relayMessage`.
    uint64 public constant LOCAL_RELAY_GAS_CHECK_BUFFER = 5_000;

    /// @notice Constant costs of `relayMessage` on L2, added to the base gas for a message.
    uint64 public constant REMOTE_RELAY_CONSTANT_OVERHEAD = 200_000;

    /// @notice Gas L2 needs for the dynamic parts of the CALL opcode in `relayMessage`.
    uint64 public constant REMOTE_RELAY_CALL_OVERHEAD = 40_000;

    /// @notice Gas reserved on L2 for finalizing `relayMessage` after the safe call. Covers
    ///         recording the outcome in a fresh storage slot, including EIP-8037 state gas.
    uint64 public constant REMOTE_RELAY_RESERVED_GAS = 125_000;

    /// @notice Gas reserved on L2 for the execution between the `hasMinGas` check and the
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

    /// @notice Contract of the SystemConfig.
    ISystemConfig public systemConfig;

    /// @notice Constructs the L1CrossDomainMessenger contract.
    constructor() ReinitializableBase(3) {
        _disableInitializers();
    }

    /// @notice Initializes the contract.
    /// @param _systemConfig Contract of the SystemConfig contract on this network.
    /// @param _portal Contract of the OptimismPortal contract on this network.
    function initialize(ISystemConfig _systemConfig, IOptimismPortal _portal) external reinitializer(initVersion()) {
        // Initialization transactions must come from the ProxyAdmin or its owner.
        _assertOnlyProxyAdminOrProxyAdminOwner();

        // Now perform initialization logic.
        systemConfig = _systemConfig;
        portal = _portal;
        __CrossDomainMessenger_init({ _otherMessenger: CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER) });
    }

    /// @inheritdoc CrossDomainMessenger
    function paused() public view override returns (bool) {
        return systemConfig.paused();
    }

    /// @notice Returns the SuperchainConfig contract.
    /// @return ISuperchainConfig The SuperchainConfig contract.
    function superchainConfig() public view returns (ISuperchainConfig) {
        return systemConfig.superchainConfig();
    }

    /// @notice Getter function for the OptimismPortal contract on this chain.
    ///         Public getter is legacy and will be removed in the future. Use `portal()` instead.
    /// @return Contract of the OptimismPortal on this chain.
    /// @custom:legacy
    function PORTAL() external view returns (IOptimismPortal) {
        return portal;
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
        portal.depositTransaction{ value: _value }({
            _to: _to, _value: _value, _gasLimit: _gasLimit, _isCreation: false, _data: _data
        });
    }

    /// @inheritdoc CrossDomainMessenger
    function _isOtherMessenger() internal view override returns (bool) {
        return msg.sender == address(portal) && portal.l2Sender() == address(otherMessenger);
    }

    /// @inheritdoc CrossDomainMessenger
    function _isUnsafeTarget(address _target) internal view override returns (bool) {
        return _target == address(this) || _target == address(portal);
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
