// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Features } from "src/libraries/Features.sol";
import "src/libraries/GasConstants.sol" as GasConstants;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";

/// @custom:proxied true
/// @title L1CrossDomainMessenger
/// @notice The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
///         for sending and receiving data on the L1 side. Users are encouraged to use this
///         interface instead of interacting with lower-level contracts directly.
contract L1CrossDomainMessenger is CrossDomainMessenger, ProxyAdminOwnedBase, ReinitializableBase, ISemver {
    /// @notice Gas limit for L2ToL2CrossDomainMessenger.expireMessage on L2, which uses about 40k
    ///         gas.
    uint32 internal constant EXPIRE_MESSAGE_GAS_LIMIT = 100_000;

    /// @notice Gas limit for ETHLiquidity.setPaused on L2.
    uint32 internal constant SET_ETH_LIQUIDITY_PAUSED_GAS_LIMIT = 200_000;

    /// @notice Thrown when `relayUndeliveredMessage` or `syncETHLiquidityPause` is called on a
    ///         chain that does not run interop.
    error L1CrossDomainMessenger_InteropNotEnabled();

    /// @notice Thrown when `relayUndeliveredMessage` is not called by another chain's
    ///         L1CrossDomainMessenger in this chain's interop cluster, relaying a withdrawal from
    ///         that chain's UndeliveredMessageExporter. A caller whose getters revert, such as an
    ///         account with no code, gets that revert instead.
    error L1CrossDomainMessenger_NotInteropMessenger();

    /// @notice Thrown when `relayUndeliveredMessage` is called while this chain is paused.
    error L1CrossDomainMessenger_Paused();

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
    /// @custom:semver 3.3.0
    string public constant version = "3.3.0";

    /// @notice See `GasConstants.L1_RELAY_RESERVED_GAS`.
    uint64 internal constant LOCAL_RELAY_RESERVED_GAS = GasConstants.L1_RELAY_RESERVED_GAS;

    /// @notice See `GasConstants.L1_RELAY_GAS_CHECK_BUFFER`.
    uint64 internal constant LOCAL_RELAY_GAS_CHECK_BUFFER = GasConstants.L1_RELAY_GAS_CHECK_BUFFER;

    /// @notice See `GasConstants.L2_RELAY_CONSTANT_OVERHEAD`.
    uint64 internal constant REMOTE_RELAY_CONSTANT_OVERHEAD = GasConstants.L2_RELAY_CONSTANT_OVERHEAD;

    /// @notice See `GasConstants.RELAY_CALL_OVERHEAD`.
    uint64 internal constant REMOTE_RELAY_CALL_OVERHEAD = GasConstants.RELAY_CALL_OVERHEAD;

    /// @notice See `GasConstants.L2_RELAY_RESERVED_GAS`.
    uint64 internal constant REMOTE_RELAY_RESERVED_GAS = GasConstants.L2_RELAY_RESERVED_GAS;

    /// @notice See `GasConstants.L2_RELAY_GAS_CHECK_BUFFER`.
    uint64 internal constant REMOTE_RELAY_GAS_CHECK_BUFFER = GasConstants.L2_RELAY_GAS_CHECK_BUFFER;

    // Public getters kept for ABI compatibility. They report the remote budgets that `baseGas` uses;
    // this contract reads the `LOCAL_` and `REMOTE_` constants directly.

    /// @inheritdoc CrossDomainMessenger
    uint64 public constant override RELAY_CONSTANT_OVERHEAD = REMOTE_RELAY_CONSTANT_OVERHEAD;

    /// @inheritdoc CrossDomainMessenger
    uint64 public constant override RELAY_CALL_OVERHEAD = REMOTE_RELAY_CALL_OVERHEAD;

    /// @inheritdoc CrossDomainMessenger
    uint64 public constant override RELAY_RESERVED_GAS = REMOTE_RELAY_RESERVED_GAS;

    /// @inheritdoc CrossDomainMessenger
    uint64 public constant override RELAY_GAS_CHECK_BUFFER = REMOTE_RELAY_GAS_CHECK_BUFFER;

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

    /// @notice Passes on word from another chain in this chain's interop cluster that a message
    ///         from this chain had not been relayed there by `_undeliveredAt`. This chain's
    ///         L2ToL2CrossDomainMessenger marks the message expired if that is past its expiry
    ///         period. It is off unless this chain runs interop. The caller must be:
    ///         - a real L1CrossDomainMessenger: its portal's SystemConfig names it as the chain's
    ///           messenger. A contract that names a real portal as its own fails this;
    ///         - of a chain in this chain's cluster: its portal is authorized by this chain's
    ///           ETHLockbox. Every such chain can already withdraw this chain's ETH, so trusting
    ///           its word adds no trust;
    ///         - relaying a withdrawal from that chain's UndeliveredMessageExporter, which sends
    ///           this call only for a message to that chain that it has not relayed. See
    ///           UndeliveredMessageExporter for why withdrawals from it can be trusted.
    ///         It reverts while this chain is paused.
    ///         The word is sent on as this contract, which no relayed message can be (see
    ///         `_isUnsafeTarget`), so L2 can trust it. If this runs out of gas, the call lands in
    ///         the caller's failed messages and can be replayed.
    /// @param _messageHash   Hash of the message.
    /// @param _undeliveredAt Timestamp on the message's destination at which it had not been
    ///                       relayed.
    function relayUndeliveredMessage(bytes32 _messageHash, uint256 _undeliveredAt) external {
        if (!systemConfig.isFeatureEnabled(Features.INTEROP)) revert L1CrossDomainMessenger_InteropNotEnabled();
        if (paused()) revert L1CrossDomainMessenger_Paused();

        L1CrossDomainMessenger caller = L1CrossDomainMessenger(msg.sender);
        IOptimismPortal callerPortal = caller.portal();
        if (
            callerPortal.systemConfig().l1CrossDomainMessenger() != msg.sender
                || !portal.ethLockbox().authorizedPortals(callerPortal)
                || caller.xDomainMessageSender() != Predeploys.UNDELIVERED_MESSAGE_EXPORTER
        ) {
            revert L1CrossDomainMessenger_NotInteropMessenger();
        }

        this.sendMessage({
            _target: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            _message: abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt)),
            _minGasLimit: EXPIRE_MESSAGE_GAS_LIMIT
        });
    }

    /// @notice Sends this chain's current pause state to its ETHLiquidity predeploy, which stops
    ///         the SuperchainETHBridge on L2 while the chain is paused. Anyone can call it, also
    ///         while paused; call it again after any pause, unpause or pause expiry. L2 ignores a
    ///         message older, by nonce, than the one it applied. It reverts unless this chain runs
    ///         interop.
    function syncETHLiquidityPause() external {
        if (!systemConfig.isFeatureEnabled(Features.INTEROP)) revert L1CrossDomainMessenger_InteropNotEnabled();

        this.sendMessage({
            _target: Predeploys.ETH_LIQUIDITY,
            _message: abi.encodeCall(IETHLiquidity.setPaused, (paused(), messageNonce())),
            _minGasLimit: SET_ETH_LIQUIDITY_PAUSED_GAS_LIMIT
        });
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
