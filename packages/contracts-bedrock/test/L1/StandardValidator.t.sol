// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";

// Scripts
import { Config } from "scripts/libraries/Config.sol";

// Target contract
import {
    StandardValidatorBase,
    StandardValidatorV180,
    StandardValidatorV200,
    StandardValidatorV300
} from "src/L1/StandardValidator.sol";

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
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";

abstract contract StandardValidatorTest is Test {
    // Common state variables used across all validator versions
    ISuperchainConfig superchainConfig;
    address l1PAOMultisig;
    address mips;
    address guardian;
    address challenger;

    // Mock contracts for validation
    IProxyAdmin proxyAdmin;
    ISystemConfig systemConfig;
    bytes32 absolutePrestate;
    uint256 l2ChainID;

    // Mock addresses for dependencies
    address disputeGameFactory;
    address permissionedDisputeGame;
    address permissionlessDisputeGame;
    address permissionedASR;
    address permissionlessASR;
    address optimismPortal;
    address l1CrossDomainMessenger;
    address l1StandardBridge;
    address l1ERC721Bridge;
    address optimismMintableERC20Factory;
    address permissionedDelayedWETH;
    address permissionlessDelayedWETH;
    address preimageOracle;

    // Abstract property that must be implemented by derived classes
    function getValidator() internal view virtual returns (StandardValidatorBase);

    // Abstract property that must be implemented by derived classes
    function validate(bool _allowFailure) internal view virtual returns (string memory);

    function setUp() public virtual {
        // Setup test addresses
        superchainConfig = ISuperchainConfig(makeAddr("superchainConfig"));
        l1PAOMultisig = makeAddr("l1PAOMultisig");
        mips = makeAddr("mips");
        guardian = makeAddr("guardian");
        challenger = makeAddr("challenger");

        // Mock superchainConfig calls needed in setup
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.guardian, ()), abi.encode(guardian));
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.paused, ()), abi.encode(false));

        // Setup mock contracts for validation
        proxyAdmin = IProxyAdmin(makeAddr("proxyAdmin"));
        systemConfig = ISystemConfig(makeAddr("systemConfig"));
        absolutePrestate = bytes32(uint256(0xdead));
        l2ChainID = 10;

        // Setup mock dependency addresses
        disputeGameFactory = makeAddr("disputeGameFactory");
        permissionedDisputeGame = makeAddr("permissionedDisputeGame");
        permissionlessDisputeGame = makeAddr("permissionlessDisputeGame");
        permissionedASR = makeAddr("anchorStateRegistry");
        permissionlessASR = makeAddr("permissionlessAnchorStateRegistry");
        optimismPortal = makeAddr("optimismPortal");
        l1CrossDomainMessenger = makeAddr("l1CrossDomainMessenger");
        l1StandardBridge = makeAddr("l1StandardBridge");
        l1ERC721Bridge = makeAddr("l1ERC721Bridge");
        optimismMintableERC20Factory = makeAddr("optimismMintableERC20Factory");
        permissionedDelayedWETH = makeAddr("delayedWETH");
        permissionlessDelayedWETH = makeAddr("permissionlessDelayedWETH");
        preimageOracle = makeAddr("preimageOracle");

        // Mock proxyAdmin owner
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(l1PAOMultisig));
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
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.paused, ()), abi.encode(true));
        assertEq("SPRCFG-10,PORTAL-70", validate(true));
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

        // Test invalid superchainConfig
        _mockValidationCalls();
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.superchainConfig, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1xDM-70", validate(true));
    }

    /// @notice Tests validation of OptimismMintableERC20Factory
    function test_validate_optimismMintableERC20Factory_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(optimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("MERC20F-10", validate(true));

        // Test invalid BRIDGE
        _mockValidationCalls();
        vm.mockCall(
            address(optimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.BRIDGE, ()),
            abi.encode(address(0xbad))
        );
        assertEq("MERC20F-30", validate(true));

        // Test invalid bridge
        _mockValidationCalls();
        vm.mockCall(
            address(optimismMintableERC20Factory),
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

        // Test invalid superchainConfig
        _mockValidationCalls();
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IL1ERC721Bridge.superchainConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("L721B-70", validate(true));
    }

    /// @notice Tests validation of OptimismPortal
    function test_validate_optimismPortal_succeeds() public {
        // Test invalid version
        _mockValidationCalls();
        vm.mockCall(address(optimismPortal), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        assertEq("PORTAL-10", validate(true));

        // Test invalid disputeGameFactory
        _mockValidationCalls();
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.disputeGameFactory, ()), abi.encode(address(0xbad))
        );
        assertEq("PORTAL-30", validate(true));

        // Test invalid systemConfig
        _mockValidationCalls();
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("PORTAL-40", validate(true));

        // Test invalid superchainConfig
        _mockValidationCalls();
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(address(0xbad))
        );
        assertEq("PORTAL-50", validate(true));

        // Test invalid guardian
        _mockValidationCalls();
        vm.mockCall(address(optimismPortal), abi.encodeCall(IOptimismPortal2.guardian, ()), abi.encode(address(0xbad)));
        assertEq("PORTAL-60", validate(true));

        // Test invalid paused
        _mockValidationCalls();
        vm.mockCall(address(optimismPortal), abi.encodeCall(IOptimismPortal2.paused, ()), abi.encode(true));
        assertEq("PORTAL-70", validate(true));

        // Test invalid l2Sender
        _mockValidationCalls();
        vm.mockCall(address(optimismPortal), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(address(0xbad)));
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
            "PDDG", permissionedDisputeGame, permissionedASR, permissionedDelayedWETH, GameTypes.PERMISSIONED_CANNON
        );
    }

    function test_validate_permissionlessDisputeGame_succeeds() public {
        _testDisputeGame(
            "PLDG", permissionlessDisputeGame, permissionlessASR, permissionlessDelayedWETH, GameTypes.CANNON
        );
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

        // Test invalid superchainConfig
        _mockValidationCalls();
        vm.mockCall(
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.superchainConfig, ()),
            abi.encode(address(0xbad))
        );
        assertEq("L1SB-70", validate(true));
    }

    function _testDisputeGame(
        string memory errorPrefix,
        address _disputeGame,
        address _asr,
        address _weth,
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
        assertEq(string.concat(errorPrefix, "-ANCHORP-10"), validate(true));

        // Test invalid anchor state registry factory
        _mockValidationCalls();
        vm.mockCall(
            address(_asr), abi.encodeCall(IAnchorStateRegistry.disputeGameFactory, ()), abi.encode(address(0xbad))
        );
        assertEq(string.concat(errorPrefix, "-ANCHORP-30"), validate(true));

        // Test invalid anchor state registry root
        _mockValidationCalls();
        vm.mockCall(
            address(_asr),
            abi.encodeCall(IAnchorStateRegistry.anchors, (_gameType)),
            abi.encode(Hash.wrap(bytes32(uint256(0xbad))), 0)
        );
        assertEq(string.concat(errorPrefix, "-ANCHORP-40"), validate(true));

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
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.owner, ()), abi.encode(address(0xbad)));
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
    }

    function _mockValidationCalls() internal virtual {
        StandardValidatorBase validator = getValidator();

        // Mock OptimismPortal superchainConfig call
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(superchainConfig)
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
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(optimismPortal));
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.l1StandardBridge, ()), abi.encode(l1StandardBridge)
        );
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.l1ERC721Bridge, ()), abi.encode(l1ERC721Bridge));
        vm.mockCall(
            address(systemConfig),
            abi.encodeCall(ISystemConfig.optimismMintableERC20Factory, ()),
            abi.encode(optimismMintableERC20Factory)
        );

        // Mock proxy implementations
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(systemConfig))),
            abi.encode(validator.systemConfigImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(optimismPortal))),
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
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(optimismMintableERC20Factory))),
            abi.encode(validator.optimismMintableERC20FactoryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(disputeGameFactory))),
            abi.encode(validator.disputeGameFactoryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(permissionedASR))),
            abi.encode(validator.anchorStateRegistryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(permissionedDelayedWETH))),
            abi.encode(validator.delayedWETHImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(permissionlessDelayedWETH))),
            abi.encode(validator.delayedWETHImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(permissionedASR))),
            abi.encode(validator.anchorStateRegistryImpl())
        );
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(permissionlessASR))),
            abi.encode(validator.anchorStateRegistryImpl())
        );

        // Mock AnchorStateRegistry
        _mockAnchorStateRegistry(
            permissionedASR, disputeGameFactory, address(superchainConfig), GameTypes.PERMISSIONED_CANNON
        );
        _mockAnchorStateRegistry(permissionlessASR, disputeGameFactory, address(superchainConfig), GameTypes.CANNON);

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
            abi.encode(permissionlessDisputeGame)
        );
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        vm.mockCall(
            address(disputeGameFactory), abi.encodeCall(IDisputeGameFactory.owner, ()), abi.encode(l1PAOMultisig)
        );

        _mockDisputeGame(
            permissionlessDisputeGame, permissionlessASR, permissionlessDelayedWETH, absolutePrestate, GameTypes.CANNON
        );
        _mockDisputeGame(
            permissionedDisputeGame,
            permissionedASR,
            permissionedDelayedWETH,
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
            abi.encode(optimismPortal)
        );
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.portal, ()),
            abi.encode(optimismPortal)
        );
        vm.mockCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(IL1CrossDomainMessenger.superchainConfig, ()),
            abi.encode(superchainConfig)
        );

        // Mock OptimismPortal
        vm.mockCall(address(optimismPortal), abi.encodeCall(ISemver.version, ()), abi.encode("3.10.0"));
        vm.mockCall(
            address(optimismPortal),
            abi.encodeCall(IOptimismPortal2.disputeGameFactory, ()),
            abi.encode(disputeGameFactory)
        );
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(systemConfig)
        );
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(superchainConfig)
        );
        vm.mockCall(
            address(optimismPortal),
            abi.encodeCall(IOptimismPortal2.guardian, ()),
            abi.encode(superchainConfig.guardian())
        );
        vm.mockCall(
            address(optimismPortal), abi.encodeCall(IOptimismPortal2.paused, ()), abi.encode(superchainConfig.paused())
        );
        vm.mockCall(
            address(optimismPortal),
            abi.encodeCall(IOptimismPortal2.l2Sender, ()),
            abi.encode(address(0x000000000000000000000000000000000000dEaD))
        );

        // Mock SuperchainConfig
        vm.mockCall(
            address(superchainConfig), abi.encodeCall(ISuperchainConfig.guardian, ()), abi.encode(makeAddr("guardian"))
        );
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.paused, ()), abi.encode(false));

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
            address(l1StandardBridge),
            abi.encodeCall(IL1StandardBridge.superchainConfig, ()),
            abi.encode(superchainConfig)
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
        vm.mockCall(
            address(l1ERC721Bridge), abi.encodeCall(IL1ERC721Bridge.superchainConfig, ()), abi.encode(superchainConfig)
        );

        // Mock OptimismMintableERC20Factory
        vm.mockCall(address(optimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.9.0"));
        vm.mockCall(
            address(optimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.BRIDGE, ()),
            abi.encode(l1StandardBridge)
        );
        vm.mockCall(
            address(optimismMintableERC20Factory),
            abi.encodeCall(IOptimismMintableERC20Factory.bridge, ()),
            abi.encode(l1StandardBridge)
        );

        _mockDelayedWETH(permissionedDelayedWETH);
        _mockDelayedWETH(permissionlessDelayedWETH);
    }

    function _mockAnchorStateRegistry(
        address _asr,
        address _disputeGameFactory,
        address _superchainConfig,
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
        vm.mockCall(
            address(_asr), abi.encodeCall(IAnchorStateRegistry.superchainConfig, ()), abi.encode(_superchainConfig)
        );
    }

    function _mockDisputeGame(
        address _disputeGame,
        address _asr,
        address _weth,
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

    function _mockDelayedWETH(address _weth) public {
        vm.mockCall(address(_weth), abi.encodeCall(ISemver.version, ()), abi.encode("1.1.0"));
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.owner, ()), abi.encode(l1PAOMultisig));
        vm.mockCall(address(_weth), abi.encodeCall(IDelayedWETH.delay, ()), abi.encode(1 weeks / 2));
    }
}

