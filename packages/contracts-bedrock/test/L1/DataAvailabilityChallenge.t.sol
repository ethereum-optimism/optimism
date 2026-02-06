// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    IDataAvailabilityChallenge,
    ChallengeStatus,
    Challenge,
    CommitmentType
} from "interfaces/L1/IDataAvailabilityChallenge.sol";
import { computeCommitmentKeccak256 } from "src/L1/DataAvailabilityChallenge.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";

/// @title DataAvailabilityChallenge Test Init
/// @notice Test initialization for DataAvailabilityChallenge tests.
abstract contract DataAvailabilityChallenge_TestInit is CommonTest {
    function setUp() public virtual override {
        super.enableAltDA();
        super.setUp();
    }
}

/// @title DataAvailabilityChallenge_SetBondSize_Test
/// @notice Test contract for DataAvailabilityChallenge `setBondSize` function.
contract DataAvailabilityChallenge_SetBondSize_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `setBondSize` function succeeds.
    function test_setBondSize_succeeds() public {
        uint256 requiredBond = dataAvailabilityChallenge.bondSize();
        uint256 actualBond = requiredBond - 1;
        dataAvailabilityChallenge.deposit{ value: actualBond }();

        // Expect the challenge to fail because the bond is too low
        bytes memory challengedCommitment = computeCommitmentKeccak256("some hash");
        vm.expectRevert(
            abi.encodeWithSelector(IDataAvailabilityChallenge.BondTooLow.selector, actualBond, requiredBond)
        );
        dataAvailabilityChallenge.challenge(0, challengedCommitment);

        // Reduce the required bond
        vm.prank(dataAvailabilityChallenge.owner());
        dataAvailabilityChallenge.setBondSize(actualBond);

        // Expect the challenge to succeed
        dataAvailabilityChallenge.challenge(0, challengedCommitment);
    }

    /// @notice Test that the `setBondSize` function reverts if the sender is not the owner.
    function test_setBondSize_onlyOwner_reverts(address notOwner, uint256 newBondSize) public {
        vm.assume(notOwner != dataAvailabilityChallenge.owner());

        // Expect setting the bond size to fail because the sender is not the owner
        vm.prank(notOwner);
        vm.expectRevert("Ownable: caller is not the owner");
        dataAvailabilityChallenge.setBondSize(newBondSize);
    }
}

/// @title DataAvailabilityChallenge_SetResolverRefundPercentage_Test
/// @notice Test contract for DataAvailabilityChallenge `setResolverRefundPercentage` function.
contract DataAvailabilityChallenge_SetResolverRefundPercentage_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `setResolverRefundPercentage` function succeeds.
    function test_setResolverRefundPercentage_succeeds(uint256 resolverRefundPercentage) public {
        resolverRefundPercentage = bound(resolverRefundPercentage, 0, 100);
        vm.prank(dataAvailabilityChallenge.owner());
        dataAvailabilityChallenge.setResolverRefundPercentage(resolverRefundPercentage);
        assertEq(dataAvailabilityChallenge.resolverRefundPercentage(), resolverRefundPercentage);
    }

    /// @notice Test that the `setResolverRefundPercentage` function reverts if the resolver refund
    ///         percentage is invalid.
    function test_setResolverRefundPercentage_invalidResolverRefundPercentage_reverts() public {
        address owner = dataAvailabilityChallenge.owner();
        vm.expectRevert(
            abi.encodeWithSelector(IDataAvailabilityChallenge.InvalidResolverRefundPercentage.selector, 101)
        );
        vm.prank(owner);
        dataAvailabilityChallenge.setResolverRefundPercentage(101);
    }

    /// @notice Test that the `setResolverRefundPercentage` function reverts if sender is not owner.
    function testFuzz_setResolverRefundPercentage_onlyOwner_reverts(address _notOwner, uint256 _percentage) public {
        vm.assume(_notOwner != dataAvailabilityChallenge.owner());
        _percentage = bound(_percentage, 0, 100);

        vm.prank(_notOwner);
        vm.expectRevert("Ownable: caller is not the owner");
        dataAvailabilityChallenge.setResolverRefundPercentage(_percentage);
    }
}

