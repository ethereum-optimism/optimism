// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import {
    OptimismPortal2_TestInit,
    OptimismPortal2_Version_Test,
    OptimismPortal2_Constructor_Test,
    OptimismPortal2_Upgrade_Test,
    OptimismPortal2_MinimumGasLimit_Test,
    OptimismPortal2_Paused_Test,
    OptimismPortal2_ProofMaturityDelaySeconds_Test,
    OptimismPortal2_DisputeGameFactory_Test,
    OptimismPortal2_SuperchainConfig_Test,
    OptimismPortal2_Guardian_Test,
    OptimismPortal2_DisputeGameFinalityDelaySeconds_Test,
    OptimismPortal2_RespectedGameType_Test,
    OptimismPortal2_RespectedGameTypeUpdatedAt_Test,
    OptimismPortal2_DisputeGameBlacklist_Test,
    OptimismPortal2_NumProofSubmitters_Test,
    OptimismPortal2_DonateETH_Test,
    OptimismPortal2_MigrateLiquidity_Test,
    OptimismPortal2_MigrateToSuperRoots_Test,
    OptimismPortal2_ProveWithdrawalTransaction_Test,
    OptimismPortal2_FinalizeWithdrawalTransaction_Test,
    OptimismPortal2_FinalizeWithdrawalTransactionExternalProof_Test,
    OptimismPortal2_CheckWithdrawal_Test,
    OptimismPortal2_DepositTransaction_Test,
    OptimismPortal2_Params_Test
} from "test/L1/OptimismPortal2.t.sol";

// Contracts
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";

// Scripts
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Constants } from "src/libraries/Constants.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { GameType, Claim } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";