contract StandardValidatorV180_Test is StandardValidatorTest {
    StandardValidatorV180 validator;

    function getValidator() internal view override returns (StandardValidatorBase) {
        return validator;
    }

    function validate(bool _allowFailure) internal view override returns (string memory) {
        StandardValidatorV180.InputV180 memory input = StandardValidatorV180.InputV180({
            proxyAdmin: proxyAdmin,
            sysCfg: systemConfig,
            absolutePrestate: absolutePrestate,
            l2ChainID: l2ChainID
        });
        return validator.validate(input, _allowFailure);
    }

    function setUp() public override {
        super.setUp();

        // Deploy validator with all required constructor args
        validator = new StandardValidatorV180(
            StandardValidatorBase.ImplementationsBase({
                systemConfigImpl: makeAddr("systemConfigImpl"),
                optimismPortalImpl: makeAddr("optimismPortalImpl"),
                l1CrossDomainMessengerImpl: makeAddr("l1CrossDomainMessengerImpl"),
                l1StandardBridgeImpl: makeAddr("l1StandardBridgeImpl"),
                l1ERC721BridgeImpl: makeAddr("l1ERC721BridgeImpl"),
                optimismMintableERC20FactoryImpl: makeAddr("optimismMintableERC20FactoryImpl"),
                disputeGameFactoryImpl: makeAddr("disputeGameFactoryImpl"),
                mipsImpl: makeAddr("mips"),
                anchorStateRegistryImpl: makeAddr("anchorStateRegistryImpl"),
                delayedWETHImpl: makeAddr("delayedWETHImpl")
            }),
            superchainConfig,
            l1PAOMultisig,
            challenger,
            302400
        );
    }

    function test_validate_opMainnet_succeeds() public {
        string memory rpcUrl = Config.mainnetRpcUrl();
        if (bytes(rpcUrl).length == 0) {
            return;
        }

        vm.createSelectFork(rpcUrl);

        // When OP Mainnet is updated this will need to be updated to the current validator version.
        StandardValidatorV200.InputV200 memory input = StandardValidatorV200.InputV200({
            proxyAdmin: IProxyAdmin(address(0x543bA4AADBAb8f9025686Bd03993043599c6fB04)),
            sysCfg: ISystemConfig(address(0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)),
            absolutePrestate: bytes32(0x039facea52b20c605c05efb0a33560a92de7074218998f75bcdf61e8989cb5d9),
            l2ChainID: 10
        });
        // Deployed March 27, 2025:
        // https://github.com/ethereum-optimism/superchain-ops/blob/5cc15911636897e8a2cebe7c2bc7cbb47d42ae11/src/improvements/tasks/eth/000-opcm-upgrade-v200/config.toml#L18
        StandardValidatorV200 mainnetValidator = StandardValidatorV200(0xECAbAeaa1D58261F1579232520C5B460ca58a164);
        mainnetValidator.validate(input, true);

        string memory errors = mainnetValidator.validate(input, true);

        // PDDG-DWETH-30: Permissioned dispute game's DelayedWETH owner must be l1PAOMultisig
        // PLDG-DWETH-30: Permissionless dispute game's DelayedWETH owner must be l1PAOMultisig
        //   DWETH-30 errors are pre-existing misconfigurations on OP Mainnet which are out of scope for this task.
        // PDDG-ANCHORP-40: Permissioned dispute game's AnchorStateRegistry root must be
        // 0xdead000000000000000000000000000000000000000000000000000000000000
        // PLDG-ANCHORP-40: Permissionless dispute game's AnchorStateRegistry root must be
        // 0xdead000000000000000000000000000000000000000000000000000000000000
        //   ANCHORP-40 errors do not apply to chains over 1 week old.
        assertEq(errors, "PDDG-DWETH-30,PDDG-ANCHORP-40,PLDG-DWETH-30,PLDG-ANCHORP-40");
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
        vm.expectRevert("StandardValidatorV180: PDDG-10");
        validate(false);
    }
}

