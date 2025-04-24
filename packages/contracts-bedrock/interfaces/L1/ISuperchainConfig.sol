// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ISuperchainConfig {
    enum UpdateType {
        GUARDIAN
    }

    event ConfigUpdate(UpdateType indexed updateType, bytes data);
    event Initialized(uint8 version);
    event Paused(address identifier);
    event Unpaused(address identifier);

    error SuperchainConfig_OnlyGuardian();
    error SuperchainConfig_AlreadyPaused(address identifier);
    error ReinitializableBase_ZeroInitVersion();

    function guardian() external view returns (address);
    function initialize(address _guardian) external;
    function upgrade() external;
    function pause(address _identifier) external;
    function unpause(address _identifier) external;
    function pausable(address _identifier) external view returns (bool);
    function paused(address _identifier) external view returns (bool);
    function expiration(address _identifier) external view returns (uint256);
    function extend(address _identifier) external;
    function version() external view returns (string memory);
    function pauseTimestamps(address) external view returns (uint256);
    function pauseExpiry() external view returns (uint256);
    function initVersion() external view returns (uint8);

    function __constructor__() external;
}