/// @title DataAvailabilityChallenge_Receive_Test
/// @notice Test contract for DataAvailabilityChallenge `receive` function.
contract DataAvailabilityChallenge_Receive_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `receive` function succeeds with various amounts.
    function testFuzz_receive_succeeds(uint256 _amount) public {
        vm.deal(address(this), _amount);

        uint256 initialBalance = dataAvailabilityChallenge.balances(address(this));
        (bool success,) = payable(address(dataAvailabilityChallenge)).call{ value: _amount }("");
        assertTrue(success);
        assertEq(dataAvailabilityChallenge.balances(address(this)), initialBalance + _amount);
    }
}

/// @title DataAvailabilityChallenge_Deposit_Test
/// @notice Test contract for DataAvailabilityChallenge `deposit` function.
contract DataAvailabilityChallenge_Deposit_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `deposit` function succeeds with various amounts.
    function testFuzz_deposit_succeeds(uint256 _amount) public {
        vm.deal(address(this), _amount);

        uint256 initialBalance = dataAvailabilityChallenge.balances(address(this));
        dataAvailabilityChallenge.deposit{ value: _amount }();
        assertEq(dataAvailabilityChallenge.balances(address(this)), initialBalance + _amount);
    }
}

/// @title DataAvailabilityChallenge_Withdraw_Test
/// @notice Test contract for DataAvailabilityChallenge `withdraw` function.
contract DataAvailabilityChallenge_Withdraw_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `withdraw` function succeeds.
    function test_withdraw_succeeds(address sender, uint256 amount) public {
        assumePayable(sender);
        assumeNotPrecompile(sender);

        // EntryPoint will revert if using amount > type(uint112).max.
        vm.assume(sender != Preinstalls.EntryPoint_v060);
        vm.assume(sender != address(dataAvailabilityChallenge));
        // OptimismPortal will send ETH into the ETHLockbox.
        vm.assume(sender != address(optimismPortal2));
        vm.deal(sender, amount);

        vm.prank(sender);
        dataAvailabilityChallenge.deposit{ value: amount }();

        assertEq(dataAvailabilityChallenge.balances(sender), amount);
        assertEq(sender.balance, 0);

        vm.prank(sender);
        dataAvailabilityChallenge.withdraw();

        assertEq(dataAvailabilityChallenge.balances(sender), 0);
        assertEq(sender.balance, amount);
    }

    /// @notice Test that the `withdraw` function reverts if the withdrawal fails (e.g., sender is
    ///         a contract).
    function test_withdraw_fails_reverts(address sender, uint256 amount) public {
        assumePayable(sender);
        assumeNotPrecompile(sender);
        // EntryPoint will revert if using amount > type(uint112).max.
        vm.assume(sender != Preinstalls.EntryPoint_v060);
        vm.assume(sender != address(dataAvailabilityChallenge));
        vm.assume(sender != artifacts.mustGetAddress("DataAvailabilityChallengeImpl"));
        vm.deal(sender, amount);

        vm.prank(sender);
        dataAvailabilityChallenge.deposit{ value: amount }();

        assertEq(dataAvailabilityChallenge.balances(sender), amount);
        assertEq(sender.balance, 0);

        vm.etch(sender, hex"fe");
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.WithdrawalFailed.selector));
        dataAvailabilityChallenge.withdraw();
    }
}

