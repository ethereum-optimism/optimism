// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { SuperchainETHBridge } from "src/L2/SuperchainETHBridge.sol";

// Libraries
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000024
/// @title SuperchainETHBridgePinned
/// @notice A SuperchainETHBridge that speaks ETH with exactly one counterpart chain, its "home"
///         chain, in both directions. Intended for a chain whose genesis we control and whose ETH
///         must have a single entrance and a single exit — a private chain in a public dependency
///         set. Pinning both directions here makes the home chain the sole mint/burn venue for this
///         chain's ETH, so the home chain's `netSent[thisChain]` is the one global running figure for
///         all ETH this chain holds, rather than a per-counterparty figure fragmented across the
///         depset.
/// @dev    `homeChainId` has NO setter and NO owner. The only thing that can write it is the chain's
///         genesis allocation, which is why it is plain storage rather than an immutable (predeploy
///         implementations are installed as deployed bytecode, which carries no constructor-set
///         immutables). An unconfigured contract is inert: with `homeChainId == 0` every send and
///         every relay reverts, so a misconfigured genesis fails closed rather than open.
contract SuperchainETHBridgePinned is SuperchainETHBridge {
    /// @notice Thrown when sending to, or relaying from, any chain other than the home chain.
    error NotHomeChain();

    /// @notice Chain ID of the only chain this bridge exchanges ETH with. Written by genesis
    ///         allocation only; zero means unconfigured, which disables the bridge entirely.
    uint256 public homeChainId;

    /// @custom:semver +home-pinned
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+home-pinned");
    }

    /// @inheritdoc SuperchainETHBridge
    /// @dev Reverts before any ETH is burned, so a misrouted send from this chain is impossible
    ///      rather than merely unprofitable: the user keeps their ETH.
    function sendETH(address _to, uint256 _chainId) public payable override returns (bytes32 msgHash_) {
        if (_chainId == 0 || _chainId != homeChainId) revert NotHomeChain();
        msgHash_ = super.sendETH(_to, _chainId);
    }

    /// @inheritdoc SuperchainETHBridge
    /// @dev Belt and braces relative to the inherited net-flow cap: because this chain can only ever
    ///      send to the home chain, `netSent` for any other chain is permanently zero and the cap
    ///      would already refuse the relay. The explicit check states the intent rather than leaving
    ///      it as a consequence.
    function relayETH(address _from, address _to, uint256 _amount) public override {
        // Repeated from the base so that a non-messenger caller still gets `Unauthorized` rather than
        // the messenger's not-entered error from the context read below.
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert Unauthorized();

        (, uint256 source) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (source == 0 || source != homeChainId) revert NotHomeChain();

        super.relayETH(_from, _to, _amount);
    }
}
