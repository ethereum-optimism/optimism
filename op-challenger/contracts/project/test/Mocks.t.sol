// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";

import { GameType, Claim } from "src/Types.sol";
import { MockDisputeGameFactory } from "src/MockDisputeGameFactory.sol";

contract MocksTest is Test {
    MockDisputeGameFactory factory;

    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    function setUp() public {
        factory = new MockDisputeGameFactory();
    }

    function testCreate(uint8 gameType, Claim rootClaim, uint256 l2BlockNumber) external {
        vm.assume(gameType <= 2);
        vm.expectEmit(false, true, true, true);
        emit DisputeGameCreated(address(0), GameType(gameType), rootClaim);
        factory.create(GameType(gameType), rootClaim, abi.encodePacked(l2BlockNumber));
    }
}