/// @title DataAvailabilityChallenge_GetChallenge_Test
/// @notice Test contract for DataAvailabilityChallenge `getChallenge` function.
contract DataAvailabilityChallenge_GetChallenge_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `getChallenge` function returns uninitialized challenge.
    function test_getChallenge_uninitializedChallenge_succeeds() public view {
        bytes memory commitment = computeCommitmentKeccak256("test data");
        uint256 blockNumber = 100;

        Challenge memory challenge = dataAvailabilityChallenge.getChallenge(blockNumber, commitment);

        assertEq(challenge.challenger, address(0));
        assertEq(challenge.lockedBond, 0);
        assertEq(challenge.startBlock, 0);
        assertEq(challenge.resolvedBlock, 0);
    }

    /// @notice Test that the `getChallenge` function returns active challenge data.
    function testFuzz_getChallenge_activeChallenge_succeeds(
        address _challenger,
        uint256 _challengedBlockNumber,
        bytes memory _preImage
    )
        public
    {
        vm.assume(_challenger != address(0));
        _challengedBlockNumber =
            bound(_challengedBlockNumber, 0, type(uint256).max - dataAvailabilityChallenge.challengeWindow() - 1);

        bytes memory commitment = computeCommitmentKeccak256(_preImage);
        uint256 bondSize = dataAvailabilityChallenge.bondSize();

        vm.roll(_challengedBlockNumber + 1);
        vm.deal(_challenger, bondSize);
        vm.prank(_challenger);
        dataAvailabilityChallenge.challenge{ value: bondSize }(_challengedBlockNumber, commitment);

        Challenge memory challenge = dataAvailabilityChallenge.getChallenge(_challengedBlockNumber, commitment);

        assertEq(challenge.challenger, _challenger);
        assertEq(challenge.lockedBond, bondSize);
        assertEq(challenge.startBlock, block.number);
        assertEq(challenge.resolvedBlock, 0);
    }
}

/// @title DataAvailabilityChallenge_GetChallengeStatus_Test
/// @notice Test contract for DataAvailabilityChallenge `getChallengeStatus` function.
contract DataAvailabilityChallenge_GetChallengeStatus_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `getChallengeStatus` function returns correct status for each state.
    function test_getChallengeStatus_allStates_succeeds() public {
        bytes memory preImage = "test data";
        bytes memory commitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 100;
        uint256 bondSize = dataAvailabilityChallenge.bondSize();

        // Test uninitialized status
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, commitment)),
            uint8(ChallengeStatus.Uninitialized)
        );

        // Create active challenge
        vm.roll(challengedBlockNumber + 1);
        vm.deal(address(this), bondSize);
        dataAvailabilityChallenge.challenge{ value: bondSize }(challengedBlockNumber, commitment);

        // Test active status
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, commitment)),
            uint8(ChallengeStatus.Active)
        );

        // Resolve the challenge
        dataAvailabilityChallenge.resolve(challengedBlockNumber, commitment, preImage);

        // Test resolved status
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, commitment)),
            uint8(ChallengeStatus.Resolved)
        );
    }

    /// @notice Test that the `getChallengeStatus` function returns expired status.
    function test_getChallengeStatus_expiredChallenge_succeeds() public {
        bytes memory commitment = computeCommitmentKeccak256("test data");
        uint256 challengedBlockNumber = 100;
        uint256 bondSize = dataAvailabilityChallenge.bondSize();

        // Create challenge
        vm.roll(challengedBlockNumber + 1);
        vm.deal(address(this), bondSize);
        dataAvailabilityChallenge.challenge{ value: bondSize }(challengedBlockNumber, commitment);

        // Move past resolve window
        vm.roll(block.number + dataAvailabilityChallenge.resolveWindow() + 1);

        // Test expired status
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, commitment)),
            uint8(ChallengeStatus.Expired)
        );
    }

    /// @notice Test status transitions with fuzz testing.
    function testFuzz_getChallengeStatus_transitions_succeeds(
        uint256 _challengedBlockNumber,
        bytes memory _preImage
    )
        public
    {
        _challengedBlockNumber = bound(
            _challengedBlockNumber,
            0,
            type(uint256).max - dataAvailabilityChallenge.challengeWindow() - dataAvailabilityChallenge.resolveWindow()
                - 10
        );

        bytes memory commitment = computeCommitmentKeccak256(_preImage);
        uint256 bondSize = dataAvailabilityChallenge.bondSize();

        // Initially uninitialized
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(_challengedBlockNumber, commitment)),
            uint8(ChallengeStatus.Uninitialized)
        );

        // Create challenge and verify active
        vm.roll(_challengedBlockNumber + 1);
        vm.deal(address(this), bondSize);
        dataAvailabilityChallenge.challenge{ value: bondSize }(_challengedBlockNumber, commitment);

        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(_challengedBlockNumber, commitment)),
            uint8(ChallengeStatus.Active)
        );
    }
}