/// @title OptimismPortal2CGT_TestInit
/// @notice Reusable test initialization for `OptimismPortal2` tests with custom gas token enabled.
contract OptimismPortal2CGT_TestInit is OptimismPortal2_TestInit {
    /// @notice Sets up the test suite with custom gas token enabled.
    function setUp() public virtual override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_Version_Test
/// @notice Tests the `version` function of the `OptimismPortal2` contract with custom gas token
///         enabled.
contract OptimismPortal2_CGT_Version_Test is OptimismPortal2_Version_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_Constructor_Test
/// @notice Tests the constructor of the `OptimismPortal2` contract with custom gas token enabled.
contract OptimismPortal2_CGT_Constructor_Test is OptimismPortal2_Constructor_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_Initialize_Test
/// @notice Test contract for OptimismPortal2 `initialize` function.
contract OptimismPortal2_Initialize_Test is OptimismPortal2CGT_TestInit {
    /// @notice Tests that the initializer sets the correct values.
    /// @dev Marked virtual to be overridden in
    ///      test/kontrol/deployment/DeploymentSummary.t.sol
    function test_initialize_succeeds() external virtual {
        assertEq(address(optimismPortal2.anchorStateRegistry()), address(anchorStateRegistry));
        assertEq(address(optimismPortal2.disputeGameFactory()), address(disputeGameFactory));
        assertEq(address(optimismPortal2.superchainConfig()), address(superchainConfig));
        assertEq(optimismPortal2.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(optimismPortal2.paused(), false);
        assertEq(address(optimismPortal2.systemConfig()), address(systemConfig));
        assertEq(address(optimismPortal2.ethLockbox()), address(ethLockbox));
        assertTrue(OptimismPortal2(payable(address(optimismPortal2))).isCustomGasToken());

        returnIfForkTest(
            "OptimismPortal2_Initialize_Test: Do not check guardian and respectedGameType on forked networks"
        );
        address guardian = superchainConfig.guardian();

        // This check is not valid for forked tests, as the guardian is not the same as the one in hardhat.json
        assertEq(guardian, deploy.cfg().superchainConfigGuardian());

        // This check is not valid on forked tests as the respectedGameType varies between OP Chains.
        assertEq(optimismPortal2.respectedGameType().raw(), deploy.cfg().respectedGameType());
    }

    /// @notice Tests that the initialize function reverts if called by a non-proxy admin or owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("OptimismPortal2", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(optimismPortal2), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector.
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `initialize` function with the sender
        vm.prank(_sender);
        optimismPortal2.initialize(systemConfig, anchorStateRegistry, ethLockbox, true);
    }
}

/// @title OptimismPortal2_CGT_Upgrade_Test
/// @notice Tests the upgrade functionality of the `OptimismPortal2` contract with custom gas token
///         enabled.
contract OptimismPortal2_CGT_Upgrade_Test is OptimismPortal2_Upgrade_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_MinimumGasLimit_Test
/// @notice Tests the `minimumGasLimit` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_MinimumGasLimit_Test is OptimismPortal2_MinimumGasLimit_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_Paused_Test
/// @notice Tests the paused state of the `OptimismPortal2` contract with custom gas token enabled.
contract OptimismPortal2_CGT_Paused_Test is OptimismPortal2_Paused_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_ProofMaturityDelaySeconds_Test
/// @notice Tests the `proofMaturityDelaySeconds` function of the `OptimismPortal2` contract with
///         custom gas token enabled.
contract OptimismPortal2_CGT_ProofMaturityDelaySeconds_Test is OptimismPortal2_ProofMaturityDelaySeconds_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_DisputeGameFactory_Test
/// @notice Tests the `disputeGameFactory` function of the `OptimismPortal2` contract with custom
///         gas token enabled.
contract OptimismPortal2_CGT_DisputeGameFactory_Test is OptimismPortal2_DisputeGameFactory_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_SuperchainConfig_Test
/// @notice Tests the `superchainConfig` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_SuperchainConfig_Test is OptimismPortal2_SuperchainConfig_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_Guardian_Test
/// @notice Tests the `guardian` function of the `OptimismPortal2` contract with custom gas token
///         enabled.
contract OptimismPortal2_CGT_Guardian_Test is OptimismPortal2_Guardian_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_DisputeGameFinalityDelaySeconds_Test
/// @notice Tests the `disputeGameFinalityDelaySeconds` function of the `OptimismPortal2` contract
///         with custom gas token enabled.
contract OptimismPortal2_CGT_DisputeGameFinalityDelaySeconds_Test is
    OptimismPortal2_DisputeGameFinalityDelaySeconds_Test
{
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_RespectedGameType_Test
/// @notice Tests the `respectedGameType` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_RespectedGameType_Test is OptimismPortal2_RespectedGameType_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_RespectedGameTypeUpdatedAt_Test
/// @notice Tests the `respectedGameTypeUpdatedAt` function of the `OptimismPortal2` contract with
///         custom gas token enabled.
contract OptimismPortal2_CGT_RespectedGameTypeUpdatedAt_Test is OptimismPortal2_RespectedGameTypeUpdatedAt_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_DisputeGameBlacklist_Test
/// @notice Tests the `disputeGameBlacklist` function of the `OptimismPortal2` contract with custom
///         gas token enabled.
contract OptimismPortal2_CGT_DisputeGameBlacklist_Test is OptimismPortal2_DisputeGameBlacklist_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_NumProofSubmitters_Test
/// @notice Tests the `numProofSubmitters` function of the `OptimismPortal2` contract with custom
///         gas token enabled.
contract OptimismPortal2_CGT_NumProofSubmitters_Test is OptimismPortal2_NumProofSubmitters_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_DonateETH_Test
/// @notice Tests the `donateETH` function of the `OptimismPortal2` contract with custom gas token
///         enabled.
contract OptimismPortal2_CGT_DonateETH_Test is OptimismPortal2_DonateETH_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_Receive_Test
/// @notice Test contract for OptimismPortal2 `receive` function.
contract OptimismPortal2_CGT_Receive_Test is OptimismPortal2CGT_TestInit {
    /// @notice Tests that receive() reverts when custom gas token is enabled
    function testFuzz_receive_customGasToken_reverts(uint256 _value) external virtual {
        _value = bound(_value, 1, type(uint128).max);
        vm.deal(alice, _value);

        address portal = address(optimismPortal2);

        vm.prank(alice);
        vm.expectRevert(IOptimismPortal.OptimismPortal_NotAllowedOnCGTMode.selector);
        assembly {
            pop(call(gas(), portal, _value, 0, 0, 0, 0))
        }
    }
}

/// @title OptimismPortal2_CGT_MigrateLiquidity_Test
/// @notice Tests the `migrateLiquidity` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_MigrateLiquidity_Test is OptimismPortal2_MigrateLiquidity_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_MigrateToSuperRoots_Test
/// @notice Tests the `migrateToSuperRoots` function of the `OptimismPortal2` contract with custom
///         gas token enabled.
contract OptimismPortal2_CGT_MigrateToSuperRoots_Test is OptimismPortal2_MigrateToSuperRoots_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_ProveWithdrawalTransaction_Test
/// @notice Tests the `proveWithdrawalTransaction` function of the `OptimismPortal2` contract with
///         custom gas token enabled.
contract OptimismPortal2_CGT_ProveWithdrawalTransaction_Test is OptimismPortal2_ProveWithdrawalTransaction_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title OptimismPortal2_CGT_FinalizeWithdrawalTransaction_Test
/// @notice Tests the `finalizeWithdrawalTransaction` function of the `OptimismPortal2` contract with
///         custom gas token enabled.
contract OptimismPortal2_CGT_FinalizeWithdrawalTransaction_Test is
    OptimismPortal2_FinalizeWithdrawalTransaction_Test
{
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();

        // Override _defaultTx to use zero value for CGT compatibility
        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 0, // Zero value for CGT mode
            gasLimit: 100_000,
            data: hex"aa"
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
        if (isForkTest()) {
            // Set the proposed block number to be the next block number on the forked network
            (, _proposedBlockNumber) = anchorStateRegistry.getAnchorRoot();
            _proposedBlockNumber += 1;

            // Set the init bond of anchor game type 0 to be 0.
            // It is a mapping so the storage slot is calculated as keccak256(abi.encode(key, slot)).
            // The storage slot for the initBond mapping is 102, see `snapshots/storageLayout/DisputeGameFactory.json`.
            vm.store(
                address(disputeGameFactory), keccak256(abi.encode(GameType.wrap(0), uint256(102))), bytes32(uint256(0))
            );
        } else {
            // Set up the dummy game.
            _proposedBlockNumber = 0xFF;
        }

        depositor = makeAddr("depositor");

        setupFaultDisputeGame(Claim.wrap(_outputRoot));

        // Warp forward in time to ensure that the game is created after the retirement timestamp.
        vm.warp(anchorStateRegistry.retirementTimestamp() + 1);

        respectedGameType = optimismPortal2.respectedGameType();
        game = IFaultDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: disputeGameFactory.initBonds(respectedGameType) }(
                        respectedGameType, Claim.wrap(_outputRoot), abi.encode(_proposedBlockNumber)
                    )
                )
            )
        );

        // Grab the index of the game we just created.
        _proposedGameIndex = disputeGameFactory.gameCount() - 1;

        // Warp beyond the chess clocks and finalize the game.
        vm.warp(block.timestamp + game.maxClockDuration().raw() + 1 seconds);

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(ethLockbox), 0xFFFFFFFF);
    }

    /// @notice Tests that `finalizeWithdrawalTransaction` succeeds when the custom gas token mode
    ///         is enabled and the withdrawal transaction has no value.
    function test_finalizeWithdrawalTransaction_withoutValueAndCustomGasToken_succeeds(
        address _sender,
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        skipIfForkTest("Skipping on forked tests because of the L2ToL1MessageParser call below");

        vm.assume(
            _target != address(optimismPortal2) // Cannot call the optimism portal or a contract
                && _target.code.length == 0 // No accounts with code
                && _target != CONSOLE // The console has no code but behaves like a contract
                && uint160(_target) > 9 // No precompiles (or zero address)
        );

        uint256 gasLimit = bound(_gasLimit, 0, 50_000_000);
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        // Get a withdrawal transaction and mock proof from the differential testing script.
        Types.WithdrawalTransaction memory _tx = Types.WithdrawalTransaction({
            nonce: nonce,
            sender: _sender,
            target: _target,
            value: 0,
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
        Types.OutputRootProof memory outputProof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });

        // Ensure the values returned from ffi are correct
        assertEq(outputRoot, Hashing.hashOutputRootProof(outputProof));
        assertEq(withdrawalHash, Hashing.hashWithdrawal(_tx));

        // Setup the dispute game to return the output root
        vm.mockCall(address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(outputRoot));

        // Prove the withdrawal transaction
        optimismPortal2.proveWithdrawalTransaction(_tx, _proposedGameIndex, outputProof, withdrawalProof);
        (IDisputeGame _game,) = optimismPortal2.provenWithdrawals(withdrawalHash, address(this));
        assertTrue(_game.rootClaim().raw() != bytes32(0));

        // Resolve the dispute game
        game.resolveClaim(0, 0);
        game.resolve();

        // Warp past the finalization period
        vm.warp(block.timestamp + optimismPortal2.proofMaturityDelaySeconds() + 1);

        // Finalize the withdrawal transaction
        vm.expectCallMinGas(_tx.target, _tx.value, uint64(_tx.gasLimit), _tx.data);
        optimismPortal2.finalizeWithdrawalTransaction(_tx);
        assertTrue(optimismPortal2.finalizedWithdrawals(withdrawalHash));
    }

    /// @notice Tests that `finalizeWithdrawalTransaction` reverts when the custom gas token mode
    ///         is enabled and the withdrawal transaction has a value.
    function test_finalizeWithdrawalTransaction_withValueAndCustomGasToken_reverts() external {
        // Set the withdrawal transaction value to a non-zero value.
        _defaultTx.value = bound(uint256(1), 1, type(uint256).max);

        // Finalize the withdrawal transaction. This should revert.
        vm.expectRevert(IOptimismPortal.OptimismPortal_NotAllowedOnCGTMode.selector);
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);
    }

    // Override tests that use non-zero values and expect different reverts in CGT mode

    /// @notice Override: In CGT mode, test with zero value and expect success
    function test_finalizeWithdrawalTransaction_badTarget_reverts() external override {
        _defaultTx.target = address(optimismPortal2);
        vm.expectRevert(IOptimismPortal.OptimismPortal_BadTarget.selector);
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);

        _defaultTx.target = address(ethLockbox);
        vm.expectRevert(IOptimismPortal.OptimismPortal_BadTarget.selector);
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);
    }

    /// @notice Override: In CGT mode, any fuzzed test with value will revert
    function testDiff_finalizeWithdrawalTransaction_succeeds(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
        override
    {
        skipIfForkTest("Skipping on forked tests because of the L2ToL1MessageParser call below");

        vm.assume(
            _target != address(optimismPortal2) // Cannot call the optimism portal or a contract
                && _target.code.length == 0 // No accounts with code
                && _target != CONSOLE // The console has no code but behaves like a contract
                && uint160(_target) > 9 // No precompiles (or zero address)
        );

        // In CGT mode, any non-zero value will cause revert
        _value = 0;

        uint256 gasLimit = bound(_gasLimit, 0, 50_000_000);
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        // Get a withdrawal transaction and mock proof from the differential testing script.
        Types.WithdrawalTransaction memory _tx = Types.WithdrawalTransaction({
            nonce: nonce,
            sender: _sender,
            target: _target,
            value: _value,
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
        optimismPortal2.proveWithdrawalTransaction(_tx, _proposedGameIndex, proof, withdrawalProof);
        (IDisputeGame _game,) = optimismPortal2.provenWithdrawals(withdrawalHash, address(this));
        assertTrue(_game.rootClaim().raw() != bytes32(0));

        // Resolve the dispute game
        game.resolveClaim(0, 0);
        game.resolve();

        // Warp past the finalization period
        vm.warp(block.timestamp + optimismPortal2.proofMaturityDelaySeconds() + 1);

        // Finalize the withdrawal transaction
        vm.expectCallMinGas(_tx.target, _tx.value, uint64(_tx.gasLimit), _tx.data);
        optimismPortal2.finalizeWithdrawalTransaction(_tx);
        assertTrue(optimismPortal2.finalizedWithdrawals(withdrawalHash));
    }

    /// @notice Tests that `finalizeWithdrawalTransaction` succeeds even if the respected game type
    ///         is changed.
    function test_finalizeWithdrawalTransaction_wasRespectedGameType_succeeds(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data,
        GameType _newGameType
    )
        external
        override
    {
        skipIfForkTest("Skipping on forked tests because of the L2ToL1MessageParser call below");

        vm.assume(
            _target != address(optimismPortal2) // Cannot call the optimism portal or a contract
                && _target.code.length == 0 // No accounts with code
                && _target != CONSOLE // The console has no code but behaves like a contract
                && uint160(_target) > 9 // No precompiles (or zero address)
        );

        // Bound to prevent changes in retirementTimestamp
        _newGameType = GameType.wrap(uint32(bound(_newGameType.raw(), 0, type(uint32).max - 1)));

        _value = 0;

        uint256 gasLimit = bound(_gasLimit, 0, 50_000_000);
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        // Get a withdrawal transaction and mock proof from the differential testing script.
        Types.WithdrawalTransaction memory _tx = Types.WithdrawalTransaction({
            nonce: nonce,
            sender: _sender,
            target: _target,
            value: _value,
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
        optimismPortal2.proveWithdrawalTransaction(_tx, _proposedGameIndex, proof, withdrawalProof);
        (IDisputeGame _game,) = optimismPortal2.provenWithdrawals(withdrawalHash, address(this));
        assertTrue(_game.rootClaim().raw() != bytes32(0));

        // Resolve the dispute game
        game.resolveClaim(0, 0);
        game.resolve();

        // Warp past the finalization period
        vm.warp(block.timestamp + optimismPortal2.proofMaturityDelaySeconds() + 1);

        // Change the respectedGameType
        vm.prank(optimismPortal2.guardian());
        anchorStateRegistry.setRespectedGameType(_newGameType);

        // Withdrawal transaction still finalizable
        vm.expectCallMinGas(_tx.target, _tx.value, uint64(_tx.gasLimit), _tx.data);
        optimismPortal2.finalizeWithdrawalTransaction(_tx);
        assertTrue(optimismPortal2.finalizedWithdrawals(withdrawalHash));
    }
}

/// @title OptimismPortal2_CGT_FinalizeWithdrawalTransactionExternalProof_Test
/// @notice Tests the `finalizeWithdrawalTransactionExternalProof` function of the `OptimismPortal2`
///         contract with custom gas token enabled.
contract OptimismPortal2_CGT_FinalizeWithdrawalTransactionExternalProof_Test is
    OptimismPortal2_FinalizeWithdrawalTransactionExternalProof_Test
{
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();

        // Override _defaultTx to use zero value for CGT compatibility
        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 0, // Zero value for CGT mode
            gasLimit: 100_000,
            data: hex"aa"
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
        if (isForkTest()) {
            // Set the proposed block number to be the next block number on the forked network
            (, _proposedBlockNumber) = anchorStateRegistry.getAnchorRoot();
            _proposedBlockNumber += 1;

            // Set the init bond of anchor game type 0 to be 0.
            // It is a mapping so the storage slot is calculated as keccak256(abi.encode(key, slot)).
            // The storage slot for the initBond mapping is 102, see `snapshots/storageLayout/DisputeGameFactory.json`.
            vm.store(
                address(disputeGameFactory), keccak256(abi.encode(GameType.wrap(0), uint256(102))), bytes32(uint256(0))
            );
        } else {
            // Set up the dummy game.
            _proposedBlockNumber = 0xFF;
        }

        depositor = makeAddr("depositor");

        setupFaultDisputeGame(Claim.wrap(_outputRoot));

        // Warp forward in time to ensure that the game is created after the retirement timestamp.
        vm.warp(anchorStateRegistry.retirementTimestamp() + 1);

        respectedGameType = optimismPortal2.respectedGameType();
        game = IFaultDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: disputeGameFactory.initBonds(respectedGameType) }(
                        respectedGameType, Claim.wrap(_outputRoot), abi.encode(_proposedBlockNumber)
                    )
                )
            )
        );

        // Grab the index of the game we just created.
        _proposedGameIndex = disputeGameFactory.gameCount() - 1;

        // Warp beyond the chess clocks and finalize the game.
        vm.warp(block.timestamp + game.maxClockDuration().raw() + 1 seconds);

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(ethLockbox), 0xFFFFFFFF);
    }
}

