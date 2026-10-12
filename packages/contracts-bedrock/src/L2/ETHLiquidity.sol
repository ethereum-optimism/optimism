// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { SafeSend } from "src/universal/SafeSend.sol";

// Libraries
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";

// Errors
import { InvalidAmount } from "src/libraries/errors/CommonErrors.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000025
/// @title ETHLiquidity
/// @notice The ETHLiquidity contract allows other contracts to access ETH liquidity without
///         needing to modify the EVM to generate new ETH. Contract comes "pre-loaded" with
///         uint128.max balance to prevent liquidity shortages. It mirrors the chain's pause on
///         L1: while that is paused, ETH can't be burned or minted, so the SuperchainETHBridge
///         can't send or relay ETH. Anyone can sync the pause from L1 through
///         `L1CrossDomainMessenger.syncETHLiquidityPause`.
contract ETHLiquidity is ISemver {
    /// @notice Thrown when burning or minting ETH liquidity while the contract is paused.
    error ETHLiquidity_Paused();

    /// @notice Thrown when `setPaused` is not relayed from this chain's L1CrossDomainMessenger.
    error ETHLiquidity_NotL1CrossDomainMessenger();

    /// @notice Emitted when an address burns ETH liquidity.
    event LiquidityBurned(address indexed caller, uint256 value);

    /// @notice Emitted when an address mints ETH liquidity.
    event LiquidityMinted(address indexed caller, uint256 value);

    /// @notice Event to emit when funds are received
    event LiquidityFunded(address indexed funder, uint256 amount);

    /// @notice Emitted when the pause state synced from L1 is applied.
    /// @param paused      Whether the contract is now paused.
    /// @param l1Timestamp L1 timestamp at which the pause state was read.
    event PausedSet(bool paused, uint256 l1Timestamp);

    /// @notice Semantic version.
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice Whether burning and minting ETH liquidity is paused, as this chain was on L1 at
    ///         `pausedL1Timestamp`.
    bool public paused;

    /// @notice L1 timestamp at which the applied pause state was read on L1. An update read
    ///         earlier is ignored.
    uint256 public pausedL1Timestamp;

    /// @notice Allows an address to lock ETH liquidity into this contract.
    function burn() external payable {
        if (msg.sender != Predeploys.SUPERCHAIN_ETH_BRIDGE) revert Unauthorized();
        if (paused) revert ETHLiquidity_Paused();
        emit LiquidityBurned(msg.sender, msg.value);
    }

    /// @notice Allows an address to unlock ETH liquidity from this contract.
    /// @param _amount The amount of liquidity to unlock.
    function mint(uint256 _amount) external {
        if (msg.sender != Predeploys.SUPERCHAIN_ETH_BRIDGE) revert Unauthorized();
        if (paused) revert ETHLiquidity_Paused();
        new SafeSend{ value: _amount }(payable(msg.sender));
        emit LiquidityMinted(msg.sender, _amount);
    }

    /// @notice Fund the contract by sending ETH
    /// @dev The function is payable to accept ETH
    function fund() external payable {
        if (msg.value == 0) revert InvalidAmount();
        emit LiquidityFunded(msg.sender, msg.value);
    }

    /// @notice Applies this chain's pause state on L1, as read there at `_l1Timestamp`. Only this
    ///         chain's L1CrossDomainMessenger can send it, through
    ///         `L1CrossDomainMessenger.syncETHLiquidityPause`. An update read before the applied
    ///         one changes nothing, so replaying an older failed message can't undo a newer state.
    /// @param _paused      Whether this chain is paused on L1.
    /// @param _l1Timestamp L1 timestamp at which `_paused` was read.
    function setPaused(bool _paused, uint256 _l1Timestamp) external {
        if (
            msg.sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER
                || ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender()
                    != address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger())
        ) {
            revert ETHLiquidity_NotL1CrossDomainMessenger();
        }
        if (_l1Timestamp < pausedL1Timestamp) return;

        paused = _paused;
        pausedL1Timestamp = _l1Timestamp;

        emit PausedSet(_paused, _l1Timestamp);
    }
}
