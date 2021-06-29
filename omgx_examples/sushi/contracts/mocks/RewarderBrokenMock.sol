// SPDX-License-Identifier: MIT

pragma solidity 0.6.12;
import "../interfaces/IRewarder.sol";


contract RewarderBrokenMock is IRewarder {

    function onSushiReward (uint256, address, address, uint256, uint256) override external {
        revert();
    }

    function pendingTokens(uint256 pid, address user, uint256 sushiAmount) override external view returns (IERC20[] memory rewardTokens, uint256[] memory rewardAmounts){
        revert();
    }
  
}