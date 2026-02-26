// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IL2ToL1MessagePasserCGT {
    error L2ToL1MessagePasserCGT_NotAllowedOnCGTMode();
    error L2ToL1MessagePasser_OnlyCompliance();
    error L2ToL1MessagePasser_OnlyProxyAdminOwner();

    event MessagePassed(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data,
        bytes32 withdrawalHash
    );
    event WithdrawerBalanceBurnt(uint256 indexed amount);

    receive() external payable;

    function MESSAGE_VERSION() external view returns (uint16);
    function approved(address _from, address _target, uint256 _value, uint64 _gasLimit, bytes memory _data, uint256 _nonce) external payable;
    function burn() external;
    function compliance() external view returns (address);
    function donateETH() external payable;
    function initiateWithdrawal(address _target, uint256 _gasLimit, bytes memory _data) external payable;
    function messageNonce() external view returns (uint256);
    function sentMessages(bytes32) external view returns (bool);
    function setCompliance(address _compliance) external;
    function version() external view returns (string memory);

    function __constructor__() external;
}
