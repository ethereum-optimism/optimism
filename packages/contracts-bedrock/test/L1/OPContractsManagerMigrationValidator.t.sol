// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Libraries
import { GameType, GameTypes, Claim, Proposal } from "src/dispute/lib/Types.sol";
import { Hash } from "src/dispute/lib/LibUDT.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IOPContractsManagerMigrationValidator } from "interfaces/L1/IOPContractsManagerMigrationValidator.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";

/// @title OPContractsManagerMigrationValidator_TestInit
/// @notice Base contract for MigrationValidator tests. Uses real opcmV2.deploy() + migrate()
///         to set up post-migration state, matching the pattern in OPContractsManagerV2_Migrate_Test.
abstract contract OPContractsManagerMigrationValidator_TestInit is CommonTest {
    /// @notice Deployed chain contracts for chain 1.
    IOPContractsManagerV2.ChainContracts chainContracts1;

    /// @notice Deployed chain contracts for chain 2.
    IOPContractsManagerV2.ChainContracts chainContracts2;

    /// @notice The shared DGF (created by migration).
    IDisputeGameFactory sharedDGF;

    /// @notice The shared ProxyAdmin discovered from the DGF.
    address sharedProxyAdmin;

    /// @notice The shared ASR discovered from SPDG game args.
    address sharedASR;

    /// @notice The shared WETH discovered from SPDG game args.
    address sharedWETH;

    /// @notice The shared lockbox created by migration (NOT ethLockbox from CommonTest).
    IETHLockbox sharedLockbox;

    /// @notice The StandardValidator instance (used to read impl addresses for refs).
    IOPContractsManagerStandardValidator standardValidator;

    /// @notice The MigrationValidator instance.
    IOPContractsManagerMigrationValidator migrationValidator;

    /// @notice Fake prestate for Cannon games (inline initializer — used before setUp completes).
    Claim cannonPrestate = Claim.wrap(bytes32(keccak256("cannonPrestate")));

    /// @notice Fake prestate for Cannon Kona games (inline initializer — used before setUp completes).
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice The proposer role for super games.
    address proposer;

    /// @notice The challenger role for super games.
    address challenger;

    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Deploy two chains via OPCMv2 for migration testing.
        chainContracts1 = _deployChainForMigration(1000001);
        chainContracts2 = _deployChainForMigration(1000002);

        // Set proposer/challenger before building migration input.
        proposer = makeAddr("superProposer");
        challenger = makeAddr("superChallenger");

        // Run real migration with both SPDG and SCKDG.
        _doMigration(_getDefaultMigrateInput());

        // Discover shared infra from real post-migration state.
        IOptimismPortal2 portal1 = chainContracts1.optimismPortal;
        IAnchorStateRegistry asr = portal1.anchorStateRegistry();
        sharedASR = address(asr);
        sharedDGF = IDisputeGameFactory(asr.disputeGameFactory());
        sharedProxyAdmin = address(IProxyAdminOwnedBase(address(sharedDGF)).proxyAdmin());
        sharedLockbox = IETHLockbox(portal1.ethLockbox());

        // Discover WETH from SPDG game args.
        LibGameArgs.GameArgs memory args = LibGameArgs.decode(sharedDGF.gameArgs(GameTypes.SUPER_PERMISSIONED_CANNON));
        sharedWETH = args.weth;

        // Get validators from OPCM.
        standardValidator = opcmV2.opcmStandardValidator();
        migrationValidator = standardValidator.migrationValidator();
    }

    /// @notice Deploys a chain via opcmV2.deploy() for subsequent migration.
    /// @param _l2ChainId The L2 chain ID for the deployed chain.
    /// @return cts_ The deployed chain contracts.
    function _deployChainForMigration(uint256 _l2ChainId)
        internal
        returns (IOPContractsManagerV2.ChainContracts memory cts_)
    {
        // Get initial proposer/challenger from existing DGF.
        address initialChallenger = DisputeGames.permissionedGameChallenger(disputeGameFactory);
        address initialProposer = DisputeGames.permissionedGameProposer(disputeGameFactory);

        IOPContractsManagerUtils.DisputeGameConfig[] memory dgConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](6);
        dgConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON,
            gameArgs: bytes("")
        });
        dgConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: cannonPrestate,
                    proposer: initialProposer,
                    challenger: initialChallenger
                })
            )
        });
        dgConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON_KONA,
            gameArgs: bytes("")
        });
        dgConfigs[3] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.SUPER_CANNON,
            gameArgs: bytes("")
        });
        dgConfigs[4] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            gameArgs: bytes("")
        });
        dgConfigs[5] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.SUPER_CANNON_KONA,
            gameArgs: bytes("")
        });

        IOPContractsManagerV2.FullConfig memory deployConfig = IOPContractsManagerV2.FullConfig({
            saltMixer: string(abi.encodePacked("migrate-test-", _l2ChainId)),
            superchainConfig: superchainConfig,
            proxyAdminOwner: opcmV2.opcmStandardValidator().l1PAOMultisig(),
            systemConfigOwner: makeAddr("migrateSystemConfigOwner"),
            unsafeBlockSigner: makeAddr("migrateUnsafeBlockSigner"),
            batcher: makeAddr("migrateBatcher"),
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(hex"1234")), l2SequenceNumber: 123 }),
            startingRespectedGameType: GameTypes.PERMISSIONED_CANNON,
            basefeeScalar: 1368,
            blobBasefeeScalar: 801949,
            gasLimit: 60_000_000,
            l2ChainId: _l2ChainId,
            resourceConfig: IResourceMetering.ResourceConfig({
                maxResourceLimit: 20_000_000,
                elasticityMultiplier: 10,
                baseFeeMaxChangeDenominator: 8,
                minimumBaseFee: 1 gwei,
                systemTxMaxGas: 1_000_000,
                maximumBaseFee: type(uint128).max
            }),
            disputeGameConfigs: dgConfigs,
            useCustomGasToken: false
        });

        cts_ = opcmV2.deploy(deployConfig);
    }

    /// @notice Creates the default migration input with both SPDG and SCKDG.
    /// @return input_ The default migration input.
    function _getDefaultMigrateInput() internal view returns (IOPContractsManagerMigrator.MigrateInput memory input_) {
        ISystemConfig[] memory chainSystemConfigs = new ISystemConfig[](2);
        chainSystemConfigs[0] = chainContracts1.systemConfig;
        chainSystemConfigs[1] = chainContracts2.systemConfig;

        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](2);
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: cannonPrestate,
                    proposer: proposer,
                    challenger: challenger
                })
            )
        });
        disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.SUPER_CANNON_KONA,
            gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate }))
        });

        input_ = IOPContractsManagerMigrator.MigrateInput({
            chainSystemConfigs: chainSystemConfigs,
            disputeGameConfigs: disputeGameConfigs,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(hex"ABBA")), l2SequenceNumber: 1234 }),
            startingRespectedGameType: GameTypes.SUPER_PERMISSIONED_CANNON
        });
    }

    /// @notice Executes a migration via delegatecall to opcmV2.
    /// @param _input The migration input.
    function _doMigration(IOPContractsManagerMigrator.MigrateInput memory _input) internal {
        address proxyAdminOwner = chainContracts1.proxyAdmin.owner();
        prankDelegateCall(proxyAdminOwner);
        (bool success,) = address(opcmV2).delegatecall(abi.encodeCall(IOPContractsManagerV2.migrate, (_input)));
        assertTrue(success, "migrate failed");
    }

    /// @notice Builds SharedImplementations from the StandardValidator's state.
    function _buildRefs() internal view returns (IOPContractsManagerMigrationValidator.SharedImplementations memory) {
        return IOPContractsManagerMigrationValidator.SharedImplementations({
            disputeGameFactoryImpl: standardValidator.disputeGameFactoryImpl(),
            anchorStateRegistryImpl: standardValidator.anchorStateRegistryImpl(),
            ethLockboxImpl: standardValidator.ethLockboxImpl(),
            delayedWETHImpl: standardValidator.delayedWETHImpl(),
            mipsImpl: standardValidator.mipsImpl(),
            l1PAOMultisig: standardValidator.l1PAOMultisig(),
            withdrawalDelaySeconds: standardValidator.withdrawalDelaySeconds()
        });
    }

    /// @notice Builds the MigrationValidationInput and calls validateMigration with 2 chains.
    function _validateMigration(bool _allowFailure) internal view returns (string memory) {
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        return _validateMigrationCustomChains(chains, _allowFailure);
    }

    /// @notice Builds MigrationValidationInput with custom chain list.
    function _validateMigrationCustomChains(
        ISystemConfig[] memory _chains,
        bool _allowFailure
    )
        internal
        view
        returns (string memory)
    {
        return migrationValidator.validateMigration(
            IOPContractsManagerMigrationValidator.MigrationValidationInput({
                dgf: sharedDGF,
                chainSystemConfigs: _chains,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                proposer: proposer,
                challenger: challenger
            }),
            _allowFailure,
            _buildRefs()
        );
    }

    /// @notice Returns the game impl address for a given game type on the shared DGF.
    function _gameImpl(GameType _gameType) internal view returns (address) {
        return address(sharedDGF.gameImpls(_gameType));
    }

    /// @notice Returns the ASR address from the SPDG game args.
    function _spdgASR() internal view returns (address) {
        bytes memory args = sharedDGF.gameArgs(GameTypes.SUPER_PERMISSIONED_CANNON);
        return LibGameArgs.decode(args).anchorStateRegistry;
    }
}

