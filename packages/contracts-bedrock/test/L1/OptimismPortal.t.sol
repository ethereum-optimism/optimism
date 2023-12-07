// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { stdError } from "forge-std/Test.sol";

import { CommonTest } from "test/setup/CommonTest.sol";
import { NextImpl } from "test/mocks/NextImpl.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Target contract dependencies
import { Proxy } from "src/universal/Proxy.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";

contract OptimismPortal_Test is CommonTest {
    address depositor;

    function setUp() public override {
        super.setUp();
        depositor = makeAddr("depositor");
    }

    /// @dev Tests that the constructor sets the correct values.
    function test_constructor_succeeds() external {
        address guardian = deploy.cfg().superchainConfigGuardian();
        assertEq(address(optimismPortal.L2_ORACLE()), address(l2OutputOracle));
        assertEq(address(optimismPortal.l2Oracle()), address(l2OutputOracle));
        assertEq(optimismPortal.GUARDIAN(), guardian);
        assertEq(optimismPortal.guardian(), guardian);
        assertEq(optimismPortal.l2Sender(), 0x000000000000000000000000000000000000dEaD);
        assertEq(optimismPortal.paused(), false);
    }

    /// @dev Tests that `pause` successfully pauses
    ///      when called by the GUARDIAN.
    function test_pause_succeeds() external {
        address guardian = optimismPortal.GUARDIAN();

        assertEq(optimismPortal.paused(), false);

        vm.expectEmit(address(superchainConfig));
        emit Paused("identifier");

        vm.prank(guardian);
        superchainConfig.pause("identifier");

        assertEq(optimismPortal.paused(), true);
    }

    /// @dev Tests that `pause` reverts when called by a non-GUARDIAN.
    function test_pause_onlyGuardian_reverts() external {
        assertEq(optimismPortal.paused(), false);

        assertTrue(optimismPortal.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        superchainConfig.pause("identifier");

        assertEq(optimismPortal.paused(), false);
    }

    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the GUARDIAN.
    function test_unpause_succeeds() external {
        address guardian = optimismPortal.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(optimismPortal.paused(), true);

        vm.expectEmit(address(superchainConfig));
        emit Unpaused();
        vm.prank(guardian);
        superchainConfig.unpause();

        assertEq(optimismPortal.paused(), false);
    }

    /// @dev Tests that `unpause` reverts when called by a non-GUARDIAN.
    function test_unpause_onlyGuardian_reverts() external {
        address guardian = optimismPortal.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(optimismPortal.paused(), true);

        assertTrue(optimismPortal.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        superchainConfig.unpause();

        assertEq(optimismPortal.paused(), true);
    }

    /// @dev Tests that `receive` successdully deposits ETH.
    function testFuzz_receive_succeeds(uint256 _value) external {
        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: alice,
            _to: alice,
            _value: _value,
            _mint: _value,
            _gasLimit: 100_000,
            _isCreation: false,
            _data: hex""
        });

        // give alice money and send as an eoa
        vm.deal(alice, _value);
        vm.prank(alice, alice);
        (bool s,) = address(optimismPortal).call{ value: _value }(hex"");

        assertTrue(s);
        assertEq(address(optimismPortal).balance, _value);
    }

    /// @dev Tests that `depositTransaction` reverts when the destination address is non-zero
    ///      for a contract creation deposit.
    function test_depositTransaction_contractCreation_reverts() external {
        // contract creation must have a target of address(0)
        vm.expectRevert("OptimismPortal: must send to address(0) when creating a contract");
        optimismPortal.depositTransaction(address(1), 1, 0, true, hex"");
    }

    /// @dev Tests that `depositTransaction` reverts when the data is too large.
    ///      This places an upper bound on unsafe blocks sent over p2p.
    function test_depositTransaction_largeData_reverts() external {
        uint256 size = 120_001;
        uint64 gasLimit = optimismPortal.minimumGasLimit(uint64(size));
        vm.expectRevert("OptimismPortal: data too large");
        optimismPortal.depositTransaction({
            _to: address(0),
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: new bytes(size)
        });
    }

    /// @dev Tests that `depositTransaction` reverts when the gas limit is too small.
    function test_depositTransaction_smallGasLimit_reverts() external {
        vm.expectRevert("OptimismPortal: gas limit too small");
        optimismPortal.depositTransaction({ _to: address(1), _value: 0, _gasLimit: 0, _isCreation: false, _data: hex"" });
    }

    /// @dev Tests that `depositTransaction` succeeds for small,
    ///      but sufficient, gas limits.
    function testFuzz_depositTransaction_smallGasLimit_succeeds(bytes memory _data, bool _shouldFail) external {
        uint64 gasLimit = optimismPortal.minimumGasLimit(uint64(_data.length));
        if (_shouldFail) {
            gasLimit = uint64(bound(gasLimit, 0, gasLimit - 1));
            vm.expectRevert("OptimismPortal: gas limit too small");
        }

        optimismPortal.depositTransaction({
            _to: address(0x40),
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: _data
        });
    }

    /// @dev Tests that `minimumGasLimit` succeeds for small calldata sizes.
    ///      The gas limit should be 21k for 0 calldata and increase linearly
    ///      for larger calldata sizes.
    function test_minimumGasLimit_succeeds() external {
        assertEq(optimismPortal.minimumGasLimit(0), 21_000);
        assertTrue(optimismPortal.minimumGasLimit(2) > optimismPortal.minimumGasLimit(1));
        assertTrue(optimismPortal.minimumGasLimit(3) > optimismPortal.minimumGasLimit(2));
    }

    /// @dev Tests that `depositTransaction` succeeds for an EOA.
    function testFuzz_depositTransaction_eoa_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data
    )
        external
    {
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        // EOA emulation
        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: depositor,
            _to: _to,
            _value: _value,
            _mint: _mint,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });

        vm.deal(depositor, _mint);
        vm.prank(depositor, depositor);
        optimismPortal.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal).balance, _mint);
    }

    /// @dev Tests that `depositTransaction` succeeds for a contract.
    function testFuzz_depositTransaction_contract_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data
    )
        external
    {
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        vm.expectEmit(address(optimismPortal));
        emitTransactionDeposited({
            _from: AddressAliasHelper.applyL1ToL2Alias(address(this)),
            _to: _to,
            _value: _value,
            _mint: _mint,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });

        vm.deal(address(this), _mint);
        vm.prank(address(this));
        optimismPortal.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal).balance, _mint);
    }

    /// @dev Tests that `isOutputFinalized` succeeds for an EOA depositing a tx with ETH and data.
    function test_simple_isOutputFinalized_succeeds() external {
        uint256 startingBlockNumber = deploy.cfg().l2OutputOracleStartingBlockNumber();

        uint256 ts = block.timestamp;
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(bytes32(uint256(1)), uint128(ts), uint128(startingBlockNumber)))
        );

        // warp to the finalization period
        vm.warp(ts + l2OutputOracle.FINALIZATION_PERIOD_SECONDS());
        assertEq(optimismPortal.isOutputFinalized(0), false);

        // warp past the finalization period
        vm.warp(ts + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        assertEq(optimismPortal.isOutputFinalized(0), true);
    }

    /// @dev Tests `isOutputFinalized` for a finalized output.
    function test_isOutputFinalized_succeeds() external {
        uint256 checkpoint = l2OutputOracle.nextBlockNumber();
        uint256 nextOutputIndex = l2OutputOracle.nextOutputIndex();
        vm.roll(checkpoint);
        vm.warp(l2OutputOracle.computeL2Timestamp(checkpoint) + 1);
        vm.prank(l2OutputOracle.PROPOSER());
        l2OutputOracle.proposeL2Output(keccak256(abi.encode(2)), checkpoint, 0, 0);

        // warp to the final second of the finalization period
        uint256 finalizationHorizon = block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS();
        vm.warp(finalizationHorizon);
        // The checkpointed block should not be finalized until 1 second from now.
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex), false);
        // Nor should a block after it
        vm.expectRevert(stdError.indexOOBError);
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex + 1), false);
        // warp past the finalization period
        vm.warp(finalizationHorizon + 1);
        // It should now be finalized.
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex), true);
        // But not the block after it.
        vm.expectRevert(stdError.indexOOBError);
        assertEq(optimismPortal.isOutputFinalized(nextOutputIndex + 1), false);
    }
}