/// @title DataAvailabilityChallenge_ValidateCommitment_Test
/// @notice Test contract for DataAvailabilityChallenge `validateCommitment` function.
contract DataAvailabilityChallenge_ValidateCommitment_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `validateCommitment` function handles valid commitment.
    function test_validateCommitment_validCommitment_succeeds() public view {
        bytes memory validCommitment = abi.encodePacked(CommitmentType.Keccak256, keccak256("test"));
        dataAvailabilityChallenge.validateCommitment(validCommitment);
    }

    /// @notice Test that the `validateCommitment` function reverts for unknown commitment types.
    function testFuzz_validateCommitment_unknownType_reverts(uint8 _unknownType, bytes32 _hash) public {
        vm.assume(_unknownType != uint8(CommitmentType.Keccak256));

        bytes memory unknownTypeCommitment = abi.encodePacked(_unknownType, _hash);

        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.UnknownCommitmentType.selector, _unknownType));
        dataAvailabilityChallenge.validateCommitment(unknownTypeCommitment);
    }

    /// @notice Test that the `validateCommitment` function reverts for invalid lengths.
    function testFuzz_validateCommitment_invalidLength_reverts(uint8 _extraBytes) public {
        _extraBytes = uint8(bound(_extraBytes, 1, 100));

        bytes memory invalidLength =
            abi.encodePacked(CommitmentType.Keccak256, keccak256("test"), new bytes(_extraBytes));

        vm.expectRevert(
            abi.encodeWithSelector(
                IDataAvailabilityChallenge.InvalidCommitmentLength.selector,
                uint8(CommitmentType.Keccak256),
                33,
                33 + _extraBytes
            )
        );
        dataAvailabilityChallenge.validateCommitment(invalidLength);
    }
}

