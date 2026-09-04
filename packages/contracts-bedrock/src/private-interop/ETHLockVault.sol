// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { SafeSend } from "src/universal/SafeSend.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { INativeMintBridge } from "interfaces/private-interop/INativeMintBridge.sol";

/// @title ETHLockVault
/// @notice Counterparty half of the application-level ETH bridge into a private chain. Deployed as
///         an ordinary contract (not a predeploy) on a public, ETH-native interop chain. ETH sent
///         to `lock` stays here and a cross domain message instructs the private chain's
///         `NativeMintBridge` to mint an equal amount of the private chain's native asset; ETH
///         leaves only through `unlock`, which the same bridge calls back after burning that
///         native asset. `totalLocked` is therefore the solvency cap of the private chain's
///         ETH-denominated supply: the private chain can never hand back more than entered through
///         `lock`, even if ETH is forced into this address by another route.
///
///         The protocol ETH path (`SuperchainETHBridge`) is deliberately not used. The private
///         chain is a custom gas token chain, so its native unit is not ETH and the protocol path
///         would mint ETH against a burn of something else. This vault is the only door.
///
///         v1 SIMPLIFYING ASSUMPTION — NO EXPIRY, NO RECLAIM, NO ESCAPE HATCH. A locked deposit is
///         released only by a message from the configured private-side bridge. Messages are
///         assumed to be received: if the operator stops rendering the private chain, stops
///         relaying, or loses the private chain's state, the ETH in this vault stays here with no
///         on-chain way out. Locked ETH depends on operator liveness. A worst-case-withdrawal
///         policy (timeout refunds, governance recovery, a proof-backed exit) is explicitly out of
///         scope for v1 and must be designed before this contract holds value anyone else owns.
contract ETHLockVault is ISemver {
    /// @notice Thrown when the caller is not the `L2ToL2CrossDomainMessenger`.
    error ETHLockVault_Unauthorized();

    /// @notice Thrown when the cross domain message sender is not the configured private-side
    ///         bridge.
    error ETHLockVault_InvalidCrossDomainSender();

    /// @notice Thrown when the cross domain message source is not the configured private chain.
    error ETHLockVault_InvalidCrossDomainSource();

    /// @notice Thrown when a zero address is supplied as a recipient.
    error ETHLockVault_ZeroAddress();

    /// @notice Thrown when a zero amount is locked or unlocked.
    error ETHLockVault_ZeroAmount();

    /// @notice Thrown when the private chain asks to unlock more ETH than entered through `lock`.
    error ETHLockVault_InsufficientLocked();

    /// @notice Emitted when ETH is locked and a mint is requested on the private chain.
    ///
    /// @param from      Address that locked the ETH.
    /// @param recipient Address that receives the minted native asset on the private chain.
    /// @param amount    Amount of ETH locked.
    /// @param msgHash   Hash of the cross domain message that was sent.
    event ETHLocked(address indexed from, address indexed recipient, uint256 amount, bytes32 msgHash);

    /// @notice Emitted when ETH is unlocked on behalf of the private chain.
    ///
    /// @param to     Address that received the unlocked ETH.
    /// @param amount Amount of ETH unlocked.
    event ETHUnlocked(address indexed to, uint256 amount);

    /// @notice Semantic version.
    /// @custom:semver 1.1.0
    string public constant version = "1.1.0";

    /// @notice Amount of ETH currently backing native asset issued through the private bridge.
    ///         Forced ETH transfers do not increase this value and therefore cannot be withdrawn
    ///         through an unbacked private-chain message.
    uint256 public totalLocked;

    /// @notice Chain ID of the private chain this vault bridges to.
    uint256 internal immutable PRIVATE_CHAIN_ID;

    /// @notice Address of the `NativeMintBridge` on the private chain. The only cross domain
    ///         message sender allowed to unlock ETH from this vault.
    address internal immutable PRIVATE_BRIDGE;

    /// @param _privateChainId Chain ID of the private chain this vault bridges to.
    /// @param _privateBridge  Address of the `NativeMintBridge` on the private chain.
    constructor(uint256 _privateChainId, address _privateBridge) {
        PRIVATE_CHAIN_ID = _privateChainId;
        PRIVATE_BRIDGE = _privateBridge;
    }

    /// @notice Getter for the chain ID of the private chain this vault bridges to.
    ///
    /// @return Chain ID of the private chain.
    function privateChainId() public view returns (uint256) {
        return PRIVATE_CHAIN_ID;
    }

    /// @notice Getter for the address of the `NativeMintBridge` on the private chain.
    ///
    /// @return Address of the private chain's `NativeMintBridge`.
    function privateBridge() public view returns (address) {
        return PRIVATE_BRIDGE;
    }

    /// @notice Locks the ETH sent with this call and asks the private chain to mint the same
    ///         amount of its native asset to `_recipient`. The ETH is held by this contract until
    ///         a matching `unlock` message comes back; see the contract-level notice for the v1
    ///         liveness assumption.
    ///
    /// @param _recipient Address to receive the minted native asset on the private chain.
    ///
    /// @return msgHash_ Hash of the cross domain message that was sent.
    function lock(address _recipient) external payable returns (bytes32 msgHash_) {
        if (_recipient == address(0)) revert ETHLockVault_ZeroAddress();
        if (msg.value == 0) revert ETHLockVault_ZeroAmount();

        totalLocked += msg.value;

        msgHash_ = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _destination: PRIVATE_CHAIN_ID,
            _target: PRIVATE_BRIDGE,
            _message: abi.encodeCall(INativeMintBridge.relayMint, (_recipient, msg.value))
        });

        emit ETHLocked(msg.sender, _recipient, msg.value, msgHash_);
    }

    /// @notice Releases locked ETH. Callable only through a relay by the
    ///         `L2ToL2CrossDomainMessenger` of a message sent by the configured private-side
    ///         bridge on the configured private chain. Reverts if this vault does not hold enough
    ///         ETH, which makes `totalLocked` a hard cap on what the private chain can
    ///         withdraw no matter what it claims.
    ///
    /// @param _to     Address to send the unlocked ETH to.
    /// @param _amount Amount of ETH to unlock.
    function unlock(address _to, uint256 _amount) external {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert ETHLockVault_Unauthorized();

        (address sender, uint256 source) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (sender != PRIVATE_BRIDGE) revert ETHLockVault_InvalidCrossDomainSender();
        if (source != PRIVATE_CHAIN_ID) revert ETHLockVault_InvalidCrossDomainSource();
        if (_to == address(0)) revert ETHLockVault_ZeroAddress();
        if (_amount == 0) revert ETHLockVault_ZeroAmount();
        if (_amount > totalLocked) revert ETHLockVault_InsufficientLocked();

        totalLocked -= _amount;

        // This is a forced ETH send to the recipient, the recipient should NOT expect to be called.
        new SafeSend{ value: _amount }(payable(_to));

        emit ETHUnlocked(_to, _amount);
    }
}
