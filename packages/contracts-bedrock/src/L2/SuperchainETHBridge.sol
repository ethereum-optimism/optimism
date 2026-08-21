// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeSend } from "src/universal/SafeSend.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { IMessageExpiryRelay } from "interfaces/L2/IMessageExpiryRelay.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000024
/// @title SuperchainETHBridge
/// @notice SuperchainETHBridge enables ETH transfers between chains within an interop cluster.
contract SuperchainETHBridge is ISemver {
    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not
    /// SuperchainETHBridge.
    error InvalidCrossDomainSender();

    /// @notice Thrown when an expiry notification arrives for a message with no pending send.
    error NoPendingSend();

    /// @notice A cross-chain ETH send awaiting delivery, refundable if its message expires
    ///         undelivered on the destination chain.
    /// @custom:field from   Address that sent the ETH.
    /// @custom:field amount Amount of ETH sent.
    struct PendingETHSend {
        address from;
        uint256 amount;
    }

    /// @notice Emitted when ETH is sent from one chain to another.
    /// @param from          Address of the sender.
    /// @param to            Address of the recipient.
    /// @param amount        Amount of ETH sent.
    /// @param destination   Chain ID of the destination chain.
    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    /// @notice Emitted whenever ETH is successfully relayed on this chain.
    /// @param from          Address of the msg.sender of sendETH on the source chain.
    /// @param to            Address of the recipient.
    /// @param amount        Amount of ETH relayed.
    /// @param source        Chain ID of the source chain.
    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Emitted when ETH from an expired cross-chain send is refunded on this chain.
    /// @param from    Address that sent the ETH and received the refund.
    /// @param amount  Amount of ETH refunded.
    /// @param msgHash Hash of the expired message.
    event RefundETH(address indexed from, uint256 amount, bytes32 indexed msgHash);

    /// @notice Semantic version.
    /// @custom:semver 1.1.0
    string public constant version = "1.1.0";

    /// @notice Mapping of message hashes to their pending cross-chain ETH sends.
    mapping(bytes32 => PendingETHSend) public pendingETHSends;

    /// @notice Sends ETH to some target address on another chain.
    /// @param _to       Address to send ETH to.
    /// @param _chainId  Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent.
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_) {
        if (_to == address(0)) revert ZeroAddress();

        // NOTE: 'burn' will soon change to 'deposit'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).burn{ value: msg.value }();

        uint256 nonce = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).messageNonce();
        bytes memory message = abi.encodeCall(this.relayETH, (msg.sender, _to, msg.value));

        msgHash_ = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _destination: _chainId,
            _target: address(this),
            _message: message
        });

        pendingETHSends[msgHash_] = PendingETHSend({ from: msg.sender, amount: msg.value });
        IMessageExpiryRelay(Predeploys.MESSAGE_EXPIRY_RELAY).recordSentMessage({
            _destination: _chainId,
            _nonce: nonce,
            _target: address(this),
            _message: message
        });

        emit SendETH(msg.sender, _to, msg.value, _chainId);
    }

    /// @notice Relays ETH received from another chain.
    /// @param _from       Address of the msg.sender of sendETH on the source chain.
    /// @param _to         Address to relay ETH to.
    /// @param _amount     Amount of ETH to relay.
    function relayETH(address _from, address _to, uint256 _amount) external {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 source) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != address(this)) revert InvalidCrossDomainSender();

        // NOTE: 'mint' will soon change to 'withdraw'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(_amount);

        // This is a forced ETH send to the recipient, the recipient should NOT expect to be called.
        new SafeSend{ value: _amount }(payable(_to));

        emit RelayETH(_from, _to, _amount, source);
    }

    /// @notice Refunds the ETH of a cross-chain send whose message verifiably expired undelivered
    ///         on the destination chain. Called by the MessageExpiryRelay exactly once per expired
    ///         message; expiry can only be attested after the interop message expiry window has
    ///         passed, at which point delivery is permanently impossible, so a refund can never
    ///         double spend against a relay of the same message.
    /// @param _msgHash Hash of the expired message.
    function onMessageExpired(bytes32 _msgHash, uint256, uint256) external {
        if (msg.sender != Predeploys.MESSAGE_EXPIRY_RELAY) revert Unauthorized();

        PendingETHSend memory pendingSend = pendingETHSends[_msgHash];
        if (pendingSend.from == address(0)) revert NoPendingSend();

        delete pendingETHSends[_msgHash];

        // NOTE: 'mint' will soon change to 'withdraw'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(pendingSend.amount);

        // This is a forced ETH send back to the sender, the sender should NOT expect to be called.
        new SafeSend{ value: pendingSend.amount }(payable(pendingSend.from));

        emit RefundETH(pendingSend.from, pendingSend.amount, _msgHash);
    }
}
