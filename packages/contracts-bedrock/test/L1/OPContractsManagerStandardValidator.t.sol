// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { SuperGameTestInit } from "test/setup/SuperGameTestInit.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";
import { DisputeGames } from "../setup/DisputeGames.sol";
import { OPContractsManagerMigrationValidator_TestInit } from "test/L1/opcm/OPContractsManagerMigrationValidator.t.sol";

// Libraries
import { GameType, Hash } from "src/dispute/lib/LibUDT.sol";
import { GameTypes, Duration, Claim } from "src/dispute/lib/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";
import { Config } from "scripts/libraries/Config.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerMigrationValidator } from "interfaces/L1/opcm/IOPContractsManagerMigrationValidator.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { IStaticERC1967Proxy } from "interfaces/universal/IStaticERC1967Proxy.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

/// @title BadVersionReturner
contract BadVersionReturner {
    /// @notice Address of the OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator public immutable validator;

    /// @notice Address of the versioned contract.
    ISemver public immutable versioned;

    /// @notice The mock semver
    string public mockVersion;

    constructor(IOPContractsManagerStandardValidator _validator, ISemver _versioned, string memory _mockVersion) {
        validator = _validator;
        versioned = _versioned;
        mockVersion = _mockVersion;
    }

    /// @notice Returns the real or fake semver
    function version() external view returns (string memory) {
        if (msg.sender == address(validator) || msg.sender == address(validator.standardValidatorUtils())) {
            return mockVersion;
        } else {
            return versioned.version();
        }
    }
}

/// @title OPContractsManagerStandardValidator_SuperMode_TestInit
/// @notice Base contract for super mode StandardValidator tests.
///         After setUp, the chain has both SUPER_PERMISSIONED and SUPER_CANNON_KONA enabled.
abstract contract OPContractsManagerStandardValidator_SuperMode_TestInit is SuperGameTestInit {
    /// @notice The l2ChainId.
    uint256 l2ChainId;

    /// @notice The cannon prestate expected by legacy Cannon validation inputs.
    Claim cannonPrestate;

    /// @notice The DisputeGameFactory instance.
    IDisputeGameFactory dgf;

    /// @notice The OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator standardValidator;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        dgf = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        standardValidator = opcmV2.opcmStandardValidator();

        l2ChainId = deploy.cfg().l2ChainID();
        cannonPrestate = Claim.wrap(bytes32(deploy.cfg().faultGameAbsolutePrestate()));
        if (isL1ForkTest()) {
            proposer = DisputeGames.permissionedGameProposer(dgf);
        } else {
            proposer = deploy.cfg().l2OutputOracleProposer();
        }

        _enableSuperCannonKona();

        _mockSuperModeForkL1PAOOwnership();
    }

    /// @notice Runs an upgrade that enables SUPER_CANNON_KONA alongside SUPER_PERMISSIONED.
    function _enableSuperCannonKona() internal override {
        address owner = proxyAdmin.owner();

        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](6);

        // Legacy types (all disabled).
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON,
            gameArgs: hex""
        });
        disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: hex""
        });
        disputeGameConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON_KONA,
            gameArgs: hex""
        });

        // Super types (enabled).
        disputeGameConfigs[3] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0,
            gameType: GameTypes.SUPER_PERMISSIONED,
            gameArgs: abi.encode(IOPContractsManagerUtils.SuperPermissionedDisputeGameConfig({ proposer: proposer }))
        });
        disputeGameConfigs[4] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
            gameType: GameTypes.SUPER_CANNON_KONA,
            gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate }))
        });
        disputeGameConfigs[5] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.ZK_DISPUTE_GAME,
            gameArgs: hex""
        });

        IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions =
            new IOPContractsManagerUtils.ExtraInstruction[](1);
        extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({
            key: "overrides.cfg.startingRespectedGameType",
            data: abi.encode(GameTypes.SUPER_PERMISSIONED)
        });

        prankDelegateCall(owner);
        (bool success,) = address(opcmV2).delegatecall(
            abi.encodeCall(
                IOPContractsManagerV2.upgrade,
                (
                    IOPContractsManagerV2.UpgradeInput({
                        systemConfig: systemConfig,
                        disputeGameConfigs: disputeGameConfigs,
                        extraInstructions: extraInstructions
                    })
                )
            )
        );
        assertTrue(success, "super mode upgrade failed");
    }

    /// @notice Normalizes fork ownership to the L1 PAO expected by the validator.
    function _mockSuperModeForkL1PAOOwnership() internal {
        if (!isL1ForkTest()) return;

        address l1PAOMultisig = standardValidator.l1PAOMultisig();
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(l1PAOMultisig));
        vm.mockCall(
            address(disputeGameFactory), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(l1PAOMultisig)
        );

        LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(dgf.gameArgs(GameTypes.SUPER_CANNON_KONA));
        vm.mockCall(gameArgs.weth, abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(l1PAOMultisig));
    }

    /// @notice Runs the OPContractsManagerStandardValidator.validate function.
    function _validate(bool _allowFailure) internal view returns (string memory) {
        return standardValidator.validate(
            IOPContractsManagerStandardValidator.ValidationInputDev({
                sysCfg: systemConfig,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                l2ChainID: l2ChainId,
                proposer: proposer
            }),
            _allowFailure
        );
    }
}

