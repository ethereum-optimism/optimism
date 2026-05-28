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
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IOPContractsManagerMigrationValidator } from "interfaces/L1/opcm/IOPContractsManagerMigrationValidator.sol";
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
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

/// @title BadSemver
/// @notice Minimal version stub used to make an expected implementation version mismatch.
contract BadSemver {
    function version() external pure returns (string memory) {
        return "0.0.0-bad";
    }
}

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

    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Deploy two chains via OPCMv2 for migration testing.
        chainContracts1 = _deployChainForMigration(1000001);
        chainContracts2 = _deployChainForMigration(1000002);

        // Get validators from OPCM.
        standardValidator = opcmV2.opcmStandardValidator();
        migrationValidator = standardValidator.migrationValidator();

        // Set proposer before building migration input.
        proposer = makeAddr("superProposer");

        // Run real migration with both SPDG and SCKDG.
        _doMigration(_getDefaultMigrateInput());

        // Discover shared infra from real post-migration state.
        IOptimismPortal2 portal1 = chainContracts1.optimismPortal;
        IAnchorStateRegistry asr = portal1.anchorStateRegistry();
        sharedASR = address(asr);
        sharedDGF = IDisputeGameFactory(asr.disputeGameFactory());
        sharedProxyAdmin = address(IProxyAdminOwnedBase(address(sharedDGF)).proxyAdmin());
        sharedLockbox = IETHLockbox(portal1.ethLockbox());

        // Discover WETH from the permissionless super game args. The simplified
        // SUPER_PERMISSIONED_CANNON no longer carries WETH in its game args.
        LibGameArgs.GameArgs memory args = LibGameArgs.decode(sharedDGF.gameArgs(GameTypes.SUPER_CANNON_KONA));
        sharedWETH = args.weth;
    }

    /// @notice Deploys a chain via opcmV2.deploy() for subsequent migration.
    /// @param _l2ChainId The L2 chain ID for the deployed chain.
    /// @return cts_ The deployed chain contracts.
    function _deployChainForMigration(uint256 _l2ChainId)
        internal
        returns (IOPContractsManagerV2.ChainContracts memory cts_)
    {
        // Get initial proposer/challenger from existing DGF.
        bool superRoot = isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        address initialChallenger = _initialPermissionedGameChallenger(superRoot);
        address initialProposer = _initialPermissionedGameProposer(superRoot);

        IOPContractsManagerUtils.DisputeGameConfig[] memory dgConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](6);
        dgConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON,
            gameArgs: bytes("")
        });
        dgConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: !superRoot,
            initBond: superRoot ? 0 : 0.08 ether,
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: superRoot
                ? bytes("")
                : abi.encode(
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
            enabled: superRoot,
            initBond: 0,
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            gameArgs: superRoot
                ? abi.encode(IOPContractsManagerUtils.SuperPermissionedDisputeGameConfig({ proposer: initialProposer }))
                : bytes("")
        });
        dgConfigs[4] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.SUPER_CANNON_KONA,
            gameArgs: bytes("")
        });
        dgConfigs[5] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.ZK_DISPUTE_GAME,
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
            startingRespectedGameType: superRoot ? GameTypes.SUPER_PERMISSIONED_CANNON : GameTypes.PERMISSIONED_CANNON,
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

    function _initialPermissionedGameChallenger(bool _superRoot) internal view returns (address challenger_) {
        if (!_superRoot) return DisputeGames.permissionedGameChallenger(disputeGameFactory);

        challenger_ = address(0);
    }

    function _initialPermissionedGameProposer(bool _superRoot) internal view returns (address proposer_) {
        if (!_superRoot) return DisputeGames.permissionedGameProposer(disputeGameFactory);

        proposer_ = DisputeGames.superPermissionedGameProposer(disputeGameFactory);
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
            initBond: 0,
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            gameArgs: abi.encode(IOPContractsManagerUtils.SuperPermissionedDisputeGameConfig({ proposer: proposer }))
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
    function _buildImpls() internal view returns (IOPContractsManagerMigrationValidator.SharedImplementations memory) {
        return IOPContractsManagerMigrationValidator.SharedImplementations({
            disputeGameFactoryImpl: standardValidator.disputeGameFactoryImpl(),
            anchorStateRegistryImpl: standardValidator.anchorStateRegistryImpl(),
            ethLockboxImpl: standardValidator.ethLockboxImpl(),
            delayedWETHImpl: standardValidator.delayedWETHImpl(),
            mipsImpl: standardValidator.mipsImpl(),
            superFaultDisputeGameImpl: standardValidator.superFaultDisputeGameImpl(),
            superPermissionedDisputeGameImpl: standardValidator.superPermissionedDisputeGameImpl(),
            standardValidatorUtils: standardValidator.standardValidatorUtils()
        });
    }

    /// @notice Builds SharedConfig from the StandardValidator's state.
    function _buildCfg() internal view returns (IOPContractsManagerMigrationValidator.SharedConfig memory) {
        return IOPContractsManagerMigrationValidator.SharedConfig({
            l1PAOMultisig: standardValidator.l1PAOMultisig(),
            challenger: standardValidator.challenger(),
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
                proposer: proposer
            }),
            _allowFailure,
            _buildImpls(),
            _buildCfg()
        );
    }

    /// @notice Returns the game impl address for a given game type on the shared DGF.
    function _gameImpl(GameType _gameType) internal view returns (address) {
        return address(sharedDGF.gameImpls(_gameType));
    }

    /// @notice Returns the ASR address from the SPDG game args.
    function _spdgASR() internal view returns (address) {
        return DisputeGames.superPermissionedGameAnchorStateRegistry(sharedDGF);
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
/// @notice Negative tests for MIG-DGF-10 through MIG-DGF-60.
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

    /// @notice MIG-DGF-60: Legacy SUPER_CANNON still registered on shared DGF.
    function test_validate_dgf60SuperCannonStillRegistered_succeeds() public {
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-DGF-60", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SPDG_Test
/// @notice Negative tests for MIG-SPDG-* error codes (Super Permissioned Dispute Game).
contract OPContractsManagerMigrationValidator_SPDG_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SPDG-20: SPDG implementation version doesn't match expected.
    function test_validate_spdg20WrongVersion_succeeds() public {
        BadSemver bad = new BadSemver();
        vm.mockCall(
            address(standardValidator),
            abi.encodeCall(IOPContractsManagerStandardValidator.superPermissionedDisputeGameImpl, ()),
            abi.encode(address(bad))
        );

        assertEq("MIG-SPDG-20", _validateMigration(true));
    }

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

    /// @notice MIG-SPDG-120: Anchor root is zero from SPDG ASR.
    ///         Also triggers MIG-SCKDG-120 if SPDG and SCKDG share the same ASR.
    function test_validate_spdg120Sckdg120ZeroAnchorRoot_succeeds() public {
        vm.mockCall(
            _spdgASR(), abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()), abi.encode(bytes32(0), uint256(0))
        );
        assertEq("MIG-SPDG-120,MIG-SCKDG-120", _validateMigration(true));
    }

    /// @notice MIG-SPDG-140: Wrong proposer in SPDG game args.
    function test_validate_spdg140WrongProposer_succeeds() public {
        DisputeGames.mockSuperPermissionedGameProposer(sharedDGF, address(0xbad));
        assertEq("MIG-SPDG-140", _validateMigration(true));
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

    /// @notice MIG-SCKDG-60: l2ChainId != 0 in SCKDG game args.
    function test_validate_sckdg60WrongL2ChainId_succeeds() public {
        DisputeGames.mockGameImplL2ChainId(sharedDGF, GameTypes.SUPER_CANNON_KONA, 42);
        assertEq("MIG-SCKDG-60", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-40: Wrong absolutePrestate in SCKDG game args.
    function test_validate_sckdg40WrongPrestate_succeeds() public {
        DisputeGames.mockGameImplPrestate(sharedDGF, GameTypes.SUPER_CANNON_KONA, bytes32(uint256(0xbad)));
        assertEq("MIG-SCKDG-40", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-VM-10/20: Wrong VM in SCKDG game args (drill-down via assertValidDisputeGame).
    function test_validate_sckdgVm10WrongVM_succeeds() public {
        address badVM = address(0xbad);
        DisputeGames.mockGameImplVM(sharedDGF, GameTypes.SUPER_CANNON_KONA, badVM);
        vm.mockCall(badVM, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(badVM, abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION));
        assertEq("MIG-SCKDG-VM-10,MIG-SCKDG-VM-20", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-GARGS-30: Wrong WETH in SCKDG game args.
    function test_validate_sckdgGargs30WrongWeth_succeeds() public {
        address badWeth = address(0xbad);
        DisputeGames.mockGameImplWeth(sharedDGF, GameTypes.SUPER_CANNON_KONA, badWeth);
        // Mock the bad WETH to satisfy drill-down so only GARGS-30 (the cross-chain check) fires.
        vm.mockCall(badWeth, abi.encodeCall(ISemver.version, ()), abi.encode(ISemver(sharedWETH).version()));
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (badWeth)),
            abi.encode(standardValidator.delayedWETHImpl())
        );
        vm.mockCall(
            badWeth, abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(standardValidator.withdrawalDelaySeconds())
        );
        vm.mockCall(
            badWeth,
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
            abi.encode(standardValidator.l1PAOMultisig())
        );
        vm.mockCall(badWeth, abi.encodeCall(IDelayedWETH.systemConfig, ()), abi.encode(chainContracts1.systemConfig));
        vm.mockCall(badWeth, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(sharedProxyAdmin));
        assertEq("MIG-SCKDG-GARGS-30", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-100: Wrong maxGameDepth on SCKDG game impl.
    function test_validate_sckdg100WrongMaxGameDepth_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()),
            abi.encode(uint256(99))
        );
        assertEq("MIG-SCKDG-100", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-90: Wrong splitDepth on SCKDG game impl.
    function test_validate_sckdg90WrongSplitDepth_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()),
            abi.encode(uint256(99))
        );
        assertEq("MIG-SCKDG-90", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-80: Wrong clockExtension on SCKDG game impl.
    function test_validate_sckdg80WrongClockExtension_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()),
            abi.encode(uint64(99))
        );
        assertEq("MIG-SCKDG-80", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-110: Wrong maxClockDuration on SCKDG game impl.
    function test_validate_sckdg110WrongMaxClockDuration_succeeds() public {
        vm.mockCall(
            _gameImpl(GameTypes.SUPER_CANNON_KONA),
            abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()),
            abi.encode(uint64(99))
        );
        assertEq("MIG-SCKDG-110", _validateMigration(true));
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

    // NOTE: Per-chain DGF clearing checks (-20 through -70) are exercised in
    // OPContractsManagerMigrationValidator_PerChainDGF_Test below, which mocks
    // systemConfig.disputeGameFactory() to return a DGF that differs from the shared DGF.

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

    /// @notice MIG-CHAIN-1-120: Second chain's SystemConfig delayedWETH doesn't match shared WETH.
    function test_validate_chain1120DelayedWethMismatch_succeeds() public {
        vm.mockCall(
            address(chainContracts2.systemConfig),
            abi.encodeCall(ISystemConfig.delayedWETH, ()),
            abi.encode(address(0xbadDE1a4ed))
        );
        assertEq("MIG-CHAIN-1-120", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SharedDGF_Test
/// @notice Negative tests for MIG-SDGF-10 through MIG-SDGF-40.
contract OPContractsManagerMigrationValidator_SharedDGF_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SDGF-10: DGF proxy version doesn't match impl version.
    function test_validate_sharedDgf10WrongVersion_succeeds() public {
        vm.mockCall(address(sharedDGF), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SDGF-10", _validateMigration(true));
    }

    /// @notice MIG-SDGF-20: DGF proxy implementation doesn't match expected.
    function test_validate_sharedDgf20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(sharedDGF))),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SDGF-20", _validateMigration(true));
    }

    /// @notice MIG-SDGF-30: DGF owner is not l1PAOMultisig.
    function test_validate_sharedDgf30WrongOwner_succeeds() public {
        vm.mockCall(address(sharedDGF), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SDGF-30", _validateMigration(true));
    }

    // Note: MIG-SDGF-40 is tautological with mock approach (discovery reads proxyAdmin from DGF,
    // then the check compares DGF's proxyAdmin to the discovered one — same mock returns same value).
    // This check catches real misconfigurations where the proxyAdmin() response changes between calls.
}

/// @title OPContractsManagerMigrationValidator_SharedASR_Test
/// @notice Negative tests covering shared AnchorStateRegistry invariants. The shared ASR is
///         reachable from both super game validation paths, so each broken ASR field surfaces
///         under both `MIG-SPDG-ANCHORP-*` and `MIG-SCKDG-ANCHORP-*`.
contract OPContractsManagerMigrationValidator_SharedASR_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-{SPDG,SCKDG}-ANCHORP-10: ASR version doesn't match impl version.
    function test_validate_sharedAnchorp10WrongVersion_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SPDG-ANCHORP-10,MIG-SCKDG-ANCHORP-10", _validateMigration(true));
    }

    /// @notice MIG-{SPDG,SCKDG}-ANCHORP-20: ASR proxy implementation doesn't match expected.
    function test_validate_sharedAnchorp20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (sharedASR)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SPDG-ANCHORP-20,MIG-SCKDG-ANCHORP-20", _validateMigration(true));
    }

    /// @notice MIG-{SPDG,SCKDG}-ANCHORP-30: ASR disputeGameFactory doesn't match shared DGF.
    function test_validate_sharedAnchorp30WrongDGF_succeeds() public {
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
        assertEq("MIG-SPDG-ANCHORP-30,MIG-SCKDG-ANCHORP-30", _validateMigration(true));
    }

    /// @notice MIG-{SPDG,SCKDG}-ANCHORP-50: ASR proxyAdmin doesn't match shared ProxyAdmin.
    function test_validate_sharedAnchorp50WrongProxyAdmin_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SPDG-ANCHORP-50,MIG-SCKDG-ANCHORP-50", _validateMigration(true));
    }

    /// @notice MIG-{SPDG,SCKDG}-ANCHORP-60: ASR retirementTimestamp is zero.
    function test_validate_sharedAnchorp60ZeroRetirementTimestamp_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(uint64(0)));
        assertEq("MIG-SPDG-ANCHORP-60,MIG-SCKDG-ANCHORP-60", _validateMigration(true));
    }

    /// @notice MIG-SASR-RGT: ASR respectedGameType is not a super game type.
    ///         Drill-down doesn't cover this — it's a migration-shape invariant.
    function test_validate_sharedAsrRgtNotSuperGameType_succeeds() public {
        vm.mockCall(sharedASR, abi.encodeCall(IAnchorStateRegistry.respectedGameType, ()), abi.encode(GameTypes.CANNON));
        assertEq("MIG-SASR-RGT", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SharedLockbox_Test
/// @notice Negative tests for MIG-SLOCKBOX-10 through MIG-SLOCKBOX-30.
contract OPContractsManagerMigrationValidator_SharedLockbox_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice MIG-SLOCKBOX-10: Lockbox version doesn't match impl version.
    function test_validate_sharedLockbox10WrongVersion_succeeds() public {
        vm.mockCall(address(sharedLockbox), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SLOCKBOX-10", _validateMigration(true));
    }

    /// @notice MIG-SLOCKBOX-20: Lockbox proxy implementation doesn't match expected.
    function test_validate_sharedLockbox20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(sharedLockbox))),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SLOCKBOX-20", _validateMigration(true));
    }

    /// @notice MIG-SLOCKBOX-30: Lockbox proxyAdmin doesn't match shared ProxyAdmin.
    function test_validate_sharedLockbox30WrongProxyAdmin_succeeds() public {
        vm.mockCall(
            address(sharedLockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("MIG-SLOCKBOX-30", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_SharedDelayedWETH_Test
/// @notice Negative tests covering shared DelayedWETH invariants. The simplified
///         SUPER_PERMISSIONED_CANNON no longer carries WETH, so these errors surface through
///         SUPER_CANNON_KONA only.
contract OPContractsManagerMigrationValidator_SharedDelayedWETH_Test is
    OPContractsManagerMigrationValidator_TestInit
{
    /// @notice MIG-SCKDG-DWETH-10: DelayedWETH version doesn't match impl version.
    function test_validate_sharedDweth10WrongVersion_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0-bad"));
        assertEq("MIG-SCKDG-DWETH-10", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-DWETH-40: DelayedWETH delay doesn't match expected withdrawalDelaySeconds.
    function test_validate_sharedDweth40WrongDelay_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(uint256(999)));
        assertEq("MIG-SCKDG-DWETH-40", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-DWETH-30: DelayedWETH proxyAdminOwner doesn't match l1PAOMultisig.
    function test_validate_sharedDweth30WrongProxyAdminOwner_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SCKDG-DWETH-30", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-DWETH-20: DelayedWETH proxy implementation doesn't match expected.
    function test_validate_sharedDweth20WrongImpl_succeeds() public {
        vm.mockCall(
            sharedProxyAdmin,
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (sharedWETH)),
            abi.encode(address(0xbad))
        );
        assertEq("MIG-SCKDG-DWETH-20", _validateMigration(true));
    }

    /// @notice MIG-SCKDG-DWETH-60: DelayedWETH proxyAdmin doesn't match shared ProxyAdmin.
    function test_validate_sharedDweth60WrongProxyAdmin_succeeds() public {
        vm.mockCall(sharedWETH, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad)));
        assertEq("MIG-SCKDG-DWETH-60", _validateMigration(true));
    }
}

/// @title OPContractsManagerMigrationValidator_AllowFailure_Test
/// @notice Tests allowFailure behavior: revert on false, return errors on true.
contract OPContractsManagerMigrationValidator_AllowFailure_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice allowFailure=false reverts with prefixed error string.
    function test_validate_allowFailureFalse_reverts() public {
        // Pre-build input and refs before vm.expectRevert (they make external calls).
        IOPContractsManagerMigrationValidator.SharedImplementations memory impls = _buildImpls();
        IOPContractsManagerMigrationValidator.SharedConfig memory cfg = _buildCfg();
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        IOPContractsManagerMigrationValidator.MigrationValidationInput memory input =
        IOPContractsManagerMigrationValidator.MigrationValidationInput({
            dgf: sharedDGF,
            chainSystemConfigs: chains,
            cannonPrestate: cannonPrestate.raw(),
            cannonKonaPrestate: cannonKonaPrestate.raw(),
            proposer: proposer
        });
        vm.mockCall(
            address(sharedDGF),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0xbad))
        );
        vm.expectRevert(bytes("OPContractsManagerMigrationValidator: MIG-DGF-30"));
        migrationValidator.validateMigration(input, false, impls, cfg);
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
        IOPContractsManagerMigrationValidator.SharedImplementations memory impls = _buildImpls();
        IOPContractsManagerMigrationValidator.SharedConfig memory cfg = _buildCfg();
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        IOPContractsManagerMigrationValidator.MigrationValidationInput memory input =
        IOPContractsManagerMigrationValidator.MigrationValidationInput({
            dgf: sharedDGF,
            chainSystemConfigs: chains,
            cannonPrestate: cannonPrestate.raw(),
            cannonKonaPrestate: cannonKonaPrestate.raw(),
            proposer: proposer
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
        migrationValidator.validateMigration(input, false, impls, cfg);
    }
}

/// @title OPContractsManagerMigrationValidator_PerChainDGF_Test
/// @notice Tests MIG-CHAIN-*-20 through MIG-CHAIN-*-70 by mocking systemConfig.disputeGameFactory()
///         to return a DGF that differs from the shared DGF, exercising the per-chain clearing checks.
///         The shared DGF is derived from chain 0 (first chain), so we mock chain 1 (second chain).
contract OPContractsManagerMigrationValidator_PerChainDGF_Test is OPContractsManagerMigrationValidator_TestInit {
    /// @notice Address used as a fake per-chain DGF distinct from the shared DGF.
    address fakeDGF = makeAddr("fakeDGF");

    /// @notice Mocks chain 1's systemConfig.disputeGameFactory() to return the fake DGF,
    ///         with all game types returning address(0) (cleared).
    function _mockPerChainDGF() internal {
        vm.mockCall(
            address(chainContracts2.systemConfig),
            abi.encodeCall(ISystemConfig.disputeGameFactory, ()),
            abi.encode(fakeDGF)
        );
        // Default: all game types cleared (return address(0)).
        vm.mockCall(fakeDGF, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)), abi.encode(address(0)));
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        vm.mockCall(
            fakeDGF, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON_KONA)), abi.encode(address(0))
        );
        vm.mockCall(
            fakeDGF, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)), abi.encode(address(0))
        );
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0))
        );
    }

    /// @notice MIG-CHAIN-1-20: Per-chain DGF has CANNON still registered.
    function test_validate_chain120CannonNotCleared_succeeds() public {
        _mockPerChainDGF();
        vm.mockCall(
            fakeDGF, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)), abi.encode(address(0xdead))
        );
        assertEq("MIG-CHAIN-1-20", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-1-30: Per-chain DGF has PERMISSIONED_CANNON still registered.
    function test_validate_chain130PermissionedCannonNotCleared_succeeds() public {
        _mockPerChainDGF();
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("MIG-CHAIN-1-30", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-1-40: Per-chain DGF has CANNON_KONA still registered.
    function test_validate_chain140CannonKonaNotCleared_succeeds() public {
        _mockPerChainDGF();
        vm.mockCall(
            fakeDGF, abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON_KONA)), abi.encode(address(0xdead))
        );
        assertEq("MIG-CHAIN-1-40", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-1-50: Per-chain DGF has SUPER_CANNON still registered.
    function test_validate_chain150SuperCannonNotCleared_succeeds() public {
        _mockPerChainDGF();
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("MIG-CHAIN-1-50", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-1-60: Per-chain DGF has SUPER_PERMISSIONED_CANNON still registered.
    function test_validate_chain160SuperPermCannonNotCleared_succeeds() public {
        _mockPerChainDGF();
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("MIG-CHAIN-1-60", _validateMigration(true));
    }

    /// @notice MIG-CHAIN-1-70: Per-chain DGF has SUPER_CANNON_KONA still registered.
    function test_validate_chain170SuperCannonKonaNotCleared_succeeds() public {
        _mockPerChainDGF();
        vm.mockCall(
            fakeDGF,
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0xdead))
        );
        assertEq("MIG-CHAIN-1-70", _validateMigration(true));
    }
}