/// @title OPContractsManagerMigrationValidator_ValidateMigration_Test
/// @notice Tests that full migration validation passes with correct setup.
contract OPContractsManagerMigrationValidator_ValidateMigration_Test is
    OPContractsManagerMigrationValidator_TestInit
{
    /// @notice Tests that validateMigration succeeds with no errors.
    function test_validateMigration_succeeds() public view {
        string memory errors = _validateMigration(false);
        assertEq(errors, "");
    }

    /// @notice Tests that validateMigration with allowFailure=true also returns empty.
    function test_validateMigration_allowFailureTrue_succeeds() public view {
        string memory errors = _validateMigration(true);
        assertEq(errors, "");
    }
}

/// @title OPContractsManagerMigrationValidator_DGFShape_Test
/// @notice Negative tests for MIG-DGF-10 through MIG-DGF-50.
contract OPContractsManagerMigrationValidator_DGFShape_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-DGF-10: SUPER_PERMISSIONED_CANNON not registered on shared DGF.
    function test_validate_dgf10SuperPermCannonMissing_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        // SPDG validation skipped (impl is 0), per-chain skipped (can't derive shared ASR).
        assertEq("MIG-DGF-10", _validateMigration(true));
    }

    /// @notice MIG-DGF-20: SUPER_CANNON_KONA not registered on shared DGF.
    function test_validate_dgf20SuperCannonKonaMissing_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0))
        );
        // SCKDG validation skipped (impl is 0).
        assertEq("MIG-DGF-20", _validateMigration(true));
    }

    /// @notice MIG-DGF-30: Legacy CANNON still registered on shared DGF.
    function test_validate_dgf30CannonStillRegistered_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-DGF-30", _validateMigration(true));
    }

    /// @notice MIG-DGF-40: Legacy PERMISSIONED_CANNON still registered on shared DGF.
    function test_validate_dgf40PermCannonStillRegistered_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-DGF-40", _validateMigration(true));
    }

    /// @notice MIG-DGF-50: Legacy CANNON_KONA still registered on shared DGF.
    function test_validate_dgf50CannonKonaStillRegistered_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON_KONA)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-DGF-50", _validateMigration(true));
    }

}