/// @title OPContractsManagerStandardValidator_SuperModeCoreValidation_Test
/// @notice Tests that full validation passes in super mode.
contract OPContractsManagerStandardValidator_SuperModeCoreValidation_Test is
    OPContractsManagerStandardValidator_SuperMode_TestInit
{
    /// @notice Tests that the validate function succeeds in super mode with all games configured.
    function test_validate_succeeds() public view {
        string memory errors = _validate(false);
        assertEq(errors, "");
    }
}

/// @title OPContractsManagerStandardValidator_SuperRootDisputeGames_Test
/// @notice Tests the renamed SUPERSHAPE error codes (now game-specific prefixes).
contract OPContractsManagerStandardValidator_SuperRootDisputeGames_Test is
    OPContractsManagerStandardValidator_SuperMode_TestInit
{
    /// @notice Tests that enabling legacy CANNON in super mode triggers PLDG-SHAPE.
    function test_validate_cannonNotDisabled_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("PLDG-SHAPE", _validate(true));
    }

    /// @notice Tests that enabling legacy PERMISSIONED_CANNON in super mode triggers PDDG-SHAPE.
    function test_validate_permissionedCannonNotDisabled_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("PDDG-SHAPE", _validate(true));
    }

    /// @notice Tests that enabling legacy CANNON_KONA in super mode triggers CKDG-SHAPE.
    function test_validate_cannonKonaNotDisabled_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON_KONA)),
            abi.encode(address(0xdead))
        );
        assertEq("CKDG-SHAPE", _validate(true));
    }

    /// @notice Tests that enabling SUPER_CANNON in super mode triggers SCDG-SHAPE.
    function test_validate_superCannonNotDisabled_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("SCDG-SHAPE", _validate(true));
    }

    /// @notice Tests that disabling SUPER_PERMISSIONED triggers SPDG-SHAPE.
    function test_validate_superPermissionedNotRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED)),
            abi.encode(address(0))
        );
        // DF-50 also fires because neither PERMISSIONED_CANNON nor SUPER_PERMISSIONED is registered.
        assertEq("DF-50,SPDG-SHAPE,SPDG-10", _validate(true));
    }

    /// @notice Tests that disabling SUPER_CANNON_KONA triggers SCKDG-SHAPE.
    function test_validate_superCannonKonaNotRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0))
        );
        assertEq("SCKDG-SHAPE,SCKDG-10", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_SuperPermissionedDisputeGame_Test
/// @notice Tests SPDG error codes for the SUPER_PERMISSIONED game validation.
contract OPContractsManagerStandardValidator_SuperPermissionedDisputeGame_Test is
    OPContractsManagerStandardValidator_SuperMode_TestInit
{
    /// @notice Tests SPDG-10 when SUPER_PERMISSIONED implementation is null.
    function test_validate_superPermissionedDisputeGameNullImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED)),
            abi.encode(address(0))
        );
        // DF-50 also fires because neither PERMISSIONED_CANNON nor SUPER_PERMISSIONED is registered.
        assertEq("DF-50,SPDG-SHAPE,SPDG-10", _validate(true));
    }

    /// @notice Tests SPDG-20 when SUPER_PERMISSIONED version is invalid.
    function test_validate_superPermissionedDisputeGameInvalidVersion_succeeds() public {
        address spdgImpl = address(disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED));
        BadVersionReturner bad = new BadVersionReturner(standardValidator, ISemver(spdgImpl), "0.0.0");
        bytes32 slot = bytes32(
            ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "superPermissionedDisputeGameImpl").slot
        );
        vm.store(address(standardValidator), slot, bytes32(uint256(uint160(address(bad)))));
        assertEq("SPDG-20", _validate(true));
    }

    /// @notice Tests SPDG-GARGS-10 when SUPER_PERMISSIONED game args are invalid.
    function test_validate_superPermissionedDisputeGameInvalidGameArgs_succeeds() public {
        vm.mockCall(
            address(dgf),
            abi.encodeCall(IDisputeGameFactory.gameArgs, (GameTypes.SUPER_PERMISSIONED)),
            abi.encode(hex"123456")
        );

        assertEq("SPDG-GARGS-10", _validate(true));
    }

    /// @notice Tests SPDG-ANCHORP-* when SUPER_PERMISSIONED's simplified ASR arg is invalid.
    function test_validate_superPermissionedDisputeGameInvalidASR_succeeds() public {
        address badASR = address(0xbad);
        DisputeGames.mockSuperPermissionedGameASR(dgf, badASR);

        vm.mockCall(badASR, abi.encodeCall(IStaticERC1967Proxy.implementation, ()), abi.encode(address(0xdeadbeef)));
        vm.mockCall(badASR, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(
            badASR,
            abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()),
            abi.encode(Hash.wrap(bytes32(uint256(0x123))), uint256(123))
        );
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(dgf));
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.systemConfig, ()), abi.encode(sysCfg));
        vm.mockCall(badASR, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(proxyAdmin));
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(uint64(100)));

        assertEq("SPDG-ANCHORP-10,SPDG-ANCHORP-20", _validate(true));
    }

    /// @notice Tests SPDG-120 when SUPER_PERMISSIONED's anchor root is zero.
    function test_validate_superPermissionedDisputeGameZeroAnchorRoot_succeeds() public {
        address spdgASR = DisputeGames.superPermissionedGameAnchorStateRegistry(dgf);
        vm.mockCall(spdgASR, abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()), abi.encode(bytes32(0), uint256(0)));

        assertEq("SPDG-120,SCKDG-120", _validate(true));
    }

    /// @notice Tests SPDG-140 when SUPER_PERMISSIONED proposer is invalid.
    function test_validate_superPermissionedDisputeGameInvalidProposer_succeeds() public {
        DisputeGames.mockSuperPermissionedGameProposer(dgf, address(0xbad));
        assertEq("SPDG-140", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_SuperPermissionlessDisputeGame_Test
/// @notice Tests SCKDG error codes for the SUPER_CANNON_KONA game validation.
contract OPContractsManagerStandardValidator_SuperPermissionlessDisputeGame_Test is
    OPContractsManagerStandardValidator_SuperMode_TestInit
{
    /// @notice Tests SCKDG-10 when SUPER_CANNON_KONA implementation is null.
    ///         Also fires SCKDG-SHAPE from the shape check in assertValidSuperRootDisputeGames.
    function test_validate_superPermissionlessDisputeGameNullImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0))
        );
        assertEq("SCKDG-SHAPE,SCKDG-10", _validate(true));
    }

    /// @notice Tests SCKDG-20 when SUPER_CANNON_KONA version is invalid.
    function test_validate_superPermissionlessDisputeGameInvalidVersion_succeeds() public {
        address sckdgImpl = address(disputeGameFactory.gameImpls(GameTypes.SUPER_CANNON_KONA));
        BadVersionReturner bad = new BadVersionReturner(standardValidator, ISemver(sckdgImpl), "0.0.0");
        bytes32 slot =
            bytes32(ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "superFaultDisputeGameImpl").slot);
        vm.store(address(standardValidator), slot, bytes32(uint256(uint160(address(bad)))));
        assertEq("SCKDG-20", _validate(true));
    }

    /// @notice Tests SCKDG-40 when SUPER_CANNON_KONA absolute prestate is invalid.
    function test_validate_superPermissionlessDisputeGameInvalidPrestate_succeeds() public {
        bytes32 badPrestate = cannonPrestate.raw(); // Use the wrong prestate
        DisputeGames.mockGameImplPrestate(dgf, GameTypes.SUPER_CANNON_KONA, badPrestate);
        assertEq("SCKDG-40", _validate(true));
    }

    /// @notice Tests SCKDG-VM-10 when SUPER_CANNON_KONA VM address is invalid.
    function test_validate_superPermissionlessDisputeGameInvalidVM_succeeds() public {
        address badVM = address(0xbad);
        DisputeGames.mockGameImplVM(dgf, GameTypes.SUPER_CANNON_KONA, badVM);
        vm.mockCall(badVM, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(badVM, abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION));
        assertEq("SCKDG-VM-10,SCKDG-VM-20", _validate(true));
    }

    /// @notice Tests SCKDG-DWETH-30 when SUPER_CANNON_KONA DelayedWETH owner is invalid.
    function test_validate_superPermissionlessDisputeGameInvalidWethOwner_succeeds() public {
        LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(dgf.gameArgs(GameTypes.SUPER_CANNON_KONA));
        vm.mockCall(gameArgs.weth, abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0xbad)));
        assertEq("SCKDG-DWETH-30", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_ZKDisputeGame_Test
/// @notice Tests that ZK dispute game validation is gated on the ZK_DISPUTE_GAME dev feature flag.
///         These tests run in non-ZK deployment mode and verify both branches of the gating logic.
contract OPContractsManagerStandardValidator_ZKDisputeGame_Test is
    OPContractsManagerStandardValidator_SuperMode_TestInit
{
    /// @notice Returns the devFeatureBitmap storage slot in standardValidator.
    function _devFeatureBitmapSlot() internal returns (bytes32) {
        return bytes32(ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "devFeatureBitmap").slot);
    }

    /// @notice Enables the ZK_DISPUTE_GAME dev feature flag in standardValidator via vm.store.
    function _enableZKFeature() internal {
        vm.store(address(standardValidator), _devFeatureBitmapSlot(), DevFeatures.ZK_DISPUTE_GAME);
    }

    /// @notice Tests ZKDG-NOSHAPE when ZK feature is not enabled but a ZK game is registered.
    ///         This is the negative test ensuring the non-ZK branch of the validation is exercised.
    function test_validate_zkDisputeGameNotExpected_succeeds() public {
        skipIfDevFeatureEnabled(DevFeatures.ZK_DISPUTE_GAME);
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.ZK_DISPUTE_GAME)),
            abi.encode(address(0xdead))
        );
        assertEq("ZKDG-NOSHAPE", _validate(true));
    }

    /// @notice Tests that ZK feature enabled + no ZK impl registered is valid.
    ///         address(0) in the factory means the chain opted out of ZK (e.g. initial deployment).
    ///         The validator should skip ZK validation entirely and report no errors.
    function test_validate_zkFeatureEnabledNoImpl_succeeds() public {
        skipIfDevFeatureEnabled(DevFeatures.ZK_DISPUTE_GAME);
        // Enable the ZK feature flag in bitmap; factory still returns address(0) for ZK_DISPUTE_GAME.
        // This is the exact state produced by an initial deployment with ZK disabled.
        _enableZKFeature();
        assertEq(address(disputeGameFactory.gameImpls(GameTypes.ZK_DISPUTE_GAME)), address(0));
        assertEq("", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_ZKMode_TestInit
/// @notice Base contract for post-super-root-migration ZK dispute game validator tests.
///         Requires ZK_DISPUTE_GAME. Configures super games plus a ZK dispute game through OPCM.
abstract contract OPContractsManagerStandardValidator_ZKMode_TestInit is CommonTest {
    /// @notice The l2ChainId from the deploy config.
    uint256 l2ChainId;

    /// @notice The cannon absolute prestate from the deploy config.
    Claim cannonPrestate;

    /// @notice The CannonKona absolute prestate.
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice The proposer role from the deploy config.
    address proposer;

    /// @notice The DisputeGameFactory instance.
    IDisputeGameFactory dgf;

    /// @notice The OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator standardValidator;

    /// @notice Sets up the ZK-mode test suite. Skips unless the ZK dev feature is enabled.
    function setUp() public virtual override {
        if (!Config.devFeatureZkDisputeGame()) {
            vm.skip(true, "Skipping: DEV_FEATURE__ZK_DISPUTE_GAME is not enabled");
        }
        super.setUp();

        dgf = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        standardValidator = opcmV2.opcmStandardValidator();

        // ZKDG-80 requires verifier.code.length > 0. Etch a dummy byte so the dummy
        // verifier address used in both fork and non-fork paths satisfies this check.
        vm.etch(address(0xBEEF), hex"01");

        if (isL1ForkTest()) {
            // Fork setup migrates the chain to super games before this fixture runs.
            GameType permissionlessGameType = DisputeGames.permissionlessGameType(dgf);
            LibGameArgs.GameArgs memory permissionlessGameArgs =
                LibGameArgs.decode(dgf.gameArgs(permissionlessGameType));
            cannonKonaPrestate = Claim.wrap(permissionlessGameArgs.absolutePrestate);
            cannonPrestate = cannonKonaPrestate;
            l2ChainId = permissionlessGameArgs.l2ChainId;
            proposer = DisputeGames.permissionedGameProposer(dgf);

            // ZK game is not deployed on mainnet. Mock it using the same ASR and WETH as the active
            // permissionless game (same on-chain infrastructure) so _assertValidZKGameArgs passes its checks.
            // ZK_DISPUTE_GAME is a super game: chain scoping comes from the SuperRootProof
            // preimage, so the 140-byte layout has no l2ChainId field.
            bytes memory zkArgs = abi.encodePacked(
                bytes32(keccak256("zkPrestate")),
                address(0xBEEF),
                uint64(7 days),
                uint64(3 days),
                uint256(0.08 ether),
                permissionlessGameArgs.anchorStateRegistry,
                permissionlessGameArgs.weth
            );
            vm.mockCall(
                address(dgf),
                abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.ZK_DISPUTE_GAME)),
                abi.encode(standardValidator.zkDisputeGameImpl())
            );
            vm.mockCall(
                address(dgf),
                abi.encodeCall(IDisputeGameFactory.gameArgs, (GameTypes.ZK_DISPUTE_GAME)),
                abi.encode(zkArgs)
            );

            address l1PAOMultisig = standardValidator.l1PAOMultisig();
            vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(l1PAOMultisig));
            vm.mockCall(address(dgf), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(l1PAOMultisig));
            vm.mockCall(
                permissionlessGameArgs.weth,
                abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
                abi.encode(l1PAOMultisig)
            );
        } else {
            l2ChainId = deploy.cfg().l2ChainID();
            cannonPrestate = cannonKonaPrestate;
            proposer = deploy.cfg().l2OutputOracleProposer();

            address owner = proxyAdmin.owner();

            // Init all game configs.
            // note: We set init bond to a non-zero value for all enabled games to avoid config validation reverts
            // in the OPCM. Other games bonds are irrelevant for the ZK specific validation tests.
            IOPContractsManagerUtils.DisputeGameConfig[] memory configs =
                new IOPContractsManagerUtils.DisputeGameConfig[](6);
            configs[0] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.CANNON,
                gameArgs: hex""
            });
            configs[1] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.PERMISSIONED_CANNON,
                gameArgs: hex""
            });
            configs[2] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.CANNON_KONA,
                gameArgs: hex""
            });
            configs[3] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: 0,
                gameType: GameTypes.SUPER_PERMISSIONED,
                gameArgs: abi.encode(IOPContractsManagerUtils.SuperPermissionedDisputeGameConfig({ proposer: proposer }))
            });
            configs[4] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.SUPER_CANNON_KONA,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate })
                )
            });
            configs[5] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.ZK_DISPUTE_GAME,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.ZKDisputeGameConfig({
                        absolutePrestate: Claim.wrap(bytes32(keccak256("zkPrestate"))),
                        verifier: IZKVerifier(address(0xBEEF)),
                        maxChallengeDuration: Duration.wrap(uint64(7 days)),
                        maxProveDuration: Duration.wrap(uint64(3 days)),
                        challengerBond: 0.08 ether
                    })
                )
            });

            IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions =
                new IOPContractsManagerUtils.ExtraInstruction[](1);
            extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({
                key: "overrides.cfg.startingRespectedGameType",
                data: abi.encode(GameTypes.SUPER_CANNON_KONA)
            });

            prankDelegateCall(owner);
            (bool success,) = address(opcmV2).delegatecall(
                abi.encodeCall(
                    IOPContractsManagerV2.upgrade,
                    (
                        IOPContractsManagerV2.UpgradeInput({
                            systemConfig: systemConfig,
                            disputeGameConfigs: configs,
                            extraInstructions: extraInstructions
                        })
                    )
                )
            );
            assertTrue(success, "ZK upgrade failed");
        }
    }

    /// @notice Runs the OPContractsManagerStandardValidator.validate function.
    function _validate(bool _allowFailure) internal view returns (string memory) {
        return standardValidator.validate(
            IOPContractsManagerStandardValidator.ValidationInputDev({
                sysCfg: systemConfig,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                l2ChainID: l2ChainId,
                proposer: proposer
            }),
            _allowFailure
        );
    }
}

