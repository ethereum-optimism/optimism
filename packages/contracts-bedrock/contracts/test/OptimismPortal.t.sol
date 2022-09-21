// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Portal_Initializer, CommonTest, NextImpl } from "./CommonTest.t.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { Types } from "../libraries/Types.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Proxy } from "../universal/Proxy.sol";

contract OptimismPortal_Test is Portal_Initializer {
    function test_OptimismPortalConstructor() external {
        assertEq(op.FINALIZATION_PERIOD_SECONDS(), 7 days);
        assertEq(address(op.L2_ORACLE()), address(oracle));
        assertEq(op.l2Sender(), 0x000000000000000000000000000000000000dEaD);
    }

    function test_OptimismPortalReceiveEth() external {
        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(alice, alice, 100, 100, 100_000, false, hex"");

        // give alice money and send as an eoa
        vm.deal(alice, 2**64);
        vm.prank(alice, alice);
        (bool s, ) = address(op).call{ value: 100 }(hex"");

        assert(s);
        assertEq(address(op).balance, 100);
    }

    // Test: depositTransaction fails when contract creation has a non-zero destination address
    function test_OptimismPortalContractCreationReverts() external {
        // contract creation must have a target of address(0)
        vm.expectRevert("OptimismPortal: must send to address(0) when creating a contract");
        op.depositTransaction(address(1), 1, 0, true, hex"");
    }

    // Test: depositTransaction should emit the correct log when an EOA deposits a tx with 0 value
    function test_depositTransaction_NoValueEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));
        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            address(this),
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    // Test: depositTransaction should emit the correct log when a contract deposits a tx with 0 value
    function test_depositTransaction_NoValueContract() external {
        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    // Test: depositTransaction should emit the correct log when an EOA deposits a contract creation with 0 value
    function test_depositTransaction_createWithZeroValueForEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            address(this),
            ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );

        op.depositTransaction(ZERO_ADDRESS, ZERO_VALUE, NON_ZERO_GASLIMIT, true, NON_ZERO_DATA);
    }

    // Test: depositTransaction should emit the correct log when a contract deposits a contract creation with 0 value
    function test_depositTransaction_createWithZeroValueForContract() external {
        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            ZERO_ADDRESS,
            ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );

        op.depositTransaction(ZERO_ADDRESS, ZERO_VALUE, NON_ZERO_GASLIMIT, true, NON_ZERO_DATA);
    }

    // Test: depositTransaction should increase its eth balance when an EOA deposits a transaction with ETH
    function test_depositTransaction_withEthValueFromEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            address(this),
            NON_ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
        assertEq(address(op).balance, NON_ZERO_VALUE);
    }

    // Test: depositTransaction should increase its eth balance when a contract deposits a transaction with ETH
    function test_depositTransaction_withEthValueFromContract() external {
        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            NON_ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA
        );
    }

    // Test: depositTransaction should increase its eth balance when an EOA deposits a contract creation with ETH
    function test_depositTransaction_withEthValueAndEOAContractCreation() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            address(this),
            ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            hex""
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            hex""
        );
        assertEq(address(op).balance, NON_ZERO_VALUE);
    }

    // Test: depositTransaction should increase its eth balance when a contract deposits a contract creation with ETH
    function test_depositTransaction_withEthValueAndContractContractCreation() external {
        vm.expectEmit(true, true, false, true);
        emitTransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(this)),
            ZERO_ADDRESS,
            NON_ZERO_VALUE,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );

        op.depositTransaction{ value: NON_ZERO_VALUE }(
            ZERO_ADDRESS,
            ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA
        );
        assertEq(address(op).balance, NON_ZERO_VALUE);
    }

    function test_simple_isBlockFinalized() external {
        vm.mockCall(
            address(op.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(bytes32(uint256(1)), startingBlockNumber))
        );

        // warp to the finalization period
        vm.warp(startingBlockNumber + op.FINALIZATION_PERIOD_SECONDS());
        assertEq(op.isBlockFinalized(startingBlockNumber), false);
        // warp past the finalization period
        vm.warp(startingBlockNumber + op.FINALIZATION_PERIOD_SECONDS() + 1);
        assertEq(op.isBlockFinalized(startingBlockNumber), true);
    }

    function test_isBlockFinalized() external {
        uint256 checkpoint = oracle.nextBlockNumber();
        vm.roll(checkpoint);
        vm.warp(oracle.computeL2Timestamp(checkpoint) + 1);
        vm.prank(oracle.proposer());
        oracle.proposeL2Output(keccak256(abi.encode(2)), checkpoint, 0, 0);

        // warp to the final second of the finalization period
        uint256 finalizationHorizon = block.timestamp + op.FINALIZATION_PERIOD_SECONDS();
        vm.warp(finalizationHorizon);
        // The checkpointed block should not be finalized until 1 second from now.
        assertEq(op.isBlockFinalized(checkpoint), false);
        // Nor should a block after it
        vm.expectRevert("L2OutputOracle: No output found for that block number.");
        assertEq(op.isBlockFinalized(checkpoint + 1), false);
        // Nor a block before it, even though the finalization period has passed, there is
        // not yet a checkpoint block on top of it for which that is true.
        assertEq(op.isBlockFinalized(checkpoint - 1), false);

        // warp past the finalization period
        vm.warp(finalizationHorizon + 1);
        // It should now be finalized.
        assertEq(op.isBlockFinalized(checkpoint), true);
        // So should the block before it.
        assertEq(op.isBlockFinalized(checkpoint - 1), true);
        // But not the block after it.
        vm.expectRevert("L2OutputOracle: No output found for that block number.");
        assertEq(op.isBlockFinalized(checkpoint + 1), false);
    }
}