contract OptimismPortal_FinalizeWithdrawal_Test is CommonTest {
    // Reusable default values for a test withdrawal
    Types.WithdrawalTransaction _defaultTx;

    uint256 _proposedOutputIndex;
    uint256 _proposedBlockNumber;
    bytes32 _stateRoot;
    bytes32 _storageRoot;
    bytes32 _outputRoot;
    bytes32 _withdrawalHash;
    bytes[] _withdrawalProof;
    Types.OutputRootProof internal _outputRootProof;

    // Use a constructor to set the storage vars above, so as to minimize the number of ffi calls.
    constructor() {
        super.setUp();
        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 100,
            gasLimit: 100_000,
            data: hex""
        });
        // Get withdrawal proof data we can use for testing.
        (_stateRoot, _storageRoot, _outputRoot, _withdrawalHash, _withdrawalProof) =
            ffi.getProveWithdrawalTransactionInputs(_defaultTx);

        // Setup a dummy output root proof for reuse.
        _outputRootProof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: _stateRoot,
            messagePasserStorageRoot: _storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });
        _proposedBlockNumber = l2OutputOracle.nextBlockNumber();
        _proposedOutputIndex = l2OutputOracle.nextOutputIndex();
    }

    /// @dev Setup the system for a ready-to-use state.
    function setUp() public override {
        // Configure the oracle to return the output root we've prepared.
        vm.warp(l2OutputOracle.computeL2Timestamp(_proposedBlockNumber) + 1);
        vm.prank(l2OutputOracle.PROPOSER());
        l2OutputOracle.proposeL2Output(_outputRoot, _proposedBlockNumber, 0, 0);

        // Warp beyond the finalization period for the block we've proposed.
        vm.warp(
            l2OutputOracle.getL2Output(_proposedOutputIndex).timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS()
                + 1
        );
        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(optimismPortal), 0xFFFFFFFF);
    }

    /// @dev Asserts that the reentrant call will revert.
    function callPortalAndExpectRevert() external payable {
        vm.expectRevert("OptimismPortal: can only trigger one withdrawal per transaction");
        // Arguments here don't matter, as the require check is the first thing that happens.
        // We assume that this has already been proven.
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);
        // Assert that the withdrawal was not finalized.
        assertFalse(optimismPortal.finalizedWithdrawals(Hashing.hashWithdrawal(_defaultTx)));
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when paused.
    function test_proveWithdrawalTransaction_paused_reverts() external {
        vm.prank(optimismPortal.GUARDIAN());
        superchainConfig.pause("identifier");

        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _l2OutputIndex: _proposedOutputIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the target is the portal contract.
    function test_proveWithdrawalTransaction_onSelfCall_reverts() external {
        _defaultTx.target = address(optimismPortal);
        vm.expectRevert("OptimismPortal: you cannot send messages to the portal contract");
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when
    ///      the outputRootProof does not match the output root
    function test_proveWithdrawalTransaction_onInvalidOutputRootProof_reverts() external {
        // Modify the version to invalidate the withdrawal proof.
        _outputRootProof.version = bytes32(uint256(1));
        vm.expectRevert("OptimismPortal: invalid output root proof");
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the withdrawal is missing.
    function test_proveWithdrawalTransaction_onInvalidWithdrawalProof_reverts() external {
        // modify the default test values to invalidate the proof.
        _defaultTx.data = hex"abcd";
        vm.expectRevert("MerkleTrie: path remainder must share all nibbles with key");
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the withdrawal has already
    ///      been proven.
    function test_proveWithdrawalTransaction_replayProve_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        vm.expectRevert("OptimismPortal: withdrawal hash has already been proven");
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @dev Tests that `proveWithdrawalTransaction` succeeds when the withdrawal has already
    ///      been proven and the output root has changed and the l2BlockNumber stays the same.
    function test_proveWithdrawalTransaction_replayProveChangedOutputRoot_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Compute the storage slot of the outputRoot corresponding to the `withdrawalHash`
        // inside of the `provenWithdrawal`s mapping.
        bytes32 slot;
        assembly {
            mstore(0x00, sload(_withdrawalHash.slot))
            mstore(0x20, 52) // 52 is the slot of the `provenWithdrawals` mapping in the OptimismPortal
            slot := keccak256(0x00, 0x40)
        }

        // Store a different output root within the `provenWithdrawals` mapping without
        // touching the l2BlockNumber or timestamp.
        vm.store(address(optimismPortal), slot, bytes32(0));

        // Warp ahead 1 second
        vm.warp(block.timestamp + 1);

        // Even though we have already proven this withdrawalHash, we should be allowed to re-submit
        // our proof with a changed outputRoot
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Ensure that the withdrawal was updated within the mapping
        (, uint128 timestamp,) = optimismPortal.provenWithdrawals(_withdrawalHash);
        assertEq(timestamp, block.timestamp);
    }

    /// @dev Tests that `proveWithdrawalTransaction` succeeds when the withdrawal has already
    ///      been proven and the output root, output index, and l2BlockNumber have changed.
    function test_proveWithdrawalTransaction_replayProveChangedOutputRootAndOutputIndex_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Compute the storage slot of the outputRoot corresponding to the `withdrawalHash`
        // inside of the `provenWithdrawal`s mapping.
        bytes32 slot;
        assembly {
            mstore(0x00, sload(_withdrawalHash.slot))
            mstore(0x20, 52) // 52 is the slot of the `provenWithdrawals` mapping in OptimismPortal
            slot := keccak256(0x00, 0x40)
        }

        // Store a dummy output root within the `provenWithdrawals` mapping without touching the
        // l2BlockNumber or timestamp.
        vm.store(address(optimismPortal), slot, bytes32(0));

        // Fetch the output proposal at `_proposedOutputIndex` from the L2OutputOracle
        Types.OutputProposal memory proposal = optimismPortal.L2_ORACLE().getL2Output(_proposedOutputIndex);

        // Propose the same output root again, creating the same output at a different index + l2BlockNumber.
        vm.startPrank(optimismPortal.L2_ORACLE().PROPOSER());
        optimismPortal.L2_ORACLE().proposeL2Output(
            proposal.outputRoot, optimismPortal.L2_ORACLE().nextBlockNumber(), blockhash(block.number), block.number
        );
        vm.stopPrank();

        // Warp ahead 1 second
        vm.warp(block.timestamp + 1);

        // Even though we have already proven this withdrawalHash, we should be allowed to re-submit
        // our proof with a changed outputRoot + a different output index
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(
            _defaultTx, _proposedOutputIndex + 1, _outputRootProof, _withdrawalProof
        );

        // Ensure that the withdrawal was updated within the mapping
        (, uint128 timestamp,) = optimismPortal.provenWithdrawals(_withdrawalHash);
        assertEq(timestamp, block.timestamp);
    }

    /// @dev Tests that `proveWithdrawalTransaction` succeeds.
    function test_proveWithdrawalTransaction_validWithdrawalProof_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` succeeds.
    function test_finalizeWithdrawalTransaction_provenWithdrawalHash_succeeds() external {
        uint256 bobBalanceBefore = address(bob).balance;

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        vm.expectEmit(true, true, false, true);
        emit WithdrawalFinalized(_withdrawalHash, true);
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore + 100);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the contract is paused.
    function test_finalizeWithdrawalTransaction_paused_reverts() external {
        vm.prank(optimismPortal.GUARDIAN());
        superchainConfig.pause("identifier");

        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal has not been
    function test_finalizeWithdrawalTransaction_ifWithdrawalNotProven_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        vm.expectRevert("OptimismPortal: withdrawal has not been proven yet");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal has not been
    ///      proven long enough ago.
    function test_finalizeWithdrawalTransaction_ifWithdrawalProofNotOldEnough_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Mock a call where the resulting output root is anything but the original output root. In
        // this case we just use bytes32(uint256(1)).
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(bytes32(uint256(1)), _proposedBlockNumber)
        );

        vm.expectRevert("OptimismPortal: proven withdrawal finalization period has not elapsed");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the provenWithdrawal's timestamp
    ///      is less than the L2 output oracle's starting timestamp.
    function test_finalizeWithdrawalTransaction_timestampLessThanL2OracleStart_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        // Prove our withdrawal
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Warp to after the finalization period
        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);

        // Mock a startingTimestamp change on the L2 Oracle
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSignature("startingTimestamp()"),
            abi.encode(block.timestamp + 1)
        );

        // Attempt to finalize the withdrawal
        vm.expectRevert("OptimismPortal: withdrawal timestamp less than L2 Oracle starting timestamp");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        // Ensure that bob's balance has remained the same
        assertEq(bobBalanceBefore, address(bob).balance);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the output root proven is not the
    ///      same as the output root at the time of finalization.
    function test_finalizeWithdrawalTransaction_ifOutputRootChanges_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        // Prove our withdrawal
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Warp to after the finalization period
        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);

        // Mock an outputRoot change on the output proposal before attempting
        // to finalize the withdrawal.
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(
                Types.OutputProposal(bytes32(uint256(0)), uint128(block.timestamp), uint128(_proposedBlockNumber))
            )
        );

        // Attempt to finalize the withdrawal
        vm.expectRevert("OptimismPortal: output root proven is not the same as current output root");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        // Ensure that bob's balance has remained the same
        assertEq(bobBalanceBefore, address(bob).balance);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the output proposal's timestamp
    ///      has not passed the finalization period.
    function test_finalizeWithdrawalTransaction_ifOutputTimestampIsNotFinalized_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        // Prove our withdrawal
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        // Warp to after the finalization period
        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);

        // Mock a timestamp change on the output proposal that has not passed the
        // finalization period.
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(_outputRoot, uint128(block.timestamp + 1), uint128(_proposedBlockNumber)))
        );

        // Attempt to finalize the withdrawal
        vm.expectRevert("OptimismPortal: output proposal finalization period has not elapsed");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        // Ensure that bob's balance has remained the same
        assertEq(bobBalanceBefore, address(bob).balance);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the target reverts.
    function test_finalizeWithdrawalTransaction_targetFails_fails() external {
        uint256 bobBalanceBefore = address(bob).balance;
        vm.etch(bob, hex"fe"); // Contract with just the invalid opcode.

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, false);
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the finalization period
    ///      has not yet passed.
    function test_finalizeWithdrawalTransaction_onRecentWithdrawal_reverts() external {
        // Setup the Oracle to return an output with a recent timestamp
        uint256 recentTimestamp = block.timestamp - 1;
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(_outputRoot, uint128(recentTimestamp), uint128(_proposedBlockNumber)))
        );

        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        vm.expectRevert("OptimismPortal: proven withdrawal finalization period has not elapsed");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal has already been
    ///      finalized.
    function test_finalizeWithdrawalTransaction_onReplay_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        optimismPortal.proveWithdrawalTransaction(_defaultTx, _proposedOutputIndex, _outputRootProof, _withdrawalProof);

        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, true);
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);

        vm.expectRevert("OptimismPortal: withdrawal has already been finalized");
        optimismPortal.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal transaction
    ///      does not have enough gas to execute.
    function test_finalizeWithdrawalTransaction_onInsufficientGas_reverts() external {
        // This number was identified through trial and error.
        uint256 gasLimit = 150_000;
        Types.WithdrawalTransaction memory insufficientGasTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 100,
            gasLimit: gasLimit,
            data: hex""
        });

        // Get updated proof inputs.
        (bytes32 stateRoot, bytes32 storageRoot,,, bytes[] memory withdrawalProof) =
            ffi.getProveWithdrawalTransactionInputs(insufficientGasTx);
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: bytes32(0),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(0)
        });

        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(
                Types.OutputProposal(
                    Hashing.hashOutputRootProof(outputRootProof),
                    uint128(block.timestamp),
                    uint128(_proposedBlockNumber)
                )
            )
        );

        optimismPortal.proveWithdrawalTransaction(
            insufficientGasTx, _proposedOutputIndex, outputRootProof, withdrawalProof
        );

        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        vm.expectRevert("SafeCall: Not enough gas");
        optimismPortal.finalizeWithdrawalTransaction{ gas: gasLimit }(insufficientGasTx);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if a sub-call attempts to finalize
    ///      another withdrawal.
    function test_finalizeWithdrawalTransaction_onReentrancy_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        // Copy and modify the default test values to attempt a reentrant call by first calling to
        // this contract's callPortalAndExpectRevert() function above.
        Types.WithdrawalTransaction memory _testTx = _defaultTx;
        _testTx.target = address(this);
        _testTx.data = abi.encodeWithSelector(this.callPortalAndExpectRevert.selector);

        // Get modified proof inputs.
        (
            bytes32 stateRoot,
            bytes32 storageRoot,
            bytes32 outputRoot,
            bytes32 withdrawalHash,
            bytes[] memory withdrawalProof
        ) = ffi.getProveWithdrawalTransactionInputs(_testTx);
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: bytes32(0),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(0)
        });

        // Setup the Oracle to return the outputRoot we want as well as a finalized timestamp.
        uint256 finalizedTimestamp = block.timestamp - l2OutputOracle.FINALIZATION_PERIOD_SECONDS() - 1;
        vm.mockCall(
            address(optimismPortal.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(outputRoot, uint128(finalizedTimestamp), uint128(_proposedBlockNumber)))
        );

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(withdrawalHash, alice, address(this));
        optimismPortal.proveWithdrawalTransaction(_testTx, _proposedBlockNumber, outputRootProof, withdrawalProof);

        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);
        vm.expectCall(address(this), _testTx.data);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(withdrawalHash, true);
        optimismPortal.finalizeWithdrawalTransaction(_testTx);

        // Ensure that bob's balance was not changed by the reentrant call.
        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` succeeds.
    function testDiff_finalizeWithdrawalTransaction_succeeds(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        vm.assume(
            _target != address(optimismPortal) // Cannot call the optimism portal or a contract
                && _target.code.length == 0 // No accounts with code
                && _target != CONSOLE // The console has no code but behaves like a contract
                && uint160(_target) > 9 // No precompiles (or zero address)
        );

        // Total ETH supply is currently about 120M ETH.
        uint256 value = bound(_value, 0, 200_000_000 ether);
        vm.deal(address(optimismPortal), value);

        uint256 gasLimit = bound(_gasLimit, 0, 50_000_000);
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        // Get a withdrawal transaction and mock proof from the differential testing script.
        Types.WithdrawalTransaction memory _tx = Types.WithdrawalTransaction({
            nonce: nonce,
            sender: _sender,
            target: _target,
            value: value,
            gasLimit: gasLimit,
            data: _data
        });
        (
            bytes32 stateRoot,
            bytes32 storageRoot,
            bytes32 outputRoot,
            bytes32 withdrawalHash,
            bytes[] memory withdrawalProof
        ) = ffi.getProveWithdrawalTransactionInputs(_tx);

        // Create the output root proof
        Types.OutputRootProof memory proof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });

        // Ensure the values returned from ffi are correct
        assertEq(outputRoot, Hashing.hashOutputRootProof(proof));
        assertEq(withdrawalHash, Hashing.hashWithdrawal(_tx));

        // Setup the Oracle to return the outputRoot
        vm.mockCall(
            address(l2OutputOracle),
            abi.encodeWithSelector(l2OutputOracle.getL2Output.selector),
            abi.encode(outputRoot, block.timestamp, 100)
        );

        // Prove the withdrawal transaction
        optimismPortal.proveWithdrawalTransaction(
            _tx,
            100, // l2BlockNumber
            proof,
            withdrawalProof
        );
        (bytes32 _root,,) = optimismPortal.provenWithdrawals(withdrawalHash);
        assertTrue(_root != bytes32(0));

        // Warp past the finalization period
        vm.warp(block.timestamp + l2OutputOracle.FINALIZATION_PERIOD_SECONDS() + 1);

        // Finalize the withdrawal transaction
        vm.expectCallMinGas(_tx.target, _tx.value, uint64(_tx.gasLimit), _tx.data);
        optimismPortal.finalizeWithdrawalTransaction(_tx);
        assertTrue(optimismPortal.finalizedWithdrawals(withdrawalHash));
    }
}