/// @title OPContractsManagerStandardValidator_ZKValidation_Test
/// @notice Tests for the ZK dispute game validation path in the standard validator.
///         Only runs when ZK_DISPUTE_GAME is enabled.
contract OPContractsManagerStandardValidator_ZKValidation_Test is
    OPContractsManagerStandardValidator_ZKMode_TestInit
{
    /// @notice Tests that ZK validation succeeds after the super-root migration.
    function test_validate_zkDisputeGameAfterSuperRootMigration_succeeds() public view {
        IOptimismPortal2 portal = IOptimismPortal2(payable(systemConfig.optimismPortal()));
        assertTrue(GameTypes.isSuperGame(portal.anchorStateRegistry().respectedGameType()));
        assertEq("", _validate(false));
    }

    // Note: Tests for address(0) game implementation are skipped since this is treated as valid
    // at the validator contract level as this indicates that the chain has opted out of ZK

    /// @notice Tests ZKDG-20 when the ZK game implementation version does not match the expected.
    function test_validate_zkDisputeGameInvalidVersion_succeeds() public {
        address zkImpl = address(dgf.gameImpls(GameTypes.ZK_DISPUTE_GAME));
        BadVersionReturner bad = new BadVersionReturner(standardValidator, ISemver(zkImpl), "0.0.0");
        bytes32 slot = bytes32(ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "zkDisputeGameImpl").slot);
        vm.store(address(standardValidator), slot, bytes32(uint256(uint160(address(bad)))));
        assertEq("ZKDG-20", _validate(true));
    }

    /// @notice Tests ZKDG-70 when the absolutePrestate encoded in the ZK game args is zero.
    function test_validate_zkDisputeGameZeroAbsolutePrestate_succeeds() public {
        // absolutePrestate occupies bytes [0-31] of the packed ZK args.
        DisputeGames.mockZKGameArg(dgf, GameTypes.ZK_DISPUTE_GAME, 0, abi.encodePacked(bytes32(0)));
        assertEq("ZKDG-70", _validate(true));
    }

    /// @notice Tests ZKDG-80 when the verifier encoded in the ZK game args is the zero address.
    function test_validate_zkDisputeGameZeroVerifier_succeeds() public {
        // verifier occupies bytes [32-51] (20-byte address).
        DisputeGames.mockZKGameArg(dgf, GameTypes.ZK_DISPUTE_GAME, 32, abi.encodePacked(address(0)));
        assertEq("ZKDG-80", _validate(true));
    }

    /// @notice Tests ZKDG-90 when the maxChallengeDuration encoded in the ZK game args is zero.
    function test_validate_zkDisputeGameZeroMaxChallengeDuration_succeeds() public {
        // maxChallengeDuration occupies bytes [52-59] (uint64).
        DisputeGames.mockZKGameArg(dgf, GameTypes.ZK_DISPUTE_GAME, 52, abi.encodePacked(uint64(0)));
        assertEq("ZKDG-90", _validate(true));
    }

    /// @notice Tests ZKDG-100 when the maxProveDuration encoded in the ZK game args is zero.
    function test_validate_zkDisputeGameZeroMaxProveDuration_succeeds() public {
        // maxProveDuration occupies bytes [60-67] (uint64).
        DisputeGames.mockZKGameArg(dgf, GameTypes.ZK_DISPUTE_GAME, 60, abi.encodePacked(uint64(0)));
        assertEq("ZKDG-100", _validate(true));
    }

    /// @notice Tests ZKDG-110 when the challengerBond encoded in the ZK game args is zero.
    function test_validate_zkDisputeGameZeroChallengerBond_succeeds() public {
        // challengerBond occupies bytes [68-99] (uint256).
        DisputeGames.mockZKGameArg(dgf, GameTypes.ZK_DISPUTE_GAME, 68, abi.encodePacked(uint256(0)));
        assertEq("ZKDG-110", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_ValidateMigratedChain_Test
/// @notice Tests the validateMigratedChain entrypoint on the StandardValidator, which delegates to
///         the MigrationValidator with SharedImplementations built from the StandardValidator's state.
contract OPContractsManagerStandardValidator_ValidateMigratedChain_Test is
    OPContractsManagerMigrationValidator_TestInit
{
    /// @notice Tests that validateMigratedChain succeeds with no errors on a valid post-migration state.
    function test_validateMigratedChain_succeeds() public view {
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        string memory errors = standardValidator.validateMigratedChain(
            IOPContractsManagerMigrationValidator.MigrationValidationInput({
                dgf: sharedDGF,
                chainSystemConfigs: chains,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                proposer: proposer
            }),
            false
        );
        assertEq(errors, "");
    }

    /// @notice Helper to build migration input with 2 chains.
    function _migrationInput()
        internal
        view
        returns (IOPContractsManagerMigrationValidator.MigrationValidationInput memory)
    {
        ISystemConfig[] memory chains = new ISystemConfig[](2);
        chains[0] = chainContracts1.systemConfig;
        chains[1] = chainContracts2.systemConfig;
        return IOPContractsManagerMigrationValidator.MigrationValidationInput({
            dgf: sharedDGF,
            chainSystemConfigs: chains,
            cannonPrestate: cannonPrestate.raw(),
            cannonKonaPrestate: cannonKonaPrestate.raw(),
            proposer: proposer
        });
    }

    /// @notice Tests that validateMigratedChainWithOverrides with l1PAOMultisig override succeeds
    ///         when DGF owner is mocked to match the overridden address.
    function test_validateMigratedChainWithOverrides_l1PAOMultisigMatch_succeeds() public {
        address overrideMultisig = makeAddr("overrideMultisig");
        vm.mockCall(address(sharedDGF), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(overrideMultisig));
        // ProxyAdmin.owner() must also match, otherwise MIG-SDGF-30 still fires via SharedContracts.
        vm.mockCall(sharedProxyAdmin, abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(overrideMultisig));
        // DelayedWETH proxyAdminOwner must also match overridden l1PAOMultisig.
        vm.mockCall(sharedWETH, abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(overrideMultisig));

        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = IOPContractsManagerStandardValidator
            .ValidationOverrides({ l1PAOMultisig: overrideMultisig, challenger: address(0) });
        string memory errors = standardValidator.validateMigratedChainWithOverrides(_migrationInput(), true, overrides);
        assertEq(errors, "");
    }

    /// @notice Tests that validateMigratedChainWithOverrides with l1PAOMultisig override triggers
    ///         MIG-SDGF-30 when DGF owner does not match the overridden address.
    function test_validateMigratedChainWithOverrides_l1PAOMultisigMismatch_succeeds() public {
        // Use a different address as override — DGF owner stays as the real l1PAOMultisig,
        // so the override causes a mismatch.
        address wrongMultisig = makeAddr("wrongMultisig");
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = IOPContractsManagerStandardValidator
            .ValidationOverrides({ l1PAOMultisig: wrongMultisig, challenger: address(0) });
        string memory errors = standardValidator.validateMigratedChainWithOverrides(_migrationInput(), true, overrides);
        // l1PAOMultisig override causes DGF owner mismatch (MIG-SDGF-30) and surfaces the shared
        // DelayedWETH proxyAdminOwner mismatch through the bonded super-game drill-down.
        assertEq("MIG-SDGF-30,MIG-SCKDG-DWETH-30", errors);
    }
}
