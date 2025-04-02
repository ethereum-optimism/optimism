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

    function GUARDIAN_SLOT() external view returns (bytes32);
    function PAUSE_EXPIRY_SLOT() external view returns (bytes32);
    function guardian() external view returns (address guardian_);
    function initialize(address _guardian, uint256 _pauseExpiry) external;
    function pause(address _identifier) external;
    function unpause(address _identifier) external;
    function pausable(address _identifier) external view returns (bool);
    function paused(address _identifier) external view returns (bool);
    function expiration(address _identifier) external view returns (uint256);
    function reset(address _identifier) external;
    function version() external view returns (string memory);

    function __constructor__() external;
}
