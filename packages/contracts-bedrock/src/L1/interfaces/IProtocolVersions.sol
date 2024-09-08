// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @notice ProtocolVersion is a numeric identifier of the protocol version.
type ProtocolVersion is uint256;

/// @title IProtocolVersions
/// @notice Interface for the IProtocolVersions contract.
interface IProtocolVersions is ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value REQUIRED_PROTOCOL_VERSION              Represents an update to the required protocol version.
    /// @custom:value RECOMMENDED_PROTOCOL_VERSION           Represents an update to the recommended protocol version.
    enum UpdateType {
        REQUIRED_PROTOCOL_VERSION,
        RECOMMENDED_PROTOCOL_VERSION
    }

    /// @notice Emitted when configuration is updated.
    /// @param version    ProtocolVersion version.
    /// @param updateType Type of update.
    /// @param data       Encoded update data.
    event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);

    function required() external view returns (ProtocolVersion out_);
    function setRequired(ProtocolVersion _required) external;
    function recommended() external view returns (ProtocolVersion out_);
    function setRecommended(ProtocolVersion _recommended) external;
}