/// @title OptimismPortal2_CGT_CheckWithdrawal_Test
/// @notice Tests the `checkWithdrawal` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_CheckWithdrawal_Test is OptimismPortal2_CheckWithdrawal_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();

        // Override _defaultTx to use zero value for CGT compatibility
        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 0, // Zero value for CGT mode
            gasLimit: 100_000,
            data: hex"aa"
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
        if (isForkTest()) {
            // Set the proposed block number to be the next block number on the forked network
            (, _proposedBlockNumber) = anchorStateRegistry.getAnchorRoot();
            _proposedBlockNumber += 1;

            // Set the init bond of anchor game type 0 to be 0.
            // It is a mapping so the storage slot is calculated as keccak256(abi.encode(key, slot)).
            // The storage slot for the initBond mapping is 102, see `snapshots/storageLayout/DisputeGameFactory.json`.
            vm.store(
                address(disputeGameFactory), keccak256(abi.encode(GameType.wrap(0), uint256(102))), bytes32(uint256(0))
            );
        } else {
            // Set up the dummy game.
            _proposedBlockNumber = 0xFF;
        }

        depositor = makeAddr("depositor");

        setupFaultDisputeGame(Claim.wrap(_outputRoot));

        // Warp forward in time to ensure that the game is created after the retirement timestamp.
        vm.warp(anchorStateRegistry.retirementTimestamp() + 1);

        respectedGameType = optimismPortal2.respectedGameType();
        game = IFaultDisputeGame(
            payable(
                address(
                    disputeGameFactory.create{ value: disputeGameFactory.initBonds(respectedGameType) }(
                        respectedGameType, Claim.wrap(_outputRoot), abi.encode(_proposedBlockNumber)
                    )
                )
            )
        );

        // Grab the index of the game we just created.
        _proposedGameIndex = disputeGameFactory.gameCount() - 1;

        // Warp beyond the chess clocks and finalize the game.
        vm.warp(block.timestamp + game.maxClockDuration().raw() + 1 seconds);

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(ethLockbox), 0xFFFFFFFF);
    }
}

