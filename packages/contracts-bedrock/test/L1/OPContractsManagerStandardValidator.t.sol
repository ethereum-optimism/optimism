// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { StandardConstants } from "scripts/deploy/StandardConstants.sol";

// Libraries
import { GameTypes, Duration, Claim } from "src/dispute/lib/Types.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
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
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

/// @title BadDisputeGameFactoryReturner
/// @notice Used to return a bad DisputeGameFactory address to the OPContractsManagerStandardValidator. Far easier
///         than the alternative ways of mocking this value since the normal vm.mockCall will cause
///         the validation function to revert.
contract BadDisputeGameFactoryReturner {
    /// @notice Address of the OPContractsManagerStandardValidator instance.
    IOPContractsManagerStandardValidator public immutable validator;

    /// @notice Address of the real DisputeGameFactory instance.
    IDisputeGameFactory public immutable realDisputeGameFactory;

    /// @notice Address of the fake DisputeGameFactory instance.
    IDisputeGameFactory public immutable fakeDisputeGameFactory;

    /// @param _validator The OPContractsManagerStandardValidator instance.
    /// @param _realDisputeGameFactory The real DisputeGameFactory instance.
    /// @param _fakeDisputeGameFactory The fake DisputeGameFactory instance.
    constructor(
        IOPContractsManagerStandardValidator _validator,
        IDisputeGameFactory _realDisputeGameFactory,
        IDisputeGameFactory _fakeDisputeGameFactory
    ) {
        validator = _validator;
        realDisputeGameFactory = _realDisputeGameFactory;
        fakeDisputeGameFactory = _fakeDisputeGameFactory;
    }

    /// @notice Returns the real or fake DisputeGameFactory address.
    function disputeGameFactory() external view returns (IDisputeGameFactory) {
        if (msg.sender == address(validator)) {
            return fakeDisputeGameFactory;
        } else {
            return realDisputeGameFactory;
        }
    }
}

