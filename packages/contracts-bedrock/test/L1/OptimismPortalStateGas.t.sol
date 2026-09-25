// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { OptimismPortal2_TestInit } from "test/L1/OptimismPortal2.t.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

import { Claim } from "src/dispute/lib/Types.sol";

/// @title OptimismPortal2_StateGas_Test
/// @notice End-to-end demonstration that under Amsterdam gas pricing a withdrawal can finalize
///         while the message it carried is left neither executed nor replayable.
///
///         The component probes size the two constants in isolation. This one drives the real path
///         a user's withdrawal takes — prove, resolve, wait, finalize — and checks the property
///         that actually matters to them.
///
///         The layering under test: `OptimismPortal2` is a one-shot execution primitive whose
///         `finalizedWithdrawals` flag is permanent by design, and `CrossDomainMessenger` supplies
///         replayability on top by recording every relay in `successfulMessages` or
///         `failedMessages`. On this leg the messenger runs as the Portal's *child*, so when its
///         bookkeeping write runs out of gas, its whole frame unwinds while the Portal's flag —
///         written in the Portal's own frame before the call — survives. `callWithMinGas` is raw
///         assembly and does not propagate the failure, so the Portal sees only a `false` return it
///         was designed to ignore, and completes normally. The transaction succeeds.
///
///         Run under both `--evm-version cancun` and `--evm-version amsterdam`. Under cancun the
///         messenger's reserve covers its status write and the invariant holds; under Amsterdam the
///         write costs ~123,694 against a 40,000 reserve and the invariant breaks.
contract OptimismPortal2_StateGas_Test is OptimismPortal2_TestInit {
    /// @notice Gas budget the message asks for its own target. Small, so the relay is a normal one
    ///         rather than something contrived.
    uint32 internal constant MESSAGE_MIN_GAS = 50_000;

    /// @notice Top-level gas for the finalize transaction.
    ///
    ///         Chosen to sit in the window where the bug is reachable, which is bounded on both
    ///         sides. EIP-150 leaves each caller 1/64 of its gas, so:
    ///
    ///         - too little and the Portal cannot finish either once its child consumes everything
    ///           forwarded. Its frame reverts, `finalizedWithdrawals` unwinds with it, and the
    ///           withdrawal stays retryable — the safe outcome.
    ///         - too much and the messenger's 1/64 floor alone exceeds what its status write needs,
    ///           so the write succeeds. That takes roughly 7.9M gas reaching the messenger.
    ///
    ///         At 2,500,000 the Portal retains ~39,000, enough to finish, while the messenger is
    ///         below the 2.56M at which its 1/64 floor would overtake `RELAY_RESERVED_GAS`, so it
    ///         retains exactly the 40,000 reserve. Supplying *more* gas here makes bricking more
    ///         likely, not less, until the upper bound is crossed.
    uint256 internal constant FINALIZE_GAS = 2_500_000;

    /// @notice A finalized withdrawal must leave its message either executed or replayable.
    ///
    ///         This is the user-facing guarantee behind the whole design: the Portal delivers a
    ///         withdrawal exactly once, and the messenger owns retries from there. If the messenger
    ///         records nothing, both doors shut at once — `finalizeWithdrawalTransaction` reverts
    ///         with `OptimismPortal_AlreadyFinalized`, and `relayMessage` reverts on
    ///         `require(failedMessages[versionedHash])`. Nothing on chain records who the funds
    ///         belonged to.
    function test_finalizeWithdrawalTransaction_leavesMessageReplayable_succeeds() external {
        skipIfForkTest("OptimismPortal2_StateGas_Test: needs locally generated withdrawal proofs");

        // A target that consumes everything forwarded to it. This is what makes
        // `RELAY_RESERVED_GAS` the binding constraint: gas the target does not spend is returned to
        // the messenger, so only a target that spends it all leaves the messenger on its reserve.
        // The repo's own baseGas fuzz test uses this same mock for the same reason.
        address target = address(new GasBurner(type(uint32).max));

        // The cross-domain message the messenger is asked to relay.
        uint256 messageNonce = Encoding.encodeVersionedNonce(0, 1);
        bytes memory message = hex"";
        bytes32 versionedHash = Hashing.hashCrossDomainMessageV1(
            messageNonce, Predeploys.L2_CROSS_DOMAIN_MESSENGER, address(target), 0, MESSAGE_MIN_GAS, message
        );

        // The withdrawal carrying it. `sender` must be the L2 messenger so the L1 messenger's
        // `_isOtherMessenger()` check passes and this is treated as a fresh delivery.
        Types.WithdrawalTransaction memory wtx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            target: address(l1CrossDomainMessenger),
            value: 0,
            gasLimit: l1CrossDomainMessenger.baseGas(message, MESSAGE_MIN_GAS),
            data: abi.encodeWithSelector(
                l1CrossDomainMessenger.relayMessage.selector,
                messageNonce,
                Predeploys.L2_CROSS_DOMAIN_MESSENGER,
                address(target),
                uint256(0),
                uint256(MESSAGE_MIN_GAS),
                message
            )
        });

        bytes32 withdrawalHash = Hashing.hashWithdrawal(wtx);

        // Prove it, against a root claim mocked to match this withdrawal.
        (bytes32 stateRoot, bytes32 storageRoot,,, bytes[] memory withdrawalProof) =
            ffi.getProveWithdrawalTransactionInputs(wtx);
        Types.OutputRootProof memory outputRootProof = Types.OutputRootProof({
            version: bytes32(0),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(0)
        });

        if (DisputeGames.isSuperGame(game.gameType())) {
            vm.mockCall(
                address(game),
                abi.encodeCall(game.rootClaimByChainId, (systemConfig.l2ChainId())),
                abi.encode(Claim.wrap(Hashing.hashOutputRootProof(outputRootProof)))
            );
        } else {
            vm.mockCall(
                address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(Hashing.hashOutputRootProof(outputRootProof))
            );
        }

        optimismPortal2.proveWithdrawalTransaction({
            _tx: wtx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: outputRootProof,
            _withdrawalProof: withdrawalProof
        });

        game.resolveClaim(0, 0);
        game.resolve();
        vm.warp(block.timestamp + optimismPortal2.proofMaturityDelaySeconds() + 1);

        optimismPortal2.finalizeWithdrawalTransaction{ gas: FINALIZE_GAS }(wtx);

        bool relayed = l1CrossDomainMessenger.successfulMessages(versionedHash);
        bool recordedFailed = l1CrossDomainMessenger.failedMessages(versionedHash);

        // The Portal finalizes regardless of what happened downstream. True under both EVMs.
        assertTrue(
            optimismPortal2.finalizedWithdrawals(withdrawalHash),
            "OptimismPortal2_StateGas_Test: withdrawal should be finalized"
        );

        assertFalse(relayed, "OptimismPortal2_StateGas_Test: gas-burning target should never relay successfully");

        // Under Amsterdam this fails with the withdrawal finalized, the target never
        // run, and no replay record written: the message is bricked and the funds unrecoverable.
        assertTrue(
            relayed || recordedFailed,
            "withdrawal finalized but message was neither relayed nor recorded failed: bricked, no replay possible"
        );
    }
}
