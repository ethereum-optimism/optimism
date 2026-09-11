// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAtomicAccountFactory {
    function __constructor__() external;
    function version() external view returns (string memory);
    function accountImplementation() external view returns (address);
    function createAccount(address _owner, uint256 _salt) external returns (address account_);
    function getAddress(address _owner, uint256 _salt) external view returns (address account_);
}
