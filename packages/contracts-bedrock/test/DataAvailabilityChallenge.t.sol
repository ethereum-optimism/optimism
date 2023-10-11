// SPDX-License-Identifier: UNLICENSED
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { DataAvailabilityChallenge, ChallengeStatus, Challenge } from "../src/L1/DataAvailabilityChallenge.sol";
import { Proxy } from "src/universal/Proxy.sol";

address constant DAC_OWNER = address(1234);
uint256 constant CHALLENGE_WINDOW = 1000;
uint256 constant RESOLVE_WINDOW = 1000;
uint256 constant BOND_SIZE = 1000;

contract DataAvailabilityChallengeTest is Test {
    DataAvailabilityChallenge public dac;

    function setUp() public virtual {
        dac = new DataAvailabilityChallenge();
        dac.initialize(DAC_OWNER, CHALLENGE_WINDOW, RESOLVE_WINDOW, BOND_SIZE);
    }

    function testInitialize() public {
        assertEq(dac.owner(), DAC_OWNER);
        assertEq(dac.challengeWindow(), CHALLENGE_WINDOW);
        assertEq(dac.resolveWindow(), RESOLVE_WINDOW);
        assertEq(dac.bondSize(), BOND_SIZE);

        vm.expectRevert("Initializable: contract is already initialized");
        dac.initialize(DAC_OWNER, CHALLENGE_WINDOW, RESOLVE_WINDOW, BOND_SIZE);
    }

    function testDeposit() public {
        assertEq(dac.balances(address(this)), 0);
        dac.deposit{ value: 1000 }();
        assertEq(dac.balances(address(this)), 1000);
    }

    function testReceive() public {
        assertEq(dac.balances(address(this)), 0);
        (bool success,) = payable(address(dac)).call{ value: 1000 }("");
        assertTrue(success);
        assertEq(dac.balances(address(this)), 1000);
    }

    function testWithdraw(address sender, uint256 amount) public {
        assumePayable(sender);
        assumeNoPrecompiles(sender);
        vm.assume(sender != address(dac));
        vm.assume(sender.balance == 0);
        vm.deal(sender, amount);

        vm.prank(sender);
        dac.deposit{ value: amount }();

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
        dac.deposit{ value: requiredBond }();

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
        dac.deposit{ value: actualBond }();

        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.BondTooLow.selector, actualBond, requiredBond));
        dac.challenge(0, "some hash");
    }

    function testChallengeFailChallengeExists() public {
        // Move to a block after the hash to challenge
        vm.roll(2);

        // First challenge succeeds
        dac.deposit{ value: dac.bondSize() }();
        dac.challenge(0, "some hash");

        // Second challenge of the same hash/blockNumber fails
        dac.deposit{ value: dac.bondSize() }();
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeExists.selector));
        dac.challenge(0, "some hash");

        // Challenge succeed if the challenged block number is different
        dac.deposit{ value: dac.bondSize() }();
        dac.challenge(1, "some hash");

        // Challenge succeed if the challenged hash is different
        dac.deposit{ value: dac.bondSize() }();
        dac.challenge(0, "some other hash");
    }

    function testChallengeFailBeforeChallengeWindow() public {
        uint256 challengeBlock = 1;
        bytes32 challengeHash = "some hash";

        // Move to challenged block
        vm.roll(challengeBlock);

        // Challenge fails because the current block number must be after the challenged block
        dac.deposit{ value: dac.bondSize() }();
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeWindowNotOpen.selector));
        dac.challenge(challengeBlock, challengeHash);
    }

    function testChallengeFailAfterChallengeWindow() public {
        uint256 challengeBlock = 1;
        bytes32 challengeHash = "some hash";

        // Move to block after the challenge window
        vm.roll(challengeBlock + dac.challengeWindow() + 1);

        // Challenge fails because the block number is after the challenge window
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
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
        dac.deposit{ value: dac.bondSize() }();
        dac.challenge(challengeBlock, challengeHash);

        vm.roll(block.number + dac.resolveWindow() - 1);

        // Expiring the challenge before the resolve window closes fails
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ResolveWindowNotClosed.selector));
        dac.expire(challengeBlock, challengeHash);
    }

    function testSetChallengeWindow(address challenger, uint256 challengedBlockNumber, bytes32 challengedHash) public {
        // Assume the block number is not close to the max uint256 value
        vm.assume(challengedBlockNumber < type(uint256).max - dac.challengeWindow() - dac.resolveWindow());

        uint256 requiredBond = dac.bondSize();
        uint256 blockDistance = CHALLENGE_WINDOW + 1;

        // Move move to a block 100 blocks after the challenged block
        vm.roll(challengedBlockNumber + blockDistance);

        // Deposit the required bond
        vm.deal(challenger, requiredBond);
        vm.prank(challenger);
        dac.deposit{ value: requiredBond }();

        // Challenge a (blockNumber,hash) tuple
        // Expect challenge to fail because the challenge window is not open
        vm.prank(challenger);
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ChallengeWindowNotOpen.selector));
        dac.challenge(challengedBlockNumber, challengedHash);

        // Extend the challenge window
        vm.prank(DAC_OWNER);
        dac.setChallengeWindow(blockDistance);

        // Expect the challenge to succeed
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

    function testSetChallengeWindowFailOnlyOwner(address notOwner, uint256 newChallengeWindow) public {
        vm.assume(notOwner != DAC_OWNER);

        // Expect setting the challenge window to fail because the sender is not the owner
        vm.prank(notOwner);
        vm.expectRevert("Ownable: caller is not the owner");
        dac.setChallengeWindow(newChallengeWindow);
    }

    function testSetResolveWindow(bytes memory preImage, uint256 challengeBlock) public {
        // Assume the block number is not close to the max uint256 value
        vm.assume(challengeBlock < type(uint256).max - dac.challengeWindow() - dac.resolveWindow() - 1);
        bytes32 challengeHash = keccak256(preImage);

        uint256 challengeStartBlock = challengeBlock + 1;

        // Move to block after challenged block
        vm.roll(challengeStartBlock);

        // Challenge the hash
        dac.deposit{ value: dac.bondSize() }();
        dac.challenge(challengeBlock, challengeHash);

        // Move to a block after the resolve window
        uint256 blockDiff = dac.resolveWindow() + 1;
        vm.roll(challengeStartBlock + blockDiff);

        // Resolve the challenge
        // Except the resolve to fail because the resolve window is not open
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.ResolveWindowNotOpen.selector));
        dac.resolve(challengeBlock, preImage);

        // Extend the resolve window
        vm.prank(DAC_OWNER);
        dac.setResolveWindow(blockDiff);

        // Expect the resolve to succeed
        dac.resolve(challengeBlock, preImage);

        // Expect the challenge to be resolved
        (ChallengeStatus _status, address _challenger, uint256 _startBlock) =
            dac.challenges(challengeBlock, challengeHash);

        assertEq(uint8(_status), uint8(ChallengeStatus.Resolved));
        assertEq(_challenger, address(this));
        assertEq(_startBlock, challengeStartBlock);
    }

    function testSetResolveWindowFailOnlyOwner(address notOwner, uint256 newResolveWindow) public {
        vm.assume(notOwner != DAC_OWNER);

        // Expect setting the resolve window to fail because the sender is not the owner
        vm.prank(notOwner);
        vm.expectRevert("Ownable: caller is not the owner");
        dac.setResolveWindow(newResolveWindow);
    }

    function testSetBondSize() public {
        uint256 requiredBond = dac.bondSize();
        uint256 actualBond = requiredBond - 1;
        dac.deposit{ value: actualBond }();

        // Expect the challenge to fail because the bond is too low
        vm.expectRevert(abi.encodeWithSelector(DataAvailabilityChallenge.BondTooLow.selector, actualBond, requiredBond));
        dac.challenge(0, "some hash");

        // Reduce the required bond
        vm.prank(DAC_OWNER);
        dac.setBondSize(actualBond);

        // Expect the challenge to succeed
        dac.challenge(0, "some hash");
    }

    function testSetBondSizeFailOnlyOwner(address notOwner, uint256 newBondSize) public {
        vm.assume(notOwner != DAC_OWNER);

        // Expect setting the bond size to fail because the sender is not the owner
        vm.prank(notOwner);
        vm.expectRevert("Ownable: caller is not the owner");
        dac.setBondSize(newBondSize);
    }
}

contract DataAvailabilityChallengeProxyTest is DataAvailabilityChallengeTest {
    function setUp() public virtual override {
        Proxy proxy = new Proxy(address(this));
        proxy.upgradeTo(address(new DataAvailabilityChallenge()));
        dac = DataAvailabilityChallenge(payable(address(proxy)));
        dac.initialize(DAC_OWNER, CHALLENGE_WINDOW, RESOLVE_WINDOW, BOND_SIZE);
    }
}