/// @title DataAvailabilityChallenge_Challenge_Test
/// @notice Test contract for DataAvailabilityChallenge `challenge` function.
contract DataAvailabilityChallenge_Challenge_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `challenge` function succeeds.
    function test_challenge_succeeds(
        address challenger,
        uint256 challengedBlockNumber,
        bytes calldata preImage
    )
        public
    {
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);

        // Assume the challenger is not the 0 address
        vm.assume(challenger != address(0));

        // Assume the block number is not close to the max uint256 value
        challengedBlockNumber = bound(
            challengedBlockNumber,
            0,
            type(uint256).max - dataAvailabilityChallenge.challengeWindow() - dataAvailabilityChallenge.resolveWindow()
                - 1
        );
        uint256 requiredBond = dataAvailabilityChallenge.bondSize();

        // Move to a block after the challenged block
        vm.roll(challengedBlockNumber + 1);

        // Deposit the required bond
        vm.deal(challenger, requiredBond);
        vm.prank(challenger);
        dataAvailabilityChallenge.deposit{ value: requiredBond }();

        // Expect the challenge status to be uninitialized
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, challengedCommitment)),
            uint8(ChallengeStatus.Uninitialized)
        );

        // Challenge a (blockNumber,hash) tuple
        vm.prank(challenger);
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Challenge should have been created
        Challenge memory challenge = dataAvailabilityChallenge.getChallenge(challengedBlockNumber, challengedCommitment);
        assertEq(challenge.challenger, challenger);
        assertEq(challenge.startBlock, block.number);
        assertEq(challenge.resolvedBlock, 0);
        assertEq(challenge.lockedBond, requiredBond);
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, challengedCommitment)),
            uint8(ChallengeStatus.Active)
        );

        // Challenge should have decreased the challenger's bond size
        assertEq(dataAvailabilityChallenge.balances(challenger), 0);
    }

    /// @notice Test that the `challenge` function succeeds when depositing bond in the same
    ///         transaction.
    function test_challenge_deposit_succeeds(
        address challenger,
        uint256 challengedBlockNumber,
        bytes memory preImage
    )
        public
    {
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);

        // Assume the challenger is not the 0 address
        vm.assume(challenger != address(0));

        // Assume the block number is not close to the max uint256 value
        challengedBlockNumber = bound(
            challengedBlockNumber,
            0,
            type(uint256).max - dataAvailabilityChallenge.challengeWindow() - dataAvailabilityChallenge.resolveWindow()
                - 1
        );
        uint256 requiredBond = dataAvailabilityChallenge.bondSize();

        // Move to a block after the challenged block
        vm.roll(challengedBlockNumber + 1);

        // Expect the challenge status to be uninitialized
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, challengedCommitment)),
            uint8(ChallengeStatus.Uninitialized)
        );

        // Deposit the required bond as part of the challenge transaction
        vm.deal(challenger, requiredBond);
        vm.prank(challenger);
        dataAvailabilityChallenge.challenge{ value: requiredBond }(challengedBlockNumber, challengedCommitment);

        // Challenge should have been created
        Challenge memory challenge = dataAvailabilityChallenge.getChallenge(challengedBlockNumber, challengedCommitment);
        assertEq(challenge.challenger, challenger);
        assertEq(challenge.startBlock, block.number);
        assertEq(challenge.resolvedBlock, 0);
        assertEq(challenge.lockedBond, requiredBond);
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, challengedCommitment)),
            uint8(ChallengeStatus.Active)
        );

        // Challenge should have decreased the challenger's bond size
        assertEq(dataAvailabilityChallenge.balances(challenger), 0);
    }

    /// @notice Test that the `challenge` function reverts if the bond is too low.
    function test_challenge_bondTooLow_reverts() public {
        uint256 requiredBond = dataAvailabilityChallenge.bondSize();
        uint256 actualBond = requiredBond - 1;
        dataAvailabilityChallenge.deposit{ value: actualBond }();

        vm.expectRevert(
            abi.encodeWithSelector(IDataAvailabilityChallenge.BondTooLow.selector, actualBond, requiredBond)
        );
        dataAvailabilityChallenge.challenge(0, computeCommitmentKeccak256("some hash"));
    }

    /// @notice Test that the `challenge` function reverts if the challenge already exists.
    function test_challenge_challengeExists_reverts() public {
        // Move to a block after the hash to challenge
        vm.roll(2);

        // First challenge succeeds
        bytes memory challengedCommitment = computeCommitmentKeccak256("some data");
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(0, challengedCommitment);

        // Second challenge of the same hash/blockNumber fails
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeExists.selector));
        dataAvailabilityChallenge.challenge(0, challengedCommitment);

        // Challenge succeed if the challenged block number is different
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(1, challengedCommitment);

        // Challenge succeed if the challenged hash is different
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(0, computeCommitmentKeccak256("some other hash"));
    }

    /// @notice Test that the `challenge` function reverts if the current block number is before
    ///         the challenged block.
    function test_challenge_beforeChallengeWindow_reverts() public {
        uint256 challengedBlockNumber = 1;
        bytes memory challengedCommitment = computeCommitmentKeccak256("some hash");

        // Move to challenged block
        vm.roll(challengedBlockNumber - 1);

        // Challenge fails because the current block number must be after the challenged block
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeWindowNotOpen.selector));
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);
    }

    /// @notice Test that the `challenge` function reverts if the current block number is after
    ///         the challenge window.
    function test_challenge_afterChallengeWindow_reverts() public {
        uint256 challengedBlockNumber = 1;
        bytes memory challengedCommitment = computeCommitmentKeccak256("some hash");

        // Move to block after the challenge window
        vm.roll(challengedBlockNumber + dataAvailabilityChallenge.challengeWindow() + 1);

        // Challenge fails because the block number is after the challenge window
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeWindowNotOpen.selector));
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);
    }
}

