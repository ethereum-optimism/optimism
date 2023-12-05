// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Types } from "src/libraries/Types.sol";
import { Constants } from "src/libraries/Constants.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @notice Entry to post to the inbox.
///         The postie may deliver multiple entries per mail delivery.
/// @custom:field chain   Chain identifier.
/// @custom:field output  Output-root of the chain.
struct InboxEntry {
    bytes32 chain;
    bytes32 output;
}

/// @custom:proxied
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox receives output-roots of any chain,
///         and makes the output-roots available for cross-L2 proving.
contract CrossL2Inbox is ISemver {
    /// @notice The address that is allowed to post into the inbox.
    /// This is temporary for Interop Milestone 0:
    /// this will be changed to a system-only address later.
    address internal immutable SUPERCHAIN_POSTIE;

    /// @notice The collection of output roots, by chain.
    /// source chain ID => output root => bool.
    /// Prototype shortcut: the "output root" is really just the storage-root of the CrossL2Outbox contract here.
    mapping(bytes32 => mapping(bytes32 => bool)) public roots;

    /// @notice Address of the cross L2 account which initiated a call in this cross L2 message.
    ///         If the of this variable is the default L2 sender address, then we are NOT inside of
    ///         a call to runCrossL2Transaction.
    address public crossL2Sender;

    /// @notice Source chain identifier from where the cross L2 call originated. Empty if not in a call.
    bytes32 public messageSourceChain;

    /// @notice A list of cross L2 message hashes which have been successfully consumed.
    mapping(bytes32 => bool) public consumedMessages;

    /// @notice Emitted when a cross L2 message has been relayed.
    /// @param messageRoot Root of the cross L2 message.
    /// @param success     Whether the cross L2 message call was successful.
    event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success);

    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    /// @notice Initialize the inbox.
    /// @param _superchainPostie  Address that will be allowed to deliver to the inbox.
    constructor(address _superchainPostie) {
        SUPERCHAIN_POSTIE = _superchainPostie;
    }

    /// @notice Getter for the SUPERCHAIN_POSTIE address.
    function superchainPostie() external view returns (address) {
        return SUPERCHAIN_POSTIE;
    }

    /// @notice The inbox receives mail from the postie.
    function deliverMail(InboxEntry[] calldata mail) external {
        require(msg.sender == SUPERCHAIN_POSTIE, "CrossL2Inbox: only postie can deliver mail");
        for (uint256 i = 0; i < mail.length; i++) {
            roots[mail[i].chain][mail[i].output] = true;
        }
    }

    /// @notice Verifies and executes a cross-L2 message.
    /// @param _msg            Cross L2 message to finalize.
    /// @param _l2OutputRoot   Cross L2 outbox root to prove against. Only previously posted output roots are accepted.
    /// @param _inclusionProof Inclusion proof of the CrossL2Outbox contract's storage root.
    function runCrossL2Message(
        Types.SuperchainMessage memory _msg,
        bytes32 _l2OutputRoot,
        bytes calldata _inclusionProof
    )
        external
    {
        // TODO: should check _msg.to to not get round-trip inbox/outbox interactions
        // that have the system contract address as _msg.from.

        require(
            roots[_msg.sourceChain][_l2OutputRoot],
            "CrossL2Inbox: must proof against known output root from message source chain"
        );

        // Prototype shortcut: we don't proof the CrossL2Outbox storage-root as part of the output root,
        // but just assume the output root is that storage-root.

        bytes32 messageRoot = Hashing.superchainMessageRoot(_msg);

        // Unlike the OptimismPortal, we do not register messages with a timestamp, nor verify any finalization period:
        // with cross-L2 messaging there is no dispute delay like on L1.

        // Compute the storage slot of the message root in the L2Outbox contract.
        // Refer to the Solidity documentation for more information on how storage layouts are
        // computed for mappings.
        bytes32 storageKey = keccak256(
            abi.encode(
                messageRoot,
                uint256(0) // The "sentMessages" mapping is at the first slot in the layout.
            )
        );

        // TODO: run precompile to verify the storageKey is part of the output root tree

        // Make sure that the crossL2Sender has not yet been set. The crossL2Sender is set to a value other
        // than the default value when a cross-L2 message call is being executed. This check is
        // a defacto reentrancy guard.
        require(
            crossL2Sender == Constants.DEFAULT_L2_SENDER, "CrossL2Inbox: can only trigger one call per cross L2 message"
        );

        // Check that this message has not already been consumed, this is replay protection.
        require(consumedMessages[messageRoot] == false, "CrossL2Inbox: message has already been consumed");

        // Mark the message as consumed so it can't be replayed.
        consumedMessages[messageRoot] = true;

        // Set the crossL2Sender so contracts know who triggered this call across L2.
        crossL2Sender = _msg.from;

        // set cross L2 source chain identifier so contracts know which L2 the message is coming from.
        messageSourceChain = _msg.sourceChain;

        // Trigger the call to the target contract. We use a custom low level method
        // SafeCall.callWithMinGas to ensure two key properties
        //   1. "To" contracts cannot force this call to run out of gas by returning a very large
        //      amount of data (and this is OK because we don't care about the returndata here).
        //   2. The amount of gas provided to the execution context of the target is at least the
        //      gas limit specified by the user. If there is not enough gas in the current context
        //      to accomplish this, `callWithMinGas` will revert.
        bool success = SafeCall.callWithMinGas(_msg.to, _msg.gasLimit, _msg.value, _msg.data);

        // Reset the crossL2Sender back to the default value.
        crossL2Sender = Constants.DEFAULT_L2_SENDER;
        // Reset the source chain back to the default value.
        messageSourceChain = bytes32(0);

        // All cross-L2 messages are unconditionally relayed. Replayability can
        // be achieved through contracts built on top of this contract
        emit CrossL2MessageRelayed(messageRoot, success);

        // Reverting here is useful for determining the exact gas cost to successfully execute the
        // sub call to the target contract if the minimum gas limit specified by the user would not
        // be sufficient to execute the sub call.
        if (success == false && tx.origin == Constants.ESTIMATION_ADDRESS) {
            revert("CrossL2Inbox: cross L2 message call execution failed");
        }
    }
}
