// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IETHLockVault } from "interfaces/private-interop/IETHLockVault.sol";

/// @title NativeMintBridge
/// @notice Private-chain half of the application-level ETH bridge. Deployed into the private
///         chain's genesis and authorized as a minter on the `LiquidityController` predeploy, so
///         it can mint and burn the chain's custom native asset. It mints against ETH locked in
///         the counterparty chain's `ETHLockVault` and burns to release that ETH again.
///
///         This is the only path by which ETH-denominated value enters or leaves the private
///         chain. The protocol path (`SuperchainETHBridge`) is closed: the private chain is a
///         custom gas token chain, so a protocol `sendETH` would burn the custom unit while asking
///         a counterparty to mint real ETH. The counterparty refuses to relay ETH from this chain,
///         and the public rendering's replay messenger refuses to render messages sent by the
///         `SuperchainETHBridge` predeploy at all.
///
///         The v1 liveness assumption documented on `ETHLockVault` applies to this half too: a
///         burn here is only made whole by a message reaching the vault. There is no expiry, no
///         reclaim and no escape hatch.
contract NativeMintBridge is ISemver {
    /// @notice Thrown when the caller is not the `L2ToL2CrossDomainMessenger`.
    error NativeMintBridge_Unauthorized();

    /// @notice Thrown when the cross domain message sender is not the configured lock vault.
    error NativeMintBridge_InvalidCrossDomainSender();

    /// @notice Thrown when the cross domain message source is not the configured counterparty
    ///         chain.
    error NativeMintBridge_InvalidCrossDomainSource();

    /// @notice Thrown when a zero address is supplied as a recipient.
    error NativeMintBridge_ZeroAddress();

    /// @notice Thrown when a zero amount is minted or burned.
    error NativeMintBridge_ZeroAmount();

    /// @notice Emitted when the native asset is minted against ETH locked on the counterparty
    ///         chain.
    ///
    /// @param to     Address that received the minted native asset.
    /// @param amount Amount of native asset minted.
    event MintRelayed(address indexed to, uint256 amount);

    /// @notice Emitted when the native asset is burned and an unlock is requested on the
    ///         counterparty chain.
    ///
    /// @param from    Address that burned the native asset.
    /// @param to      Address that receives the unlocked ETH on the counterparty chain.
    /// @param amount  Amount of native asset burned.
    /// @param msgHash Hash of the cross domain message that was sent.
    event BurnAndUnlockSent(address indexed from, address indexed to, uint256 amount, bytes32 msgHash);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Chain ID of the counterparty chain that holds the lock vault.
    uint256 internal immutable COUNTERPARTY_CHAIN_ID;

    /// @notice Address of the `ETHLockVault` on the counterparty chain. The only cross domain
    ///         message sender allowed to mint native asset on this chain.
    address internal immutable LOCK_VAULT;

    /// @param _counterpartyChainId Chain ID of the counterparty chain that holds the lock vault.
    /// @param _lockVault           Address of the `ETHLockVault` on the counterparty chain.
    constructor(uint256 _counterpartyChainId, address _lockVault) {
        COUNTERPARTY_CHAIN_ID = _counterpartyChainId;
        LOCK_VAULT = _lockVault;
    }

    /// @notice Getter for the chain ID of the counterparty chain that holds the lock vault.
    ///
    /// @return Chain ID of the counterparty chain.
    function counterpartyChainId() public view returns (uint256) {
        return COUNTERPARTY_CHAIN_ID;
    }

    /// @notice Getter for the address of the `ETHLockVault` on the counterparty chain.
    ///
    /// @return Address of the counterparty chain's `ETHLockVault`.
    function lockVault() public view returns (address) {
        return LOCK_VAULT;
    }

    /// @notice Mints native asset to `_to`. Callable only through a relay by the
    ///         `L2ToL2CrossDomainMessenger` of a message sent by the configured lock vault on the
    ///         configured counterparty chain, which is what makes the vault's ETH balance the cap
    ///         on the native asset this bridge can create.
    ///
    /// @param _to     Address to receive the minted native asset.
    /// @param _amount Amount of native asset to mint.
    function relayMint(address _to, uint256 _amount) external {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert NativeMintBridge_Unauthorized();

        (address sender, uint256 source) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (sender != LOCK_VAULT) revert NativeMintBridge_InvalidCrossDomainSender();
        if (source != COUNTERPARTY_CHAIN_ID) revert NativeMintBridge_InvalidCrossDomainSource();
        if (_to == address(0)) revert NativeMintBridge_ZeroAddress();
        if (_amount == 0) revert NativeMintBridge_ZeroAmount();

        ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).mint(_to, _amount);

        emit MintRelayed(_to, _amount);
    }

    /// @notice Burns the native asset sent with this call and asks the counterparty chain to
    ///         release the same amount of ETH to `_to` from the lock vault. The burn happens
    ///         first, so the private chain's native supply is reduced before the unlock message
    ///         exists; see the contract-level notice for the v1 liveness assumption.
    ///
    /// @param _to Address to receive the unlocked ETH on the counterparty chain.
    ///
    /// @return msgHash_ Hash of the cross domain message that was sent.
    function burnAndUnlock(address _to) external payable returns (bytes32 msgHash_) {
        if (_to == address(0)) revert NativeMintBridge_ZeroAddress();
        if (msg.value == 0) revert NativeMintBridge_ZeroAmount();

        ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).burn{ value: msg.value }();

        msgHash_ = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _destination: COUNTERPARTY_CHAIN_ID,
            _target: LOCK_VAULT,
            _message: abi.encodeCall(IETHLockVault.unlock, (_to, msg.value))
        });

        emit BurnAndUnlockSent(msg.sender, _to, msg.value, msgHash_);
    }
}
