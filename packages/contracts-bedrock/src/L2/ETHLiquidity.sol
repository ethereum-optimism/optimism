// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { SafeSend } from "src/universal/SafeSend.sol";
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

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
///         uint128.max balance to prevent liquidity shortages. While it is paused, ETH can't be
///         burned or minted, so the SuperchainETHBridge can't send, relay or refund ETH. By
///         default it follows the chain's pause on L1, which anyone can sync through
///         `L1CrossDomainMessenger.syncETHLiquidityPause`. The L2 ProxyAdmin owner can override
///         that, pausing or unpausing the contract regardless of L1. A pause longer than the
///         message expiry window leaves sends in flight to expire, and only senders that can call
///         `refundETH` get that ETH back.
contract ETHLiquidity is ProxyAdminOwnedBase, ISemver {
    /// @notice The L2 ProxyAdmin owner's override of the pause synced from L1.
    /// @custom:value None     The contract follows the pause synced from L1.
    /// @custom:value Paused   The contract is paused, whatever L1's pause.
    /// @custom:value Unpaused The contract is unpaused, whatever L1's pause.
    enum PauseOverride {
        None,
        Paused,
        Unpaused
    }

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
    /// @param paused       Whether this chain was paused on L1.
    /// @param messageNonce Nonce of the L1CrossDomainMessenger message that sent it.
    event PausedSet(bool paused, uint256 messageNonce);

    /// @notice Emitted when the L2 ProxyAdmin owner sets its override of the synced pause.
    /// @param pauseOverride The override now set.
    event PauseOverrideSet(PauseOverride pauseOverride);

    /// @notice Semantic version.
    /// @custom:semver 2.1.0
    string public constant version = "2.1.0";

    /// @notice Whether this chain was paused on L1 when the L1CrossDomainMessenger sent message
    ///         `pausedMessageNonce`.
    bool public l1Paused;

    /// @notice Nonce of the L1CrossDomainMessenger message that sent the applied pause state. An
    ///         update from an earlier message is ignored.
    uint256 public pausedMessageNonce;

    /// @notice The L2 ProxyAdmin owner's override of the pause synced from L1.
    PauseOverride public pauseOverride;

    /// @notice Allows an address to lock ETH liquidity into this contract.
    function burn() external payable {
        if (msg.sender != Predeploys.SUPERCHAIN_ETH_BRIDGE) revert Unauthorized();
        if (paused()) revert ETHLiquidity_Paused();
        emit LiquidityBurned(msg.sender, msg.value);
    }

    /// @notice Allows an address to unlock ETH liquidity from this contract.
    /// @param _amount The amount of liquidity to unlock.
    function mint(uint256 _amount) external {
        if (msg.sender != Predeploys.SUPERCHAIN_ETH_BRIDGE) revert Unauthorized();
        if (paused()) revert ETHLiquidity_Paused();
        new SafeSend{ value: _amount }(payable(msg.sender));
        emit LiquidityMinted(msg.sender, _amount);
    }

    /// @notice Fund the contract by sending ETH
    /// @dev The function is payable to accept ETH
    function fund() external payable {
        if (msg.value == 0) revert InvalidAmount();
        emit LiquidityFunded(msg.sender, msg.value);
    }

    /// @notice Applies this chain's pause state on L1, as sent by this chain's
    ///         L1CrossDomainMessenger through `L1CrossDomainMessenger.syncETHLiquidityPause`. That
    ///         messenger sends it as itself, which no message it relays can be, so no one else
    ///         can. An update from an earlier message than the applied one changes nothing, so
    ///         replaying an older failed message can't undo a newer state. While the L2
    ///         ProxyAdmin owner's override is set, the synced state is recorded but doesn't change
    ///         `paused()`.
    /// @param _paused       Whether this chain is paused on L1.
    /// @param _messageNonce Nonce of the L1CrossDomainMessenger message that sends this.
    function setPaused(bool _paused, uint256 _messageNonce) external {
        if (
            msg.sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER
                || ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender()
                    != address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger())
        ) {
            revert ETHLiquidity_NotL1CrossDomainMessenger();
        }
        if (_messageNonce <= pausedMessageNonce) return;

        l1Paused = _paused;
        pausedMessageNonce = _messageNonce;

        emit PausedSet(_paused, _messageNonce);
    }

    /// @notice Sets the L2 ProxyAdmin owner's override of the pause synced from L1. `None` hands
    ///         the pause back to the synced state.
    /// @param _pauseOverride The override to set.
    function setPauseOverride(PauseOverride _pauseOverride) external {
        _assertOnlyProxyAdminOwner();

        pauseOverride = _pauseOverride;

        emit PauseOverrideSet(_pauseOverride);
    }

    /// @notice Whether burning and minting ETH liquidity is paused: as the L2 ProxyAdmin owner's
    ///         override says if it is set, otherwise as this chain was paused on L1 when last
    ///         synced.
    /// @return Whether burning and minting ETH liquidity is paused.
    function paused() public view returns (bool) {
        if (pauseOverride == PauseOverride.None) return l1Paused;
        return pauseOverride == PauseOverride.Paused;
    }
}
