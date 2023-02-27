pragma solidity 0.8.15;

import { Portal_Initializer } from "../CommonTest.t.sol";
import { Vm } from "forge-std/Vm.sol";
import { Proxy } from "../../universal/Proxy.sol";
import { OptimismPortal } from "../../L1/OptimismPortal.sol";
import { Types } from "../../libraries/Types.sol";

contract OptimismPortal_Invariant_Harness is Portal_Initializer {
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

    function setUp() public virtual override {
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
            .getProveWithdrawalTransactionInputs(_defaultTx);

        // Setup a dummy output root proof for reuse.
        _outputRootProof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: _stateRoot,
            messagePasserStorageRoot: _storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });
        _proposedBlockNumber = oracle.nextBlockNumber();
        _proposedOutputIndex = oracle.nextOutputIndex();

        // Configure the oracle to return the output root we've prepared.
        vm.warp(oracle.computeL2Timestamp(_proposedBlockNumber) + 1);
        vm.prank(oracle.PROPOSER());
        oracle.proposeL2Output(_outputRoot, _proposedBlockNumber, 0, 0);

        // Warp beyond the finalization period for the block we've proposed.
        vm.warp(
            oracle.getL2Output(_proposedOutputIndex).timestamp +
                op.FINALIZATION_PERIOD_SECONDS() +
                1
        );
        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(op), 0xFFFFFFFF);
    }
}

contract OptimismPortal_CannotTimeTravel is OptimismPortal_Invariant_Harness {
    function setUp() public override {
        super.setUp();

        // Prove the withdrawal transaction
        op.proveWithdrawalTransaction(
            _defaultTx,
            _proposedOutputIndex,
            _outputRootProof,
            _withdrawalProof
        );

        // Set the target contract to the portal proxy
        targetContract(address(op));
        // Exclude the proxy multisig from the senders so that the proxy cannot be upgraded
        excludeSender(address(multisig));
    }

    /**
     * @custom:invariant `finalizeWithdrawalTransaction` should revert if the finalization
     * period has not elapsed.
     *
     * A withdrawal that has been proven should not be able to be finalized until after
     * the finalization period has elapsed.
     */
    function invariant_cannotFinalizeBeforePeriodHasPassed() external {
        vm.expectRevert("OptimismPortal: proven withdrawal finalization period has not elapsed");
        op.finalizeWithdrawalTransaction(_defaultTx);
    }
}

contract OptimismPortal_CannotFinalizeTwice is OptimismPortal_Invariant_Harness {
    function setUp() public override {
        super.setUp();

        // Prove the withdrawal transaction
        op.proveWithdrawalTransaction(
            _defaultTx,
            _proposedOutputIndex,
            _outputRootProof,
            _withdrawalProof
        );

        // Warp past the finalization period.
        vm.warp(block.timestamp + op.FINALIZATION_PERIOD_SECONDS() + 1);

        // Finalize the withdrawal transaction.
        op.finalizeWithdrawalTransaction(_defaultTx);

        // Set the target contract to the portal proxy
        targetContract(address(op));
        // Exclude the proxy multisig from the senders so that the proxy cannot be upgraded
        excludeSender(address(multisig));
    }

    /**
     * @custom:invariant `finalizeWithdrawalTransaction` should revert if the withdrawal
     * has already been finalized.
     *
     * Ensures that there is no chain of calls that can be made that allows a withdrawal
     * to be finalized twice.
     */
    function invariant_cannotFinalizeTwice() external {
        vm.expectRevert("OptimismPortal: withdrawal has already been finalized");
        op.finalizeWithdrawalTransaction(_defaultTx);
    }
}

contract OptimismPortal_FinalizeFuzzGasActor {
    OptimismPortal internal immutable OP;
    Vm internal immutable VM;
    Types.WithdrawalTransaction internal _defaultTx;

    constructor(
        OptimismPortal _op,
        Vm _vm,
        Types.WithdrawalTransaction memory _tx
    ) {
        OP = _op;
        VM = _vm;
        _defaultTx = _tx;
    }

    function finalize(uint64 gas) external {
        // Ensure that the portal has at least the transaction value as a balance.
        // For some weird reason, the invariant fuzzer sometimes zeroes out the portal
        // proxy's balance if a call to it reverts, even though it should have 0xFFFFFFFF ETH
        // prior to finalizing alice's withdrawal to bob.
        VM.deal(address(OP), _defaultTx.value);

        (bool success, bytes memory returndata) = address(OP).call{ gas: gas }(
            abi.encodeWithSelector(OP.finalizeWithdrawalTransaction.selector, _defaultTx)
        );

        // With the set up of this test, the only way the above call should fail is if:
        // 1. The call ran out of gas
        // 2. The portal reverted with the "insufficient gas to finalize withdrawal" error
        // 3. The withdrawal has already been finalized.
        if (!success) {
            bytes32 returnDataHash = keccak256(returndata);
            require(
                returnDataHash ==
                    keccak256(
                        abi.encodeWithSignature(
                            "Error(string)",
                            "OptimismPortal: insufficient gas to finalize withdrawal"
                        )
                    ) ||
                    returnDataHash ==
                    keccak256(abi.encodeWithSignature("Error(string)", "EvmError: OutOfGas")) ||
                    returnDataHash ==
                    keccak256(
                        abi.encodeWithSignature(
                            "Error(string)",
                            "OptimismPortal: withdrawal has already been finalized"
                        )
                    )
            );
        }
    }
}

contract OptimismPortal_CanAlwaysFinalizeAfterWindow is OptimismPortal_Invariant_Harness {
    uint256 internal bobBalanceBefore;

    function setUp() public override {
        super.setUp();

        // Create a finalize actor
        OptimismPortal_FinalizeFuzzGasActor finalizeActor = new OptimismPortal_FinalizeFuzzGasActor(
            op,
            vm,
            _defaultTx
        );

        // Prove the withdrawal transaction
        op.proveWithdrawalTransaction(
            _defaultTx,
            _proposedOutputIndex,
            _outputRootProof,
            _withdrawalProof
        );

        // Warp past the finalization period.
        vm.warp(block.timestamp + op.FINALIZATION_PERIOD_SECONDS() + 1);

        // Target the portal proxy
        targetContract(address(op));

        // Target the finalize actor
        targetContract(address(finalizeActor));

        // Exclude the proxy multisig from the senders so that the proxy cannot be upgraded
        excludeSender(address(multisig));

        // Exclude bob as a sender so that his balance remains static.
        excludeSender(address(bob));

        // Exclude the guardian as a sender so that they may not pause the portal.
        excludeSender(address(guardian));

        // Keep track of bob's balance before any calls were made.
        bobBalanceBefore = address(bob).balance;
    }

    /**
     * @custom:invariant A withdrawal should **always** be able to be finalized
     * `FINALIZATION_PERIOD_SECONDS` after it was successfully proven.
     *
     * This invariant asserts that there is no chain of calls that can be made that
     * will prevent a withdrawal from being finalized exactly `FINALIZATION_PERIOD_SECONDS`
     * after it was successfully proven if enough gas is provided.
     */
    function invariant_canAlwaysFinalize() external {
        // If the withdrawal was marked as finalized, we need to ensure that the external
        // call made within `finalizeWithdrawalTransaction` was successful. We do so by
        // ensuring that bob's balance has increased by the amount of the withdrawal.
        if (op.finalizedWithdrawals(_withdrawalHash)) {
            assertEq(address(bob).balance, bobBalanceBefore + _defaultTx.value);
        }
    }
}