/// @title OPContractsManagerMigrationValidator_SPDG_Test
/// @notice Negative tests for MIG-SPDG-* error codes (Super Permissioned Dispute Game).
contract OPContractsManagerMigrationValidator_SPDG_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SPDG-GARGS-10: Invalid game args length for SPDG.
    function test_validate_spdgGargs10InvalidArgsLength_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameArgs, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(hex"deadbeef")
        );
        // Per-chain also can't decode SPDG args, skips per-chain checks.
        assertEq("MIG-SPDG-GARGS-10", _validateMigration(true));
    }

    /// @notice MIG-SPDG-10: l2ChainId != 0 in SPDG game args.
    function test_validate_spdg10WrongL2ChainId_succeeds() public {
        DisputeGames.mockGameImplL2ChainId(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, 42);
        assertEq("MIG-SPDG-10", _validateMigration(true));
    }

    /// @notice MIG-SPDG-20: Wrong absolutePrestate in SPDG game args.
    function test_validate_spdg20WrongPrestate_succeeds() public {
        DisputeGames.mockGameImplPrestate(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, bytes32(uint256(0xbad)));
        assertEq("MIG-SPDG-20", _validateMigration(true));
    }

    /// @notice MIG-SPDG-GARGS-20: Wrong VM in SPDG game args.
    function test_validate_spdgGargs20WrongVM_succeeds() public {
        DisputeGames.mockGameImplVM(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, address(0xbad));
        assertEq("MIG-SPDG-GARGS-20", _validateMigration(true));
    }

    /// @notice MIG-SPDG-GARGS-30: Wrong WETH in SPDG game args (doesn't match discovered WETH).
    function test_validate_spdgGargs30WrongWeth_succeeds() public {
        DisputeGames.mockGameImplWeth(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, address(0xbad));
        assertEq("MIG-SPDG-GARGS-30", _validateMigration(true));
    }

    /// @notice MIG-SPDG-30: Wrong maxGameDepth on SPDG game impl.
    function test_validate_spdg30WrongMaxGameDepth_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_PERMISSIONED_CANNON),
            abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()),
            abi.encode(uint256(99))
        );
        assertEq("MIG-SPDG-30", _validateMigration(true));
    }

    /// @notice MIG-SPDG-40: Wrong splitDepth on SPDG game impl.
    function test_validate_spdg40WrongSplitDepth_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_PERMISSIONED_CANNON),
            abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()),
            abi.encode(uint256(99))
        );
        assertEq("MIG-SPDG-40", _validateMigration(true));
    }

    /// @notice MIG-SPDG-50: Wrong clockExtension on SPDG game impl.
    function test_validate_spdg50WrongClockExtension_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_PERMISSIONED_CANNON),
            abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()),
            abi.encode(uint64(99))
        );
        assertEq("MIG-SPDG-50", _validateMigration(true));
    }

    /// @notice MIG-SPDG-60: Wrong maxClockDuration on SPDG game impl.
    function test_validate_spdg60WrongMaxClockDuration_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_PERMISSIONED_CANNON),
            abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()),
            abi.encode(uint64(99))
        );
        assertEq("MIG-SPDG-60", _validateMigration(true));
    }

    /// @notice MIG-SPDG-70: l2SequenceNumber != 0 on SPDG game impl.
    function test_validate_spdg70WrongL2SequenceNumber_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_PERMISSIONED_CANNON),
            abi.encodeCall(IDisputeGame.l2SequenceNumber, ()),
            abi.encode(uint256(1))
        );
        assertEq("MIG-SPDG-70", _validateMigration(true));
    }

    /// @notice MIG-SPDG-80: Anchor root is zero from SPDG ASR.
    ///         Also triggers MIG-SCKDG-80 if SPDG and SCKDG share the same ASR.
    function test_validate_spdg80Sckdg80ZeroAnchorRoot_succeeds() public {
        vm.mockCall(
            _spdgASR(), abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()), abi.encode(bytes32(0), uint256(0))
        );
        assertEq("MIG-SPDG-80,MIG-SCKDG-80", _validateMigration(true));
    }

    /// @notice MIG-SPDG-90: Wrong proposer in SPDG game args.
    function test_validate_spdg90WrongProposer_succeeds() public {
        DisputeGames.mockGameImplProposer(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, address(0xbad));
        assertEq("MIG-SPDG-90", _validateMigration(true));
    }

    /// @notice MIG-SPDG-100: Wrong challenger in SPDG game args.
    function test_validate_spdg100WrongChallenger_succeeds() public {
        DisputeGames.mockGameImplChallenger(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, address(0xbad));
        assertEq("MIG-SPDG-100", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SCKDG_Test
/// @notice Negative tests for MIG-SCKDG-* error codes (Super Cannon Kona Dispute Game).
contract OPContractsManagerMigrationValidator_SCKDG_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SCKDG-GARGS-10: Invalid game args length for SCKDG.
    function test_validate_sckdgGargs10InvalidArgsLength_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameArgs, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(hex"deadbeef")
        );
        assertEq("MIG-SCKDG-GARGS-10", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-10: l2ChainId != 0 in SCKDG game args.
    function test_validate_sckdg10WrongL2ChainId_succeeds() public {
        DisputeGames.mockGameImplL2ChainId(sharedDGF, GameTypes.SUPER_CANNON_KONA, 42);
        assertEq("MIG-SCKDG-10", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-20: Wrong absolutePrestate in SCKDG game args.
    function test_validate_sckdg20WrongPrestate_succeeds() public {
        DisputeGames.mockGameImplPrestate(sharedDGF, GameTypes.SUPER_CANNON_KONA, bytes32(uint256(0xbad)));
        assertEq("MIG-SCKDG-20", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-GARGS-20: Wrong VM in SCKDG game args.
    function test_validate_sckdgGargs20WrongVM_succeeds() public {
        DisputeGames.mockGameImplVM(sharedDGF, GameTypes.SUPER_CANNON_KONA, address(0xbad));
        assertEq("MIG-SCKDG-GARGS-20", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-GARGS-30: Wrong WETH in SCKDG game args.
    function test_validate_sckdgGargs30WrongWeth_succeeds() public {
        DisputeGames.mockGameImplWeth(sharedDGF, GameTypes.SUPER_CANNON_KONA, address(0xbad));
        assertEq("MIG-SCKDG-GARGS-30", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-30: Wrong maxGameDepth on SCKDG game impl.
    function test_validate_sckdg30WrongMaxGameDepth_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()),
            abi.encode(uint256(99))
        );
        assertEq("MIG-SCKDG-30", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-40: Wrong splitDepth on SCKDG game impl.
    function test_validate_sckdg40WrongSplitDepth_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()),
            abi.encode(uint256(99))
        );
        assertEq("MIG-SCKDG-40", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-50: Wrong clockExtension on SCKDG game impl.
    function test_validate_sckdg50WrongClockExtension_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()),
            abi.encode(uint64(99))
        );
        assertEq("MIG-SCKDG-50", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-60: Wrong maxClockDuration on SCKDG game impl.
    function test_validate_sckdg60WrongMaxClockDuration_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()),
            abi.encode(uint64(99))
        );
        assertEq("MIG-SCKDG-60", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-70: l2SequenceNumber != 0 on SCKDG game impl.
    function test_validate_sckdg70WrongL2SequenceNumber_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IDisputeGame.l2SequenceNumber, ()),
            abi.encode(uint256(1))
        );
        assertEq("MIG-SCKDG-70", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_PerChain_Test
/// @notice Negative tests for MIG-CHAIN-* and MIG-LOCKBOX-MISSING error codes.
contract OPContractsManagerMigrationValidator_PerChain_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-CHAIN-EMPTY: Empty chainSystemConfigs array.
    function test_validate_chainEmpty_succeeds() public view {
        ISystemConfig[] memory chains = new ISystemConfig[](0);
        assertEq("MIG-CHAIN-EMPTY", _validateMigrationCustomChains(chains, true));
    }

    // NOTE: MIG-CHAIN-0-10 is tautological — shared ASR is discovered from portal[0],
    // so portal[0].anchorStateRegistry() always equals itself. Same pattern as MIG-SDGF-40.

    /// @notice MIG-CHAIN-1-10: Second chain's portal ASR does not match shared ASR.
    function test_validate_chain110PortalAsrMismatch_succeeds() public {
        vm.mockCall(
            address(chainContracts2.optimismPortal),
            abi.encodeCall(IOptimismPortal2.anchorStateRegistry, ()),
            abi.encode(address(0xbadA5B))
        );
        assertEq("MIG-CHAIN-1-10", _validateMigration(true));
    }

    // NOTE: Per-chain DGF tests (-20 through -70) are not applicable here.
    // After migration, systemConfig.disputeGameFactory() resolves through
    // portal → shared ASR → shared DGF. The per-chain DGF IS the shared DGF,
    // so the validator skips per-chain DGF clearing checks. The shared DGF's
    // game type registration is validated by the DGF shape tests (MIG-DGF-*).

    /// @notice MIG-CHAIN-0-80: Portal not authorized in shared lockbox.
    function test_validate_chain080PortalNotAuthorized_succeeds() public {
        vm.mockCall(
            address(sharedLockbox),
            abi.encodeCall(IETHLockbox.authorizedPortals, (chainContracts1.optimismPortal)),
            abi.encode(false)
        );
        assertEq("MIG-CHAIN-0-80", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-1-90: Second chain's portal lockbox doesn't match shared lockbox.
    function test_validate_chain190PortalLockboxMismatch_succeeds() public {
        vm.mockCall(
            address(chainContracts2.optimismPortal),
            abi.encodeCall(IOptimismPortal2.ethLockbox, ()),
            abi.encode(address(0xbadB0C))
        );
        assertEq("MIG-CHAIN-1-90", _validateMigration(true));
    }

    /// @notice MIG-LOCKBOX-MISSING: Portal's ethLockbox is address(0).
    function test_validate_lockboxMissing_succeeds() public {
        vm.mockCall(
            address(chainContracts1.optimismPortal),
            abi.encodeCall(IOptimismPortal2.ethLockbox, ()),
            abi.encode(address(0))
        );
        assertEq("MIG-LOCKBOX-MISSING", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-0-100: INTEROP feature not enabled.
    function test_validate_chain0100InteropNotEnabled_succeeds() public {
        vm.mockCall(
            address(chainContracts1.systemConfig),
            abi.encodeCall(ISystemConfig.isFeatureEnabled, (Features.INTEROP)),
            abi.encode(false)
        );
        assertEq("MIG-CHAIN-0-100", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-0-110: ETH_LOCKBOX feature not enabled.
    function test_validate_chain0110EthLockboxNotEnabled_succeeds() public {
        vm.mockCall(
            address(chainContracts1.systemConfig),
            abi.encodeCall(ISystemConfig.isFeatureEnabled, (Features.ETH_LOCKBOX)),
            abi.encode(false)
        );
        assertEq("MIG-CHAIN-0-110", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SharedDGF_Test
/// @notice Negative tests for MIG-SDGF-10 through MIG-SDGF-40.
contract OPContractsManagerMigrationValidator_SharedDGF_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SDGF-10: DGF proxy version doesn't match impl version.
    function test_validate_sdgf10WrongVersion_succeeds() public {
        vm.mockCall(address(sharedDGF), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SDGF-10", _validateMigration(true));
    }

    /// @notice MIG-SDGF-20: DGF proxy implementation doesn't match expected.
    function test_validate_sdgf20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(sharedDGF))),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SDGF-20", _validateMigration(true));
    }

    /// @notice MIG-SDGF-30: DGF owner is not l1PAOMultisig.
    function test_validate_sdgf30WrongOwner_succeeds() public {
        vm.mockCall(address(sharedDGF), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SDGF-30", _validateMigration(true));
    }

    // Note: MIG-SDGF-40 is tautological with mock approach (discovery reads proxyAdmin from DGF,
    // then the check compares DGF's proxyAdmin to the discovered one — same mock returns same value).
    // This check catches real misconfigurations where the proxyAdmin() response changes between calls.
}

/// @title OPContractsManagerMigrationValidator_SharedASR_Test
/// @notice Negative tests for MIG-SASR-10 through MIG-SASR-80.
contract OPContractsManagerMigrationValidator_SharedASR_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SASR-10: ASR version doesn't match impl version.
    function test_validate_sasr10WrongVersion_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SASR-10", _validateMigration(true));
    }

    /// @notice MIG-SASR-20: ASR proxy implementation doesn't match expected.
    function test_validate_sasr20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (sharedASR)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SASR-20", _validateMigration(true));
    }

    /// @notice MIG-SASR-30: ASR disputeGameFactory doesn't match shared DGF.
    function test_validate_sasr30WrongDGF_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(address(0xbad)));
        // Mocking sharedASR.disputeGameFactory() cascades through portal → ASR → DGF.
        // Break the cascade so systemConfig.disputeGameFactory() still returns sharedDGF.
        vm.mockCall(
            address(chainContracts1.optimismPortal),
            abi.encodeCall(IOptimismPortal2.disputeGameFactory, ()),
            abi.encode(address(sharedDGF))
        );
        vm.mockCall(
            address(chainContracts2.optimismPortal),
            abi.encodeCall(IOptimismPortal2.disputeGameFactory, ()),
            abi.encode(address(sharedDGF))
        );
        assertEq("MIG-SASR-30", _validateMigration(true));
    }

    /// @notice MIG-SASR-40: ASR proxyAdmin doesn't match shared ProxyAdmin.
    function test_validate_sasr40WrongProxyAdmin_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SASR-40", _validateMigration(true));
    }

    /// @notice MIG-SASR-50: ASR retirementTimestamp is zero.
    function test_validate_sasr50ZeroRetirementTimestamp_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(uint64(0)));
        assertEq("MIG-SASR-50", _validateMigration(true));
    }

    /// @notice MIG-SASR-60: ASR respectedGameType is not a super game type.
    function test_validate_sasr60NotSuperGameType_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IAnchorStateRegistry.respectedGameType, ()), abi.encode(GameTypes.CANNON));
        assertEq("MIG-SASR-60", _validateMigration(true));
    }

    /// @notice MIG-SASR-70: SCKDG game args ASR doesn't match discovered ASR.
    function test_validate_sasr70SckdgAsrMismatch_succeeds() public {
        address badASR = address(0xbadA5B);
        DisputeGames.mockGameImplASR(sharedDGF, GameTypes.SUPER_CANNON_KONA, badASR);
        // Mock getAnchorRoot on the bad ASR so the super game check doesn't revert.
        vm.mockCall(
            badASR, abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()), abi.encode(bytes32(uint256(1)), uint256(0))
        );
        assertEq("MIG-SASR-70", _validateMigration(true));
    }

    /// @notice MIG-SASR-80: SPDG game args ASR doesn't match discovered ASR.
    function test_validate_sasr80SpdgAsrMismatch_succeeds() public {
        address badASR = address(0xbadA5B);
        DisputeGames.mockGameImplASR(sharedDGF, GameTypes.SUPER_PERMISSIONED_CANNON, badASR);
        // Mock getAnchorRoot on the bad ASR so the super game check doesn't revert.
        vm.mockCall(
            badASR, abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()), abi.encode(bytes32(uint256(1)), uint256(0))
        );
        assertEq("MIG-SASR-80", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SharedLockbox_Test
/// @notice Negative tests for MIG-SLOCKBOX-10 through MIG-SLOCKBOX-30.
contract OPContractsManagerMigrationValidator_SharedLockbox_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SLOCKBOX-10: Lockbox version doesn't match impl version.
    function test_validate_slockbox10WrongVersion_succeeds() public {
        vm.mockCall(address(sharedLockbox), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SLOCKBOX-10", _validateMigration(true));
    }

    /// @notice MIG-SLOCKBOX-20: Lockbox proxy implementation doesn't match expected.
    function test_validate_slockbox20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(sharedLockbox))),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SLOCKBOX-20", _validateMigration(true));
    }

    /// @notice MIG-SLOCKBOX-30: Lockbox proxyAdmin doesn't match shared ProxyAdmin.
    function test_validate_slockbox30WrongProxyAdmin_succeeds() public {
        vm.mockCall(
            address(sharedLockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("MIG-SLOCKBOX-30", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SharedDelayedWETH_Test
/// @notice Negative tests for MIG-SDWETH-10 through MIG-SDWETH-30.
contract OPContractsManagerMigrationValidator_SharedDelayedWETH_Test is
    OPContractsManagerMigrationValidator_TestInit
{
    /// @notice MIG-SDWETH-10: DelayedWETH version doesn't match impl version.
    function test_validate_sdweth10WrongVersion_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SDWETH-10", _validateMigration(true));
    }

    /// @notice MIG-SDWETH-20: DelayedWETH delay doesn't match expected withdrawalDelaySeconds.
    function test_validate_sdweth20WrongDelay_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(uint256(999)));
        assertEq("MIG-SDWETH-20", _validateMigration(true));
    }

    /// @notice MIG-SDWETH-30: DelayedWETH proxyAdminOwner doesn't match l1PAOMultisig.
    function test_validate_sdweth30WrongProxyAdminOwner_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SDWETH-30", _validateMigration(true));
    }

    /// @notice MIG-SDWETH-40: DelayedWETH proxy implementation doesn't match expected.
    function test_validate_sdweth40WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (sharedWETH)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SDWETH-40", _validateMigration(true));
    }

    /// @notice MIG-SDWETH-50: DelayedWETH proxyAdmin doesn't match shared ProxyAdmin.
    function test_validate_sdweth50WrongProxyAdmin_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SDWETH-50", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_AllowFailure_Test