/// @title OPContractsManagerStandardValidator_TestInit
/// @notice Base contract for `OPContractsManagerStandardValidator` tests, handles common setup.
contract OPContractsManagerStandardValidator_TestInit is CommonTest {
    /// @notice Deploy input that was used to deploy the contracts being tested.
    IOPContractsManager.DeployInput deployInput;

    /// @notice The l2ChainId, either from config or from registry if fork test.
    uint256 l2ChainId;

    /// @notice The absolute prestate, either from config or dummy value if fork test.
    Claim absolutePrestate;

    /// @notice The PermissionedDisputeGame instance.
    IPermissionedDisputeGame pdg;

    /// @notice The FaultDisputeGame instance.
    IFaultDisputeGame fdg;

    /// @notice The PreimageOracle instance.
    IPreimageOracle preimageOracle;

    /// @notice The BadDisputeGameFactoryReturner instance.
    BadDisputeGameFactoryReturner badDisputeGameFactoryReturner;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        // Grab the deploy input for later use.
        deployInput = deploy.getDeployInput();

        // Load the PermissionedDisputeGame once, we'll need it later.
        pdg = IPermissionedDisputeGame(artifacts.mustGetAddress("PermissionedDisputeGame"));

        // Load the PreimageOracle once, we'll need it later.
        preimageOracle = IPreimageOracle(artifacts.mustGetAddress("PreimageOracle"));

        // Values are slightly different for fork tests vs local tests. Most we can get from
        // reasonable sources, challenger we need to get from live system because there's no other
        // consistent way to get it right now. Means we're cheating a tiny bit for the challenger
        // address in fork tests but it's fine.
        if (isForkTest()) {
            l2ChainId = uint256(uint160(address(artifacts.mustGetAddress("L2ChainId"))));
            absolutePrestate = Claim.wrap(bytes32(keccak256("absolutePrestate")));

            vm.mockCall(
                address(proxyAdmin),
                abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1OptimismMintableERC20Factory))),
                abi.encode(opcm.opcmStandardValidator().optimismMintableERC20FactoryImpl())
            );
            vm.mockCall(
                address(pdg),
                abi.encodeCall(IPermissionedDisputeGame.challenger, ()),
                abi.encode(opcm.opcmStandardValidator().challenger())
            );
            vm.mockCall(
                address(proxyAdmin),
                abi.encodeCall(IProxyAdmin.owner, ()),
                abi.encode(opcm.opcmStandardValidator().l1PAOMultisig())
            );
            vm.mockCall(
                address(delayedWeth),
                abi.encodeCall(IDelayedWETH.proxyAdminOwner, ()),
                abi.encode(opcm.opcmStandardValidator().l1PAOMultisig())
            );
            // Use vm.store so that the .setImplementation call below works.
            vm.store(
                address(disputeGameFactory),
                // this assumes that it is not packed with any other value
                bytes32(ForgeArtifacts.getSlot("DisputeGameFactory", "_owner").slot),
                bytes32(uint256(uint160(opcm.opcmStandardValidator().l1PAOMultisig())))
            );
        } else {
            l2ChainId = deployInput.l2ChainId;
            absolutePrestate = deployInput.disputeAbsolutePrestate;
        }

        // Deploy the BadDisputeGameFactoryReturner once.
        badDisputeGameFactoryReturner = new BadDisputeGameFactoryReturner(
            opcm.opcmStandardValidator(), disputeGameFactory, IDisputeGameFactory(address(0xbad))
        );

        if (isForkTest()) {
            // Load the FaultDisputeGame once, we'll need it later.
            fdg = IFaultDisputeGame(artifacts.mustGetAddress("FaultDisputeGame"));
        } else {
            // Deploy the FaultDisputeGame.
            fdg = IFaultDisputeGame(
                DeployUtils.create1({
                    _name: "FaultDisputeGame",
                    _args: DeployUtils.encodeConstructor(
                        abi.encodeCall(
                            IFaultDisputeGame.__constructor__,
                            (
                                IFaultDisputeGame.GameConstructorParams({
                                    gameType: GameTypes.CANNON,
                                    absolutePrestate: absolutePrestate,
                                    maxGameDepth: 73,
                                    splitDepth: 30,
                                    clockExtension: Duration.wrap(10800),
                                    maxClockDuration: Duration.wrap(302400),
                                    vm: mips,
                                    weth: delayedWeth,
                                    anchorStateRegistry: anchorStateRegistry,
                                    l2ChainId: l2ChainId
                                })
                            )
                        )
                    )
                })
            );
        }

        // Add the FaultDisputeGame to the DisputeGameFactory.
        vm.prank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(GameTypes.CANNON, IDisputeGame(address(fdg)));
    }

    /// @notice Runs the OPContractsManagerStandardValidator.validate function.
    /// @param _allowFailure Whether to allow failure.
    /// @return The error message(s) from the validate function.
    function _validate(bool _allowFailure) internal view returns (string memory) {
        return opcm.validate(
            IOPContractsManagerStandardValidator.ValidationInput({
                proxyAdmin: proxyAdmin,
                sysCfg: systemConfig,
                absolutePrestate: absolutePrestate.raw(),
                l2ChainID: l2ChainId
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
        return opcm.validateWithOverrides(
            IOPContractsManagerStandardValidator.ValidationInput({
                proxyAdmin: proxyAdmin,
                sysCfg: systemConfig,
                absolutePrestate: absolutePrestate.raw(),
                l2ChainID: l2ChainId
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
            "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,PROXYA-10,DF-30,PDDG-DWETH-30,PDDG-130,PLDG-DWETH-30",
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
            address(delayedWeth), abi.encodeCall(IDelayedWETH.proxyAdminOwner, ()), abi.encode(overrides.l1PAOMultisig)
        );
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(overrides.l1PAOMultisig));
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.owner, ()),
            abi.encode(overrides.l1PAOMultisig)
        );
        vm.mockCall(
            address(pdg), abi.encodeCall(IPermissionedDisputeGame.challenger, ()), abi.encode(overrides.challenger)
        );

        assertEq("OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER", _validateWithOverrides(true, overrides));
    }

    /// @notice Tests that the validate function (with overrides) and allow failure set to false,
    ///         returns the errors with the overrides prepended.
    function test_validateOverrides_notAllowFailurePrependsOverrides_succeeds() public {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = IOPContractsManagerStandardValidator
            .ValidationOverrides({ l1PAOMultisig: address(0xbad), challenger: address(0xc0ffee) });

        vm.expectRevert(
            bytes(
                "OPContractsManagerStandardValidator: OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,PROXYA-10,DF-30,PDDG-DWETH-30,PDDG-130,PLDG-DWETH-30"
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
        vm.mockCall(address(delayedWeth), abi.encodeCall(IDelayedWETH.proxyAdminOwner, ()), abi.encode(address(0xbad)));
        assertEq("PROXYA-10,PDDG-DWETH-30,PLDG-DWETH-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right overrides error
    ///         when the ProxyAdmin owner is overridden but is correct.
    function test_validate_overridenProxyAdminOwner_succeeds() public {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.l1PAOMultisig = address(0xbad);
        vm.mockCall(address(delayedWeth), abi.encodeCall(IDelayedWETH.proxyAdminOwner, ()), abi.encode(0xbad));
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
            "OVERRIDES-L1PAOMULTISIG,PROXYA-10,DF-30,PDDG-DWETH-30,PLDG-DWETH-30",
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
    ///         SystemConfig proxyAdmin is invalid.
    function test_validate_systemConfigInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(systemConfig), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("SYSCON-130", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         SystemConfig superchainConfig is invalid.
    function test_validate_systemConfigInvalidSuperchainConfig_succeeds() public {
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.superchainConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("SYSCON-140", _validate(true));
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
        assertEq("LOCKBOX-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox implementation is invalid.
    function test_validate_ethLockboxInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(ethLockbox))),
            abi.encode(address(0xbad))
        );
        assertEq("LOCKBOX-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox proxyAdmin is invalid.
    function test_validate_ethLockboxInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(ethLockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );
        assertEq("LOCKBOX-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox systemConfig is invalid.
    function test_validate_ethLockboxInvalidSystemConfig_succeeds() public {
        vm.mockCall(address(ethLockbox), abi.encodeCall(IETHLockbox.systemConfig, ()), abi.encode(address(0xbad)));
        assertEq("LOCKBOX-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         ETHLockbox does not have the OptimismPortal as an authorized portal.
    function test_validate_ethLockboxPortalUnauthorized_succeeds() public {
        vm.mockCall(
            address(ethLockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (optimismPortal2)), abi.encode(false)
        );
        assertEq("LOCKBOX-50", _validate(true));
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
        assertEq("PDDG-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame version is invalid.
    function test_validate_permissionedDisputeGameInvalidVersion_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        assertEq("PDDG-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame game type is invalid.
    function test_validate_permissionedDisputeGameInvalidGameType_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IDisputeGame.gameType, ()), abi.encode(GameTypes.CANNON));
        assertEq("PDDG-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame absolute prestate is invalid.
    function test_validate_permissionedDisputeGameInvalidAbsolutePrestate_succeeds() public {
        vm.mockCall(
            address(pdg),
            abi.encodeCall(IPermissionedDisputeGame.absolutePrestate, ()),
            abi.encode(bytes32(uint256(0xbad)))
        );
        assertEq("PDDG-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame VM address is invalid.
    function test_validate_permissionedDisputeGameInvalidVM_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.vm, ()), abi.encode(address(0xbad)));
        vm.mockCall(address(0xbad), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(
            address(0xbad), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION)
        );
        assertEq("PDDG-VM-10,PDDG-VM-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame VM's state version is invalid.
    function test_validate_permissionedDisputeGameInvalidVMStateVersion_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.vm, ()), abi.encode(address(mips)));
        vm.mockCall(address(mips), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(6));
        assertEq("PDDG-VM-30,PLDG-VM-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame L2 Chain ID is invalid.
    function test_validate_permissionedDisputeGameInvalidL2ChainId_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.l2ChainId, ()), abi.encode(l2ChainId + 1));
        assertEq("PDDG-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame L2 Sequence Number is invalid.
    function test_validate_permissionedDisputeGameInvalidL2SequenceNumber_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IDisputeGame.l2SequenceNumber, ()), abi.encode(123));
        assertEq("PDDG-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame clockExtension is invalid.
    function test_validate_permissionedDisputeGameInvalidClockExtension_succeeds() public {
        vm.mockCall(
            address(pdg), abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()), abi.encode(Duration.wrap(1000))
        );
        assertEq("PDDG-80", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame splitDepth is invalid.
    function test_validate_permissionedDisputeGameInvalidSplitDepth_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()), abi.encode(20));
        assertEq("PDDG-90", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame maxGameDepth is invalid.
    function test_validate_permissionedDisputeGameInvalidMaxGameDepth_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()), abi.encode(50));
        assertEq("PDDG-100", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame maxClockDuration is invalid.
    function test_validate_permissionedDisputeGameInvalidMaxClockDuration_succeeds() public {
        vm.mockCall(
            address(pdg), abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()), abi.encode(Duration.wrap(1000))
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
        assertEq("PDDG-120,PLDG-120", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PermissionedDisputeGame challenger is invalid.
    function test_validate_permissionedDisputeGameInvalidChallenger_succeeds() public {
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.challenger, ()), abi.encode(address(0xbad)));
        assertEq("PDDG-130", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right overrides error when the
    ///         PermissionedDisputeGame challenger is overridden but is correct.
    function test_validate_overridenPermissionedDisputeGameChallenger_succeeds() public {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.challenger = address(0xbad);
        vm.mockCall(address(pdg), abi.encodeCall(IPermissionedDisputeGame.challenger, ()), abi.encode(address(0xbad)));
        assertEq("OVERRIDES-CHALLENGER", _validateWithOverrides(true, overrides));
    }

    /// @notice Tests that the validate function (with an overridden PermissionedDisputeGame challenger) successfully
    ///         returns the right error when the PermissionedDisputeGame challenger is invalid.
    function test_validateOverridesChallenger_permissionedDisputeGameInvalidChallenger_succeeds() public view {
        IOPContractsManagerStandardValidator.ValidationOverrides memory overrides = _defaultValidationOverrides();
        overrides.challenger = address(0xbad);
        assertEq("OVERRIDES-CHALLENGER,PDDG-130", _validateWithOverrides(true, overrides));
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
        assertEq("PDDG-ANCHORP-10,PLDG-ANCHORP-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry implementation is invalid.
    function test_validate_anchorStateRegistryInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(anchorStateRegistry))),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-20,PLDG-ANCHORP-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry disputeGameFactory is invalid.
    function test_validate_anchorStateRegistryInvalidDisputeGameFactory_succeeds() public {
        vm.mockFunction(
            address(anchorStateRegistry),
            address(badDisputeGameFactoryReturner),
            abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ())
        );
        assertEq("PDDG-ANCHORP-30,PLDG-ANCHORP-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry systemConfig is invalid.
    function test_validate_anchorStateRegistryInvalidSystemConfig_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IAnchorStateRegistry.systemConfig, ()),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-40,PLDG-ANCHORP-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry proxyAdmin is invalid.
    function test_validate_anchorStateRegistryInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-50,PLDG-ANCHORP-50", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         AnchorStateRegistry retirementTimestamp is invalid.
    function test_validate_anchorStateRegistryInvalidRetirementTimestamp_succeeds() public {
        vm.mockCall(
            address(anchorStateRegistry), abi.encodeCall(IAnchorStateRegistry.retirementTimestamp, ()), abi.encode(0)
        );
        assertEq("PDDG-ANCHORP-60,PLDG-ANCHORP-60", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_DelayedWETH_Test
/// @notice Tests validation of `DelayedWETH` configuration
contract OPContractsManagerStandardValidator_DelayedWETH_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH version is invalid.
    function test_validate_delayedWETHInvalidVersion_succeeds() public {
        vm.mockCall(address(delayedWeth), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.1"));

        // One last mess here, during local tests delayedWeth refers to the contract attached to
        // the FaultDisputeGame, but during fork tests it refers to the one attached to the
        // PermissionedDisputeGame. We'll just branch based on the test type.
        if (isForkTest()) {
            assertEq("PDDG-DWETH-10", _validate(true));
        } else {
            assertEq("PLDG-DWETH-10", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH implementation is invalid.
    function test_validate_delayedWETHInvalidImplementation_succeeds() public {
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(delayedWeth))),
            abi.encode(address(0xbad))
        );

        if (isForkTest()) {
            assertEq("PDDG-DWETH-20", _validate(true));
        } else {
            assertEq("PLDG-DWETH-20", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH proxyAdminOwner is invalid.
    function test_validate_delayedWETHInvalidProxyAdminOwner_succeeds() public {
        vm.mockCall(
            address(delayedWeth), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0xbad))
        );

        if (isForkTest()) {
            assertEq("PDDG-DWETH-30", _validate(true));
        } else {
            assertEq("PLDG-DWETH-30", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH delay is invalid.
    function test_validate_delayedWETHInvalidDelay_succeeds() public {
        vm.mockCall(address(delayedWeth), abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(1000));

        if (isForkTest()) {
            assertEq("PDDG-DWETH-40", _validate(true));
        } else {
            assertEq("PLDG-DWETH-40", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH systemConfig is invalid.
    function test_validate_delayedWETHInvalidSystemConfig_succeeds() public {
        vm.mockCall(address(delayedWeth), abi.encodeCall(IDelayedWETH.systemConfig, ()), abi.encode(address(0xbad)));

        if (isForkTest()) {
            assertEq("PDDG-DWETH-50", _validate(true));
        } else {
            assertEq("PLDG-DWETH-50", _validate(true));
        }
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         DelayedWETH proxyAdmin is invalid.
    function test_validate_delayedWETHInvalidProxyAdmin_succeeds() public {
        vm.mockCall(
            address(delayedWeth), abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()), abi.encode(address(0xbad))
        );

        if (isForkTest()) {
            assertEq("PDDG-DWETH-60", _validate(true));
        } else {
            assertEq("PLDG-DWETH-60", _validate(true));
        }
    }
}

/// @title OPContractsManagerStandardValidator_PreimageOracle_Test
/// @notice Tests validation of `PreimageOracle` configuration
contract OPContractsManagerStandardValidator_PreimageOracle_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PreimageOracle version is invalid.
    function test_validate_preimageOracleInvalidVersion_succeeds() public {
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.1"));
        assertEq("PDDG-PIMGO-10,PLDG-PIMGO-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PreimageOracle challengePeriod is invalid.
    function test_validate_preimageOracleInvalidChallengePeriod_succeeds() public {
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.challengePeriod, ()), abi.encode(1000));
        assertEq("PDDG-PIMGO-20,PLDG-PIMGO-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         PreimageOracle minProposalSize is invalid.
    function test_validate_preimageOracleInvalidMinProposalSize_succeeds() public {
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.minProposalSize, ()), abi.encode(1000));
        assertEq("PDDG-PIMGO-30,PLDG-PIMGO-30", _validate(true));
    }
}

/// @title OPContractsManagerStandardValidator_FaultDisputeGame_Test
/// @notice Tests validation of `FaultDisputeGame` configuration
contract OPContractsManagerStandardValidator_FaultDisputeGame_Test is OPContractsManagerStandardValidator_TestInit {
    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) implementation is null.
    function test_validate_faultDisputeGameNullImplementation_succeeds() public {
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(address(0))
        );
        assertEq("PLDG-10", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) version is invalid.
    function test_validate_faultDisputeGameInvalidVersion_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        assertEq("PLDG-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) game type is invalid.
    function test_validate_faultDisputeGameInvalidGameType_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IDisputeGame.gameType, ()), abi.encode(GameTypes.PERMISSIONED_CANNON));
        assertEq("PLDG-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) absolute prestate is invalid.
    function test_validate_faultDisputeGameInvalidAbsolutePrestate_succeeds() public {
        vm.mockCall(
            address(fdg),
            abi.encodeCall(IFaultDisputeGame.absolutePrestate, ()),
            abi.encode(bytes32(uint256(0xbadbad))) // Different from the expected absolutePrestate
        );
        assertEq("PLDG-40", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) VM address is invalid.
    function test_validate_faultDisputeGameInvalidVM_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IFaultDisputeGame.vm, ()), abi.encode(address(0xbad)));
        vm.mockCall(address(0xbad), abi.encodeCall(ISemver.version, ()), abi.encode("0.0.0"));
        vm.mockCall(
            address(0xbad), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(StandardConstants.MIPS_VERSION)
        );
        assertEq("PLDG-VM-10,PLDG-VM-20", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) VM's state version is invalid.
    function test_validate_faultDisputeGameInvalidVMStateVersion_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IFaultDisputeGame.vm, ()), abi.encode(address(mips)));
        vm.mockCall(address(mips), abi.encodeCall(IMIPS64.stateVersion, ()), abi.encode(6));
        assertEq("PDDG-VM-30,PLDG-VM-30", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) L2 Chain ID is invalid.
    function test_validate_faultDisputeGameInvalidL2ChainId_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IFaultDisputeGame.l2ChainId, ()), abi.encode(l2ChainId + 1));
        assertEq("PLDG-60", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) L2 Sequence Number is invalid.
    function test_validate_faultDisputeGameInvalidL2SequenceNumber_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IDisputeGame.l2SequenceNumber, ()), abi.encode(123));
        assertEq("PLDG-70", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) clockExtension is invalid.
    function test_validate_faultDisputeGameInvalidClockExtension_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IFaultDisputeGame.clockExtension, ()), abi.encode(Duration.wrap(1000)));
        assertEq("PLDG-80", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) splitDepth is invalid.
    function test_validate_faultDisputeGameInvalidSplitDepth_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IFaultDisputeGame.splitDepth, ()), abi.encode(20));
        assertEq("PLDG-90", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) maxGameDepth is invalid.
    function test_validate_faultDisputeGameInvalidMaxGameDepth_succeeds() public {
        vm.mockCall(address(fdg), abi.encodeCall(IFaultDisputeGame.maxGameDepth, ()), abi.encode(50));
        assertEq("PLDG-100", _validate(true));
    }

    /// @notice Tests that the validate function successfully returns the right error when the
    ///         FaultDisputeGame (permissionless) maxClockDuration is invalid.
    function test_validate_faultDisputeGameInvalidMaxClockDuration_succeeds() public {
        vm.mockCall(
            address(fdg), abi.encodeCall(IFaultDisputeGame.maxClockDuration, ()), abi.encode(Duration.wrap(1000))
        );
        assertEq("PLDG-110", _validate(true));
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
        assertTrue(bytes(opcm.opcmStandardValidator().systemConfigVersion()).length > 0, "systemConfigVersion empty");
        assertTrue(
            bytes(opcm.opcmStandardValidator().optimismPortalVersion()).length > 0, "optimismPortalVersion empty"
        );
        assertTrue(
            bytes(opcm.opcmStandardValidator().l1CrossDomainMessengerVersion()).length > 0,
            "l1CrossDomainMessengerVersion empty"
        );
        assertTrue(
            bytes(opcm.opcmStandardValidator().l1ERC721BridgeVersion()).length > 0, "l1ERC721BridgeVersion empty"
        );
        assertTrue(
            bytes(opcm.opcmStandardValidator().l1StandardBridgeVersion()).length > 0, "l1StandardBridgeVersion empty"
        );
        assertTrue(bytes(opcm.opcmStandardValidator().mipsVersion()).length > 0, "mipsVersion empty");
        assertTrue(
            bytes(opcm.opcmStandardValidator().optimismMintableERC20FactoryVersion()).length > 0,
            "optimismMintableERC20FactoryVersion empty"
        );
        assertTrue(
            bytes(opcm.opcmStandardValidator().disputeGameFactoryVersion()).length > 0,
            "disputeGameFactoryVersion empty"
        );
        assertTrue(
            bytes(opcm.opcmStandardValidator().anchorStateRegistryVersion()).length > 0,
            "anchorStateRegistryVersion empty"
        );
        assertTrue(bytes(opcm.opcmStandardValidator().delayedWETHVersion()).length > 0, "delayedWETHVersion empty");
        assertTrue(
            bytes(opcm.opcmStandardValidator().permissionedDisputeGameVersion()).length > 0,
            "permissionedDisputeGameVersion empty"
        );
        assertTrue(
            bytes(opcm.opcmStandardValidator().preimageOracleVersion()).length > 0, "preimageOracleVersion empty"
        );
        assertTrue(bytes(opcm.opcmStandardValidator().ethLockboxVersion()).length > 0, "ethLockboxVersion empty");
    }
}
