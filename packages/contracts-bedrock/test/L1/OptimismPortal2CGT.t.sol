// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import {
    OptimismPortal2_TestInit,
    OptimismPortal2_Version_Test,
    OptimismPortal2_Initialize_Test,
    OptimismPortal2_Constructor_Test,
    OptimismPortal2_UpgradeInterop_Test,
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

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Interfaces
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";

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
contract OptimismPortal2_CGT_Initialize_Test is OptimismPortal2_Initialize_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }

    /// @notice Tests that the initializer sets the correct values.
    /// @dev Marked virtual to be overridden in
    ///      test/kontrol/deployment/DeploymentSummary.t.sol
    function test_initialize_succeeds() public override {
        skipIfForkTest("OptimismPortal2_Initialize_Test: isCustomGasToken() not available on forked networks");
        super.test_initialize_succeeds();
    }
}

/// @title OptimismPortal2_CGT_UpgradeInterop_Test
/// @notice Tests the upgrade functionality of the `OptimismPortal2` contract with custom gas token
///         enabled.
contract OptimismPortal2_CGT_UpgradeInterop_Test is OptimismPortal2_UpgradeInterop_Test {
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
        skipIfForkTest("OptimismPortal2_Initialize_Test: isCustomGasToken() not available on forked networks");

        // Set the withdrawal transaction value to a non-zero value.
        _defaultTx.value = bound(uint256(1), 1, type(uint256).max);

        // Finalize the withdrawal transaction. This should revert.
        vm.expectRevert(IOptimismPortal.OptimismPortal_NotAllowedOnCGTMode.selector);
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);
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
    }
}

/// @title OptimismPortal2_CGT_CheckWithdrawal_Test
/// @notice Tests the `checkWithdrawal` function of the `OptimismPortal2` contract with custom gas
///         token enabled.
contract OptimismPortal2_CGT_CheckWithdrawal_Test is OptimismPortal2_CheckWithdrawal_Test {
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
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
        skipIfForkTest("OptimismPortal2_Initialize_Test: isCustomGasToken() not available on forked networks");

        // Prevent overflow on an upgrade context
        _value = bound(_value, 1, type(uint256).max - address(optimismPortal2).balance);
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
