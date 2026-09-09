// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAtomicCallExample {
    error AtomicCallExample_LimitExceeded();

    function version() external view returns (string memory);
    function result() external view returns (uint256);
    function catchRemoteFailure(address _proxy) external;
    function run(address _proxy, uint256 _amount, uint256 _limit) external returns (uint256 result_);
    function runAcross(address _first, address _second, uint256 _amount) external returns (uint256 result_);
}
