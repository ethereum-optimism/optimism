// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAtomicCounter {
    error AtomicCounter_LimitExceeded(uint256 value, uint256 limit);

    function version() external view returns (string memory);
    function value() external view returns (uint256);
    function limit() external view returns (uint256);
    function setLimit(uint256 _limit) external;
    function add(uint256 _amount) external returns (uint256);
}