/// @notice Tests allowFailure behavior: revert on false, return errors on true.
contract OPContractsManagerMigrationValidator_AllowFailure_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice allowFailure=false reverts with prefixed error string.
    function test_validate_allowFailureFalse_reverts() public {
        // Pre-build input and refs before vm.expectRevert (they make external calls).
        IOPContractsManagerMigrationValidator.SharedImplementations memory refs = _buildRefs();
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        IOPContractsManagerMigrationValidator.MigrationValidationInput memory input =
        IOPContractsManagerMigrationValidator.MigrationValidationInput({
            dgf: sharedDGF,
            chainSystemConfigs: chains,
            cannonPrestate: cannonPrestate.raw(),
            cannonKonaPrestate: cannonKonaPrestate.raw(),
            proposer: proposer,
            challenger: challenger
        });
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0xbad))
        );
        vm.expectRevert(bytes("OPContractsManagerMigrationValidator: MIG-DGF-30"));
        migrationValidator.validateMigration(input, false, refs);
    }

    /// @notice allowFailure=true returns error string without revert.
    function test_validate_allowFailureTrue_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-DGF-30", _validateMigration(true));
    }

    /// @notice allowFailure=false with multiple errors reverts with all errors.
    function test_validate_allowFailureFalseMultipleErrors_reverts() public {
        // Pre-build input and refs before vm.expectRevert (they make external calls).
        IOPContractsManagerMigrationValidator.SharedImplementations memory refs = _buildRefs();
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        IOPContractsManagerMigrationValidator.MigrationValidationInput memory input =
        IOPContractsManagerMigrationValidator.MigrationValidationInput({
            dgf: sharedDGF,
            chainSystemConfigs: chains,
            cannonPrestate: cannonPrestate.raw(),
            cannonKonaPrestate: cannonKonaPrestate.raw(),
            proposer: proposer,
            challenger: challenger
        });
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0xbad))
        );
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0xbad))
        );
        vm.expectRevert(bytes("OPContractsManagerMigrationValidator: MIG-DGF-30,MIG-DGF-40"));
        migrationValidator.validateMigration(input, false, refs);
    }
}
