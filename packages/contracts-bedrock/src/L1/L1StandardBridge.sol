// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";
import { StandardBridge } from "src/universal/StandardBridge.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { WithdrawalThrottle } from "src/libraries/WithdrawalThrottle.sol";

// Interfaces
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @custom:proxied true
/// @title L1StandardBridge
/// @notice The L1StandardBridge is responsible for transferring ETH and ERC20 tokens between L1 and
///         L2. In the case that an ERC20 token is native to L1, it will be escrowed within this
///         contract. If the ERC20 token is native to L2, it will be burnt. Before Bedrock, ETH was
///         stored within this contract. After Bedrock, ETH is instead stored inside the
///         OptimismPortal contract.
///         NOTE: this contract is not intended to support all variations of ERC20 tokens. Examples
///         of some token types that may not be properly supported by this contract include, but are
///         not limited to: tokens with transfer fees, rebasing tokens, and tokens with blocklists.
contract L1StandardBridge is StandardBridge, ProxyAdminOwnedBase, ReinitializableBase, ISemver {
    using WithdrawalThrottle for WithdrawalThrottle.State;

    /// @notice Thrown when a withdrawal throttle is configured for the zero address.
    error L1StandardBridge_InvalidWithdrawalThrottleToken();

    /// @notice Thrown when a withdrawal throttle basis point value is zero or exceeds 100%.
    error L1StandardBridge_InvalidWithdrawalThrottleBps();

    /// @notice Thrown when a withdrawal throttle is configured for a non-escrowed token.
    error L1StandardBridge_UnsupportedWithdrawalThrottleToken();

    /// @notice Thrown when a withdrawal throttle refill period is zero.
    error L1StandardBridge_InvalidWithdrawalThrottlePeriod();

    /// @notice Thrown when a withdrawal throttle is not enabled for a token.
    error L1StandardBridge_WithdrawalThrottleNotEnabled();

    /// @notice Thrown when a withdrawal exceeds the token's available withdrawal capacity.
    error L1StandardBridge_WithdrawalThrottled(
        address token, uint256 requestedAmount, uint256 availableCapacity, uint256 totalCapacity
    );

    /// @notice Logical withdrawal throttle state for an L1 token.
    struct WithdrawalThrottleConfig {
        uint256 capacity;
        uint256 available;
        uint64 refillPeriod;
        uint64 lastUpdated;
        uint64 refillRemainder;
        uint16 maxBps;
        bool enabled;
    }

    /// @notice Emitted when a withdrawal throttle is configured for a token.
    event WithdrawalThrottleConfigured(
        address indexed token,
        uint16 maxBps,
        uint64 refillPeriod,
        uint256 stockSnapshot,
        uint256 capacity,
        uint256 available
    );

    /// @notice Emitted when a withdrawal throttle's stock snapshot is refreshed.
    event WithdrawalThrottleRefreshed(
        address indexed token, uint256 stockSnapshot, uint256 capacity, uint256 available
    );

    /// @notice Emitted when a withdrawal throttle is disabled for a token.
    event WithdrawalThrottleDisabled(address indexed token);

    /// @notice Emitted when withdrawal capacity is consumed.
    event WithdrawalThrottleCapacityConsumed(address indexed token, uint256 amount, uint256 remaining);

    /// @notice Emitted when all currently available withdrawal capacity is consumed.
    event WithdrawalThrottleCapacityExhausted(address indexed token);

    /// @custom:legacy
    /// @notice Emitted whenever a deposit of ETH from L1 into L2 is initiated.
    /// @param from      Address of the depositor.
    /// @param to        Address of the recipient on L2.
    /// @param amount    Amount of ETH deposited.
    /// @param extraData Extra data attached to the deposit.
    event ETHDepositInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData);

    /// @custom:legacy
    /// @notice Emitted whenever a withdrawal of ETH from L2 to L1 is finalized.
    /// @param from      Address of the withdrawer.
    /// @param to        Address of the recipient on L1.
    /// @param amount    Amount of ETH withdrawn.
    /// @param extraData Extra data attached to the withdrawal.
    event ETHWithdrawalFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData);

    /// @custom:legacy
    /// @notice Emitted whenever an ERC20 deposit is initiated.
    /// @param l1Token   Address of the token on L1.
    /// @param l2Token   Address of the corresponding token on L2.
    /// @param from      Address of the depositor.
    /// @param to        Address of the recipient on L2.
    /// @param amount    Amount of the ERC20 deposited.
    /// @param extraData Extra data attached to the deposit.
    event ERC20DepositInitiated(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes extraData
    );

    /// @custom:legacy
    /// @notice Emitted whenever an ERC20 withdrawal is finalized.
    /// @param l1Token   Address of the token on L1.
    /// @param l2Token   Address of the corresponding token on L2.
    /// @param from      Address of the withdrawer.
    /// @param to        Address of the recipient on L1.
    /// @param amount    Amount of the ERC20 withdrawn.
    /// @param extraData Extra data attached to the withdrawal.
    event ERC20WithdrawalFinalized(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes extraData
    );

    /// @notice Semantic version.
    /// @custom:semver 3.1.0
    string public constant version = "3.1.0";

    /// @custom:legacy
    /// @custom:spacer superchainConfig
    /// @notice Spacer taking up the legacy `superchainConfig` slot.
    address private spacer_50_0_20;

    /// @custom:legacy
    /// @custom:spacer systemConfig
    /// @notice Spacer taking up the legacy `systemConfig` slot.
    address private spacer_51_0_20;

    /// @notice Address of the SystemConfig contract.
    ISystemConfig public systemConfig;

    /// @notice Withdrawal throttle state keyed by L1 token address.
    mapping(address => WithdrawalThrottle.State) internal _withdrawalThrottles;

    /// @notice Constructs the L1StandardBridge contract.
    constructor() StandardBridge() ReinitializableBase(3) {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _messenger        Contract for the CrossDomainMessenger on this network.
    /// @param _systemConfig Contract for the SystemConfig on this network.
    function initialize(
        ICrossDomainMessenger _messenger,
        ISystemConfig _systemConfig
    )
        external
        reinitializer(initVersion())
    {
        // Initialization transactions must come from the ProxyAdmin or its owner.
        _assertOnlyProxyAdminOrProxyAdminOwner();

        // Now perform initialization logic.
        systemConfig = _systemConfig;
        __StandardBridge_init({
            _messenger: _messenger,
            _otherBridge: StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE))
        });
    }

    /// @inheritdoc StandardBridge
    function paused() public view override returns (bool) {
        return systemConfig.paused();
    }

    /// @notice Returns the SuperchainConfig contract.
    /// @return ISuperchainConfig The SuperchainConfig contract.
    function superchainConfig() public view returns (ISuperchainConfig) {
        return systemConfig.superchainConfig();
    }

    /// @notice Configures a token's withdrawal capacity as a percentage of its current bridge stock.
    ///         Reconfiguration preserves accrued whole-unit capacity, resets the fractional
    ///         remainder, and clamps availability to the new maximum.
    /// @param _token        Address of the L1 token to throttle.
    /// @param _maxBps       Maximum withdrawable stock in basis points.
    /// @param _refillPeriod Time in seconds for the bucket to refill from empty to full.
    function setWithdrawalThrottle(address _token, uint16 _maxBps, uint64 _refillPeriod) external {
        _assertOnlyProxyAdminOwner();

        if (_token == address(0)) revert L1StandardBridge_InvalidWithdrawalThrottleToken();
        if (_isOptimismMintableERC20(_token)) revert L1StandardBridge_UnsupportedWithdrawalThrottleToken();
        if (_maxBps == 0 || _maxBps > WithdrawalThrottle.MAX_BPS) {
            revert L1StandardBridge_InvalidWithdrawalThrottleBps();
        }
        if (_refillPeriod == 0) revert L1StandardBridge_InvalidWithdrawalThrottlePeriod();

        WithdrawalThrottle.State storage throttle = _withdrawalThrottles[_token];
        uint256 stockSnapshot = IERC20(_token).balanceOf(address(this));
        (uint128 capacity, uint128 available) =
            throttle.configure(stockSnapshot, _maxBps, _refillPeriod, block.timestamp);

        emit WithdrawalThrottleConfigured(_token, _maxBps, _refillPeriod, stockSnapshot, capacity, available);
    }

    /// @notice Recomputes a token's capacity from its current bridge stock without refilling it.
    /// @param _token Address of the L1 token whose stock snapshot should be refreshed.
    function refreshWithdrawalThrottle(address _token) external {
        _assertOnlyProxyAdminOwner();

        WithdrawalThrottle.State storage throttle = _withdrawalThrottles[_token];
        uint256 config_ = throttle.config;
        if (!WithdrawalThrottle.enabled(config_)) revert L1StandardBridge_WithdrawalThrottleNotEnabled();

        uint256 stockSnapshot = IERC20(_token).balanceOf(address(this));
        (uint128 capacity, uint128 available) = throttle.refresh(config_, stockSnapshot, block.timestamp);

        emit WithdrawalThrottleRefreshed(_token, stockSnapshot, capacity, available);
    }

    /// @notice Disables a token's withdrawal throttle.
    /// @param _token Address of the L1 token whose throttle should be disabled.
    function disableWithdrawalThrottle(address _token) external {
        _assertOnlyProxyAdminOwner();

        WithdrawalThrottle.State storage throttle = _withdrawalThrottles[_token];
        if (!WithdrawalThrottle.enabled(throttle.config)) revert L1StandardBridge_WithdrawalThrottleNotEnabled();
        throttle.disable();

        emit WithdrawalThrottleDisabled(_token);
    }

    /// @notice Returns the stored withdrawal throttle state for a token before pending refill is materialized.
    /// @param _token Address of the L1 token.
    /// @return Withdrawal throttle state.
    function withdrawalThrottle(address _token) external view returns (WithdrawalThrottleConfig memory) {
        WithdrawalThrottle.Snapshot memory throttle = _withdrawalThrottles[_token].snapshot();
        return WithdrawalThrottleConfig({
            capacity: throttle.capacity,
            available: throttle.available,
            refillPeriod: throttle.refillPeriod,
            lastUpdated: throttle.lastUpdated,
            refillRemainder: throttle.refillRemainder,
            maxBps: throttle.maxBps,
            enabled: throttle.enabled
        });
    }

    /// @notice Returns the currently available withdrawal capacity for a token.
    /// @param _token Address of the L1 token.
    /// @return Available capacity, or the maximum uint256 value when throttling is disabled.
    function availableWithdrawalCapacity(address _token) external view returns (uint256) {
        WithdrawalThrottle.State storage throttle = _withdrawalThrottles[_token];
        uint256 config_ = throttle.config;
        if (!WithdrawalThrottle.enabled(config_)) return type(uint256).max;
        return throttle.availableCapacity(config_, IERC20(_token).balanceOf(address(this)), block.timestamp);
    }

    /// @notice Allows EOAs to bridge ETH by sending directly to the bridge.
    receive() external payable override onlyEOA {
        _initiateETHDeposit(msg.sender, msg.sender, RECEIVE_DEFAULT_GAS_LIMIT, bytes(""));
    }

    /// @custom:legacy
    /// @notice Deposits some amount of ETH into the sender's account on L2.
    /// @param _minGasLimit Minimum gas limit for the deposit message on L2.
    /// @param _extraData   Optional data to forward to L2.
    ///                     Data supplied here will not be used to execute any code on L2 and is
    ///                     only emitted as extra data for the convenience of off-chain tooling.
    function depositETH(uint32 _minGasLimit, bytes calldata _extraData) external payable onlyEOA {
        _initiateETHDeposit(msg.sender, msg.sender, _minGasLimit, _extraData);
    }

    /// @custom:legacy
    /// @notice Deposits some amount of ETH into a target account on L2.
    ///         Note that if ETH is sent to a contract on L2 and the call fails, then that ETH will
    ///         be locked in the L2StandardBridge. ETH may be recoverable if the call can be
    ///         successfully replayed by increasing the amount of gas supplied to the call. If the
    ///         call will fail for any amount of gas, then the ETH will be locked permanently.
    /// @param _to          Address of the recipient on L2.
    /// @param _minGasLimit Minimum gas limit for the deposit message on L2.
    /// @param _extraData   Optional data to forward to L2.
    ///                     Data supplied here will not be used to execute any code on L2 and is
    ///                     only emitted as extra data for the convenience of off-chain tooling.
    function depositETHTo(address _to, uint32 _minGasLimit, bytes calldata _extraData) external payable {
        _initiateETHDeposit(msg.sender, _to, _minGasLimit, _extraData);
    }

    /// @custom:legacy
    /// @notice Deposits some amount of ERC20 tokens into the sender's account on L2.
    /// @param _l1Token     Address of the L1 token being deposited.
    /// @param _l2Token     Address of the corresponding token on L2.
    /// @param _amount      Amount of the ERC20 to deposit.
    /// @param _minGasLimit Minimum gas limit for the deposit message on L2.
    /// @param _extraData   Optional data to forward to L2.
    ///                     Data supplied here will not be used to execute any code on L2 and is
    ///                     only emitted as extra data for the convenience of off-chain tooling.
    function depositERC20(
        address _l1Token,
        address _l2Token,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    )
        external
        virtual
        onlyEOA
    {
        _initiateERC20Deposit(_l1Token, _l2Token, msg.sender, msg.sender, _amount, _minGasLimit, _extraData);
    }

    /// @custom:legacy
    /// @notice Deposits some amount of ERC20 tokens into a target account on L2.
    /// @param _l1Token     Address of the L1 token being deposited.
    /// @param _l2Token     Address of the corresponding token on L2.
    /// @param _to          Address of the recipient on L2.
    /// @param _amount      Amount of the ERC20 to deposit.
    /// @param _minGasLimit Minimum gas limit for the deposit message on L2.
    /// @param _extraData   Optional data to forward to L2.
    ///                     Data supplied here will not be used to execute any code on L2 and is
    ///                     only emitted as extra data for the convenience of off-chain tooling.
    function depositERC20To(
        address _l1Token,
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    )
        external
        virtual
    {
        _initiateERC20Deposit(_l1Token, _l2Token, msg.sender, _to, _amount, _minGasLimit, _extraData);
    }

    /// @custom:legacy
    /// @notice Finalizes a withdrawal of ETH from L2.
    /// @param _from      Address of the withdrawer on L2.
    /// @param _to        Address of the recipient on L1.
    /// @param _amount    Amount of ETH to withdraw.
    /// @param _extraData Optional data forwarded from L2.
    function finalizeETHWithdrawal(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        external
        payable
    {
        finalizeBridgeETH(_from, _to, _amount, _extraData);
    }

    /// @custom:legacy
    /// @notice Finalizes a withdrawal of ERC20 tokens from L2.
    /// @param _l1Token   Address of the token on L1.
    /// @param _l2Token   Address of the corresponding token on L2.
    /// @param _from      Address of the withdrawer on L2.
    /// @param _to        Address of the recipient on L1.
    /// @param _amount    Amount of the ERC20 to withdraw.
    /// @param _extraData Optional data forwarded from L2.
    function finalizeERC20Withdrawal(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    )
        external
    {
        finalizeBridgeERC20(_l1Token, _l2Token, _from, _to, _amount, _extraData);
    }

    /// @custom:legacy
    /// @notice Retrieves the access of the corresponding L2 bridge contract.
    /// @return Address of the corresponding L2 bridge contract.
    function l2TokenBridge() external view returns (address) {
        return address(otherBridge);
    }

    /// @notice Internal function for initiating an ETH deposit.
    /// @param _from        Address of the sender on L1.
    /// @param _to          Address of the recipient on L2.
    /// @param _minGasLimit Minimum gas limit for the deposit message on L2.
    /// @param _extraData   Optional data to forward to L2.
    function _initiateETHDeposit(address _from, address _to, uint32 _minGasLimit, bytes memory _extraData) internal {
        _initiateBridgeETH(_from, _to, msg.value, _minGasLimit, _extraData);
    }

    /// @notice Internal function for initiating an ERC20 deposit.
    /// @param _l1Token     Address of the L1 token being deposited.
    /// @param _l2Token     Address of the corresponding token on L2.
    /// @param _from        Address of the sender on L1.
    /// @param _to          Address of the recipient on L2.
    /// @param _amount      Amount of the ERC20 to deposit.
    /// @param _minGasLimit Minimum gas limit for the deposit message on L2.
    /// @param _extraData   Optional data to forward to L2.
    function _initiateERC20Deposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    )
        internal
    {
        _initiateBridgeERC20(_l1Token, _l2Token, _from, _to, _amount, _minGasLimit, _extraData);
    }

    /// @inheritdoc StandardBridge
    function _beforeFinalizeBridgeERC20(address _localToken, uint256 _amount) internal override {
        WithdrawalThrottle.State storage throttle = _withdrawalThrottles[_localToken];
        uint256 config_ = throttle.config;
        if (!WithdrawalThrottle.enabled(config_) || _amount == 0) return;

        uint256 stockSnapshot = IERC20(_localToken).balanceOf(address(this));
        (uint128 capacity, uint128 available, uint128 remaining, bool capacityChanged) =
            throttle.consume(config_, stockSnapshot, _amount, block.timestamp);
        if (_amount > available) {
            revert L1StandardBridge_WithdrawalThrottled(_localToken, _amount, available, capacity);
        }

        if (capacityChanged) emit WithdrawalThrottleRefreshed(_localToken, stockSnapshot, capacity, available);
        emit WithdrawalThrottleCapacityConsumed(_localToken, _amount, remaining);
        if (remaining == 0) emit WithdrawalThrottleCapacityExhausted(_localToken);
    }

    /// @inheritdoc StandardBridge
    /// @notice Emits the legacy ETHDepositInitiated event followed by the ETHBridgeInitiated event.
    ///         This is necessary for backwards compatibility with the legacy bridge.
    function _emitETHBridgeInitiated(
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        internal
        override
    {
        emit ETHDepositInitiated(_from, _to, _amount, _extraData);
        super._emitETHBridgeInitiated(_from, _to, _amount, _extraData);
    }

    /// @inheritdoc StandardBridge
    /// @notice Emits the legacy ERC20DepositInitiated event followed by the ERC20BridgeInitiated
    ///         event. This is necessary for backwards compatibility with the legacy bridge.
    function _emitETHBridgeFinalized(
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        internal
        override
    {
        emit ETHWithdrawalFinalized(_from, _to, _amount, _extraData);
        super._emitETHBridgeFinalized(_from, _to, _amount, _extraData);
    }

    /// @inheritdoc StandardBridge
    /// @notice Emits the legacy ERC20WithdrawalFinalized event followed by the ERC20BridgeFinalized
    ///         event. This is necessary for backwards compatibility with the legacy bridge.
    function _emitERC20BridgeInitiated(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        internal
        override
    {
        emit ERC20DepositInitiated(_localToken, _remoteToken, _from, _to, _amount, _extraData);
        super._emitERC20BridgeInitiated(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }

    /// @inheritdoc StandardBridge
    /// @notice Emits the legacy ERC20WithdrawalFinalized event followed by the ERC20BridgeFinalized
    ///         event. This is necessary for backwards compatibility with the legacy bridge.
    function _emitERC20BridgeFinalized(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        internal
        override
    {
        emit ERC20WithdrawalFinalized(_localToken, _remoteToken, _from, _to, _amount, _extraData);
        super._emitERC20BridgeFinalized(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }
}
