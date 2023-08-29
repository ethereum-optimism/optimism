// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface ILockedGold {
    function lock() external payable;
    function incrementNonvotingAccountBalance(address, uint256) external;
    function decrementNonvotingAccountBalance(address, uint256) external;
    function getAccountTotalLockedGold(address) external view returns (uint256);
    function getTotalLockedGold() external view returns (uint256);
    function getPendingWithdrawals(address) external view returns (uint256[] memory, uint256[] memory);
    function getPendingWithdrawal(address account, uint256 index) external view returns (uint256, uint256);
    function getTotalPendingWithdrawals(address) external view returns (uint256);
    function unlock(uint256) external;
    function relock(uint256, uint256) external;
    function withdraw(uint256) external;
    function slash(
        address account,
        uint256 penalty,
        address reporter,
        uint256 reward,
        address[] calldata lessers,
        address[] calldata greaters,
        uint256[] calldata indices
    )
        external;
    function isSlasher(address) external view returns (bool);
    function unlockingPeriod() external view returns (uint256);
    function getAccountNonvotingLockedGold(address account) external view returns (uint256);
}
