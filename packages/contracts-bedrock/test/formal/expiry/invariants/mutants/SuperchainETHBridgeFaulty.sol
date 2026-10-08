// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// MUTANT FOR TESTING ONLY. NOT A REAL CONTRACT. NEVER DEPLOY.
// Copy of src/L2/SuperchainETHBridge.sol at c7c51d79e2 with three faults: refundETH skips the expiry check and the
// already-refunded check, and relayETH pays `_from` instead of `_to`. Used only by the deterministic failing
// witnesses for RefundImpliesExpired, AtMostOneRefund and ETH conservation (ExpiryInvariants_FailingWitness_Test).

// Libraries
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeSend } from "src/universal/SafeSend.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";

/// @title SuperchainETHBridgeFaulty (MUTANT, test only)
/// @notice SuperchainETHBridge enables ETH transfers between chains within an interop cluster.
contract SuperchainETHBridgeFaulty is ISemver {
    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not
    /// SuperchainETHBridge.
    error InvalidCrossDomainSender();

    /// @notice Thrown when refunding a send whose message has not expired.
    error SuperchainETHBridge_MessageNotExpired();

    /// @notice Thrown when refunding a send that was already refunded.
    error SuperchainETHBridge_AlreadyRefunded();

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

    /// @notice Emitted when the ETH of an expired send is returned to its sender.
    /// @param from        Address that sent the ETH, and got it back.
    /// @param amount      Amount of ETH returned.
    /// @param messageHash Hash of the expired message.
    event RefundETH(address indexed from, uint256 amount, bytes32 indexed messageHash);

    /// @notice Semantic version.
    /// @custom:semver 1.1.0
    string public constant version = "1.1.0";

    /// @notice Mapping of message hashes to whether the ETH of that send was refunded.
    mapping(bytes32 => bool) public refunded;

    /// @notice Sends ETH to some target address on another chain.
    /// @param _to       Address to send ETH to.
    /// @param _chainId  Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent.
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_) {
        if (_to == address(0)) revert ZeroAddress();

        // NOTE: 'burn' will soon change to 'deposit'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).burn{ value: msg.value }();

        msgHash_ = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _destination: _chainId,
            _target: address(this),
            _message: abi.encodeCall(this.relayETH, (msg.sender, _to, msg.value))
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
        // MUTANT: pays the sender instead of the recipient.
        new SafeSend{ value: _amount }(payable(_from));

        emit RelayETH(_from, _to, _amount, source);
    }

    /// @notice Returns the ETH of a send whose message expired: its destination never relayed it,
    ///         and now never can. Anyone can call it, and the ETH goes to the sender. The arguments
    ///         are those of the send's message, which the message hash binds.
    /// @param _destination Chain ID of the destination chain of the send.
    /// @param _nonce       Nonce of the send's message.
    /// @param _from        Address that sent the ETH.
    /// @param _to          Address the ETH was sent to.
    /// @param _amount      Amount of ETH sent.
    function refundETH(uint256 _destination, uint256 _nonce, address _from, address _to, uint256 _amount) external {
        bytes32 messageHash = Hashing.hashL2toL2CrossDomainMessage({
            _destination: _destination,
            _source: block.chainid,
            _nonce: _nonce,
            _sender: address(this),
            _target: address(this),
            _message: abi.encodeCall(this.relayETH, (_from, _to, _amount))
        });

        // MUTANT: expiry check removed.
        // MUTANT: already-refunded check removed.

        refunded[messageHash] = true;

        // NOTE: 'mint' will soon change to 'withdraw'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(_amount);

        // This is a forced ETH send back to the sender, the sender should NOT expect to be called.
        new SafeSend{ value: _amount }(payable(_from));

        emit RefundETH(_from, _amount, messageHash);
    }
}