contract OptimismPortal_FinalizeWithdrawal_Test is Portal_Initializer {
    // Reusable default values for a test withdrawal
    Types.WithdrawalTransaction _defaultTx;

    uint256 _proposedBlockNumber;
    bytes32 _stateRoot;
    bytes32 _storageRoot;
    bytes32 _outputRoot;
    bytes32 _withdrawalHash;
    bytes _withdrawalProof;
    Types.OutputRootProof internal _outputRootProof;

    event WithdrawalFinalized(bytes32 indexed, bool success);

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
        (_stateRoot, _storageRoot, _outputRoot, _withdrawalHash, _withdrawalProof) = ffi
            .getFinalizeWithdrawalTransactionInputs(_defaultTx);

        // Setup a dummy output root proof for reuse.
        _outputRootProof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: _stateRoot,
            messagePasserStorageRoot: _storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });
        _proposedBlockNumber = oracle.nextBlockNumber();
    }

    // Get the system into a nice ready-to-use state.
    function setUp() public override {
        // Configure the oracle to return the output root we've prepared.
        vm.warp(oracle.computeL2Timestamp(_proposedBlockNumber) + 1);
        vm.prank(oracle.proposer());
        oracle.proposeL2Output(_outputRoot, _proposedBlockNumber, 0, 0);

        // Warp beyond the finalization period for the block we've proposed.
        vm.warp(
            oracle.getL2Output(_proposedBlockNumber).timestamp +
                op.FINALIZATION_PERIOD_SECONDS() +
                1
        );
        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(op), 0xFFFFFFFF);
    }

    // Test: finalizeWithdrawalTransaction succeeds and emits the WithdrawalFinalized event.
    function test_finalizeWithdrawalTransaction_succeeds() external {
        uint256 bobBalanceBefore = address(bob).balance;
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, true);
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
        assert(address(bob).balance == bobBalanceBefore + 100);
    }

    // Test: finalizeWithdrawalTransaction fails because the target reverts,
    // and emits the WithdrawalFinalized event with success=false.
    function test_finalizeWithdrawalTransaction_targetFails() external {
        uint256 bobBalanceBefore = address(bob).balance;
        vm.etch(bob, hex"fe"); // Contract with just the invalid opcode.

        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, false);
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
        assert(address(bob).balance == bobBalanceBefore);
    }

    // Test: finalizeWithdrawalTransaction cannot finalize a withdrawal with itself (the OptimismPortal) as the target.
    function test_finalizeWithdrawalTransaction_revertsOnSelfCall() external {
        _defaultTx.target = address(op);
        vm.expectRevert("OptimismPortal: you cannot send messages to the portal contract");
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
    }

    // Test: finalizeWithdrawalTransaction reverts if the outputRootProof does not match the output root
    function test_finalizeWithdrawalTransaction_revertsOnInvalidOutputRootProof() external {
        // Modify the version to invalidate the withdrawal proof.
        _outputRootProof.version = bytes32(uint256(1));
        vm.expectRevert("OptimismPortal: invalid output root proof");
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
    }

    // Test: finalizeWithdrawalTransaction reverts if the finalization period has not yet passed.
    function test_finalizeWithdrawalTransaction_revertsOnRecentWithdrawal() external {
        // Setup the Oracle to return an output with a recent timestamp
        uint256 recentTimestamp = block.timestamp - 1000;
        vm.mockCall(
            address(op.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(bytes32(uint256(1)), recentTimestamp)
        );

        vm.expectRevert("OptimismPortal: proposal is not yet finalized");
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
    }

    // Test: finalizeWithdrawalTransaction reverts if the withdrawal has already been finalized.
    function test_finalizeWithdrawalTransaction_revertsOnReplay() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, true);
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
        vm.expectRevert("OptimismPortal: withdrawal has already been finalized");
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
    }

    // Test: finalizeWithdrawalTransaction reverts if insufficient gas is supplied.
    function test_finalizeWithdrawalTransaction_revertsOnInsufficientGas() external {
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
        (
            bytes32 stateRoot,
            bytes32 storageRoot,
            ,
            ,
            bytes memory withdrawalProof
        ) = ffi.getFinalizeWithdrawalTransactionInputs(insufficientGasTx);
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: bytes32(0),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(0)
        });
        vm.mockCall(
            address(op.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Hashing.hashOutputRootProof(outputRootProof), _proposedBlockNumber)
        );
        vm.expectRevert("OptimismPortal: insufficient gas to finalize withdrawal");
        op.finalizeWithdrawalTransaction{ gas: gasLimit }(
            insufficientGasTx,
            _proposedBlockNumber,
            outputRootProof,
            withdrawalProof
        );
    }

    // Test: finalizeWithdrawalTransaction reverts if the proof is invalid due to non-existence of
    // the withdrawal.
    function test_finalizeWithdrawalTransaction_revertsOninvalidWithdrawalProof() external {
        // modify the default test values to invalidate the proof.
        _defaultTx.data = hex"abcd";
        vm.expectRevert("OptimismPortal: invalid withdrawal inclusion proof");
        op.finalizeWithdrawalTransaction(
            _defaultTx,
            _proposedBlockNumber,
            _outputRootProof,
            _withdrawalProof
        );
    }

    // Utility function used in the subsequent test. This is necessary to assert that the
    // reentrant call will revert.
    function callPortalAndExpectRevert() external payable {
        vm.expectRevert("OptimismPortal: can only trigger one withdrawal per transaction");
        // Arguments here don't matter, as the require check is the first thing that happens.
        op.finalizeWithdrawalTransaction(_defaultTx, 0, _outputRootProof, hex"");
        // Assert that the withdrawal was not finalized.
        assertFalse(op.finalizedWithdrawals(Hashing.hashWithdrawal(_defaultTx)));
    }

    // Test: finalizeWithdrawalTransaction reverts if a sub-call attempts to finalize another
    // withdrawal.
    function test_finalizeWithdrawalTransaction_revertsOnReentrancy() external {
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
            bytes memory withdrawalProof
        ) = ffi.getFinalizeWithdrawalTransactionInputs(_testTx);
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: bytes32(0),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(0)
        });

        // Setup the Oracle to return the outputRoot we want as well as a finalized timestamp.
        uint256 finalizedTimestamp = block.timestamp - op.FINALIZATION_PERIOD_SECONDS() - 1;
        vm.mockCall(
            address(op.L2_ORACLE()),
            abi.encodeWithSelector(L2OutputOracle.getL2Output.selector),
            abi.encode(Types.OutputProposal(outputRoot, finalizedTimestamp))
        );

        // Assert that this contract is called with the expected data (i.e. the function signature of
        // callPortalAndExpectRevert).
        vm.expectCall(address(this), _testTx.data);
        vm.expectEmit(true, true, true, true);
        // Assert that the withdrawal should be finalized, and that the sub-call passes (because the
        // assertions in callPortalAndExpectRevert pass).
        emit WithdrawalFinalized(withdrawalHash, true);
        op.finalizeWithdrawalTransaction(
            _testTx,
            _proposedBlockNumber,
            outputRootProof,
            withdrawalProof
        );
        // Ensure that bob's balance was not changed by the reentrant call.
        assert(address(bob).balance == bobBalanceBefore);
    }

    function test_finalizeWithdrawalTransaction_differential(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external {
        // Cannot call the optimism portal
        vm.assume(_target != address(op));
        // Total ETH supply is currently about 120M ETH.
        vm.assume(_value < 200_000_000 ether);
        vm.assume(_gasLimit < 50_000_000);
        uint256 _nonce = messagePasser.nonce();
        Types.WithdrawalTransaction memory _tx = Types.WithdrawalTransaction({
            nonce: _nonce,
            sender: _sender,
            target: _target,
            value: _value,
            gasLimit: _gasLimit,
            data: _data
        });
        (
            bytes32 stateRoot,
            bytes32 storageRoot,
            bytes32 outputRoot,
            bytes32 withdrawalHash,
            bytes memory withdrawalProof
        ) = ffi.getFinalizeWithdrawalTransactionInputs(_tx);

        Types.OutputRootProof memory proof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });

        // Ensure the values returned from ffi are correct
        assertEq(outputRoot, Hashing.hashOutputRootProof(proof));
        assertEq(withdrawalHash, Hashing.hashWithdrawal(_tx));

        // Mock the call to the oracle
        vm.mockCall(
            address(oracle),
            abi.encodeWithSelector(oracle.getL2Output.selector),
            abi.encode(outputRoot, 0)
        );

        // Start the withdrawal, it must be initiated by the _sender and the
        // correct value must be passed along
        vm.deal(_tx.sender, _tx.value);
        vm.prank(_tx.sender);
        messagePasser.initiateWithdrawal{ value: _tx.value }(_tx.target, _tx.gasLimit, _tx.data);
        // Ensure that the sentMessages is correct
        assertEq(messagePasser.sentMessages(withdrawalHash), true);

        vm.warp(op.FINALIZATION_PERIOD_SECONDS() + 1);
        op.finalizeWithdrawalTransaction(
            _tx,
            100, // l2BlockNumber
            proof,
            withdrawalProof
        );
    }
}

