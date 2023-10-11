// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.8.15;

import {Test} from "forge-std/Test.sol";
import {DataAvailabilityChallenge, ChallengeStatus, Challenge} from "../src/L1/DataAvailabilityChallenge.sol";

address constant DAC_OWNER = address(1234);
uint256 constant CHALLENGE_WINDOW = 1000;
uint256 constant RESOLVE_WINDOW = 1000;
uint256 constant BOND_SIZE = 1000;

contract DataAvailabilityChallengeTest is Test {
    DataAvailabilityChallenge public dac;

    function setUp() public {
        dac = new DataAvailabilityChallenge();
        dac.initialize(DAC_OWNER, CHALLENGE_WINDOW, RESOLVE_WINDOW, BOND_SIZE);
    }

    function testDeposit() public {
        assertEq(dac.balances(address(this)), 0);
        dac.deposit{value: 1000}();
        assertEq(dac.balances(address(this)), 1000);
    }

    function testWithdraw(address sender, uint256 amount) public {
        assumePayable(sender);
        vm.assume(sender.balance == 0);
        vm.deal(sender, amount);

        vm.prank(sender);
        dac.deposit{value: amount}();

        assertEq(dac.balances(sender), amount);
        assertEq(sender.balance, 0);

        vm.prank(sender);
        dac.withdraw();

        assertEq(dac.balances(sender), 0);
        assertEq(sender.balance, amount);
    }

    function testChallengeSuccess(address challenger, uint256 challengedBlockNumber, bytes32 challengedHash) public {
        // Assume the block number is not close to the max uint256 value
        vm.assume(challengedBlockNumber < type(uint256).max - dac.challengeWindow() - dac.resolveWindow());
        uint256 requiredBond = dac.bondSize();

        // Move to a block after the challenged block
        vm.roll(challengedBlockNumber + 1);

        // Deposit the required bond
        vm.deal(challenger, requiredBond);
        vm.prank(challenger);
        dac.deposit{value: requiredBond}();

        // Challenge a (blockNumber,hash) tuple
        vm.prank(challenger);
        dac.challenge(challengedBlockNumber, challengedHash);

        // Challenge should have been created
        (ChallengeStatus _status, address _challenger, uint256 _startBlock) =
            dac.challenges(challengedBlockNumber, challengedHash);
        assertEq(uint8(_status), uint8(ChallengeStatus.Active));
        assertEq(_challenger, challenger);
        assertEq(_startBlock, block.number);

        // Challenge should have decreased the challenger's bond size
        assertEq(dac.balances(challenger), 0);
    }

    function testChallengeFailBondTooLow() public {
        uint256 requiredBond = dac.bondSize();
        uint256 actualBond = requiredBond - 1;
        dac.deposit{value: actualBond}();

        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.BondTooLow.selector, actualBond, requiredBond));
        dac.challenge(0, "some hash");
    }

    function testChallengeFailChallengeExists() public {
        // Move to a block after the hash to challenge
        vm.roll(2);

        // First challenge succeeds
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(0, "some hash");

        // Second challenge of the same hash/blockNumber fails
        dac.deposit{value: dac.bondSize()}();
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeExists.selector));
        dac.challenge(0, "some hash");

        // Challenge succeed if the challenged block number is different
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(1, "some hash");

        // Challenge succeed if the challenged hash is different
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(0, "some other hash");
    }

    function testChallengeFailBeforeChallengeWindow() public {
        uint256 challengeBlock = 1;
        bytes32 challengeHash = "some hash";

        // Move to challenged block
        vm.roll(challengeBlock);

        // Challenge fails because the current block number must be after the challenged block
        dac.deposit{value: dac.bondSize()}();
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeWindowNotOpen.selector));
        dac.challenge(challengeBlock, challengeHash);
    }

    function testChallengeFailAfterChallengeWindow() public {
        uint256 challengeBlock = 1;
        bytes32 challengeHash = "some hash";

        // Move to block after the challenge window
        vm.roll(challengeBlock + dac.challengeWindow() + 1);

        // Challenge fails because the block number is after the challenge window
        dac.deposit{value: dac.bondSize()}();
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeWindowNotOpen.selector));
        dac.challenge(challengeBlock, challengeHash);
    }

    function testResolveSuccess(bytes memory preImage, uint256 challengeBlock) public {
        // Assume the block number is not close to the max uint256 value
        vm.assume(challengeBlock < type(uint256).max - dac.challengeWindow() - dac.resolveWindow());
        bytes32 challengeHash = keccak256(preImage);

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Resolve the challenge
        dac.resolve(challengeBlock, preImage);

        // Expect the challenge to be resolved
        (ChallengeStatus _status, address _challenger, uint256 _startBlock) =
            dac.challenges(challengeBlock, challengeHash);

        assertEq(uint8(_status), uint8(ChallengeStatus.Resolved));
        assertEq(_challenger, address(this));
        assertEq(_startBlock, block.number);
    }

    function testResolveFailNonExistentChallenge() public {
        bytes memory preImage = "some preimage";
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Resolving a non-existent challenge fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeNotActive.selector));
        dac.resolve(challengeBlock, preImage);
    }

    function testResolveFailResolved() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Resolve the challenge
        dac.resolve(challengeBlock, preImage);

        // Resolving an already resolved challenge fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeNotActive.selector));
        dac.resolve(challengeBlock, preImage);
    }

    function testResolveFailExpired() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Move to a block after the resolve window
        vm.roll(block.number + dac.resolveWindow() + 1);

        // Expire the challenge
        dac.expire(challengeBlock, challengeHash);

        // Resolving an expired challenge fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeNotActive.selector));
        dac.resolve(challengeBlock, preImage);
    }

    function testResolveFailAfterResolveWindow() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Move to block after resolve window
        vm.roll(block.number + dac.resolveWindow() + 1);

        // Resolve the challenge
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ResolveWindowNotOpen.selector));
        dac.resolve(challengeBlock, preImage);
    }

    function testExpireSuccess(bytes memory preImage, uint256 challengeBlock) public {
        // Assume the block number is not close to the max uint256 value
        vm.assume(challengeBlock < type(uint256).max - dac.challengeWindow() - dac.resolveWindow());
        bytes32 challengeHash = keccak256(preImage);

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Move to a block after the resolve window
        vm.roll(block.number + dac.resolveWindow() + 1);

        // Expire the challenge
        dac.expire(challengeBlock, challengeHash);
    }

    function testExpireFailNonExistentChallenge() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Expiring a non-existent challenge fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeNotActive.selector));
        dac.expire(challengeBlock, challengeHash);
    }

    function testExpireFailResolvedChallenge() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Resolve the challenge
        dac.resolve(challengeBlock, preImage);

        // Expiring a resolved challenge fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeNotActive.selector));
        dac.expire(challengeBlock, challengeHash);
    }

    function testExpireFailExpiredChallenge() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        // Move to a block after the challenge window
        vm.roll(block.number + dac.resolveWindow() + 1);

        // Expire the challenge
        dac.expire(challengeBlock, challengeHash);

        // Expiring an expired challenge fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeNotActive.selector));
        dac.expire(challengeBlock, challengeHash);
    }

    function testExpireFailResolveWindowNotClosed() public {
        bytes memory preImage = "some preimage";
        bytes32 challengeHash = keccak256(preImage);
        uint256 challengeBlock = 1;

        // Move to block after challenged block
        vm.roll(challengeBlock + 1);

        // Challenge the hash
        dac.deposit{value: dac.bondSize()}();
        dac.challenge(challengeBlock, challengeHash);

        vm.roll(block.number + dac.resolveWindow() - 1);

        // Expiring the challenge before the resolve window closes fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ResolveWindowNotClosed.selector));
        dac.expire(challengeBlock, challengeHash);
    }
}
