// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract
import { StandardValidator } from "src/L1/StandardValidator.sol";

// Libraries
import { GameType, GameTypes, Hash } from "src/dispute/lib/Types.sol";
import { Duration } from "src/dispute/lib/LibUDT.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { OPContractsManager } from "src/L1/OPContractsManager.sol";

contract StandardValidatorTest is CommonTest {
    StandardValidator validator;
    address l1PAOMultisig;
    address mips;
    address guardian;
    address challenger;

    bytes32 absolutePrestate;
    uint256 l2ChainID;

    address preimageOracle;

    function setUp() public virtual override {
        super.setUp();

        // Setup test addresses
        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        l1PAOMultisig = makeAddr("l1PAOMultisig");
        guardian = makeAddr("guardian");
        challenger = makeAddr("challenger");

        // Mock superchainConfig calls needed in setup
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.guardian, ()), abi.encode(guardian));
        vm.mockCall(
            address(superchainConfig), abi.encodeCall(ISuperchainConfig.paused, (address(0))), abi.encode(false)
        );

        // Setup mock contracts for validation
        vm.prank(address(0));
        proxyAdmin = IProxyAdmin(IProxy(payable(address(systemConfig))).admin());

        absolutePrestate = bytes32(uint256(0xdead));
        l2ChainID = 10;

        // Setup mock dependency addresses
        permissionedDisputeGame = IPermissionedDisputeGame(
            address(IDisputeGameFactory(disputeGameFactory).gameImpls(GameTypes.PERMISSIONED_CANNON))
        );
        faultDisputeGame =
            IFaultDisputeGame(address(IDisputeGameFactory(disputeGameFactory).gameImpls(GameTypes.CANNON)));
        delayedWETHPermissionedGameProxy = IDelayedWETH(IFaultDisputeGame(address(permissionedDisputeGame)).weth());
        delayedWeth = IDelayedWETH(IFaultDisputeGame(address(faultDisputeGame)).weth());
        mips = address(IFaultDisputeGame(address(permissionedDisputeGame)).vm());
        preimageOracle = address(IBigStepper(mips).oracle());

        // Mock proxyAdmin owner
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(l1PAOMultisig));

        // Get the OPContractsManager and its implementations struct
        OPContractsManager opcm = OPContractsManager(artifacts.mustGetAddress("OPContractsManager"));
        OPContractsManager.Implementations memory impls = opcm.implementations();

        // Deploy validator with implementations from OPCM
        validator = new StandardValidator(
            StandardValidator.Implementations({
                systemConfigImpl: impls.systemConfigImpl,
                optimismPortalImpl: impls.optimismPortalImpl,
                l1CrossDomainMessengerImpl: impls.l1CrossDomainMessengerImpl,
                l1StandardBridgeImpl: impls.l1StandardBridgeImpl,
                l1ERC721BridgeImpl: impls.l1ERC721BridgeImpl,
                optimismMintableERC20FactoryImpl: impls.optimismMintableERC20FactoryImpl,
                disputeGameFactoryImpl: impls.disputeGameFactoryImpl,
                mipsImpl: impls.mipsImpl,
                anchorStateRegistryImpl: impls.anchorStateRegistryImpl,
                delayedWETHImpl: impls.delayedWETHImpl
            }),
            superchainConfig,
            l1PAOMultisig,
            challenger,
            302400
        );
    }

    /// @notice Tests that validation succeeds with valid inputs and mocked dependencies
    function test_validate_allowFailureTrue_succeeds() public {
        // Mock all necessary calls for validation
        _mockValidationCalls();

        // Validate with allowFailure = true
        string memory errors = validate(true);
        assertEq(errors, "");
    }

    /// @notice Tests validation of SuperchainConfig
    function test_validate_superchainConfig_succeeds() public {
        // Test invalid paused
        _mockValidationCalls();
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.paused, (address(0))), abi.encode(true));
        assertEq("SPRCFG-10", validate(true));
    }

    /// @notice Tests that validation fails with invalid proxy admin owner
    function test_validate_proxyAdmin_succeeds() public {
        _mockValidationCalls();
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(address(0xbad)));

        // Mocking the proxy admin like this will also break ownership checks
        // for the DGF, DelayedWETH, and other contracts.
        assertEq("PROXYA-10", validate(true));
    }

    /// @notice Tests validation of SystemConfig
    function test_validate_systemConfig_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("SYSCON-10", validate(true));

        // Test invalid gas limit
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.gasLimit, ()), abi.encode(uint64(200_000_001)));
        assertEq("SYSCON-20", validate(true));

        // Test invalid scalar
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.scalar, ()), abi.encode(uint256(2) << 248));
        assertEq("SYSCON-30", validate(true));

        // Test invalid proxy implementation
        _mockValidationCalls();
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(systemConfig))),
            abi.encode(address(0xbad))
        );
        assertEq("SYSCON-40", validate(true));

        // Test invalid resource config - maxResourceLimit
        _mockValidationCalls();
        IResourceMetering.ResourceConfig memory badConfig = IResourceMetering.ResourceConfig({
            maxResourceLimit: 1_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 1 gwei,
            maximumBaseFee: type(uint128).max
        });
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-50", validate(true));

        // Test invalid resource config - elasticityMultiplier
        _mockValidationCalls();
        badConfig.maxResourceLimit = 20_000_000;
        badConfig.elasticityMultiplier = 5;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-60", validate(true));

        // Test invalid resource config - baseFeeMaxChangeDenominator
        _mockValidationCalls();
        badConfig.elasticityMultiplier = 10;
        badConfig.baseFeeMaxChangeDenominator = 4;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-70", validate(true));

        // Test invalid resource config - systemTxMaxGas
        _mockValidationCalls();
        badConfig.baseFeeMaxChangeDenominator = 8;
        badConfig.systemTxMaxGas = 500_000;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-80", validate(true));

        // Test invalid resource config - minimumBaseFee
        _mockValidationCalls();
        badConfig.systemTxMaxGas = 1_000_000;
        badConfig.minimumBaseFee = 2 gwei;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-90", validate(true));

        // Test invalid resource config - maximumBaseFee
        _mockValidationCalls();
        badConfig.minimumBaseFee = 1 gwei;
        badConfig.maximumBaseFee = 1_000_000;
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(badConfig));
        assertEq("SYSCON-100", validate(true));
    }

    /// @notice Tests validation of L1CrossDomainMessenger
    function test_validate_l1CrossDomainMessenger_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(l1CrossDomainMessenger), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("L1xDM-10", validate(true));

        // Test invalid OTHER_MESSENGER
        _mockValidationCalls();
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.OTHER_MESSENGER, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-30", validate(true));

        // Test invalid otherMessenger
        _mockValidationCalls();
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.otherMessenger, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-40", validate(true));

        // Test invalid PORTAL
        _mockValidationCalls();
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.PORTAL, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-50", validate(true));

        // Test invalid portal
        _mockValidationCalls();
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.portal, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-60", validate(true));

        // Test invalid systemConfig
        _mockValidationCalls();
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-70", validate(true));
    }

    /// @notice Tests validation of OptimismMintableERC20Factory
    function test_validate_optimismMintableERC20Factory_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(l1OptimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("MERC20F-10", validate(true));

        // Test invalid BRIDGE
        _mockValidationCalls();
        vm.mockCall(
            address(l1OptimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.BRIDGE, ()),
            abi.encode(address(0xbad))
        );
        assertEq("MERC20F-30", validate(true));

        // Test invalid bridge
        _mockValidationCalls();
        vm.mockCall(
            address(l1OptimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.bridge, ()),
            abi.encode(address(0xbad))
        );
        assertEq("MERC20F-40", validate(true));
    }

    /// @notice Tests validation of L1ERC721Bridge
    function test_validate_l1ERC721Bridge_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("L721B-10", validate(true));

        // Test invalid OTHER_BRIDGE
        _mockValidationCalls();
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.OTHER_BRIDGE, ()), abi.encode(address(0xbad)));
        assertEq("L721B-30", validate(true));

        // Test invalid otherBridge
        _mockValidationCalls();
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.otherBridge, ()), abi.encode(address(0xbad)));
        assertEq("L721B-40", validate(true));

        // Test invalid MESSENGER
        _mockValidationCalls();
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.MESSENGER, ()), abi.encode(address(0xbad)));
        assertEq("L721B-50", validate(true));

        // Test invalid messenger
        _mockValidationCalls();
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.messenger, ()), abi.encode(address(0xbad)));
        assertEq("L721B-60", validate(true));

        // Test invalid systemConfig
        _mockValidationCalls();
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IL1ERC721Bridge.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("L721B-70", validate(true));
    }

    /// @notice Tests validation of OptimismPortal
    function test_validate_optimismPortal_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(optimismPortal2), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("PORTAL-10", validate(true));

        // Test invalid disputeGameFactory
        _mockValidationCalls();
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.disputeGameFactory, ()),
            abi.encode(address(0xbad))
        );
        assertEq("PORTAL-30", validate(true));

        // Test invalid systemConfig
        _mockValidationCalls();
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("PORTAL-40", validate(true));

        // Test invalid l2Sender
        _mockValidationCalls();
        vm.mockCall(address(optimismPortal2), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(address(0xbad)));
        assertEq("PORTAL-80", validate(true));
    }

    /// @notice Tests validation of DisputeGameFactory
    function test_validate_disputeGameFactory_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("0.9.0"));
        assertEq("DF-10", validate(true));

        // Test invalid implementation
        _mockValidationCalls();
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(disputeGameFactory))),
            abi.encode(address(0xbad))
        );
        assertEq("DF-20", validate(true));

        // Test invalid owner
        _mockValidationCalls();
        vm.mockCall(
            address(disputeGameFactory), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(address(0xbad))
        );
        assertEq("DF-30", validate(true));
    }

    /// @notice Tests validation of PermissionedDisputeGame. The ASR, PreimageOracle, and DelayedWETH are
    /// validated for each PDG and so are included here.
    function test_validate_permissionedDisputeGame_succeeds() public {
        _testDisputeGame(
            "PDDG",
            address(permissionedDisputeGame),
            anchorStateRegistry,
            delayedWETHPermissionedGameProxy,
            GameTypes.PERMISSIONED_CANNON
        );
    }

    function test_validate_permissionlessDisputeGame_succeeds() public {
        _testDisputeGame("PLDG", address(faultDisputeGame), anchorStateRegistry, delayedWeth, GameTypes.CANNON);
    }

    /// @notice Tests validation of L1StandardBridge
    function test_validate_l1StandardBridge_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(l1StandardBridge), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("L1SB-10", validate(true));

        // Test invalid MESSENGER
        _mockValidationCalls();
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.MESSENGER, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-30", validate(true));

        // Test invalid messenger
        _mockValidationCalls();
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.messenger, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-40", validate(true));

        // Test invalid OTHER_BRIDGE
        _mockValidationCalls();
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.OTHER_BRIDGE, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-50", validate(true));

        // Test invalid otherBridge
        _mockValidationCalls();
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.otherBridge, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-60", validate(true));

        // Test invalid systemConfig
        _mockValidationCalls();
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IL1StandardBridge.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("L1SB-70", validate(true));
    }

    function _testDisputeGame(
        string memory errorPrefix,
        address _disputeGame,
        IAnchorStateRegistry _asr,
        IDelayedWETH _weth,
        GameType _gameType
    )
        public
        virtual
    {
        // Test null implementation
        _mockValidationCalls();
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (_gameType)),
            abi.encode(address(0))
        );
        assertEq(string.concat(errorPrefix, "-10"), validate(true));

        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq(string.concat(errorPrefix, "-20"), validate(true));

        // Test invalid game type
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(IDisputeGame.gameType, ()), abi.encode(GameType.wrap(123)));
        assertEq(string.concat(errorPrefix, "-30"), validate(true));

        // Test invalid absolute prestate
        _mockValidationCalls();
        vm.mockCall(
            address(_disputeGame),
            abi.encodeCall(IPermissionedDisputeGame.absolutePrestate, ()),
            abi.encode(bytes32(uint256(0xbad)))
        );
        assertEq(string.concat(errorPrefix, "-40"), validate(true));

        // Test invalid vm
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.vm, ()), abi.encode(address(0xbad)));
        assertEq(string.concat(errorPrefix, "-50"), validate(true));

        // Test invalid l2ChainId
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.l2ChainId, ()), abi.encode(123));
        assertEq(string.concat(errorPrefix, "-60"), validate(true));

        // Test invalid l2BlockNumber
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.l2BlockNumber, ()), abi.encode(1));
        assertEq(string.concat(errorPrefix, "-70"), validate(true));

        // Test invalid clockExtension
        _mockValidationCalls();
        vm.mockCall(
            address(_disputeGame),
            abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()),
            abi.encode(Duration.wrap(1000))
        );
        assertEq(string.concat(errorPrefix, "-80"), validate(true));

        // Test invalid splitDepth
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()), abi.encode(20));
        assertEq(string.concat(errorPrefix, "-90"), validate(true));

        // Test invalid maxGameDepth
        _mockValidationCalls();
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()), abi.encode(50));
        assertEq(string.concat(errorPrefix, "-100"), validate(true));

        // Test invalid maxClockDuration
        _mockValidationCalls();
        vm.mockCall(
            address(_disputeGame),
            abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()),
            abi.encode(Duration.wrap(1000))
        );
        assertEq(string.concat(errorPrefix, "-110"), validate(true));

        if (_gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
            _mockValidationCalls();
            vm.mockCall(
                address(_disputeGame),
                abi.encodeCall(IPermissionedDisputeGame.challenger, ()),
                abi.encode(address(0xbad))
            );
            assertEq(string.concat(errorPrefix, "-120"), validate(true));
        }

        // Test invalid anchor state registry version
        _mockValidationCalls();
        vm.mockCall(address(_asr), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("PDDG-ANCHORP-10,PLDG-ANCHORP-10", validate(true));

        // Test invalid anchor state registry factory
        _mockValidationCalls();
        vm.mockCall(
            address(_asr), abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-30,PLDG-ANCHORP-30", validate(true));

        // Test invalid DelayedWETH version
        _mockValidationCalls();
        vm.mockCall(address(_weth), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq(string.concat(errorPrefix, "-DWETH-10"), validate(true));

        // Test invalid DelayedWETH implementation for permissioned game
        _mockValidationCalls();
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(_weth))),
            abi.encode(address(0xbad))
        );
        assertEq(string.concat(errorPrefix, "-DWETH-20"), validate(true));

        // Test invalid DelayedWETH owner
        _mockValidationCalls();
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.proxyAdminOwner, ()), abi.encode(address(0xbad)));
        assertEq(string.concat(errorPrefix, "-DWETH-30"), validate(true));

        // Test invalid DelayedWETH delay
        _mockValidationCalls();
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(2));
        assertEq(string.concat(errorPrefix, "-DWETH-40"), validate(true));

        // Since the preimage oracle is shared, the errors need to include both
        // the permissioned and permissionless game type.

        // Test invalid PreimageOracle version
        _mockValidationCalls();
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("PDDG-PIMGO-10,PLDG-PIMGO-10", validate(true));

        // Test invalid PreimageOracle challenge period
        _mockValidationCalls();
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.challengePeriod, ()), abi.encode(1000));
        assertEq("PDDG-PIMGO-20,PLDG-PIMGO-20", validate(true));

        // Test invalid PreimageOracle min proposal size for permissioned game
        _mockValidationCalls();
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.minProposalSize, ()), abi.encode(1000));
        assertEq("PDDG-PIMGO-30,PLDG-PIMGO-30", validate(true));

        // Test invalid anchor state registry implementation
        _mockValidationCalls();
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(_asr))),
            abi.encode(address(0xbad))
        );
        assertEq("PDDG-ANCHORP-20,PLDG-ANCHORP-20", validate(true));
    }

    function _mockValidationCalls() internal virtual {
        // Mock OptimismPortal superchainConfig call
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(systemConfig)
        );

        // Mock SystemConfig dependencies
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.disputeGameFactory, ()), abi.encode(disputeGameFactory)
        );
        vm.mockCall(address(systemConfig), abi.encodeCall(ISemver.version, ()), abi.encode("2.3.0"));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.gasLimit, ()), abi.encode(uint64(60_000_000)));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.scalar, ()), abi.encode(uint256(1) << 248));
        vm.mockCall(
            address(systemConfig),
            abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()),
            abi.encode(l1CrossDomainMessenger)
        );
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(optimismPortal2)
        );
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.l1StandardBridge, ()), abi.encode(l1StandardBridge)
        );
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.l1ERC721Bridge, ()), abi.encode(l1ERC721Bridge));
        vm.mockCall(
            address(systemConfig),
            abi.encodeCall(ISystemConfig.optimismMintableERC20Factory, ()),
            abi.encode(l1OptimismMintableERC20Factory)
        );
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.superchainConfig, ()), abi.encode(superchainConfig)
        );

        // Mock proxy implementations
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(systemConfig))),
            abi.encode(validator.systemConfigImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(optimismPortal2))),
            abi.encode(validator.optimismPortalImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1CrossDomainMessenger))),
            abi.encode(validator.l1CrossDomainMessengerImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1StandardBridge))),
            abi.encode(validator.l1StandardBridgeImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1ERC721Bridge))),
            abi.encode(validator.l1ERC721BridgeImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(l1OptimismMintableERC20Factory))),
            abi.encode(validator.optimismMintableERC20FactoryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(disputeGameFactory))),
            abi.encode(validator.disputeGameFactoryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(anchorStateRegistry))),
            abi.encode(validator.anchorStateRegistryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(delayedWETHPermissionedGameProxy))),
            abi.encode(validator.delayedWETHImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(delayedWeth))),
            abi.encode(validator.delayedWETHImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(anchorStateRegistry))),
            abi.encode(validator.anchorStateRegistryImpl())
        );

        // Mock AnchorStateRegistry
        _mockAnchorStateRegistry(anchorStateRegistry, disputeGameFactory, GameTypes.PERMISSIONED_CANNON);

        // Mock resource config
        IResourceMetering.ResourceConfig memory config = IResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            systemTxMaxGas: 1_000_000,
            minimumBaseFee: 1e9,
            maximumBaseFee: type(uint128).max
        });
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.resourceConfig, ()), abi.encode(config));

        // Mock DisputeGameFactory
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(permissionedDisputeGame)
        );
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.CANNON)),
            abi.encode(faultDisputeGame)
        );
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        vm.mockCall(
            address(disputeGameFactory), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(l1PAOMultisig)
        );

        _mockDisputeGame(
            address(faultDisputeGame), anchorStateRegistry, delayedWeth, absolutePrestate, GameTypes.CANNON
        );
        _mockDisputeGame(
            address(permissionedDisputeGame),
            anchorStateRegistry,
            delayedWETHPermissionedGameProxy,
            absolutePrestate,
            GameTypes.PERMISSIONED_CANNON
        );
        vm.mockCall(
            address(permissionedDisputeGame),
            abi.encodeCall(IPermissionedDisputeGame.challenger, ()),
            abi.encode(challenger)
        );

        // Mock MIPS
        vm.mockCall(address(mips), abi.encodeCall(IMIPS.oracle, ()), abi.encode(preimageOracle));

        // Mock PreimageOracle
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("1.1.2"));
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.challengePeriod, ()), abi.encode(86400));
        vm.mockCall(address(preimageOracle), abi.encodeCall(IPreimageOracle.minProposalSize, ()), abi.encode(126000));

        // Mock L1CrossDomainMessenger
        vm.mockCall(address(l1CrossDomainMessenger), abi.encodeCall(ISemver.version, ()), abi.encode("2.3.0"));
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.OTHER_MESSENGER, ()),
            abi.encode(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER))
        );
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.otherMessenger, ()),
            abi.encode(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER))
        );
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.PORTAL, ()),
            abi.encode(optimismPortal2)
        );
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.portal, ()),
            abi.encode(optimismPortal2)
        );
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.systemConfig, ()),
            abi.encode(systemConfig)
        );

        // Mock OptimismPortal
        vm.mockCall(address(optimismPortal2), abi.encodeCall(ISemver.version, ()), abi.encode("3.10.0"));
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.disputeGameFactory, ()),
            abi.encode(disputeGameFactory)
        );
        vm.mockCall(
            address(optimismPortal2), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(systemConfig)
        );
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.guardian, ()),
            abi.encode(superchainConfig.guardian())
        );
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.paused, ()),
            abi.encode(superchainConfig.paused(address(0)))
        );
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.l2Sender, ()),
            abi.encode(address(0x000000000000000000000000000000000000dEaD))
        );

        // Mock SuperchainConfig
        vm.mockCall(
            address(superchainConfig), abi.encodeCall(ISuperchainConfig.guardian, ()), abi.encode(makeAddr("guardian"))
        );
        vm.mockCall(
            address(superchainConfig), abi.encodeCall(ISuperchainConfig.paused, (address(0))), abi.encode(false)
        );

        // Mock L1StandardBridge
        vm.mockCall(address(l1StandardBridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.1.0"));
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.MESSENGER, ()), abi.encode(l1CrossDomainMessenger)
        );
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IStandardBridge.messenger, ()), abi.encode(l1CrossDomainMessenger)
        );
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IStandardBridge.OTHER_BRIDGE, ()),
            abi.encode(Predeploys.L2_STANDARD_BRIDGE)
        );
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IStandardBridge.otherBridge, ()),
            abi.encode(Predeploys.L2_STANDARD_BRIDGE)
        );
        vm.mockCall(
            address(l1StandardBridge), abi.encodeCall(IL1StandardBridge.systemConfig, ()), abi.encode(systemConfig)
        );

        // Mock L1ERC721Bridge
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.1.0"));
        vm.mockCall(
            address(l1ERC721Bridge),
            abi.encodeCall(IERC721Bridge.OTHER_BRIDGE, ()),
            abi.encode(Predeploys.L2_ERC721_BRIDGE)
        );
        vm.mockCall(
            address(l1ERC721Bridge),
            abi.encodeCall(IERC721Bridge.otherBridge, ()),
            abi.encode(Predeploys.L2_ERC721_BRIDGE)
        );
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.MESSENGER, ()), abi.encode(l1CrossDomainMessenger)
        );
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IERC721Bridge.messenger, ()), abi.encode(l1CrossDomainMessenger)
        );
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(IL1ERC721Bridge.systemConfig, ()), abi.encode(systemConfig));

        // Mock OptimismMintableERC20Factory
        vm.mockCall(address(l1OptimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.9.0"));
        vm.mockCall(
            address(l1OptimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.BRIDGE, ()),
            abi.encode(l1StandardBridge)
        );
        vm.mockCall(
            address(l1OptimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.bridge, ()),
            abi.encode(l1StandardBridge)
        );

        _mockDelayedWETH(delayedWETHPermissionedGameProxy);
        _mockDelayedWETH(delayedWeth);

        // Mock operator fee calls with valid values
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeScalar, ()), abi.encode(0));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeConstant, ()), abi.encode(0));

        // Override version numbers for V300
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.4.0"));
        vm.mockCall(address(optimismPortal2), abi.encodeCall(ISemver.version, ()), abi.encode("3.14.0"));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISemver.version, ()), abi.encode("2.5.0"));
        vm.mockCall(address(l1OptimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.10.1"));
        vm.mockCall(address(l1CrossDomainMessenger), abi.encodeCall(ISemver.version, ()), abi.encode("2.6.0"));
        vm.mockCall(address(l1StandardBridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.3.0"));
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.1"));
        vm.mockCall(address(anchorStateRegistry), abi.encodeCall(ISemver.version, ()), abi.encode("2.2.2"));
        vm.mockCall(address(delayedWETHPermissionedGameProxy), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(delayedWeth), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(mips), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        vm.mockCall(address(permissionedDisputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.4.1"));
        vm.mockCall(address(faultDisputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.4.1"));
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("1.1.4"));
    }

    function _mockAnchorStateRegistry(
        IAnchorStateRegistry _asr,
        IDisputeGameFactory _disputeGameFactory,
        GameType _gameType
    )
        internal
    {
        vm.mockCall(address(_asr), abi.encodeCall(ISemver.version, ()), abi.encode("2.0.0"));
        vm.mockCall(
            address(_asr), abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(_disputeGameFactory)
        );
        vm.mockCall(
            address(_asr),
            abi.encodeCall(IAnchorStateRegistry.anchors, (_gameType)),
            abi.encode(Hash.wrap(0xdead000000000000000000000000000000000000000000000000000000000000), 0)
        );
        vm.mockCall(address(_asr), abi.encodeCall(IAnchorStateRegistry.systemConfig, ()), abi.encode(systemConfig));
    }

    function _mockDisputeGame(
        address _disputeGame,
        IAnchorStateRegistry _asr,
        IDelayedWETH _weth,
        bytes32 _absolutePrestate,
        GameType _gameType
    )
        internal
    {
        vm.mockCall(address(_disputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.1"));
        vm.mockCall(address(_disputeGame), abi.encodeCall(IDisputeGame.gameType, ()), abi.encode(_gameType));
        vm.mockCall(
            address(_disputeGame),
            abi.encodeCall(IPermissionedDisputeGame.absolutePrestate, ()),
            abi.encode(_absolutePrestate)
        );
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.vm, ()), abi.encode(mips));
        vm.mockCall(
            address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.anchorStateRegistry, ()), abi.encode(_asr)
        );
        vm.mockCall(
            address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.l2ChainId, ()), abi.encode(l2ChainID)
        );
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.l2BlockNumber, ()), abi.encode(0));
        vm.mockCall(
            address(_disputeGame),
            abi.encodeCall(IPermissionedDisputeGame.clockExtension, ()),
            abi.encode(Duration.wrap(10800))
        );
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.splitDepth, ()), abi.encode(30));
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.maxGameDepth, ()), abi.encode(73));
        vm.mockCall(
            address(_disputeGame),
            abi.encodeCall(IPermissionedDisputeGame.maxClockDuration, ()),
            abi.encode(Duration.wrap(302400))
        );
        vm.mockCall(address(_disputeGame), abi.encodeCall(IPermissionedDisputeGame.weth, ()), abi.encode(_weth));
    }

    function _mockDelayedWETH(IDelayedWETH _weth) public {
        vm.mockCall(address(_weth), abi.encodeCall(ISemver.version, ()), abi.encode("1.1.0"));
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.proxyAdminOwner, ()), abi.encode(l1PAOMultisig));
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(1 weeks / 2));
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.systemConfig, ()), abi.encode(systemConfig));
    }

    function validate(bool _allowFailure) internal view returns (string memory) {
        StandardValidator.ValidationInput memory input = StandardValidator.ValidationInput({
            proxyAdmin: proxyAdmin,
            sysCfg: systemConfig,
            absolutePrestate: absolutePrestate,
            l2ChainID: l2ChainID
        });
        return validator.validate(input, _allowFailure);
    }

    /// @notice Tests that validation reverts with error message when allowFailure is false
    function test_validate_allowFailureFalse_reverts() public {
        _mockValidationCalls();

        // Mock null implementation for permissioned dispute game
        vm.mockCall(
            address(disputeGameFactory),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.PERMISSIONED_CANNON)),
            abi.encode(address(0))
        );

        // Expect revert with PDDG-10 error message
        vm.expectRevert("StandardValidator: PDDG-10");
        validate(false);
    }

    /// @notice Tests validation of operator fee settings in SystemConfig
    function test_validate_systemConfigOperatorFees_succeeds() public {
        // Test invalid operator fee scalar
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeScalar, ()), abi.encode(1));
        assertEq("SYSCON-110", validate(true));

        // Test invalid operator fee constant
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeConstant, ()), abi.encode(1));
        assertEq("SYSCON-120", validate(true));

        // Test both invalid
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeScalar, ()), abi.encode(1));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeConstant, ()), abi.encode(1));
        assertEq("SYSCON-110,SYSCON-120", validate(true));

        // Test both valid (should be included in _mockValidationCalls, but let's be explicit)
        _mockValidationCalls();
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeScalar, ()), abi.encode(0));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeConstant, ()), abi.encode(0));
        assertEq("", validate(true));
    }

    function test_versions_succeeds() public view {
        // Assert that each version function returns the expected string value.
        assertTrue(bytes(validator.systemConfigVersion()).length > 0);
        assertTrue(bytes(validator.optimismPortalVersion()).length > 0);
        assertTrue(bytes(validator.l1CrossDomainMessengerVersion()).length > 0);
        assertTrue(bytes(validator.l1ERC721BridgeVersion()).length > 0);
        assertTrue(bytes(validator.l1StandardBridgeVersion()).length > 0);
        assertTrue(bytes(validator.mipsVersion()).length > 0);
        assertTrue(bytes(validator.optimismMintableERC20FactoryVersion()).length > 0);
        assertTrue(bytes(validator.disputeGameFactoryVersion()).length > 0);
        assertTrue(bytes(validator.anchorStateRegistryVersion()).length > 0);
        assertTrue(bytes(validator.delayedWETHVersion()).length > 0);
        assertTrue(bytes(validator.permissionedDisputeGameVersion()).length > 0);
        assertTrue(bytes(validator.preimageOracleVersion()).length > 0);
    }
}