// The V200 validator is the same as the V180 validator except for the version numbers. Therefore
// we just inherit from the V180 test to ensure that all tests run again.
contract StandardValidatorV200_Test is StandardValidatorTest {
    StandardValidatorV200 validator;

    function getValidator() internal view override returns (StandardValidatorBase) {
        return validator;
    }

    function validate(bool _allowFailure) internal view override returns (string memory) {
        StandardValidatorV200.InputV200 memory input = StandardValidatorV200.InputV200({
            proxyAdmin: proxyAdmin,
            sysCfg: systemConfig,
            absolutePrestate: absolutePrestate,
            l2ChainID: l2ChainID
        });
        return validator.validate(input, _allowFailure);
    }

    function setUp() public override {
        super.setUp();

        // Deploy validator with all required constructor args
        validator = new StandardValidatorV200(
            StandardValidatorBase.ImplementationsBase({
                systemConfigImpl: makeAddr("systemConfigImpl"),
                optimismPortalImpl: makeAddr("optimismPortalImpl"),
                l1CrossDomainMessengerImpl: makeAddr("l1CrossDomainMessengerImpl"),
                l1StandardBridgeImpl: makeAddr("l1StandardBridgeImpl"),
                l1ERC721BridgeImpl: makeAddr("l1ERC721BridgeImpl"),
                optimismMintableERC20FactoryImpl: makeAddr("optimismMintableERC20FactoryImpl"),
                disputeGameFactoryImpl: makeAddr("disputeGameFactoryImpl"),
                mipsImpl: makeAddr("mips"),
                anchorStateRegistryImpl: makeAddr("anchorStateRegistryImpl"),
                delayedWETHImpl: makeAddr("delayedWETHImpl")
            }),
            superchainConfig,
            l1PAOMultisig,
            challenger,
            302400
        );
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
        vm.expectRevert("StandardValidatorV200: PDDG-10");
        validate(false);
    }

    function _testDisputeGame(
        string memory errorPrefix,
        address _disputeGame,
        address _asr,
        address _weth,
        GameType _gameType
    )
        public
        override
    {
        super._testDisputeGame(errorPrefix, _disputeGame, _asr, _weth, _gameType);

        // Test invalid anchor state registry implementation
        _mockValidationCalls();
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(_asr))),
            abi.encode(address(0xbad))
        );
        assertEq(string.concat(errorPrefix, "-ANCHORP-20"), validate(true));
    }

    function _mockValidationCalls() internal virtual override {
        super._mockValidationCalls();

        // Override version numbers for V200
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.3.1"));
        vm.mockCall(address(optimismPortal), abi.encodeCall(ISemver.version, ()), abi.encode("3.13.0"));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISemver.version, ()), abi.encode("2.4.0"));
        vm.mockCall(address(optimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.10.1"));
        vm.mockCall(address(l1CrossDomainMessenger), abi.encodeCall(ISemver.version, ()), abi.encode("2.5.0"));
        vm.mockCall(address(l1StandardBridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.2.2"));
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.1"));
        vm.mockCall(address(permissionedASR), abi.encodeCall(ISemver.version, ()), abi.encode("2.2.2"));
        vm.mockCall(address(permissionedDelayedWETH), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(permissionlessASR), abi.encodeCall(ISemver.version, ()), abi.encode("2.2.2"));
        vm.mockCall(address(permissionlessDelayedWETH), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(mips), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(permissionedDisputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.4.1"));
        vm.mockCall(address(permissionlessDisputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.4.1"));
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("1.1.4"));
    }
}

contract StandardValidatorV300_Test is StandardValidatorTest {
    StandardValidatorV300 validator;

    function getValidator() internal view override returns (StandardValidatorBase) {
        return validator;
    }

    function validate(bool _allowFailure) internal view override returns (string memory) {
        StandardValidatorV300.InputV300 memory input = StandardValidatorV300.InputV300({
            proxyAdmin: proxyAdmin,
            sysCfg: systemConfig,
            absolutePrestate: absolutePrestate,
            l2ChainID: l2ChainID
        });
        return validator.validate(input, _allowFailure);
    }

    function setUp() public override {
        super.setUp();

        // Deploy validator with all required constructor args
        validator = new StandardValidatorV300(
            StandardValidatorBase.ImplementationsBase({
                systemConfigImpl: makeAddr("systemConfigImpl"),
                optimismPortalImpl: makeAddr("optimismPortalImpl"),
                l1CrossDomainMessengerImpl: makeAddr("l1CrossDomainMessengerImpl"),
                l1StandardBridgeImpl: makeAddr("l1StandardBridgeImpl"),
                l1ERC721BridgeImpl: makeAddr("l1ERC721BridgeImpl"),
                optimismMintableERC20FactoryImpl: makeAddr("optimismMintableERC20FactoryImpl"),
                disputeGameFactoryImpl: makeAddr("disputeGameFactoryImpl"),
                mipsImpl: makeAddr("mips"),
                anchorStateRegistryImpl: makeAddr("anchorStateRegistryImpl"),
                delayedWETHImpl: makeAddr("delayedWETHImpl")
            }),
            superchainConfig,
            l1PAOMultisig,
            challenger,
            302400
        );
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
        vm.expectRevert("StandardValidatorV300: PDDG-10");
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

    function _testDisputeGame(
        string memory errorPrefix,
        address _disputeGame,
        address _asr,
        address _weth,
        GameType _gameType
    )
        public
        override
    {
        super._testDisputeGame(errorPrefix, _disputeGame, _asr, _weth, _gameType);

        // Test invalid anchor state registry implementation
        _mockValidationCalls();
        vm.mockCall(
            address(proxyAdmin),
            abi.encodeCall(IProxyAdmin.getProxyImplementation, (address(_asr))),
            abi.encode(address(0xbad))
        );
        assertEq(string.concat(errorPrefix, "-ANCHORP-20"), validate(true));
    }

    function _mockValidationCalls() internal virtual override {
        super._mockValidationCalls();

        // Mock operator fee calls with valid values
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeScalar, ()), abi.encode(0));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.operatorFeeConstant, ()), abi.encode(0));

        // Override version numbers for V300
        vm.mockCall(address(l1ERC721Bridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.4.0"));
        vm.mockCall(address(optimismPortal), abi.encodeCall(ISemver.version, ()), abi.encode("3.14.0"));
        vm.mockCall(address(systemConfig), abi.encodeCall(ISemver.version, ()), abi.encode("2.5.0"));
        vm.mockCall(address(optimismMintableERC20Factory), abi.encodeCall(ISemver.version, ()), abi.encode("1.10.1"));
        vm.mockCall(address(l1CrossDomainMessenger), abi.encodeCall(ISemver.version, ()), abi.encode("2.6.0"));
        vm.mockCall(address(l1StandardBridge), abi.encodeCall(ISemver.version, ()), abi.encode("2.3.0"));
        vm.mockCall(address(disputeGameFactory), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.1"));
        vm.mockCall(address(permissionedASR), abi.encodeCall(ISemver.version, ()), abi.encode("2.2.2"));
        vm.mockCall(address(permissionedDelayedWETH), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(permissionlessASR), abi.encodeCall(ISemver.version, ()), abi.encode("2.2.2"));
        vm.mockCall(address(permissionlessDelayedWETH), abi.encodeCall(ISemver.version, ()), abi.encode("1.3.0"));
        vm.mockCall(address(mips), abi.encodeCall(ISemver.version, ()), abi.encode("1.0.0"));
        vm.mockCall(address(permissionedDisputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.4.1"));
        vm.mockCall(address(permissionlessDisputeGame), abi.encodeCall(ISemver.version, ()), abi.encode("1.4.1"));
        vm.mockCall(address(preimageOracle), abi.encodeCall(ISemver.version, ()), abi.encode("1.1.4"));
    }
}
