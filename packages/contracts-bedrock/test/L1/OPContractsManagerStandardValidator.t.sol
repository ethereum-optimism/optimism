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
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";
import { Features } from "src/libraries/Features.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Config } from "scripts/libraries/Config.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerMigrationValidator } from "interfaces/L1/opcm/IOPContractsManagerMigrationValidator.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { IStaticERC1967Proxy } from "interfaces/universal/IStaticERC1967Proxy.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";
import { IStandardValidatorUtils } from "interfaces/L1/opcm/IStandardValidatorUtils.sol";

/// @title BadDisputeGameFactoryReturner
/// @notice Used to return a bad DisputeGameFactory address to the OPContractsManagerStandardValidator. Far easier
///         than the alternative ways of mocking this value since the normal vm.mockCall will cause
///         the validation function to revert.
contract BadDisputeGameFactoryReturner {
    /// @notice Address of the OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator public immutable validator;

    /// @notice Address of the IStandardValidatorUtils instance.
    IStandardValidatorUtils public immutable validatorUtils;

    /// @notice Address of the real DisputeGameFactory instance.
    IDisputeGameFactory public immutable realDisputeGameFactory;

    /// @notice Address of the fake DisputeGameFactory instance.
    IDisputeGameFactory public immutable fakeDisputeGameFactory;

    /// @param _validator The OPContractsManagerStandardValidator instance.
    /// @param _validatorUtils The IStandardValidatorUtils instance.
    /// @param _realDisputeGameFactory The real DisputeGameFactory instance.
    /// @param _fakeDisputeGameFactory The fake DisputeGameFactory instance.
    constructor(
        IOPContractsManagerStandardValidator _validator,
        IStandardValidatorUtils _validatorUtils,
        IDisputeGameFactory _realDisputeGameFactory,
        IDisputeGameFactory _fakeDisputeGameFactory
    ) {
        validator = _validator;
        validatorUtils = _validatorUtils;
        realDisputeGameFactory = _realDisputeGameFactory;
        fakeDisputeGameFactory = _fakeDisputeGameFactory;
    }

    /// @notice Returns the real or fake DisputeGameFactory address.
    function disputeGameFactory() external view returns (IDisputeGameFactory) {
        if (msg.sender == address(validator) || msg.sender == address(validatorUtils)) {
            return fakeDisputeGameFactory;
        } else {
            return realDisputeGameFactory;
        }
    }
}

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

/// @title OPContractsManagerStandardValidator_TestInit
/// @notice Base contract for `OPContractsManagerStandardValidator` tests, handles common setup.
abstract contract OPContractsManagerStandardValidator_TestInit is CommonTest {
    /// @notice The l2ChainId, either from config or from registry if fork test.
    uint256 l2ChainId;

    /// @notice The absolute prestate, either from config or dummy value if fork test.
    Claim cannonPrestate;

    /// @notice The CannonKona absolute prestate.
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice The proposer role set on the PermissionedDisputeGame instance.
    address proposer;

    /// @notice The challenger role set on the PermissionedDisputeGame instance.
    address challenger;

    /// @notice The DisputeGameFactory instance.
    IDisputeGameFactory dgf;

    /// @notice The PermissionedDisputeGame implementation.
    IPermissionedDisputeGame pdgImpl;

    /// @notice The FaultDisputeGame implementation.
    IFaultDisputeGame fdgImpl;

    /// @notice The PreimageOracle instance.
    IPreimageOracle preimageOracle;

    /// @notice The BadDisputeGameFactoryReturner instance.
    BadDisputeGameFactoryReturner badDisputeGameFactoryReturner;

    /// @notice The OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator standardValidator;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        // Standard validator tests use standard game configs incompatible with migration mode.
        if (Config.devFeatureSuperRootGamesMigration()) {
            vm.skip(true, "Skipping: standard configs incompatible with SUPER_ROOT_GAMES_MIGRATION");
        }
        // Standard validator tests do not deploy a ZK dispute game, so they are incompatible
        // with ZK_DISPUTE_GAME mode which expects one to be registered.
        if (Config.devFeatureZkDisputeGame()) {
            vm.skip(true, "Skipping: standard configs incompatible with ZK_DISPUTE_GAME");
        }
        super.setUp();

        // Load the dgf
        dgf = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));

        // Load the PermissionedDisputeGame once, we'll need it later.
        pdgImpl = IPermissionedDisputeGame(artifacts.mustGetAddress("PermissionedDisputeGame"));

        // Load the PreimageOracle once, we'll need it later.
        preimageOracle = IPreimageOracle(artifacts.mustGetAddress("PreimageOracle"));

        standardValidator = opcmV2.opcmStandardValidator();

        // Values are slightly different for fork tests vs local tests. Most we can get from
        // reasonable sources, challenger we need to get from live system because there's no other
        // consistent way to get it right now. Means we're cheating a tiny bit for the challenger
        // address in fork tests but it's fine.
        if (isL1ForkTest()) {
            l2ChainId = uint256(uint160(address(artifacts.mustGetAddress("L2ChainId"))));
            cannonPrestate = Claim.wrap(bytes32(keccak256("cannonPrestate")));
            proposer = address(123);
            challenger = address(456);

            vm.mockCall(
                address(proxyAdmin),
                abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1OptimismMintableERC20Factory))),
                abi.encode(standardValidator.optimismMintableERC20FactoryImpl())
            );

            // Mock getProxyImplementation for DelayedWETH and ETHLockbox proxies when running
            // with an unoptimized Foundry profile. See Setup.mockUnoptimizedProxyImplementations.
            mockUnoptimizedProxyImplementations(
                dgf,
                proxyAdmin,
                address(ethLockbox),
                standardValidator.delayedWETHImpl(),
                standardValidator.ethLockboxImpl()
            );

            DisputeGames.mockGameImplChallenger(
                disputeGameFactory, GameTypes.PERMISSIONED_CANNON, standardValidator.challenger()
            );
            DisputeGames.mockGameImplProposer(disputeGameFactory, GameTypes.PERMISSIONED_CANNON, proposer);
            vm.mockCall(
                address(proxyAdmin),
                abi.encodeCall(IProxyAdmin.owner, ()),
                abi.encode(standardValidator.l1PAOMultisig())
            );
            vm.mockCall(
                address(delayedWeth),
                abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
                abi.encode(standardValidator.l1PAOMultisig())
            );
            // Use vm.store so that the .setImplementation call below works.
            vm.store(
                address(disputeGameFactory),
                // this assumes that it is not packed with any other value
                bytes32(ForgeArtifacts.getSlot("DisputeGameFactory", "_owner").slot),
                bytes32(uint256(uint160(standardValidator.l1PAOMultisig())))
            );
        } else {
            l2ChainId = deploy.cfg().l2ChainID();
            cannonPrestate = Claim.wrap(bytes32(deploy.cfg().faultGameAbsolutePrestate()));
            proposer = deploy.cfg().l2OutputOracleProposer();
            challenger = deploy.cfg().l2OutputOracleChallenger();
        }

        // Deploy the BadDisputeGameFactoryReturner once.
        badDisputeGameFactoryReturner = new BadDisputeGameFactoryReturner(
            standardValidator,
            standardValidator.standardValidatorUtils(),
            disputeGameFactory,
            IDisputeGameFactory(address(0xbad))
        );

        if (isL1ForkTest()) {
            // Load the FaultDisputeGame once, we'll need it later.
            fdgImpl = IFaultDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.CANNON)));
        } else {
            // Get the ProxyAdmin owner.
            address owner = proxyAdmin.owner();

            // Prepare the upgrade input.
            // note: Init bond is set to a non-zero value and not fetched from the factory
            // as enabled games with 0 init bond are rejected by the OPCM.
            IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
                new IOPContractsManagerUtils.DisputeGameConfig[](6);
            disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.CANNON,
                gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonPrestate }))
            });
            disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.PERMISSIONED_CANNON,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                        absolutePrestate: cannonPrestate,
                        proposer: proposer,
                        challenger: challenger
                    })
                )
            });
            disputeGameConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.CANNON_KONA,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate })
                )
            });
            disputeGameConfigs[3] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
                gameArgs: hex""
            });
            disputeGameConfigs[4] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.SUPER_CANNON_KONA,
                gameArgs: hex""
            });
            disputeGameConfigs[5] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.ZK_DISPUTE_GAME,
                gameArgs: hex""
            });

            // Call upgrade to all games to be enabled.
            prankDelegateCall(owner);
            (bool success,) = address(opcmV2).delegatecall(
                abi.encodeCall(
                    IOPContractsManagerV2.upgrade,
                    (
                        IOPContractsManagerV2.UpgradeInput({
                            systemConfig: systemConfig,
                            disputeGameConfigs: disputeGameConfigs,
                            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
                        })
                    )
                )
            );
            assertTrue(success, "upgrade failed");

            // Grab the FaultDisputeGame implementation.
            fdgImpl = IFaultDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.CANNON)));
        }
    }

    /// @notice Runs the OPContractsManagerStandardValidator.validate function.
    /// @param _allowFailure Whether to allow failure.
    /// @return The error message(s) from the validate function.
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

    /// @notice Runs the OPContractsManagerStandardValidator.validateWithOverrides function.
    /// @param _allowFailure Whether to allow failure.
    /// @return The error message(s) from the validate function.
    function _validateWithOverrides(
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides
    )
        internal
        view
        returns (string memory)
    {
        return standardValidator.validateWithOverrides(
            IOPContractsManagerStandardValidator.ValidationInputDev({
                sysCfg: systemConfig,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                l2ChainID: l2ChainId,
                proposer: proposer
            }),
            _allowFailure,
            _overrides
        );
    }

    function _defaultValidationOverrides()
        internal
        pure
        returns (IOPContractsManagerStandardValidator.ValidationOverrides memory)
    {
        return IOPContractsManagerStandardValidator.ValidationOverrides({
            l1PAOMultisig: address(0),
            challenger: address(0)
        });
    }
}