/// @title DataAvailabilityChallenge_Resolve_Test
/// @notice Test contract for DataAvailabilityChallenge `resolve` function.
contract DataAvailabilityChallenge_Resolve_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `resolve` function succeeds.
    function test_resolve_succeeds(
        address challenger,
        address resolver,
        bytes memory preImage,
        uint256 challengedBlockNumber,
        uint256 resolverRefundPercentage,
        uint64 bondSize,
        uint128 txGasPrice
    )
        public
    {
        // Assume neither the challenger nor resolver is address(0) and that they're not the same
        // entity
        vm.assume(challenger != address(0));
        vm.assume(resolver != address(0));
        vm.assume(challenger != resolver);

        vm.prank(dataAvailabilityChallenge.owner());
        dataAvailabilityChallenge.setBondSize(bondSize);

        // Bound the resolver refund percentage to 100
        resolverRefundPercentage = bound(resolverRefundPercentage, 0, 100);

        // Set the gas price to a fuzzed value to test bond distribution logic
        vm.txGasPrice(txGasPrice);

        // Change the resolver refund percentage
        vm.prank(dataAvailabilityChallenge.owner());
        dataAvailabilityChallenge.setResolverRefundPercentage(resolverRefundPercentage);

        // Assume the block number is not close to the max uint256 value
        challengedBlockNumber = bound(
            challengedBlockNumber,
            0,
            type(uint256).max - dataAvailabilityChallenge.challengeWindow() - dataAvailabilityChallenge.resolveWindow()
                - 1
        );
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        vm.deal(challenger, bondSize);
        vm.prank(challenger);
        dataAvailabilityChallenge.challenge{ value: bondSize }(challengedBlockNumber, challengedCommitment);

        // Store the address(0) balance before resolving to assert the burned amount later
        uint256 zeroAddressBalanceBeforeResolve = address(0).balance;

        // Assert challenger balance after bond distribution
        uint256 resolutionCost =
            (dataAvailabilityChallenge.fixedResolutionCost()
                    + preImage.length
                    * dataAvailabilityChallenge.variableResolutionCost()
                    / dataAvailabilityChallenge.variableResolutionCostPrecision()) * block.basefee;
        uint256 challengerRefund = bondSize > resolutionCost ? bondSize - resolutionCost : 0;
        uint256 resolverRefund = resolutionCost * dataAvailabilityChallenge.resolverRefundPercentage() / 100;
        resolverRefund = resolverRefund > resolutionCost ? resolutionCost : resolverRefund;
        resolverRefund = resolverRefund > bondSize ? bondSize : resolverRefund;

        if (challengerRefund > 0) {
            vm.expectEmit(true, true, true, true);
            emit BalanceChanged(challenger, challengerRefund);
        }
        if (resolverRefund > 0) {
            vm.expectEmit(true, true, true, true);
            emit BalanceChanged(resolver, resolverRefund);
        }

        // Resolve the challenge
        vm.prank(resolver);
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);

        // Expect the challenge to be resolved
        Challenge memory challenge = dataAvailabilityChallenge.getChallenge(challengedBlockNumber, challengedCommitment);

        assertEq(challenge.challenger, challenger);
        assertEq(challenge.lockedBond, 0);
        assertEq(challenge.startBlock, block.number);
        assertEq(challenge.resolvedBlock, block.number);
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, challengedCommitment)),
            uint8(ChallengeStatus.Resolved)
        );
        address _challenger = challenger;
        address _resolver = resolver;
        assertEq(dataAvailabilityChallenge.balances(_challenger), challengerRefund, "challenger refund");
        assertEq(dataAvailabilityChallenge.balances(_resolver), resolverRefund, "resolver refund");

        // Assert burned amount after bond distribution
        uint256 burned = bondSize - challengerRefund - resolverRefund;
        assertEq(address(0).balance - zeroAddressBalanceBeforeResolve, burned, "burned bond");
    }

    /// @notice Test that the `resolve` function reverts if the input data is invalid.
    function test_resolve_invalidInputData_reverts(
        address challenger,
        address resolver,
        bytes memory preImage,
        bytes memory wrongPreImage,
        uint256 challengedBlockNumber,
        uint256 resolverRefundPercentage,
        uint128 txGasPrice
    )
        public
    {
        // Assume neither the challenger nor resolver is address(0) and that they're not the same
        // entity
        vm.assume(challenger != address(0));
        vm.assume(resolver != address(0));
        vm.assume(challenger != resolver);
        vm.assume(keccak256(preImage) != keccak256(wrongPreImage));

        // Bound the resolver refund percentage to 100
        resolverRefundPercentage = bound(resolverRefundPercentage, 0, 100);

        // Set the gas price to a fuzzed value to test bond distribution logic
        vm.txGasPrice(txGasPrice);

        // Change the resolver refund percentage
        vm.prank(dataAvailabilityChallenge.owner());
        dataAvailabilityChallenge.setResolverRefundPercentage(resolverRefundPercentage);

        // Assume the block number is not close to the max uint256 value
        challengedBlockNumber = bound(
            challengedBlockNumber,
            0,
            type(uint256).max - dataAvailabilityChallenge.challengeWindow() - dataAvailabilityChallenge.resolveWindow()
                - 1
        );
        bytes memory challengedCommitment = computeCommitmentKeccak256(wrongPreImage);

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        uint256 bondSize = dataAvailabilityChallenge.bondSize();
        vm.deal(challenger, bondSize);
        vm.prank(challenger);
        dataAvailabilityChallenge.challenge{ value: bondSize }(challengedBlockNumber, challengedCommitment);

        // Resolve the challenge
        vm.prank(resolver);
        vm.expectRevert(
            abi.encodeWithSelector(
                IDataAvailabilityChallenge.InvalidInputData.selector,
                computeCommitmentKeccak256(preImage),
                challengedCommitment
            )
        );
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);
    }

    /// @notice Test that the `resolve` function reverts if the challenge does not exist.
    function test_resolve_nonExistentChallenge_reverts() public {
        bytes memory preImage = "some preimage";
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Resolving a non-existent challenge fails
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotActive.selector));
        dataAvailabilityChallenge.resolve(challengedBlockNumber, computeCommitmentKeccak256(preImage), preImage);
    }

    /// @notice Test that the `resolve` function reverts if the challenge is already resolved.
    function test_resolve_resolved_reverts() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Resolve the challenge
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);

        // Resolving an already resolved challenge fails
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotActive.selector));
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);
    }

    /// @notice Test that the `resolve` function reverts if the challenge is expired.
    function test_resolve_expired_reverts() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Move to a block after the resolve window
        vm.roll(block.number + dataAvailabilityChallenge.resolveWindow() + 1);

        // Resolving an expired challenge fails
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotActive.selector));
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);
    }

    /// @notice Test that the `resolve` function reverts if the challenge is after the resolve
    ///         window.
    function test_resolve_afterResolveWindow_reverts() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Move to block after resolve window
        vm.roll(block.number + dataAvailabilityChallenge.resolveWindow() + 1);

        // Resolve the challenge
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotActive.selector));
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);
    }
}

