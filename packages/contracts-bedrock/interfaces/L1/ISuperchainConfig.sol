// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ISuperchainConfig {
    enum UpdateType {
        GUARDIAN
    }

    event ConfigUpdate(UpdateType indexed updateType, bytes data);
    event Initialized(uint8 version);
    event Paused(string identifier);
    event Unpaused();

    error OnlyGuardian();
    error PauseAlreadyUsed();

    function guardian() external view returns (address);
    function initialize(address _guardian, uint256 _pauseExpiry) external;
    function pause(address _identifier) external;
    function unpause(address _identifier) external;
    function pausable(address _identifier) external view returns (bool);
    function paused(address _identifier) external view returns (bool);
    function expiration(address _identifier) external view returns (uint256);
    function reset(address _identifier) external;
    function version() external view returns (string memory);
    function pauseUsed(address) external view returns (bool);
    function pauseTimestamps(address) external view returns (uint256);
    function pauseExpiry() external view returns (uint256);

    function __constructor__() external;
}
