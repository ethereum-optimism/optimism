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
import { Constants } from "src/libraries/Constants.sol";

// Target contract dependencies
import { Proxy } from "src/universal/Proxy.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { L1ToL2MessagePasser } from "src/L1/L1ToL2MessagePasser.sol";

import { FaultDisputeGame, IDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import "src/libraries/DisputeTypes.sol";

contract L1ToL2MessagePasser_Test is CommonTest {
    address depositor;

    function setUp() public override {
        super.enableFaultProofs();
        super.setUp();

        depositor = makeAddr("depositor");
    }

    /// @dev Tests that the constructor sets the correct values.
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function test_constructor_succeeds() external virtual {
        L1ToL2MessagePasser opImpl = L1ToL2MessagePasser(payable(deploy.mustGetAddress("L1ToL2MessagePasser")));
        assertEq(address(opImpl.disputeGameFactory()), address(0));
        assertEq(address(opImpl.SYSTEM_CONFIG()), address(0));
        assertEq(address(opImpl.systemConfig()), address(0));
        assertEq(address(opImpl.superchainConfig()), address(0));
        assertEq(opImpl.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(opImpl.respectedGameType().raw(), deploy.cfg().respectedGameType());
    }

    /// @dev Tests that the initializer sets the correct values.
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function test_initialize_succeeds() external virtual {
        address guardian = deploy.cfg().superchainConfigGuardian();
        assertEq(address(l1ToL2MessagePasser.disputeGameFactory()), address(disputeGameFactory));
        assertEq(address(l1ToL2MessagePasser.SYSTEM_CONFIG()), address(systemConfig));
        assertEq(address(l1ToL2MessagePasser.systemConfig()), address(systemConfig));
        assertEq(l1ToL2MessagePasser.GUARDIAN(), guardian);
        assertEq(l1ToL2MessagePasser.guardian(), guardian);
        assertEq(address(l1ToL2MessagePasser.superchainConfig()), address(superchainConfig));
        assertEq(l1ToL2MessagePasser.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(l1ToL2MessagePasser.paused(), false);
        assertEq(l1ToL2MessagePasser.respectedGameType().raw(), deploy.cfg().respectedGameType());
    }

    /// @dev Tests that `pause` successfully pauses
    ///      when called by the GUARDIAN.
    function test_pause_succeeds() external {
        address guardian = l1ToL2MessagePasser.GUARDIAN();

        assertEq(l1ToL2MessagePasser.paused(), false);

        vm.expectEmit(address(superchainConfig));
        emit Paused("identifier");

        vm.prank(guardian);
        superchainConfig.pause("identifier");

        assertEq(l1ToL2MessagePasser.paused(), true);
    }

    /// @dev Tests that `pause` reverts when called by a non-GUARDIAN.
    function test_pause_onlyGuardian_reverts() external {
        assertEq(l1ToL2MessagePasser.paused(), false);

        assertTrue(l1ToL2MessagePasser.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        superchainConfig.pause("identifier");

        assertEq(l1ToL2MessagePasser.paused(), false);
    }

    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the GUARDIAN.
    function test_unpause_succeeds() external {
        address guardian = l1ToL2MessagePasser.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(l1ToL2MessagePasser.paused(), true);

        vm.expectEmit(address(superchainConfig));
        emit Unpaused();
        vm.prank(guardian);
        superchainConfig.unpause();

        assertEq(l1ToL2MessagePasser.paused(), false);
    }

    /// @dev Tests that `unpause` reverts when called by a non-GUARDIAN.
    function test_unpause_onlyGuardian_reverts() external {
        address guardian = l1ToL2MessagePasser.GUARDIAN();

        vm.prank(guardian);
        superchainConfig.pause("identifier");
        assertEq(l1ToL2MessagePasser.paused(), true);

        assertTrue(l1ToL2MessagePasser.GUARDIAN() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        superchainConfig.unpause();

        assertEq(l1ToL2MessagePasser.paused(), true);
    }

    /// @dev Tests that `receive` successdully deposits ETH.
    function testFuzz_receive_succeeds(uint256 _value) external {
        vm.expectEmit(address(l1ToL2MessagePasser));
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
        (bool s,) = address(l1ToL2MessagePasser).call{ value: _value }(hex"");

        assertTrue(s);
        assertEq(address(l1ToL2MessagePasser).balance, _value);
    }

    /// @dev Tests that `depositTransaction` reverts when the destination address is non-zero
    ///      for a contract creation deposit.
    function test_depositTransaction_contractCreation_reverts() external {
        // contract creation must have a target of address(0)
        vm.expectRevert("L1ToL2MessagePasser: must send to address(0) when creating a contract");
        l1ToL2MessagePasser.depositTransaction(address(1), 1, 0, true, hex"");
    }

    /// @dev Tests that `depositTransaction` reverts when the data is too large.
    ///      This places an upper bound on unsafe blocks sent over p2p.
    function test_depositTransaction_largeData_reverts() external {
        uint256 size = 120_001;
        uint64 gasLimit = l1ToL2MessagePasser.minimumGasLimit(uint64(size));
        vm.expectRevert("L1ToL2MessagePasser: data too large");
        l1ToL2MessagePasser.depositTransaction({
            _to: address(0),
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: new bytes(size)
        });
    }

    /// @dev Tests that `depositTransaction` reverts when the gas limit is too small.
    function test_depositTransaction_smallGasLimit_reverts() external {
        vm.expectRevert("L1ToL2MessagePasser: gas limit too small");
        l1ToL2MessagePasser.depositTransaction({
            _to: address(1),
            _value: 0,
            _gasLimit: 0,
            _isCreation: false,
            _data: hex""
        });
    }

    /// @dev Tests that `depositTransaction` succeeds for small,
    ///      but sufficient, gas limits.
    function testFuzz_depositTransaction_smallGasLimit_succeeds(bytes memory _data, bool _shouldFail) external {
        uint64 gasLimit = l1ToL2MessagePasser.minimumGasLimit(uint64(_data.length));
        if (_shouldFail) {
            gasLimit = uint64(bound(gasLimit, 0, gasLimit - 1));
            vm.expectRevert("L1ToL2MessagePasser: gas limit too small");
        }

        l1ToL2MessagePasser.depositTransaction({
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
        assertEq(l1ToL2MessagePasser.minimumGasLimit(0), 21_000);
        assertTrue(l1ToL2MessagePasser.minimumGasLimit(2) > l1ToL2MessagePasser.minimumGasLimit(1));
        assertTrue(l1ToL2MessagePasser.minimumGasLimit(3) > l1ToL2MessagePasser.minimumGasLimit(2));
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
                l1ToL2MessagePasser.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        // EOA emulation
        vm.expectEmit(address(l1ToL2MessagePasser));
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
        l1ToL2MessagePasser.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(l1ToL2MessagePasser).balance, _mint);
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
                l1ToL2MessagePasser.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        vm.expectEmit(address(l1ToL2MessagePasser));
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
        l1ToL2MessagePasser.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(l1ToL2MessagePasser).balance, _mint);
    }
}

contract L1ToL2MessagePasser_FinalizeWithdrawal_Test is CommonTest {
    // Reusable default values for a test withdrawal
    Types.WithdrawalTransaction _defaultTx;

    FaultDisputeGame game;
    uint256 _proposedGameIndex;
    uint256 _proposedBlockNumber;
    bytes32 _stateRoot;
    bytes32 _storageRoot;
    bytes32 _outputRoot;
    bytes32 _withdrawalHash;
    bytes[] _withdrawalProof;
    Types.OutputRootProof internal _outputRootProof;

    // Use a constructor to set the storage vars above, so as to minimize the number of ffi calls.
    constructor() {
        super.enableFaultProofs();
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
    }

    /// @dev Setup the system for a ready-to-use state.
    function setUp() public override {
        _proposedBlockNumber = 0xFF;
        game = FaultDisputeGame(
            address(
                disputeGameFactory.create(
                    l1ToL2MessagePasser.respectedGameType(), Claim.wrap(_outputRoot), abi.encode(_proposedBlockNumber)
                )
            )
        );
        _proposedGameIndex = disputeGameFactory.gameCount() - 1;

        // Warp beyond the chess clocks and finalize the game.
        vm.warp(block.timestamp + game.gameDuration().raw() / 2 + 1 seconds);

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(l1ToL2MessagePasser), 0xFFFFFFFF);
    }

    /// @dev Asserts that the reentrant call will revert.
    function callPortalAndExpectRevert() external payable {
        vm.expectRevert("L1ToL2MessagePasser: can only trigger one withdrawal per transaction");
        // Arguments here don't matter, as the require check is the first thing that happens.
        // We assume that this has already been proven.
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
        // Assert that the withdrawal was not finalized.
        assertFalse(l1ToL2MessagePasser.finalizedWithdrawals(Hashing.hashWithdrawal(_defaultTx)));
    }

    /// @dev Tests that `deleteProvenWithdrawal` reverts when called by a non-SAURON.
    function testFuzz_deleteProvenWithdrawal_onlySauron_reverts(address _act, bytes32 _wdHash) external {
        vm.assume(_act != address(l1ToL2MessagePasser.sauron()));

        vm.expectRevert("L1ToL2MessagePasser: only sauron can delete proven withdrawals");
        l1ToL2MessagePasser.deleteProvenWithdrawal(_wdHash);
    }

    /// @dev Tests that the SAURON role can delete any proven withdrawal.
    function test_deleteProvenWithdrawal_sauron_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Ensure the withdrawal has been proven.
        (, uint64 timestamp) = l1ToL2MessagePasser.provenWithdrawals(_withdrawalHash);
        assertEq(timestamp, block.timestamp);

        // Delete the proven withdrawal.
        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.deleteProvenWithdrawal(_withdrawalHash);

        // Ensure the withdrawal has been deleted
        (, timestamp) = l1ToL2MessagePasser.provenWithdrawals(_withdrawalHash);
        assertEq(timestamp, 0);
    }

    /// @dev Tests that `deleteProvenWithdrawal` reverts when called by a non-SAURON.
    function testFuzz_blacklist_onlySauron_reverts(address _act) external {
        vm.assume(_act != address(l1ToL2MessagePasser.sauron()));

        vm.expectRevert("L1ToL2MessagePasser: only sauron can blacklist dispute games");
        l1ToL2MessagePasser.blacklistDisputeGame(IDisputeGame(address(0xdead)));
    }

    /// @dev Tests that the SAURON role can blacklist any dispute game.
    function testFuzz_blacklist_sauron_succeeds(address _addr) external {
        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.blacklistDisputeGame(IDisputeGame(_addr));

        assertTrue(l1ToL2MessagePasser.disputeGameBlacklist(IDisputeGame(_addr)));
    }

    /// @dev Tests that `setRespectedGameType` reverts when called by a non-SAURON.
    function testFuzz_setRespectedGameType_onlySauron_reverts(address _act, GameType _ty) external {
        vm.assume(_act != address(l1ToL2MessagePasser.sauron()));

        vm.prank(_act);
        vm.expectRevert("L1ToL2MessagePasser: only sauron can set the respected game type");
        l1ToL2MessagePasser.setRespectedGameType(_ty);
    }

    /// @dev Tests that the SAURON role can set the respected game type to anything they want.
    function testFuzz_setRespectedGameType_sauron_succeeds(GameType _ty) external {
        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.setRespectedGameType(_ty);

        assertEq(l1ToL2MessagePasser.respectedGameType().raw(), _ty.raw());
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when paused.
    function test_proveWithdrawalTransaction_paused_reverts() external {
        vm.prank(l1ToL2MessagePasser.GUARDIAN());
        superchainConfig.pause("identifier");

        vm.expectRevert("L1ToL2MessagePasser: paused");
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the target is the portal contract.
    function test_proveWithdrawalTransaction_onSelfCall_reverts() external {
        _defaultTx.target = address(l1ToL2MessagePasser);
        vm.expectRevert("L1ToL2MessagePasser: you cannot send messages to the portal contract");
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the outputRootProof does not match the output root
    function test_proveWithdrawalTransaction_onInvalidOutputRootProof_reverts() external {
        // Modify the version to invalidate the withdrawal proof.
        _outputRootProof.version = bytes32(uint256(1));
        vm.expectRevert("L1ToL2MessagePasser: invalid output root proof");
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the withdrawal is missing.
    function test_proveWithdrawalTransaction_onInvalidWithdrawalProof_reverts() external {
        // modify the default test values to invalidate the proof.
        _defaultTx.data = hex"abcd";
        vm.expectRevert("MerkleTrie: path remainder must share all nibbles with key");
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts when the withdrawal has already
    ///      been proven.
    function test_proveWithdrawalTransaction_replayProve_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        vm.expectRevert("L1ToL2MessagePasser: withdrawal hash has already been proven, and dispute game is not invalid");
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` reverts if the dispute game being proven against is not of the
    ///      respected game type.
    function test_proveWithdrawalTransaction_badGameType_reverts() external {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(disputeGameFactory.gameAtIndex, (_proposedGameIndex)),
            abi.encode(GameType.wrap(0xFF), Timestamp.wrap(uint64(block.timestamp)), IDisputeGame(address(game)))
        );

        vm.expectRevert("L1ToL2MessagePasser: invalid game type");
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` can be re-executed if the dispute game proven against has been
    ///      blacklisted.
    function test_proveWithdrawalTransaction_replayProveBlacklisted_suceeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Blacklist the dispute dispute game.
        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.blacklistDisputeGame(IDisputeGame(address(game)));

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` can be re-executed if the dispute game proven against has resolved
    ///      against the favor of the root claim.
    function test_proveWithdrawalTransaction_replayProveBadProposal_suceeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        vm.mockCall(address(game), abi.encodeCall(game.status, ()), abi.encode(GameStatus.CHALLENGER_WINS));

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` can be re-executed if the dispute game proven against is no longer
    ///      of the respected game type.
    function test_proveWithdrawalTransaction_replayRespectedGameTypeChanged_suceeds() external {
        // Prove the withdrawal against a game with the current respected game type.
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Update the respected game type to 0xbeef.
        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.setRespectedGameType(GameType.wrap(0xbeef));

        // Create a new game and mock the game type as 0xbeef in the factory.
        IDisputeGame newGame =
            disputeGameFactory.create(GameType.wrap(0), Claim.wrap(_outputRoot), abi.encode(_proposedBlockNumber + 1));
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(disputeGameFactory.gameAtIndex, (_proposedGameIndex + 1)),
            abi.encode(GameType.wrap(0xbeef), Timestamp.wrap(uint64(block.timestamp)), IDisputeGame(address(newGame)))
        );

        // Re-proving should be successful against the new game.
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex + 1,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `proveWithdrawalTransaction` succeeds.
    function test_proveWithdrawalTransaction_validWithdrawalProof_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` succeeds.
    function test_finalizeWithdrawalTransaction_provenWithdrawalHash_succeeds() external {
        uint256 bobBalanceBefore = address(bob).balance;

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Warp and resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1 seconds);

        vm.expectEmit(true, true, false, true);
        emit WithdrawalFinalized(_withdrawalHash, true);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore + 100);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the contract is paused.
    function test_finalizeWithdrawalTransaction_paused_reverts() external {
        vm.prank(l1ToL2MessagePasser.GUARDIAN());
        superchainConfig.pause("identifier");

        vm.expectRevert("L1ToL2MessagePasser: paused");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal has not been
    function test_finalizeWithdrawalTransaction_ifWithdrawalNotProven_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        vm.expectRevert("L1ToL2MessagePasser: withdrawal has not been proven yet");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal has not been
    ///      proven long enough ago.
    function test_finalizeWithdrawalTransaction_ifWithdrawalProofNotOldEnough_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        vm.expectRevert("L1ToL2MessagePasser: proven withdrawal has not matured yet");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the provenWithdrawal's timestamp
    ///      is less than the dispute game's creation timestamp.
    function test_finalizeWithdrawalTransaction_timestampLessThanGameCreation_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        // Prove our withdrawal
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Warp to after the finalization period
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);

        // Mock a createdAt change in the dispute game.
        vm.mockCall(address(game), abi.encodeWithSignature("createdAt()"), abi.encode(block.timestamp + 1));

        // Attempt to finalize the withdrawal
        vm.expectRevert("L1ToL2MessagePasser: withdrawal timestamp less than dispute game creation timestamp");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        // Ensure that bob's balance has remained the same
        assertEq(bobBalanceBefore, address(bob).balance);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the dispute game has not resolved in favor of the
    ///      root claim.
    function test_finalizeWithdrawalTransaction_ifDisputeGameNotResolved_reverts() external {
        uint256 bobBalanceBefore = address(bob).balance;

        // Prove our withdrawal
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Warp to after the finalization period
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);

        // Attempt to finalize the withdrawal
        vm.expectRevert("L1ToL2MessagePasser: output proposal has not been finalized yet");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        // Ensure that bob's balance has remained the same
        assertEq(bobBalanceBefore, address(bob).balance);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the target reverts.
    function test_finalizeWithdrawalTransaction_targetFails_fails() external {
        uint256 bobBalanceBefore = address(bob).balance;
        vm.etch(bob, hex"fe"); // Contract with just the invalid opcode.

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, false);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        assert(address(bob).balance == bobBalanceBefore);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal has already been
    ///      finalized.
    function test_finalizeWithdrawalTransaction_onReplay_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(_withdrawalHash, true);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        vm.expectRevert("L1ToL2MessagePasser: withdrawal has already been finalized");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
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
            address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(Hashing.hashOutputRootProof(outputRootProof))
        );

        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: insufficientGasTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: outputRootProof,
            _withdrawalProof: withdrawalProof
        });

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);
        vm.expectRevert("SafeCall: Not enough gas");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction{ gas: gasLimit }(insufficientGasTx);
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

        // Return a mock output root from the game.
        vm.mockCall(address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(outputRoot));

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(withdrawalHash, alice, address(this));
        l1ToL2MessagePasser.proveWithdrawalTransaction(_testTx, _proposedGameIndex, outputRootProof, withdrawalProof);

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);
        vm.expectCall(address(this), _testTx.data);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(withdrawalHash, true);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_testTx);

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
            _target != address(l1ToL2MessagePasser) // Cannot call the optimism portal or a contract
                && _target.code.length == 0 // No accounts with code
                && _target != CONSOLE // The console has no code but behaves like a contract
                && uint160(_target) > 9 // No precompiles (or zero address)
        );

        // Total ETH supply is currently about 120M ETH.
        uint256 value = bound(_value, 0, 200_000_000 ether);
        vm.deal(address(l1ToL2MessagePasser), value);

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

        // Setup the dispute game to return the output root
        vm.mockCall(address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(outputRoot));

        // Prove the withdrawal transaction
        l1ToL2MessagePasser.proveWithdrawalTransaction(_tx, _proposedGameIndex, proof, withdrawalProof);
        (IDisputeGame _game,) = l1ToL2MessagePasser.provenWithdrawals(withdrawalHash);
        assertTrue(_game.rootClaim().raw() != bytes32(0));

        // Resolve the dispute game
        game.resolveClaim(0);
        game.resolve();

        // Warp past the finalization period
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);

        // Finalize the withdrawal transaction
        vm.expectCallMinGas(_tx.target, _tx.value, uint64(_tx.gasLimit), _tx.data);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_tx);
        assertTrue(l1ToL2MessagePasser.finalizedWithdrawals(withdrawalHash));
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal's dispute game has been blacklisted.
    function test_finalizeWithdrawalTransaction_blacklisted_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.blacklistDisputeGame(IDisputeGame(address(game)));

        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);

        vm.expectRevert("L1ToL2MessagePasser: dispute game has been blacklisted");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the withdrawal's dispute game is still in the air
    ///      gap.
    function test_finalizeWithdrawalTransaction_gameInAirGap_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Warp past the finalization period.
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        // Attempt to finalize the withdrawal directly after the game resolves. This should fail.
        vm.expectRevert("L1ToL2MessagePasser: output proposal in air-gap");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        // Finalize the withdrawal transaction. This should succeed.
        vm.warp(block.timestamp + l1ToL2MessagePasser.disputeGameFinalityDelaySeconds() + 1);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
        assertTrue(l1ToL2MessagePasser.finalizedWithdrawals(_withdrawalHash));
    }

    /// @dev Tests that `finalizeWithdrawalTransaction` reverts if the respected game type has changed since the
    ///      withdrawal was proven.
    function test_finalizeWithdrawalTransaction_respectedTypeChangedSinceProving_reverts() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Warp past the finalization period.
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds() + 1);

        // Resolve the dispute game.
        game.resolveClaim(0);
        game.resolve();

        // Change the respected game type in the portal.
        vm.prank(l1ToL2MessagePasser.sauron());
        l1ToL2MessagePasser.setRespectedGameType(GameType.wrap(0xFF));

        vm.expectRevert("L1ToL2MessagePasser: invalid game type");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @dev Tests an e2e prove -> finalize path, checking the edges of each delay for correctness.
    function test_finalizeWithdrawalTransaction_delayEdges_succeeds() external {
        // Prove the withdrawal transaction.
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(_withdrawalHash, alice, bob);
        l1ToL2MessagePasser.proveWithdrawalTransaction({
            _tx: _defaultTx,
            _disputeGameIndex: _proposedGameIndex,
            _outputRootProof: _outputRootProof,
            _withdrawalProof: _withdrawalProof
        });

        // Attempt to finalize the withdrawal transaction 1 second before the proof has matured. This should fail.
        vm.warp(block.timestamp + l1ToL2MessagePasser.proofMaturityDelaySeconds());
        vm.expectRevert("L1ToL2MessagePasser: proven withdrawal has not matured yet");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        // Warp 1 second in the future, past the proof maturity delay, and attempt to finalize the withdrawal.
        // This should also fail, since the dispute game has not resolved yet.
        vm.warp(block.timestamp + 1 seconds);
        vm.expectRevert("L1ToL2MessagePasser: output proposal has not been finalized yet");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        // Finalize the dispute game and attempt to finalize the withdrawal again. This should also fail, since the
        // air gap dispute game delay has not elapsed.
        game.resolveClaim(0);
        game.resolve();
        vm.warp(block.timestamp + l1ToL2MessagePasser.disputeGameFinalityDelaySeconds());
        vm.expectRevert("L1ToL2MessagePasser: output proposal in air-gap");
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);

        // Warp 1 second in the future, past the air gap dispute game delay, and attempt to finalize the withdrawal.
        // This should succeed.
        vm.warp(block.timestamp + 1 seconds);
        l1ToL2MessagePasser.finalizeWithdrawalTransaction(_defaultTx);
        assertTrue(l1ToL2MessagePasser.finalizedWithdrawals(_withdrawalHash));
    }
}

contract L1ToL2MessagePasser_Upgradeable_Test is CommonTest {
    function setUp() public override {
        super.enableFaultProofs();
        super.setUp();
    }

    /// @dev Tests that the proxy is initialized correctly.
    function test_params_initValuesOnProxy_succeeds() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = l1ToL2MessagePasser.params();
        ResourceMetering.ResourceConfig memory rcfg = systemConfig.resourceConfig();

        assertEq(prevBaseFee, rcfg.minimumBaseFee);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, block.number);
    }

    /// @dev Tests that the proxy can be upgraded.
    function test_upgradeToAndCall_upgrading_succeeds() external {
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(l1ToL2MessagePasser), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();

        vm.startPrank(EIP1967Helper.getAdmin(address(l1ToL2MessagePasser)));
        // The value passed to the initialize must be larger than the last value
        // that initialize was called with.
        Proxy(payable(address(l1ToL2MessagePasser))).upgradeToAndCall(
            address(nextImpl), abi.encodeWithSelector(NextImpl.initialize.selector, 2)
        );
        assertEq(Proxy(payable(address(l1ToL2MessagePasser))).implementation(), address(nextImpl));

        // Verify that the NextImpl contract initialized its values according as expected
        bytes32 slot21After = vm.load(address(l1ToL2MessagePasser), bytes32(uint256(21)));
        bytes32 slot21Expected = NextImpl(address(l1ToL2MessagePasser)).slot21Init();
        assertEq(slot21Expected, slot21After);
    }
}

/// @title L1ToL2MessagePasser_ResourceFuzz_Test
/// @dev Test various values of the resource metering config to ensure that deposits cannot be
///      broken by changing the config.
contract L1ToL2MessagePasser_ResourceFuzz_Test is CommonTest {
    /// @dev The max gas limit observed throughout this test. Setting this too high can cause
    ///      the test to take too long to run.
    uint256 constant MAX_GAS_LIMIT = 30_000_000;

    function setUp() public override {
        super.enableFaultProofs();
        super.setUp();
    }

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
            address(l1ToL2MessagePasser),
            bytes32(uint256(1)),
            bytes32((_prevBlockNum << 192) | (uint256(_prevBoughtGas) << 128) | _prevBaseFee)
        );
        // Ensure that the storage setting is correct
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = l1ToL2MessagePasser.params();
        assertEq(prevBaseFee, _prevBaseFee);
        assertEq(prevBoughtGas, _prevBoughtGas);
        assertEq(prevBlockNum, _prevBlockNum);

        // Do a deposit, should not revert
        l1ToL2MessagePasser.depositTransaction{ gas: MAX_GAS_LIMIT }({
            _to: address(0x20),
            _value: 0x40,
            _gasLimit: _gasLimit,
            _isCreation: false,
            _data: hex""
        });
    }
}
