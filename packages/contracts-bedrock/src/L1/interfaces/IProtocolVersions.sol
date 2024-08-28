// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOwnableUpgradeable } from "src/universal/interfaces/IOwnableUpgradeable.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

type ProtocolVersion is uint256;

/// @title IProtocolVersions
/// @notice Interface for the ProtocolVersions contract.
interface IProtocolVersions is IOwnableUpgradeable, ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value REQUIRED_PROTOCOL_VERSION              Represents an update to the required protocol version.
    /// @custom:value RECOMMENDED_PROTOCOL_VERSION           Represents an update to the recommended protocol version.
    enum UpdateType {
        REQUIRED_PROTOCOL_VERSION,
        RECOMMENDED_PROTOCOL_VERSION
    }

    event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);

    function RECOMMENDED_SLOT() external view returns (bytes32);
    function REQUIRED_SLOT() external view returns (bytes32);
    function VERSION() external view returns (uint256);
    function initialize(address _owner, ProtocolVersion _required, ProtocolVersion _recommended) external;
    function recommended() external view returns (ProtocolVersion out_);
    function required() external view returns (ProtocolVersion out_);
    function setRecommended(ProtocolVersion _recommended) external;
    function setRequired(ProtocolVersion _required) external;
}
