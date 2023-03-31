// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/**
 * @title OracleBondManager
 * @notice The Bond Manager holds ether posted as a bond for a bond id.
 */
interface IBondManager {
    function post(bytes32 _id) external payable;

    function call(bytes32 _id, address _to) external returns (uint256);

    function next() external returns (uint256);
}