contract OptimismPortalUpgradeable_Test is Portal_Initializer {
    Proxy internal proxy;
    uint64 initialBlockNum;

    function setUp() public override {
        super.setUp();
        initialBlockNum = uint64(block.number);
        proxy = Proxy(payable(address(op)));
    }

    function test_initValuesOnProxy() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = OptimismPortal(
            payable(address(proxy))
        ).params();
        assertEq(prevBaseFee, opImpl.INITIAL_BASE_FEE());
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_cannotInitProxy() external {
        vm.expectRevert("Initializable: contract is already initialized");
        OptimismPortal(payable(proxy)).initialize();
    }

    function test_cannotInitImpl() external {
        vm.expectRevert("Initializable: contract is already initialized");
        OptimismPortal(opImpl).initialize();
    }

    function test_upgrading() external {
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(op), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();
        vm.startPrank(multisig);
        proxy.upgradeToAndCall(
            address(nextImpl),
            abi.encodeWithSelector(NextImpl.initialize.selector)
        );
        assertEq(proxy.implementation(), address(nextImpl));

        // Verify that the NextImpl contract initialized its values according as expected
        bytes32 slot21After = vm.load(address(op), bytes32(uint256(21)));
        bytes32 slot21Expected = NextImpl(address(op)).slot21Init();
        assertEq(slot21Expected, slot21After);
    }
}