contract OptimismPortalUpgradeable_Test is CommonTest {
    /// @dev Tests that the proxy is initialized correctly.
    function test_params_initValuesOnProxy_succeeds() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = optimismPortal.params();
        ResourceMetering.ResourceConfig memory rcfg = systemConfig.resourceConfig();

        assertEq(prevBaseFee, rcfg.minimumBaseFee);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, block.number);
    }

    /// @dev Tests that the proxy cannot be initialized twice.
    function test_initialize_cannotInitProxy_reverts() external {
        L2OutputOracle l2OutputOracle = L2OutputOracle(deploy.mustGetAddress("L2OutputOracle"));
        SystemConfig systemConfig = SystemConfig(deploy.mustGetAddress("SystemConfig"));
        SuperchainConfig superchainConfig = SuperchainConfig(deploy.mustGetAddress("SuperchainConfig"));
        vm.expectRevert("Initializable: contract is already initialized");
        optimismPortal.initialize(l2OutputOracle, systemConfig, superchainConfig);
    }

    /// @dev Tests that the implementation cannot be initialized twice.
    function test_initialize_cannotInitImpl_reverts() external {
        address opImpl = deploy.mustGetAddress("OptimismPortal");
        L2OutputOracle l2OutputOracle = L2OutputOracle(deploy.mustGetAddress("L2OutputOracle"));
        SystemConfig systemConfig = SystemConfig(deploy.mustGetAddress("SystemConfig"));
        SuperchainConfig superchainConfig = SuperchainConfig(deploy.mustGetAddress("SuperchainConfig"));
        vm.expectRevert("Initializable: contract is already initialized");
        OptimismPortal(payable(opImpl)).initialize(l2OutputOracle, systemConfig, superchainConfig);
    }

    /// @dev Tests that the proxy can be upgraded.
    function test_upgradeToAndCall_upgrading_succeeds() external {
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(optimismPortal), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();

        vm.startPrank(EIP1967Helper.getAdmin(address(optimismPortal)));
        // The value passed to the initialize must be larger than the last value
        // that initialize was called with.
        Proxy(payable(address(optimismPortal))).upgradeToAndCall(
            address(nextImpl),
            abi.encodeWithSelector(NextImpl.initialize.selector, 3) // TODO: check that this change makes sense
        );
        assertEq(Proxy(payable(address(optimismPortal))).implementation(), address(nextImpl));

        // Verify that the NextImpl contract initialized its values according as expected
        bytes32 slot21After = vm.load(address(optimismPortal), bytes32(uint256(21)));
        bytes32 slot21Expected = NextImpl(address(optimismPortal)).slot21Init();
        assertEq(slot21Expected, slot21After);
    }
}

