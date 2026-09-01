// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title EventReplayer
/// @notice Generic batch-authenticated log emitter, deployed at a fixed address in the genesis of a
///         private chain's public rendering. `L2ToL2CrossDomainMessengerReplay` renders the
///         messenger's exports at the messenger's own predeploy address; this contract renders
///         everything else a devnet export policy chooses to make public, at one well-known
///         address, without the operator having to ship a bespoke contract per event shape.
///
///         Logs emitted here carry this contract's address as their emitter, so they are never
///         mistaken for the private chain's messenger traffic. Nothing consumes them as protocol
///         input; they exist for indexers, explorers and tests observing the public rendering.
contract EventReplayer is ISemver {
    /// @notice Thrown when more than four topics are supplied. The EVM has no `log5`.
    error EventReplayer_TooManyTopics();

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Emits an arbitrary log with zero to four topics. The log is emitted verbatim: no
    ///         topic is derived, added or reordered, so an operator can reproduce any log shape
    ///         the private chain produced.
    ///
    /// @param _topics Topics of the log, in order. At most four.
    /// @param _data   Data section of the log.
    function replayEvent(bytes32[] calldata _topics, bytes calldata _data) external {
        uint256 count = _topics.length;
        if (count > 4) revert EventReplayer_TooManyTopics();

        bytes32 topic0;
        bytes32 topic1;
        bytes32 topic2;
        bytes32 topic3;
        if (count > 0) topic0 = _topics[0];
        if (count > 1) topic1 = _topics[1];
        if (count > 2) topic2 = _topics[2];
        if (count > 3) topic3 = _topics[3];

        assembly {
            // Copy the data section into scratch space above the free memory pointer. Nothing is
            // allocated afterwards, so the free memory pointer does not need to be advanced.
            let ptr := mload(0x40)
            calldatacopy(ptr, _data.offset, _data.length)

            switch count
            case 0 { log0(ptr, _data.length) }
            case 1 { log1(ptr, _data.length, topic0) }
            case 2 { log2(ptr, _data.length, topic0, topic1) }
            case 3 { log3(ptr, _data.length, topic0, topic1, topic2) }
            default { log4(ptr, _data.length, topic0, topic1, topic2, topic3) }
        }
    }
}
