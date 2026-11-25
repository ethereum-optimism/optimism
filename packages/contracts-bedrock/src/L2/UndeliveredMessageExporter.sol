// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000030
/// @title UndeliveredMessageExporter
/// @notice Tells a message's source chain, through the withdrawal path, that the message has not
///         been relayed on this chain. The source chain's L1CrossDomainMessenger trusts withdrawals
///         from this predeploy and passes the word on to the source chain's
///         L2ToL2CrossDomainMessenger, which marks the message expired once its expiry period has
///         passed. This predeploy had no implementation before, so no earlier withdrawal from it
///         exists. Every withdrawal it sends carries only `relayUndeliveredMessage` calldata for a
///         message it checked.
contract UndeliveredMessageExporter is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Thrown when exporting a message that was relayed on this chain.
    error UndeliveredMessageExporter_MessageRelayed();

    /// @notice Emitted when a message to this chain is exported as not relayed.
    /// @param messageHash     Hash of the message.
    /// @param source          Chain ID of the message's source chain.
    /// @param sourceMessenger The source chain's L1CrossDomainMessenger the word is sent to.
    /// @param undeliveredAt   Timestamp at which the message had not been relayed.
    event UndeliveredMessageExported(
        bytes32 indexed messageHash, uint256 indexed source, address sourceMessenger, uint256 undeliveredAt
    );

    /// @notice Tells the source chain that a message to this chain has not been relayed by now.
    ///         Anyone can call it, and since it is not an executing message it can be forced in as
    ///         a deposit. The message hash is computed with this chain as the destination, so a
    ///         chain can only speak for messages to itself. An export before the source's expiry
    ///         period has passed is harmless: it becomes a failed message on the source chain that
    ///         can never succeed, and the message can be exported again later.
    /// @param _sourceMessenger The source chain's L1CrossDomainMessenger. If it is wrong, nothing
    ///                         happens and the message can be exported again.
    /// @param _source          Chain ID of the source chain.
    /// @param _nonce           Nonce of the message.
    /// @param _sender          Address that sent the message.
    /// @param _target          Target contract or wallet address.
    /// @param _message         Message payload.
    /// @param _minGasLimit     Minimum gas limit for the call on L1.
    /// @return messageHash_ Hash of the message.
    function exportUndeliveredMessage(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
        returns (bytes32 messageHash_)
    {
        messageHash_ = Hashing.hashL2toL2CrossDomainMessage({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });

        if (IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).successfulMessages(messageHash_)) {
            revert UndeliveredMessageExporter_MessageRelayed();
        }

        ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _target: _sourceMessenger,
            _message: abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (messageHash_, block.timestamp)),
            _minGasLimit: _minGasLimit
        });

        emit UndeliveredMessageExported(messageHash_, _source, _sourceMessenger, block.timestamp);
    }
}