/// @title OptimismPortalResourceFuzz_Test
/// @dev Test various values of the resource metering config to ensure that deposits cannot be
///      broken by changing the config.
contract OptimismPortalResourceFuzz_Test is CommonTest {
    /// @dev The max gas limit observed throughout this test. Setting this too high can cause
    ///      the test to take too long to run.
    uint256 constant MAX_GAS_LIMIT = 30_000_000;

    /// @dev Test that various values of the resource metering config will not break deposits.
    function testFuzz_systemConfigDeposit_succeeds(
        uint32 _maxResourceLimit,
        uint8 _elasticityMultiplier,
        uint8 _baseFeeMaxChangeDenominator,
        uint32 _minimumBaseFee,
        uint32 _systemTxMaxGas,
        uint128 _maximumBaseFee,
        uint64 _gasLimit,
        uint64 _prevBoughtGas,
        uint128 _prevBaseFee,
        uint8 _blockDiff
    )
        external
    {
        // Get the set system gas limit
        uint64 gasLimit = systemConfig.gasLimit();
        // Bound resource config
        _maxResourceLimit = uint32(bound(_maxResourceLimit, 21000, MAX_GAS_LIMIT / 8));
        _gasLimit = uint64(bound(_gasLimit, 21000, _maxResourceLimit));
        _prevBaseFee = uint128(bound(_prevBaseFee, 0, 3 gwei));
        // Prevent values that would cause reverts
        vm.assume(gasLimit >= _gasLimit);
        vm.assume(_minimumBaseFee < _maximumBaseFee);
        vm.assume(_baseFeeMaxChangeDenominator > 1);
        vm.assume(uint256(_maxResourceLimit) + uint256(_systemTxMaxGas) <= gasLimit);
        vm.assume(_elasticityMultiplier > 0);
        vm.assume(((_maxResourceLimit / _elasticityMultiplier) * _elasticityMultiplier) == _maxResourceLimit);
        _prevBoughtGas = uint64(bound(_prevBoughtGas, 0, _maxResourceLimit - _gasLimit));
        _blockDiff = uint8(bound(_blockDiff, 0, 3));
        // Pick a pseudorandom block number
        vm.roll(uint256(keccak256(abi.encode(_blockDiff))) % uint256(type(uint16).max) + uint256(_blockDiff));

        // Create a resource config to mock the call to the system config with
        ResourceMetering.ResourceConfig memory rcfg = ResourceMetering.ResourceConfig({
            maxResourceLimit: _maxResourceLimit,
            elasticityMultiplier: _elasticityMultiplier,
            baseFeeMaxChangeDenominator: _baseFeeMaxChangeDenominator,
            minimumBaseFee: _minimumBaseFee,
            systemTxMaxGas: _systemTxMaxGas,
            maximumBaseFee: _maximumBaseFee
        });
        vm.mockCall(
            address(systemConfig), abi.encodeWithSelector(systemConfig.resourceConfig.selector), abi.encode(rcfg)
        );

        // Set the resource params
        uint256 _prevBlockNum = block.number - _blockDiff;
        vm.store(
            address(optimismPortal),
            bytes32(uint256(1)),
            bytes32((_prevBlockNum << 192) | (uint256(_prevBoughtGas) << 128) | _prevBaseFee)
        );
        // Ensure that the storage setting is correct
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = optimismPortal.params();
        assertEq(prevBaseFee, _prevBaseFee);
        assertEq(prevBoughtGas, _prevBoughtGas);
        assertEq(prevBlockNum, _prevBlockNum);

        // Do a deposit, should not revert
        optimismPortal.depositTransaction{ gas: MAX_GAS_LIMIT }({
            _to: address(0x20),
            _value: 0x40,
            _gasLimit: _gasLimit,
            _isCreation: false,
            _data: hex""
        });
    }
}