/// @title DataAvailabilityChallenge_UnlockBond_Test
/// @notice Test contract for DataAvailabilityChallenge `unlockBond` function.
contract DataAvailabilityChallenge_UnlockBond_Test is DataAvailabilityChallenge_TestInit {
    /// @notice Test that the `unlockBond` function succeeds.
    function test_unlockBond_succeeds(bytes memory preImage, uint256 challengedBlockNumber) public {
        // Assume the block number is not close to the max uint256 value
        challengedBlockNumber = bound(
            challengedBlockNumber,
            0,
            type(uint256).max - dataAvailabilityChallenge.challengeWindow() - dataAvailabilityChallenge.resolveWindow()
                - 1
        );
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Move to a block after the resolve window
        vm.roll(block.number + dataAvailabilityChallenge.resolveWindow() + 1);

        uint256 balanceBeforeUnlock = dataAvailabilityChallenge.balances(address(this));

        // Unlock the bond associated with the challenge
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);

        // Expect the balance to be increased by the bond size
        uint256 balanceAfterUnlock = dataAvailabilityChallenge.balances(address(this));
        assertEq(balanceAfterUnlock, balanceBeforeUnlock + dataAvailabilityChallenge.bondSize());

        // Expect the bond to be unlocked
        Challenge memory challenge = dataAvailabilityChallenge.getChallenge(challengedBlockNumber, challengedCommitment);

        assertEq(challenge.challenger, address(this));
        assertEq(challenge.lockedBond, 0);
        assertEq(challenge.startBlock, challengedBlockNumber + 1);
        assertEq(challenge.resolvedBlock, 0);
        assertEq(
            uint8(dataAvailabilityChallenge.getChallengeStatus(challengedBlockNumber, challengedCommitment)),
            uint8(ChallengeStatus.Expired)
        );

        // Unlock the bond again, expect the balance to remain the same
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);
        assertEq(dataAvailabilityChallenge.balances(address(this)), balanceAfterUnlock);
    }

    /// @notice Test that the `unlockBond` function reverts if the challenge does not exist.
    function test_unlockBond_nonExistentChallenge_reverts() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Unlock a bond of a non-existent challenge fails
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotExpired.selector));
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);
    }

    /// @notice Test that the `unlockBond` function reverts if the challenge is resolved.
    function test_unlockBond_resolvedChallenge_reverts() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Resolve the challenge
        dataAvailabilityChallenge.resolve(challengedBlockNumber, challengedCommitment, preImage);

        // Attempting to unlock a bond of a resolved challenge fails
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotExpired.selector));
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);
    }

    /// @notice Test that the `unlockBond` function handles double-unlocking of expired challenges
    ///         correctly.
    function test_unlockBond_expiredChallengeTwice_fails() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        // Move to a block after the challenge window
        vm.roll(block.number + dataAvailabilityChallenge.resolveWindow() + 1);

        // Unlock the bond
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);

        uint256 balanceAfterUnlock = dataAvailabilityChallenge.balances(address(this));

        // Unlock the bond again doesn't change the balance
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);
        assertEq(dataAvailabilityChallenge.balances(address(this)), balanceAfterUnlock);
    }

    /// @notice Test that the `unlockBond` function reverts if the resolve window is not closed.
    function test_unlockBond_resolveWindowNotClosed_reverts() public {
        bytes memory preImage = "some preimage";
        bytes memory challengedCommitment = computeCommitmentKeccak256(preImage);
        uint256 challengedBlockNumber = 1;

        // Move to block after challenged block
        vm.roll(challengedBlockNumber + 1);

        // Challenge the hash
        dataAvailabilityChallenge.deposit{ value: dataAvailabilityChallenge.bondSize() }();
        dataAvailabilityChallenge.challenge(challengedBlockNumber, challengedCommitment);

        vm.roll(block.number + dataAvailabilityChallenge.resolveWindow() - 1);

        // Expiring the challenge before the resolve window closes fails
        vm.expectRevert(abi.encodeWithSelector(IDataAvailabilityChallenge.ChallengeNotExpired.selector));
        dataAvailabilityChallenge.unlockBond(challengedBlockNumber, challengedCommitment);
    }
}
