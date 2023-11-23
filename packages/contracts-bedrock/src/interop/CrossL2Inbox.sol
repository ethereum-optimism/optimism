// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

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
    /// chain ID => output root => bool.
    mapping(bytes32 => mapping(bytes32 => bool)) public roots;

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
}