/// @title OptimismPortal2_CGT_DepositTransaction_Test
/// @notice Tests the `depositTransaction` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_DepositTransaction_Test is OptimismPortal2_DepositTransaction_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }

    /// @notice Override: Tests that depositTransaction with EOA succeeds only with zero value in CGT mode
    function testFuzz_depositTransaction_eoa_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data
    )
        external
        override
    {
        // In CGT mode, only allow zero value transactions
        _value = 0;
        _mint = 0;

        // Prevent overflow on an upgrade context
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal2.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        uint256 balanceBefore = address(optimismPortal2).balance;
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // EOA emulation
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited({
            _from: depositor,
            _to: _to,
            _value: _value,
            _mint: _mint,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });

        // Expect call to the ETHLockbox to lock the funds only if the value is greater than 0.
        vm.expectCall(address(ethLockbox), _mint, abi.encodeCall(ethLockbox.lockETH, ()), _mint > 0 ? 1 : 0);

        vm.deal(depositor, _mint);
        vm.prank(depositor, depositor);
        optimismPortal2.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal2).balance, balanceBefore);
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore + _mint);
    }

    /// @notice Override: Tests that depositTransaction with contract succeeds only with zero value in CGT mode
    function testFuzz_depositTransaction_contract_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data
    )
        external
        override
    {
        // In CGT mode, only allow zero value transactions
        _value = 0;
        _mint = 0;

        // Prevent overflow on an upgrade context
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal2.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        uint256 balanceBefore = address(optimismPortal2).balance;
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // EOA emulation
        vm.expectEmit(address(optimismPortal2));
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
        optimismPortal2.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal2).balance, balanceBefore);
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore + _mint);
    }

    /// @notice Override: Tests that depositTransaction with 7702 delegated contract succeeds only with zero value in
    /// CGT mode
    function testFuzz_depositTransaction_eoa7702_succeeds(
        address _to,
        uint64 _gasLimit,
        uint256 _value,
        uint256 _mint,
        bool _isCreation,
        bytes memory _data,
        address _7702Target
    )
        external
        override
    {
        // In CGT mode, only allow zero value transactions
        _value = 0;
        _mint = 0;

        assumeNotForgeAddress(_7702Target);

        // Prevent overflow on an upgrade context
        _gasLimit = uint64(
            bound(
                _gasLimit,
                optimismPortal2.minimumGasLimit(uint64(_data.length)),
                systemConfig.resourceConfig().maxResourceLimit
            )
        );
        if (_isCreation) _to = address(0);

        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // EOA emulation
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited({
            _from: depositor,
            _to: _to,
            _value: _value,
            _mint: _mint,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });

        // 7702 delegation using the 7702 prefix
        vm.etch(depositor, abi.encodePacked(hex"EF0100", _7702Target));

        vm.deal(depositor, _mint);
        vm.prank(depositor, address(0x0420));
        optimismPortal2.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: _gasLimit,
            _isCreation: _isCreation,
            _data: _data
        });
        assertEq(address(optimismPortal2).balance, portalBalanceBefore);
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore + _mint);
    }

    /// @notice Tests that depositTransaction succeeds when value = 0 and custom gas token is enabled
    function testFuzz_depositTransaction_withZeroValue_succeeds(bytes memory _data, address _to) external {
        uint64 gasLimit = optimismPortal2.minimumGasLimit(uint64(_data.length));
        assumeNotForgeAddress(_to);
        assumeNotZeroAddress(_to);

        vm.prank(alice);
        optimismPortal2.depositTransaction({ _to: _to, _value: 0, _gasLimit: gasLimit, _isCreation: false, _data: hex"" });
    }

    /// @notice Tests that `depositTransaction` reverts when the value is greater than 0 and the
    ///         custom gas token is active.
    function test_depositTransaction_withValue_reverts(bytes memory _data, uint256 _value) external {
        // Prevent overflow on an upgrade context
        _value = bound(_value, 1, type(uint256).max - address(ethLockbox).balance);
        uint64 gasLimit = optimismPortal2.minimumGasLimit(uint64(_data.length));

        vm.deal(alice, _value);
        vm.prank(alice);
        vm.expectRevert(IOptimismPortal.OptimismPortal_NotAllowedOnCGTMode.selector);
        optimismPortal2.depositTransaction{ value: _value }({
            _to: address(0x40),
            _value: _value,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: _data
        });
    }
}

/// @title OptimismPortal2_CGT_Params_Test
/// @notice Tests the parameter functions of the `OptimismPortal2` contract with custom gas token
///         enabled.
contract OptimismPortal2_CGT_Params_Test is OptimismPortal2_Params_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}