/// @title OPContractsManagerStandardValidator_CoreValidation_Test
/// @notice Tests the basic functionality of the `validate` function when all parameters are valid
contract OPContractsManagerStandardValidator_CoreValidation_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function succeeds when all parameters are valid.
    function test_validate_succeeds() public view {
        string memory errors = _validate(false);
        assertEq(errors, "");
    }

    /// @notice Tests that the validate function succeeds when failures are allowed but no failures
    ///         are present in the result.
    function test_validate_allowFailureTrue_succeeds() public view {
        string memory errors = _validate(true);
        assertEq(errors, "");
    }
}

/// @title OPContractsManagerStandardValidator_GeneralOverride_Test
/// @notice Tests behavior of validation overrides when multiple parameters are overridden
///         simultaneously
contract OPContractsManagerStandardValidator_GeneralOverride_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function (with the L1PAOMultisig and Challenger overridden)
    ///         successfully returns the right error when both are invalid.
    function test_validateL1PAOMultisigAndChallengerOverrides_succeeds() public view {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.l1PAOMultisig = address(0xace);
        overrides.challenger = address(0xbad);
        assertEq(
            "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,PROXYA-10,DF-30,PDDG-DWETH-30,PDDG-130,PLDG-DWETH-30,CKDG-DWETH-30",
            _validateWithOverrides(true, overrides)
        );
    }

    /// @notice Tests that the validate function (with the L1PAOMultisig and Challenger overridden)
    ///         successfully returns no error when there is none. That is, it never returns the
    ///         overridden strings alone.
    function test_validateOverrides_noErrors_succeeds() public {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = IOPContractsManagerStandardValidator
            .ValidationOverrides({ l1PAOMultisig: address(0xbad), challenger: address(0xc0ffee) });
        vm.mockCall(
            address(delayedWeth),
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
            abi.encode(overrides.l1PAOMultisig)
        );
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(overrides.l1PAOMultisig));
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.owner, ()),
            abi.encode(overrides.l1PAOMultisig)
        );
        DisputeGames.mockGameImplChallenger(dgf, GameTypes.PERMISSIONED_CANNON, overrides.challenger);

        assertEq("OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER", _validateWithOverrides(true, overrides));
    }

    /// @notice Tests that the validate function (with overrides) and allow failure set to false,
    ///         returns the errors with the overrides prepended.
    function test_validateOverrides_notAllowFailurePrependsOverrides_succeeds() public {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = IOPContractsManagerStandardValidator
            .ValidationOverrides({ l1PAOMultisig: address(0xbad), challenger: address(0xc0ffee) });

        vm.expectRevert(
            bytes(
                "OPContractsManagerStandardValidator: OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,PROXYA-10,DF-30,PDDG-DWETH-30,PDDG-130,PLDG-DWETH-30,CKDG-DWETH-30"
            )
        );

        _validateWithOverrides(false, overrides);
    }
}
/// @title OPContractsManagerStandardValidator_SuperchainConfig_Test
/// @notice Tests validation of `SuperchainConfig` contract configuration

contract OPContractsManagerStandardValidator_SuperchainConfig_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SuperchainConfig contract is paused.
    function test_validate_superchainConfigPaused_succeeds() public {
        // We use abi.encodeWithSignature because paused is overloaded.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(
            address(superchainConfig), abi.encodeWithSignature("paused(address)", (address(0))), abi.encode(true)
        );
        assertEq("SPRCFG-10", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_ProxyAdmin_Test
/// @notice Tests validation of `ProxyAdmin` configuration
contract OPContractsManagerStandardValidator_ProxyAdmin_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ProxyAdmin owner is not correct.
    function test_validate_invalidProxyAdminOwner_succeeds() public {
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(address(0xbad)));
        vm.mockCall(
            address(delayedWeth), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0xbad))
        );
        assertEq("PROXYA-10,PDDG-DWETH-30,PLDG-DWETH-30,CKDG-DWETH-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right overrides error
    ///         when the ProxyAdmin owner is overridden but is correct.
    function test_validate_overriddenProxyAdminOwner_succeeds() public {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.l1PAOMultisig = address(0xbad);
        vm.mockCall(address(delayedWeth), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(0xbad));
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(address(0xbad)));
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.owner, ()),
            abi.encode(overrides.l1PAOMultisig)
        );
        assertEq("OVERRIDES-L1PAOMULTISIG", _validateWithOverrides(true, overrides));
    }

    /// @notice Tests that the validate function (with an overridden ProxyAdmin owner) successfully
    ///         returns the right error when the ProxyAdmin owner is not correct.
    function test_validateOverrideL1PAOMultisig_invalidProxyAdminOwner_succeeds() public view {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.l1PAOMultisig = address(0xbad);
        assertEq(
            "OVERRIDES-L1PAOMULTISIG,PROXYA-10,DF-30,PDDG-DWETH-30,PLDG-DWETH-30,CKDG-DWETH-30",
            _validateWithOverrides(true, overrides)
        );
    }
}

/// @title OPContractsManagerStandardValidator_SystemConfig_Test
/// @notice Tests validation of `SystemConfig` configuration
contract OPContractsManagerStandardValidator_SystemConfig_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig version is invalid.
    function test_validate_systemConfigInvalidVersion_succeeds() public {
        vm.mockCall(address(systemConfig), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("SYSCON-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig gas limit is invalid.
    function test_validate_systemConfigInvalidGasLimit_succeeds() public {
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.gasLimit, ()), abi.encode(uint64(500_000_001)));
        assertEq("SYSCON-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig scalar is invalid.
    function test_validate_systemConfigInvalidScalar_succeeds() public {
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.scalar, ()), abi.encode(0));
        assertEq("SYSCON-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig proxy implementation is invalid.
    function test_validate_systemConfigInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(systemConfig))),
            abi.encode(address(0xbad))
        );
        assertEq("SYSCON-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig resourceConfig.maxResourceLimit is invalid.
    function test_validate_systemConfigInvalidResourceConfigMaxResourceLimit_succeeds() public {
        IResourceMetering.ResourceConfig memory badConfig = systemConfig.resourceConfig();
        badConfig.maxResourceLimit = 1_000_000;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig resourceConfig.elasticityMultiplier is invalid.
    function test_validate_systemConfigInvalidResourceConfigElasticityMultiplier_succeeds() public {
        IResourceMetering.ResourceConfig memory badConfig = systemConfig.resourceConfig();
        badConfig.elasticityMultiplier = 5;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig resourceConfig.baseFeeMaxChangeDenominator is invalid.
    function test_validate_systemConfigInvalidResourceConfigBaseFeeMaxChangeDenominator_succeeds() public {
        IResourceMetering.ResourceConfig memory badConfig = systemConfig.resourceConfig();
        badConfig.baseFeeMaxChangeDenominator = 4;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig resourceConfig.systemTxMaxGas is invalid.
    function test_validate_systemConfigInvalidResourceConfigSystemTxMaxGas_succeeds() public {
        IResourceMetering.ResourceConfig memory badConfig = systemConfig.resourceConfig();
        badConfig.systemTxMaxGas = 500_000;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-80", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig resourceConfig.minimumBaseFee is invalid.
    function test_validate_systemConfigInvalidResourceConfigMinimumBaseFee_succeeds() public {
        IResourceMetering.ResourceConfig memory badConfig = systemConfig.resourceConfig();
        badConfig.minimumBaseFee = 2 gwei;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-90", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig resourceConfig.maximumBaseFee is invalid.
    function test_validate_systemConfigInvalidResourceConfigMaximumBaseFee_succeeds() public {
        IResourceMetering.ResourceConfig memory badConfig = systemConfig.resourceConfig();
        badConfig.maximumBaseFee = 1_000_000;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-100", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig operatorFeeScalar is invalid.
    function test_validate_systemConfigInvalidOperatorFeeScalar_succeeds() public {
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeScalar, ()), abi.encode(1));
        assertEq("SYSCON-110", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig operatorFeeConstant is invalid.
    function test_validate_systemConfigInvalidOperatorFeeConstant_succeeds() public {
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeConstant, ()), abi.encode(1));
        assertEq("SYSCON-120", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig superchainConfig is invalid.
    function test_validate_systemConfigInvalidSuperchainConfig_succeeds() public {
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.superchainConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("SYSCON-130", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_L1CrossDomainMessenger_Test
/// @notice Tests validation of `L1CrossDomainMessenger` configuration
contract OPContractsManagerStandardValidator_L1CrossDomainMessenger_Test is
    OPContractsManagerStandardValidator_TestInit
{
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger version is invalid.
    function test_validate_l1CrossDomainMessengerInvalidVersion_succeeds() public {
        vm.mockCall(address(l1CrossDomainMessenger), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("L1xDM-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger implementation is invalid.
    function test_validate_l1CrossDomainMessengerBadImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1CrossDomainMessenger))),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger otherMessenger is invalid (legacy function).
    function test_validate_l1CrossDomainMessengerInvalidOtherMessengerLegacy_succeeds() public {
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.OTHER_MESSENGER, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger otherMessenger is invalid.
    function test_validate_l1CrossDomainMessengerInvalidOtherMessenger_succeeds() public {
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.otherMessenger, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger portal is invalid (legacy function).
    function test_validate_l1CrossDomainMessengerInvalidPortalLegacy_succeeds() public {
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.PORTAL, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger portal is invalid.
    function test_validate_l1CrossDomainMessengerInvalidPortal_succeeds() public {
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.portal, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger systemConfig is invalid.
    function test_validate_l1CrossDomainMessengerInvalidSystemConfig_succeeds() public {
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1CrossDomainMessenger proxyAdmin is invalid.
    function test_validate_l1CrossDomainMessengerInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-80", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_OptimismMintableERC20Factory_Test
/// @notice Tests validation of `OptimismMintableERC20Factory` configuration
contract OPContractsManagerStandardValidator_OptimismMintableERC20Factory_Test is
    OPContractsManagerStandardValidator_TestInit
{
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismMintableERC20Factory version is invalid.
    function test_validate_optimismMintableERC20FactoryInvalidVersion_succeeds() public {
        vm.mockCall(address(l1OptimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("MERC20F-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismMintableERC20Factory implementation is invalid.
    function test_validate_optimismMintableERC20FactoryInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1OptimismMintableERC20Factory))),
            abi.encode(address(0xbad))
        );
        assertEq("MERC20F-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismMintableERC20Factory bridge is invalid (legacy function).
    function test_validate_optimismMintableERC20FactoryInvalidBridgeLegacy_succeeds() public {
        vm.mockCall(
            address(l1OptimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.BRIDGE, ()),
            abi.encode(address(0xbad))
        );
        assertEq("MERC20F-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismMintableERC20Factory bridge is invalid.
    function test_validate_optimismMintableERC20FactoryInvalidBridge_succeeds() public {
        vm.mockCall(
            address(l1OptimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.bridge, ()),
            abi.encode(address(0xbad))
        );
        assertEq("MERC20F-40", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_L1ERC721Bridge_Test
/// @notice Tests validation of `L1ERC721Bridge` configuration
contract OPContractsManagerStandardValidator_L1ERC721Bridge_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge version is invalid.
    function test_validate_l1ERC721BridgeInvalidVersion_succeeds() public {
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("L721B-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge implementation is invalid.
    function test_validate_l1ERC721BridgeInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1ERC721Bridge))),
            abi.encode(address(0xbad))
        );
        assertEq("L721B-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge otherBridge is invalid (legacy function).
    function test_validate_l1ERC721BridgeInvalidOtherBridgeLegacy_succeeds() public {
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.OTHER_BRIDGE, ()), abi.encode(address(0xbad)));
        assertEq("L721B-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge otherBridge is invalid.
    function test_validate_l1ERC721BridgeInvalidOtherBridge_succeeds() public {
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.otherBridge, ()), abi.encode(address(0xbad)));
        assertEq("L721B-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge messenger is invalid (legacy function).
    function test_validate_l1ERC721BridgeInvalidMessengerLegacy_succeeds() public {
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.MESSENGER, ()), abi.encode(address(0xbad)));
        assertEq("L721B-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge messenger is invalid.
    function test_validate_l1ERC721BridgeInvalidMessenger_succeeds() public {
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.messenger, ()), abi.encode(address(0xbad)));
        assertEq("L721B-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge systemConfig is invalid.
    function test_validate_l1ERC721BridgeInvalidSystemConfig_succeeds() public {
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IL1ERC721Bridge.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("L721B-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1ERC721Bridge proxyAdmin is invalid.
    function test_validate_l1ERC721BridgeInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("L721B-80", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_OptimismPortal_Test
/// @notice Tests validation of `OptimismPortal` configuration
contract OPContractsManagerStandardValidator_OptimismPortal_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismPortal version is invalid.
    function test_validate_optimismPortalInvalidVersion_succeeds() public {
        vm.mockCall(address(optimismPortal2), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("PORTAL-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismPortal implementation is invalid.
    function test_validate_optimismPortalInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(optimismPortal2))),
            abi.encode(address(0xbad))
        );
        assertEq("PORTAL-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismPortal disputeGameFactory is invalid.
    function test_validate_optimismPortalInvalidDisputeGameFactory_succeeds() public {
        vm.mockFunction(
            address(optimismPortal2),
            address(badDisputeGameFactoryReturner),
            abi.encodeCall(IOptimismPortal2.disputeGameFactory, ())
        );
        assertEq("PORTAL-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismPortal systemConfig is invalid.
    function test_validate_optimismPortalInvalidSystemConfig_succeeds() public {
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("PORTAL-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismPortal l2Sender is invalid.
    function test_validate_optimismPortalInvalidL2Sender_succeeds() public {
        vm.mockCall(address(optimismPortal2), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(address(0xbad)));
        assertEq("PORTAL-80", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         OptimismPortal proxyAdmin is invalid.
    function test_validate_optimismPortalInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("PORTAL-90", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_ETHLockbox_Test
/// @notice Tests validation of `ETHLockbox` configuration
contract OPContractsManagerStandardValidator_ETHLockbox_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox version is invalid.
    function test_validate_ethLockboxInvalidVersion_succeeds() public {
        vm.mockCall(address(ethLockbox), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq("LOCKBOX-10", _validate(true));
        } else {
            assertEq("", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox implementation is invalid.
    function test_validate_ethLockboxInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(ethLockbox))),
            abi.encode(address(0xbad))
        );

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq("LOCKBOX-20", _validate(true));
        } else {
            assertEq("", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox proxyAdmin is invalid.
    function test_validate_ethLockboxInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(ethLockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq("LOCKBOX-30", _validate(true));
        } else {
            assertEq("", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox systemConfig is invalid.
    function test_validate_ethLockboxInvalidSystemConfig_succeeds() public {
        vm.mockCall(address(ethLockbox), abi.encodeCall(IETHLockbox.systemConfig, ()), abi.encode(address(0xbad)));

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq("LOCKBOX-40", _validate(true));
        } else {
            assertEq("", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox does not have the OptimismPortal as an authorized portal.
    function test_validate_ethLockboxPortalUnauthorized_succeeds() public {
        vm.mockCall(
            address(ethLockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (optimismPortal2)), abi.encode(false)
        );

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq("LOCKBOX-50", _validate(true));
        } else {
            assertEq("", _validate(true));
        }
    }
}

/// @title OPContractsManagerStandardValidator_DisputeGameFactory_Test
/// @notice Tests validation of `DisputeGameFactory` configuration
contract OPContractsManagerStandardValidator_DisputeGameFactory_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DisputeGameFactory version is invalid.
    function test_validate_disputeGameFactoryInvalidVersion_succeeds() public {
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("0.9.0"));
        assertEq("DF-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DisputeGameFactory implementation is invalid.
    function test_validate_disputeGameFactoryInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(disputeGameFactory))),
            abi.encode(address(0xbad))
        );
        assertEq("DF-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DisputeGameFactory owner is invalid.
    function test_validate_disputeGameFactoryInvalidOwner_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(address(0xbad))
        );
        assertEq("DF-30", _validate(true));
    }

    /// @notice Tests that the validate function returns DF-50 when neither PERMISSIONED_CANNON nor
    ///         SUPER_PERMISSIONED_CANNON has a registered implementation in the DGF.
    function test_validate_disputeGameFactoryNoPermissionedGame_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        // SUPER_PERMISSIONED_CANNON is not registered in non-super mode, so DF-50 fires.
        // PDDG-NOSHAPE fires because PERMISSIONED_CANNON is not registered in non-super mode.
        // PDDG-10 also fires because PERMISSIONED_CANNON impl is null.
        assertEq("DF-50,PDDG-NOSHAPE,PDDG-10", _validate(true));
    }

    /// @notice Tests that SCDG-NOSHAPE fires when SUPER_CANNON has a registered impl in non-super mode.
    function test_validate_nonSuperModeSuperCannonRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("SCDG-NOSHAPE", _validate(true));
    }

    /// @notice Tests that SPDG-NOSHAPE fires when SUPER_PERMISSIONED_CANNON has a registered impl in non-super mode.
    function test_validate_nonSuperModeSuperPermissionedCannonRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0xdead))
        );
        assertEq("SPDG-NOSHAPE", _validate(true));
    }

    /// @notice Tests that SCKDG-NOSHAPE fires when SUPER_CANNON_KONA has a registered impl in non-super mode.
    function test_validate_nonSuperModeSuperCannonKonaRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0xdead))
        );
        assertEq("SCKDG-NOSHAPE", _validate(true));
    }

    /// @notice Tests that all three NOSHAPE errors fire when all super game types are registered in non-super mode.
    function test_validate_nonSuperModeAllSuperGamesRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON)),
            abi.encode(address(0xdead))
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0xdead))
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_CANNON_KONA)),
            abi.encode(address(0xdead))
        );
        assertEq("SCDG-NOSHAPE,SPDG-NOSHAPE,SCKDG-NOSHAPE", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_PermissionedDisputeGame_Test
/// @notice Tests validation of `PermissionedDisputeGame` configuration
contract OPContractsManagerStandardValidator_PermissionedDisputeGame_Test is
    OPContractsManagerStandardValidator_TestInit
{
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame implementation is null.
    function test_validate_permissionedDisputeGameNullImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        // DF-50 also fires because neither PERMISSIONED_CANNON nor SUPER_PERMISSIONED_CANNON is registered.
        // PDDG-NOSHAPE fires because PERMISSIONED_CANNON is not registered in non-super mode.
        assertEq("DF-50,PDDG-NOSHAPE,PDDG-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame version is invalid.
    function test_validate_permissionedDisputeGameInvalidVersion_succeeds() public {
        BadVersionReturner bad = new BadVersionReturner(standardValidator, ISemver(address(pdgImpl)), "0.0.0");
        bytes32 slot =
            bytes32(ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "permissionedDisputeGameImpl").slot);
        vm.store(address(standardValidator), slot, bytes32(uint256(uint160(address(bad)))));
        assertEq("PDDG-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame game args are invalid.
    function test_validate_permissionedDisputeGameInvalidGameArgs_succeeds() public {
        bytes memory invalidGameArgs = hex"123456";
        GameType gameType = GameTypes.PERMISSIONED_CANNON;
        vm.mockCall(address(dgf), abi.encodeCall(IDisputeGameFactory.gameArgs, (gameType)), abi.encode(invalidGameArgs));

        assertEq("PDDG-GARGS-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame absolute prestate is invalid.
    function test_validate_permissionedDisputeGameInvalidAbsolutePrestate_succeeds() public {
        bytes32 badPrestate = bytes32(uint256(0xbadbad));
        DisputeGames.mockGameImplPrestate(dgf, GameTypes.PERMISSIONED_CANNON, badPrestate);
        assertEq("PDDG-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame VM address is invalid.
    function test_validate_permissionedDisputeGameInvalidVM_succeeds() public {
        address badVM = address(0xbad);
        DisputeGames.mockGameImplVM(dgf, GameTypes.PERMISSIONED_CANNON, badVM);
        vm.mockCall(badVM, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(
            address(0xbad), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION)
        );
        assertEq("PDDG-VM-10,PDDG-VM-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame ASR address is invalid.
    function test_validate_permissionedDisputeGameInvalidASR_succeeds() public {
        address badASR = address(0xbad);
        DisputeGames.mockGameImplASR(dgf, GameTypes.PERMISSIONED_CANNON, badASR);

        // Mock invalid values
        vm.mockCall(badASR, abi.encodeCall(IStaticERC1967Proxy.implementation, ()), abi.encode(address(0xdeadbeef)));
        vm.mockCall(badASR, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));

        // Mock valid return values
        vm.mockCall(
            badASR,
            abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()),
            abi.encode(Hash.wrap(bytes32(uint256(0x123))), uint256(123))
        );
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(dgf));
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.systemConfig, ()), abi.encode(sysCfg));
        vm.mockCall(badASR, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(proxyAdmin));
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(uint64(100)));

        assertEq("PDDG-ANCHORP-10,PDDG-ANCHORP-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame Weth address is invalid.
    function test_validate_permissionedDisputeGameInvalidWeth_succeeds() public {
        address badWeth = address(0xbad);
        DisputeGames.mockGameImplWeth(dgf, GameTypes.PERMISSIONED_CANNON, badWeth);

        // Mock invalid values
        vm.mockCall(badWeth, abi.encodeCall(IStaticERC1967Proxy.implementation, ()), abi.encode(address(0xdeadbeef)));
        vm.mockCall(badWeth, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));

        // Mock valid return values
        vm.mockCall(
            badWeth,
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
            abi.encode(standardValidator.l1PAOMultisig())
        );
        vm.mockCall(
            badWeth, abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(standardValidator.withdrawalDelaySeconds())
        );
        vm.mockCall(badWeth, abi.encodeCall(IDelayedWETH.systemConfig, ()), abi.encode(sysCfg));
        vm.mockCall(badWeth, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(proxyAdmin));

        assertEq("PDDG-DWETH-10,PDDG-DWETH-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame VM's state version is invalid.
    function test_validate_permissionedDisputeGameInvalidVMStateVersion_succeeds() public {
        vm.mockCall(address(mips), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(6));
        assertEq("PDDG-VM-30,PLDG-VM-30,CKDG-VM-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame L2 Chain ID is invalid.
    function test_validate_permissionedDisputeGameInvalidL2ChainId_succeeds() public {
        uint256 badChainId = l2ChainId + 1;
        DisputeGames.mockGameImplL2ChainId(dgf, GameTypes.PERMISSIONED_CANNON, badChainId);
        assertEq("PDDG-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame L2 Sequence Number is invalid.
    function test_validate_permissionedDisputeGameInvalidL2SequenceNumber_succeeds() public {
        vm.mockCall(address(pdgImpl), abi.encodeCall(IDisputeGame.l2SequenceNumber, ()), abi.encode(123));
        assertEq("PDDG-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame clockExtension is invalid.
    function test_validate_permissionedDisputeGameInvalidClockExtension_succeeds() public {
        vm.mockCall(
            address(pdgImpl),
            abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()),
            abi.encode(Duration.wrap(1000))
        );
        assertEq("PDDG-80", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame splitDepth is invalid.
    function test_validate_permissionedDisputeGameInvalidSplitDepth_succeeds() public {
        vm.mockCall(address(pdgImpl), abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()), abi.encode(20));
        assertEq("PDDG-90", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame maxGameDepth is invalid.
    function test_validate_permissionedDisputeGameInvalidMaxGameDepth_succeeds() public {
        vm.mockCall(address(pdgImpl), abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()), abi.encode(50));
        assertEq("PDDG-100", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame maxClockDuration is invalid.
    function test_validate_permissionedDisputeGameInvalidMaxClockDuration_succeeds() public {
        vm.mockCall(
            address(pdgImpl),
            abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()),
            abi.encode(Duration.wrap(1000))
        );
        assertEq("PDDG-110", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame anchor root is 0.
    function test_validate_permissionedDisputeGameZeroAnchorRoot_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()),
            abi.encode(bytes32(0), 1)
        );
        assertEq("PDDG-120,PLDG-120,CKDG-120", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame challenger is invalid.
    function test_validate_permissionedDisputeGameInvalidChallenger_succeeds() public {
        address badChallenger = address(0xbad);
        DisputeGames.mockGameImplChallenger(dgf, GameTypes.PERMISSIONED_CANNON, badChallenger);
        assertEq("PDDG-130", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right overrides error when the
    ///         PermissionedDisputeGame challenger is overridden but is correct.
    function test_validate_overriddenPermissionedDisputeGameChallenger_succeeds() public {
        address challengerOverride = address(0xbad);

        DisputeGames.mockGameImplChallenger(dgf, GameTypes.PERMISSIONED_CANNON, challengerOverride);
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.challenger = challengerOverride;

        assertEq("OVERRIDES-CHALLENGER", _validateWithOverrides(true, overrides));
    }

    /// @notice Tests that the validate function (with an overridden PermissionedDisputeGame challenger) successfully
    ///         returns the right error when the PermissionedDisputeGame challenger is invalid.
    function test_validateOverridesChallenger_permissionedDisputeGameInvalidChallenger_succeeds() public view {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.challenger = address(0xbad);
        assertEq("OVERRIDES-CHALLENGER,PDDG-130", _validateWithOverrides(true, overrides));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame proposer is invalid.
    function test_validate_permissionedDisputeGameInvalidProposer_succeeds() public {
        address badProposer = address(0xbad);
        DisputeGames.mockGameImplProposer(dgf, GameTypes.PERMISSIONED_CANNON, badProposer);
        assertEq("PDDG-140", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_AnchorStateRegistry_Test
/// @notice Tests validation of `AnchorStateRegistry` configuration
contract OPContractsManagerStandardValidator_AnchorStateRegistry_Test is
    OPContractsManagerStandardValidator_TestInit
{
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry version is invalid.
    function test_validate_anchorStateRegistryInvalidVersion_succeeds() public {
        vm.mockCall(address(anchorStateRegistry), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.1"));
        assertEq("PDDG-ANCHORP-10,PLDG-ANCHORP-10,CKDG-ANCHORP-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry implementation is invalid.
    function test_validate_anchorStateRegistryInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(anchorStateRegistry))),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-20,PLDG-ANCHORP-20,CKDG-ANCHORP-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry disputeGameFactory is invalid.
    function test_validate_anchorStateRegistryInvalidDisputeGameFactory_succeeds() public {
        vm.mockFunction(
            address(anchorStateRegistry),
            address(badDisputeGameFactoryReturner),
            abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ())
        );
        assertEq("PDDG-ANCHORP-30,PLDG-ANCHORP-30,CKDG-ANCHORP-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry systemConfig is invalid.
    function test_validate_anchorStateRegistryInvalidSystemConfig_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IAnchorStateRegistry.systemConfig, ()),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-40,PLDG-ANCHORP-40,CKDG-ANCHORP-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry proxyAdmin is invalid.
    function test_validate_anchorStateRegistryInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-50,PLDG-ANCHORP-50,CKDG-ANCHORP-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry retirementTimestamp is invalid.
    function test_validate_anchorStateRegistryInvalidRetirementTimestamp_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry), abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(0)
        );
        assertEq("PDDG-ANCHORP-60,PLDG-ANCHORP-60,CKDG-ANCHORP-60", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_DelayedWETH_Test
/// @notice Tests validation of `DelayedWETH` configuration
contract OPContractsManagerStandardValidator_DelayedWETH_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH version is invalid.
    function test_validate_delayedWETHInvalidVersion_succeeds() public {
        vm.mockCall(address(delayedWeth), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.1"));
        assertEq("PDDG-DWETH-10,PLDG-DWETH-10,CKDG-DWETH-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH implementation is invalid.
    function test_validate_delayedWETHInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(delayedWeth))),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-DWETH-20,PLDG-DWETH-20,CKDG-DWETH-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH proxyAdminOwner is invalid.
    function test_validate_delayedWETHInvalidProxyAdminOwner_succeeds() public {
        vm.mockCall(
            address(delayedWeth), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0xbad))
        );
        assertEq("PDDG-DWETH-30,PLDG-DWETH-30,CKDG-DWETH-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH delay is invalid.
    function test_validate_delayedWETHInvalidDelay_succeeds() public {
        vm.mockCall(address(delayedWeth), abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(1000));
        assertEq("PDDG-DWETH-40,PLDG-DWETH-40,CKDG-DWETH-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH systemConfig is invalid.
    function test_validate_delayedWETHInvalidSystemConfig_succeeds() public {
        vm.mockCall(address(delayedWeth), abi.encodeCall(IDelayedWETH.systemConfig, ()), abi.encode(address(0xbad)));
        assertEq("PDDG-DWETH-50,PLDG-DWETH-50,CKDG-DWETH-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH proxyAdmin is invalid.
    function test_validate_delayedWETHInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(delayedWeth), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("PDDG-DWETH-60,PLDG-DWETH-60,CKDG-DWETH-60", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_PreimageOracle_Test
/// @notice Tests validation of `PreimageOracle` configuration
contract OPContractsManagerStandardValidator_PreimageOracle_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PreimageOracle version is invalid.
    function test_validate_preimageOracleInvalidVersion_succeeds() public {
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.1"));
        assertEq("PDDG-PIMGO-10,PLDG-PIMGO-10,CKDG-PIMGO-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PreimageOracle challengePeriod is invalid.
    function test_validate_preimageOracleInvalidChallengePeriod_succeeds() public {
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.challengePeriod, ()), abi.encode(1000));
        assertEq("PDDG-PIMGO-20,PLDG-PIMGO-20,CKDG-PIMGO-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PreimageOracle minProposalSize is invalid.
    function test_validate_preimageOracleInvalidMinProposalSize_succeeds() public {
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.minProposalSize, ()), abi.encode(1000));
        assertEq("PDDG-PIMGO-30,PLDG-PIMGO-30,CKDG-PIMGO-30", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_FaultDisputeGame_Test
/// @notice Tests validation of `FaultDisputeGame` configuration
contract OPContractsManagerStandardValidator_FaultDisputeGame_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon implementation is null.
    function test_validate_faultDisputeGameNullCannonImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0))
        );
        assertEq("PLDG-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona implementation is null.
    function test_validate_faultDisputeGameNullCannonKonaImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON_KONA)),
            abi.encode(address(0))
        );
        assertEq("CKDG-NOSHAPE,CKDG-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) version is invalid.
    function test_validate_faultDisputeGameInvalidVersion_succeeds() public {
        BadVersionReturner bad = new BadVersionReturner(standardValidator, ISemver(address(pdgImpl)), "0.0.0");
        bytes32 slot =
            bytes32(ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "faultDisputeGameImpl").slot);
        vm.store(address(standardValidator), slot, bytes32(uint256(uint160(address(bad)))));
        assertEq("PLDG-20,CKDG-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon game args are invalid.
    function test_validate_faultDisputeGameInvalidCannonGameArgs_succeeds() public {
        bytes memory invalidGameArgs = hex"123456";
        GameType gameType = GameTypes.CANNON;
        vm.mockCall(address(dgf), abi.encodeCall(IDisputeGameFactory.gameArgs, (gameType)), abi.encode(invalidGameArgs));

        assertEq("PLDG-GARGS-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona game args are invalid.
    function test_validate_faultDisputeGameInvalidCannonKonaGameArgs_succeeds() public {
        bytes memory invalidGameArgs = hex"123456";
        GameType gameType = GameTypes.CANNON_KONA;
        vm.mockCall(address(dgf), abi.encodeCall(IDisputeGameFactory.gameArgs, (gameType)), abi.encode(invalidGameArgs));

        assertEq("CKDG-GARGS-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon absolute prestate is invalid.
    function test_validate_faultDisputeGameInvalidCannonAbsolutePrestate_succeeds() public {
        bytes32 badPrestate = bytes32(uint256(0xbadbad));
        DisputeGames.mockGameImplPrestate(dgf, GameTypes.CANNON, badPrestate);

        assertEq("PLDG-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona absolute prestate is invalid.
    function test_validate_faultDisputeGameInvalidCannonKonaAbsolutePrestate_succeeds() public {
        bytes32 badPrestate = cannonPrestate.raw(); // Use the wrong prestate
        DisputeGames.mockGameImplPrestate(dgf, GameTypes.CANNON_KONA, badPrestate);

        assertEq("CKDG-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon VM address is invalid.
    function test_validate_faultDisputeGameInvalidCannonVM_succeeds() public {
        address badVM = address(0xbad);
        DisputeGames.mockGameImplVM(dgf, GameTypes.CANNON, badVM);
        vm.mockCall(badVM, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(badVM, abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION));

        assertEq("PLDG-VM-10,PLDG-VM-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona VM address is invalid.
    function test_validate_faultDisputeGameInvalidCannonKonaVM_succeeds() public {
        address badVM = address(0xbad);
        DisputeGames.mockGameImplVM(dgf, GameTypes.CANNON_KONA, badVM);
        vm.mockCall(badVM, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(badVM, abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION));

        assertEq("CKDG-VM-10,CKDG-VM-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon ASR address is invalid.
    function test_validate_faultDisputeGameInvalidCannonASR_succeeds() public {
        _mockInvalidASR(GameTypes.CANNON);
        assertEq("PLDG-ANCHORP-10,PLDG-ANCHORP-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona ASR address is invalid.
    function test_validate_faultDisputeGameInvalidCannonKonaASR_succeeds() public {
        _mockInvalidASR(GameTypes.CANNON_KONA);
        assertEq("CKDG-ANCHORP-10,CKDG-ANCHORP-20", _validate(true));
    }

    function _mockInvalidASR(GameType _gameType) internal {
        address badASR = address(0xbad);
        DisputeGames.mockGameImplASR(dgf, _gameType, badASR);

        // Mock invalid values
        vm.mockCall(badASR, abi.encodeCall(IStaticERC1967Proxy.implementation, ()), abi.encode(address(0xdeadbeef)));
        vm.mockCall(badASR, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));

        // Mock valid return values
        vm.mockCall(
            badASR,
            abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()),
            abi.encode(Hash.wrap(bytes32(uint256(0x123))), uint256(123))
        );
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(dgf));
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.systemConfig, ()), abi.encode(sysCfg));
        vm.mockCall(badASR, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(proxyAdmin));
        vm.mockCall(badASR, abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(uint64(100)));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon Weth address is invalid.
    function test_validate_faultDisputeGameInvalidCannonWeth_succeeds() public {
        _mockInvalidWeth(GameTypes.CANNON);
        assertEq("PLDG-DWETH-10,PLDG-DWETH-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona Weth address is invalid.
    function test_validate_faultDisputeGameInvalidCannonKonaWeth_succeeds() public {
        _mockInvalidWeth(GameTypes.CANNON_KONA);
        assertEq("CKDG-DWETH-10,CKDG-DWETH-20", _validate(true));
    }

    function _mockInvalidWeth(GameType _gameType) internal {
        address badWeth = address(0xbad);
        DisputeGames.mockGameImplWeth(dgf, _gameType, badWeth);

        // Mock invalid values
        vm.mockCall(badWeth, abi.encodeCall(IStaticERC1967Proxy.implementation, ()), abi.encode(address(0xdeadbeef)));
        vm.mockCall(badWeth, abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));

        // Mock valid return values
        vm.mockCall(
            badWeth,
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
            abi.encode(standardValidator.l1PAOMultisig())
        );
        vm.mockCall(
            badWeth, abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(standardValidator.withdrawalDelaySeconds())
        );
        vm.mockCall(badWeth, abi.encodeCall(IDelayedWETH.systemConfig, ()), abi.encode(sysCfg));
        vm.mockCall(badWeth, abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(proxyAdmin));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) VM's state version is invalid.
    function test_validate_faultDisputeGameInvalidVMStateVersion_succeeds() public {
        vm.mockCall(address(mips), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(6));
        assertEq("PDDG-VM-30,PLDG-VM-30,CKDG-VM-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) Cannon L2 Chain ID is invalid.
    function test_validate_faultDisputeGameInvalidCannonL2ChainId_succeeds() public {
        uint256 badChainId = l2ChainId + 1;
        DisputeGames.mockGameImplL2ChainId(dgf, GameTypes.CANNON, badChainId);

        assertEq("PLDG-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) CannonKona L2 Chain ID is invalid.
    function test_validate_faultDisputeGameInvalidCannonKonaL2ChainId_succeeds() public {
        uint256 badChainId = l2ChainId + 1;
        DisputeGames.mockGameImplL2ChainId(dgf, GameTypes.CANNON_KONA, badChainId);

        assertEq("CKDG-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) L2 Sequence Number is invalid.
    function test_validate_faultDisputeGameInvalidL2SequenceNumber_succeeds() public {
        vm.mockCall(address(fdgImpl), abi.encodeCall(IDisputeGame.l2SequenceNumber, ()), abi.encode(123));
        assertEq("PLDG-70,CKDG-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) clockExtension is invalid.
    function test_validate_faultDisputeGameInvalidClockExtension_succeeds() public {
        vm.mockCall(
            address(fdgImpl), abi.encodeCall(IFaultDisputeGame.clockExtension, ()), abi.encode(Duration.wrap(1000))
        );
        assertEq("PLDG-80,CKDG-80", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) splitDepth is invalid.
    function test_validate_faultDisputeGameInvalidSplitDepth_succeeds() public {
        vm.mockCall(address(fdgImpl), abi.encodeCall(IFaultDisputeGame.splitDepth, ()), abi.encode(20));
        assertEq("PLDG-90,CKDG-90", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) maxGameDepth is invalid.
    function test_validate_faultDisputeGameInvalidMaxGameDepth_succeeds() public {
        vm.mockCall(address(fdgImpl), abi.encodeCall(IFaultDisputeGame.maxGameDepth, ()), abi.encode(50));
        assertEq("PLDG-100,CKDG-100", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) maxClockDuration is invalid.
    function test_validate_faultDisputeGameInvalidMaxClockDuration_succeeds() public {
        vm.mockCall(
            address(fdgImpl), abi.encodeCall(IFaultDisputeGame.maxClockDuration, ()), abi.encode(Duration.wrap(1000))
        );
        assertEq("PLDG-110,CKDG-110", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_L1StandardBridge_Test
/// @notice Tests validation of `L1StandardBridge` configuration
contract OPContractsManagerStandardValidator_L1StandardBridge_Test is OPContractsManagerStandardValidator_TestInit {
    // L1StandardBridge Tests
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge version is invalid.
    function test_validate_l1StandardBridgeInvalidVersion_succeeds() public {
        vm.mockCall(address(l1StandardBridge), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("L1SB-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge MESSENGER immutable is invalidly reported (mocked).
    function test_validate_l1StandardBridgeInvalidMessengerImmutable_succeeds() public {
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.MESSENGER, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge messenger getter is invalid.
    function test_validate_l1StandardBridgeInvalidMessengerGetter_succeeds() public {
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.messenger, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge OTHER_BRIDGE immutable is invalidly reported (mocked).
    function test_validate_l1StandardBridgeInvalidOtherBridgeImmutable_succeeds() public {
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.OTHER_BRIDGE, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge otherBridge getter is invalid.
    function test_validate_l1StandardBridgeInvalidOtherBridgeGetter_succeeds() public {
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.otherBridge, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge systemConfig is invalid.
    function test_validate_l1StandardBridgeInvalidSystemConfig_succeeds() public {
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IL1StandardBridge.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         L1StandardBridge proxyAdmin is invalid.
    function test_validate_l1StandardBridgeInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-80", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_Versions_Test
/// @notice Tests the `version` functions on `OPContractsManagerStandardValidator`.
contract OPContractsManagerStandardValidator_Versions_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the version getter functions on `OPContractsManagerStandardValidator` return non-empty
    ///         strings.
    function test_versions_succeeds() public view {
        assertTrue(
            bytes(ISemver(standardValidator.systemConfigImpl()).version()).length > 0, "systemConfigVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.optimismPortalImpl()).version()).length > 0, "optimismPortalVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.l1CrossDomainMessengerImpl()).version()).length > 0,
            "l1CrossDomainMessengerVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.l1ERC721BridgeImpl()).version()).length > 0, "l1ERC721BridgeVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.l1StandardBridgeImpl()).version()).length > 0,
            "l1StandardBridgeVersion empty"
        );
        assertTrue(bytes(ISemver(standardValidator.mipsImpl()).version()).length > 0, "mipsVersion empty");
        assertTrue(
            bytes(ISemver(standardValidator.faultDisputeGameImpl()).version()).length > 0,
            "faultDisputeGameVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.permissionedDisputeGameImpl()).version()).length > 0,
            "permissionedDisputeGameVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.optimismMintableERC20FactoryImpl()).version()).length > 0,
            "optimismMintableERC20FactoryVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.disputeGameFactoryImpl()).version()).length > 0,
            "disputeGameFactoryVersion empty"
        );
        assertTrue(
            bytes(ISemver(standardValidator.anchorStateRegistryImpl()).version()).length > 0,
            "anchorStateRegistryVersion empty"
        );
        assertTrue(bytes(ISemver(standardValidator.delayedWETHImpl()).version()).length > 0, "delayedWETHVersion empty");
        assertTrue(bytes(standardValidator.preimageOracleVersion()).length > 0, "preimageOracleVersion empty");
        assertTrue(bytes(ISemver(standardValidator.ethLockboxImpl()).version()).length > 0, "ethLockboxVersion empty");
    }
}

/// @title OPContractsManagerStandardValidator_SuperMode_TestInit
/// @notice Base contract for super mode StandardValidator tests. Requires SUPER_ROOT_GAMES_MIGRATION flag.
///         After setUp, the chain has both SUPER_PERMISSIONED_CANNON and SUPER_CANNON_KONA enabled.
abstract contract OPContractsManagerStandardValidator_SuperMode_TestInit is SuperGameTestInit {
    /// @notice The l2ChainId.
    uint256 l2ChainId;

    /// @notice The DisputeGameFactory instance.
    IDisputeGameFactory dgf;

    /// @notice The OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator standardValidator;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        // Requires migration flag — inverse of the standard test init skip.
        if (!Config.devFeatureSuperRootGamesMigration()) {
            vm.skip(true, "Skipping: requires SUPER_ROOT_GAMES_MIGRATION");
        }
        super.setUp();

        dgf = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        standardValidator = opcmV2.opcmStandardValidator();

        l2ChainId = deploy.cfg().l2ChainID();
        cannonPrestate = Claim.wrap(bytes32(deploy.cfg().faultGameAbsolutePrestate()));
        proposer = deploy.cfg().l2OutputOracleProposer();
        challenger = deploy.cfg().l2OutputOracleChallenger();

        // The deploy created SUPER_PERMISSIONED_CANNON (enabled) + SUPER_CANNON_KONA (disabled).
        // Run an upgrade to also enable SUPER_CANNON_KONA so that full validation passes.
        _enableSuperCannonKona();
    }

    /// @notice Runs an upgrade that enables SUPER_CANNON_KONA alongside SUPER_PERMISSIONED_CANNON.
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
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
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
            data: abi.encode(GameTypes.SUPER_PERMISSIONED_CANNON)
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

    /// @notice Tests that disabling SUPER_PERMISSIONED_CANNON triggers SPDG-SHAPE.
    function test_validate_superPermissionedCannonNotRegistered_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        // DF-50 also fires because neither PERMISSIONED_CANNON nor SUPER_PERMISSIONED_CANNON is registered.
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
/// @notice Tests SPDG error codes for the SUPER_PERMISSIONED_CANNON game validation.
contract OPContractsManagerStandardValidator_SuperPermissionedDisputeGame_Test is
    OPContractsManagerStandardValidator_SuperMode_TestInit
{
    /// @notice Tests SPDG-10 when SUPER_PERMISSIONED_CANNON implementation is null.
    function test_validate_superPermissionedDisputeGameNullImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );
        // DF-50 also fires because neither PERMISSIONED_CANNON nor SUPER_PERMISSIONED_CANNON is registered.
        assertEq("DF-50,SPDG-SHAPE,SPDG-10", _validate(true));
    }

    /// @notice Tests SPDG-20 when SUPER_PERMISSIONED_CANNON version is invalid.
    function test_validate_superPermissionedDisputeGameInvalidVersion_succeeds() public {
        address spdgImpl = address(disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON));
        BadVersionReturner bad = new BadVersionReturner(standardValidator, ISemver(spdgImpl), "0.0.0");
        bytes32 slot = bytes32(
            ForgeArtifacts.getSlot("OPContractsManagerStandardValidator", "superPermissionedDisputeGameImpl").slot
        );
        vm.store(address(standardValidator), slot, bytes32(uint256(uint160(address(bad)))));
        assertEq("SPDG-20", _validate(true));
    }

    /// @notice Tests SPDG-GARGS-10 when SUPER_PERMISSIONED_CANNON game args are invalid.
    function test_validate_superPermissionedDisputeGameInvalidGameArgs_succeeds() public {
        vm.mockCall(
            address(dgf),
            abi.encodeCall(IDisputeGameFactory.gameArgs, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(hex"123456")
        );

        assertEq("SPDG-GARGS-10", _validate(true));
    }

    /// @notice Tests SPDG-ANCHORP-* when SUPER_PERMISSIONED_CANNON's simplified ASR arg is invalid.
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

    /// @notice Tests SPDG-120 when SUPER_PERMISSIONED_CANNON's anchor root is zero.
    function test_validate_superPermissionedDisputeGameZeroAnchorRoot_succeeds() public {
        address spdgASR = DisputeGames.superPermissionedGameAnchorStateRegistry(dgf);
        vm.mockCall(spdgASR, abi.encodeCall(IAnchorStateRegistry.getAnchorRoot, ()), abi.encode(bytes32(0), uint256(0)));

        assertEq("SPDG-120,SCKDG-120", _validate(true));
    }

    /// @notice Tests SPDG-140 when SUPER_PERMISSIONED_CANNON proposer is invalid.
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
}

/// @title OPContractsManagerStandardValidator_ZKDisputeGame_Test
/// @notice Tests that ZK dispute game validation is gated on the ZK_DISPUTE_GAME dev feature flag.
///         These tests run in non-ZK deployment mode and verify both branches of the gating logic.
contract OPContractsManagerStandardValidator_ZKDisputeGame_Test is OPContractsManagerStandardValidator_TestInit {
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
/// @notice Base contract for ZK dispute game validator tests.
///         Skips unless DEV_FEATURE__ZK_DISPUTE_GAME is enabled.
///         Deploys the chain with a ZK dispute game via OPCM so the full validation path is exercised.
abstract contract OPContractsManagerStandardValidator_ZKMode_TestInit is CommonTest {
    /// @notice The l2ChainId from the deploy config.
    uint256 l2ChainId;

    /// @notice The cannon absolute prestate from the deploy config.
    Claim cannonPrestate;

    /// @notice The CannonKona absolute prestate.
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice The proposer role from the deploy config.
    address proposer;

    /// @notice The challenger role from the deploy config.
    address challenger;

    /// @notice The DisputeGameFactory instance.
    IDisputeGameFactory dgf;

    /// @notice The OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator standardValidator;

    /// @notice Sets up the ZK-mode test suite. Skips if the ZK feature is not enabled.
    function setUp() public virtual override {
        if (!Config.devFeatureZkDisputeGame()) {
            vm.skip(true, "Skipping: DEV_FEATURE__ZK_DISPUTE_GAME is not enabled");
        }
        if (Config.devFeatureSuperRootGamesMigration()) {
            vm.skip(true, "Skipping: standard configs incompatible with SUPER_ROOT_GAMES_MIGRATION");
        }
        super.setUp();

        dgf = IDisputeGameFactory(artifacts.mustGetAddress("DisputeGameFactoryProxy"));
        standardValidator = opcmV2.opcmStandardValidator();

        // ZKDG-80 requires verifier.code.length > 0. Etch a dummy byte so the dummy
        // verifier address used in both fork and non-fork paths satisfies this check.
        vm.etch(address(0xBEEF), hex"01");

        if (isL1ForkTest()) {
            // In fork mode read the actual values from the deployed contracts so _validate()
            // is consistent with the real on-chain state.
            LibGameArgs.GameArgs memory cannonArgs = LibGameArgs.decode(dgf.gameArgs(GameTypes.CANNON));
            cannonPrestate = Claim.wrap(cannonArgs.absolutePrestate);
            l2ChainId = cannonArgs.l2ChainId;

            LibGameArgs.GameArgs memory pddgArgs = LibGameArgs.decode(dgf.gameArgs(GameTypes.PERMISSIONED_CANNON));
            proposer = pddgArgs.proposer;
            challenger = pddgArgs.challenger;

            cannonKonaPrestate = Claim.wrap(LibGameArgs.decode(dgf.gameArgs(GameTypes.CANNON_KONA)).absolutePrestate);

            // ZK game is not deployed on mainnet. Mock it using the same ASR and WETH as CANNON
            // (same on-chain infrastructure) so _assertValidZKGameArgs passes its checks.
            bytes memory zkArgs = abi.encodePacked(
                bytes32(keccak256("zkPrestate")),
                address(0xBEEF),
                uint64(7 days),
                uint64(3 days),
                uint256(0.08 ether),
                cannonArgs.anchorStateRegistry,
                cannonArgs.weth,
                l2ChainId
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
        } else {
            l2ChainId = deploy.cfg().l2ChainID();
            cannonPrestate = Claim.wrap(bytes32(deploy.cfg().faultGameAbsolutePrestate()));
            proposer = deploy.cfg().l2OutputOracleProposer();
            challenger = deploy.cfg().l2OutputOracleChallenger();

            address owner = proxyAdmin.owner();

            // Init all game configs.
            // note: We set init bond to a non-zero value for all enabled games to avoid config validation reverts
            // in the OPCM. Other games bonds are irrelevant for the ZK specific validation tests.
            IOPContractsManagerUtils.DisputeGameConfig[] memory configs =
                new IOPContractsManagerUtils.DisputeGameConfig[](6);
            configs[0] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.CANNON,
                gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonPrestate }))
            });
            configs[1] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.PERMISSIONED_CANNON,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                        absolutePrestate: cannonPrestate,
                        proposer: proposer,
                        challenger: challenger
                    })
                )
            });
            configs[2] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
                gameType: GameTypes.CANNON_KONA,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate })
                )
            });
            configs[3] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
                gameArgs: hex""
            });
            configs[4] = IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.SUPER_CANNON_KONA,
                gameArgs: hex""
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

            prankDelegateCall(owner);
            (bool success,) = address(opcmV2).delegatecall(
                abi.encodeCall(
                    IOPContractsManagerV2.upgrade,
                    (
                        IOPContractsManagerV2.UpgradeInput({
                            systemConfig: systemConfig,
                            disputeGameConfigs: configs,
                            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
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
///         Only runs when DEV_FEATURE__ZK_DISPUTE_GAME is enabled.
contract OPContractsManagerStandardValidator_ZKValidation_Test is
    OPContractsManagerStandardValidator_ZKMode_TestInit
{
    /// @notice Tests that validate succeeds when the ZK game is properly configured.
    function test_validate_zkDisputeGame_succeeds() public view {
        string memory errors = _validate(false);
        assertEq(errors, "");
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

    /// @notice Tests ZKDG-60 when the l2ChainId encoded in the ZK game args does not match.
    function test_validate_zkDisputeGameWrongChainId_succeeds() public {
        DisputeGames.mockZKGameImplL2ChainId(dgf, GameTypes.ZK_DISPUTE_GAME, l2ChainId + 1);
        assertEq("ZKDG-60", _validate(true));
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
