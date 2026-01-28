// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { ReadImplementationAddresses } from "scripts/deploy/ReadImplementationAddresses.s.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { MockEIP1967Proxy, MockL1ChugSplashProxy } from "test/mocks/ProxyMocks.sol";
import { MockAddressManager } from "test/mocks/MockAddressManager.sol";
import { MockMIPS } from "test/mocks/MockMIPS.sol";
import { Constants } from "src/libraries/Constants.sol";

// Test addresses declared as constants here for convenience, so both mock contract managers can use them.
address constant TEST_DELAYED_WETH_IMPL = address(0x1000);
address constant TEST_OPTIMISM_PORTAL_IMPL = address(0x1001);
address constant TEST_OPTIMISM_PORTAL_INTEROP_IMPL = address(0x1002);
address constant TEST_ETH_LOCKBOX_IMPL = address(0x1003);
address constant TEST_SYSTEM_CONFIG_IMPL = address(0x1004);
address constant TEST_ANCHOR_STATE_REGISTRY_IMPL = address(0x1005);
address constant TEST_L1_CROSS_DOMAIN_MESSENGER_IMPL = address(0x1006);
address constant TEST_L1_ERC721_BRIDGE_IMPL = address(0x1007);
address constant TEST_L1_STANDARD_BRIDGE_IMPL = address(0x1008);
address constant TEST_OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL = address(0x1009);
address constant TEST_DISPUTE_GAME_FACTORY_IMPL = address(0x100A);
address constant TEST_MIPS_SINGLETON = address(0x100B);
address constant TEST_PREIMAGE_ORACLE_SINGLETON = address(0x100C);
address constant TEST_FAULT_DISPUTE_GAME = address(0x100D);
address constant TEST_PERMISSIONED_DISPUTE_GAME = address(0x100E);
address constant TEST_SUPER_FAULT_DISPUTE_GAME = address(0x100F);
address constant TEST_SUPER_PERMISSIONED_DISPUTE_GAME = address(0x1010);
address constant TEST_OPCM_DEPLOYER = address(0x2000);
address constant TEST_OPCM_UPGRADER = address(0x2001);
address constant TEST_OPCM_GAME_TYPE_ADDER = address(0x2002);
address constant TEST_OPCM_STANDARD_VALIDATOR = address(0x2003);
address constant TEST_OPCM_INTEROP_MIGRATOR = address(0x2004);

/// @title MockOPContractsManagerV1
/// @notice Mock implementation of an OPCM V1 that returns the test addresses for the implementations.
contract MockOPContractsManagerV1 {
    IOPContractsManager.Implementations private _implementations;
    address public opcmDeployerAddr;
    address public opcmUpgraderAddr;
    address public opcmGameTypeAdderAddr;
    address public opcmStandardValidatorAddr;
    address public opcmInteropMigratorAddr;
    bytes32 public devFeatureBitmap;

    constructor() {
        opcmDeployerAddr = TEST_OPCM_DEPLOYER;
        opcmUpgraderAddr = TEST_OPCM_UPGRADER;
        opcmGameTypeAdderAddr = TEST_OPCM_GAME_TYPE_ADDER;
        opcmStandardValidatorAddr = TEST_OPCM_STANDARD_VALIDATOR;
        opcmInteropMigratorAddr = TEST_OPCM_INTEROP_MIGRATOR;
        devFeatureBitmap = bytes32(0);
    }

    function implementations() external pure returns (IOPContractsManager.Implementations memory) {
        return IOPContractsManager.Implementations({
            superchainConfigImpl: address(0),
            protocolVersionsImpl: address(0),
            l1ERC721BridgeImpl: address(0),
            optimismPortalImpl: address(0),
            optimismPortalInteropImpl: TEST_OPTIMISM_PORTAL_INTEROP_IMPL,
            ethLockboxImpl: TEST_ETH_LOCKBOX_IMPL,
            systemConfigImpl: address(0),
            optimismMintableERC20FactoryImpl: address(0),
            l1CrossDomainMessengerImpl: address(0),
            l1StandardBridgeImpl: address(0),
            disputeGameFactoryImpl: address(0),
            anchorStateRegistryImpl: TEST_ANCHOR_STATE_REGISTRY_IMPL,
            delayedWETHImpl: TEST_DELAYED_WETH_IMPL,
            mipsImpl: TEST_MIPS_SINGLETON,
            faultDisputeGameImpl: TEST_FAULT_DISPUTE_GAME,
            permissionedDisputeGameImpl: TEST_PERMISSIONED_DISPUTE_GAME,
            superFaultDisputeGameImpl: TEST_SUPER_FAULT_DISPUTE_GAME,
            superPermissionedDisputeGameImpl: TEST_SUPER_PERMISSIONED_DISPUTE_GAME
        });
    }

    function opcmDeployer() external view returns (address) {
        return opcmDeployerAddr;
    }

    function opcmUpgrader() external view returns (address) {
        return opcmUpgraderAddr;
    }

    function opcmGameTypeAdder() external view returns (address) {
        return opcmGameTypeAdderAddr;
    }

    function opcmStandardValidator() external view returns (address) {
        return opcmStandardValidatorAddr;
    }

    function opcmInteropMigrator() external view returns (address) {
        return opcmInteropMigratorAddr;
    }

    function version() external pure returns (string memory) {
        return "6.0.0";
    }
}

/// @title MockOPContractsManagerV2
/// @notice Mock implementation of an OPCM V2 that returns the test addresses its implementations.
contract MockOPContractsManagerV2 {
    address public opcmStandardValidatorAddr;
    address public opcmMigratorAddr;
    bytes32 public devFeatureBitmap;

    constructor() {
        opcmStandardValidatorAddr = TEST_OPCM_STANDARD_VALIDATOR;
        opcmMigratorAddr = TEST_OPCM_INTEROP_MIGRATOR;
        devFeatureBitmap = DevFeatures.OPCM_V2;
    }

    function implementations() external pure returns (IOPContractsManagerContainer.Implementations memory) {
        return IOPContractsManagerContainer.Implementations({
            superchainConfigImpl: address(0),
            protocolVersionsImpl: address(0),
            l1ERC721BridgeImpl: address(0),
            optimismPortalImpl: address(0),
            optimismPortalInteropImpl: TEST_OPTIMISM_PORTAL_INTEROP_IMPL,
            ethLockboxImpl: TEST_ETH_LOCKBOX_IMPL,
            systemConfigImpl: address(0),
            optimismMintableERC20FactoryImpl: address(0),
            l1CrossDomainMessengerImpl: address(0),
            l1StandardBridgeImpl: address(0),
            disputeGameFactoryImpl: address(0),
            anchorStateRegistryImpl: TEST_ANCHOR_STATE_REGISTRY_IMPL,
            delayedWETHImpl: TEST_DELAYED_WETH_IMPL,
            mipsImpl: TEST_MIPS_SINGLETON,
            faultDisputeGameImpl: TEST_FAULT_DISPUTE_GAME,
            permissionedDisputeGameImpl: TEST_PERMISSIONED_DISPUTE_GAME,
            superFaultDisputeGameImpl: TEST_SUPER_FAULT_DISPUTE_GAME,
            superPermissionedDisputeGameImpl: TEST_SUPER_PERMISSIONED_DISPUTE_GAME,
            storageSetterImpl: address(0)
        });
    }

    function opcmStandardValidator() external view returns (address) {
        return opcmStandardValidatorAddr;
    }

    function opcmMigrator() external view returns (address) {
        return opcmMigratorAddr;
    }

    function version() external pure returns (string memory) {
        return Constants.OPCM_V2_MIN_VERSION;
    }
}

/// @title ReadImplementationAddressesTest
/// @notice Tests that ReadImplementationAddresses.run and ReadImplementationAddresses.runWithBytes succeed with OPCM V1
/// and OPCM V2. Also tests that all proxy types and OPCM implementations are correctly read.
contract ReadImplementationAddressesTest is Test {
    ReadImplementationAddresses script;
    ReadImplementationAddresses.Input input;

    function setUp() public {
        script = new ReadImplementationAddresses();
    }

    function _setupMockProxies() internal returns (ReadImplementationAddresses.Input memory) {
        ReadImplementationAddresses.Input memory _input;

        // Setup mock proxies
        _input.delayedWETHPermissionedGameProxy = address(new MockEIP1967Proxy(TEST_DELAYED_WETH_IMPL));
        _input.optimismPortalProxy = address(new MockEIP1967Proxy(TEST_OPTIMISM_PORTAL_IMPL));
        _input.systemConfigProxy = address(new MockEIP1967Proxy(TEST_SYSTEM_CONFIG_IMPL));
        _input.l1ERC721BridgeProxy = address(new MockEIP1967Proxy(TEST_L1_ERC721_BRIDGE_IMPL));
        _input.optimismMintableERC20FactoryProxy =
            address(new MockEIP1967Proxy(TEST_OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL));
        _input.disputeGameFactoryProxy = address(new MockEIP1967Proxy(TEST_DISPUTE_GAME_FACTORY_IMPL));
        _input.l1StandardBridgeProxy = address(new MockL1ChugSplashProxy(TEST_L1_STANDARD_BRIDGE_IMPL));

        // Setup AddressManager
        MockAddressManager am = new MockAddressManager();
        am.setAddress("OVM_L1CrossDomainMessenger", TEST_L1_CROSS_DOMAIN_MESSENGER_IMPL);
        _input.addressManager = address(am);

        return _input;
    }

    /// @notice Tests that ReadImplementationAddresses.run succeeds with OPCM V1.
    function test_run_withOPCMV1_succeeds() public {
        input = _setupMockProxies();

        // Setup OPCM V1
        MockOPContractsManagerV1 opcmV1 = new MockOPContractsManagerV1();
        input.opcm = address(opcmV1);

        // Setup MIPS with oracle
        MockMIPS mips = new MockMIPS(TEST_PREIMAGE_ORACLE_SINGLETON);
        vm.etch(TEST_MIPS_SINGLETON, address(mips).code);

        // Run the script
        ReadImplementationAddresses.Output memory output = script.run(input);

        // Assert EIP-1967 proxy implementations
        assertEq(output.delayedWETH, TEST_DELAYED_WETH_IMPL, "DelayedWETH should match");
        assertEq(output.optimismPortal, TEST_OPTIMISM_PORTAL_IMPL, "OptimismPortal should match");
        assertEq(output.systemConfig, TEST_SYSTEM_CONFIG_IMPL, "SystemConfig should match");
        assertEq(output.l1ERC721Bridge, TEST_L1_ERC721_BRIDGE_IMPL, "L1ERC721Bridge should match");
        assertEq(
            output.optimismMintableERC20Factory,
            TEST_OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL,
            "OptimismMintableERC20Factory should match"
        );
        assertEq(output.disputeGameFactory, TEST_DISPUTE_GAME_FACTORY_IMPL, "DisputeGameFactory should match");

        // Assert L1StandardBridge
        assertEq(output.l1StandardBridge, TEST_L1_STANDARD_BRIDGE_IMPL, "L1StandardBridge should match");

        // Assert L1CrossDomainMessenger from AddressManager
        assertEq(
            output.l1CrossDomainMessenger, TEST_L1_CROSS_DOMAIN_MESSENGER_IMPL, "L1CrossDomainMessenger should match"
        );

        // Assert OPCM V1 specific implementations
        assertEq(output.opcmDeployer, TEST_OPCM_DEPLOYER, "OPCM Deployer should match");
        assertEq(output.opcmUpgrader, TEST_OPCM_UPGRADER, "OPCM Upgrader should match");
        assertEq(output.opcmGameTypeAdder, TEST_OPCM_GAME_TYPE_ADDER, "OPCM GameTypeAdder should match");
        assertEq(output.opcmStandardValidator, TEST_OPCM_STANDARD_VALIDATOR, "OPCM StandardValidator should match");
        assertEq(output.opcmInteropMigrator, TEST_OPCM_INTEROP_MIGRATOR, "OPCM InteropMigrator should match");

        // Assert implementations from OPCM
        assertEq(output.mipsSingleton, TEST_MIPS_SINGLETON, "MIPS singleton should match");
        assertEq(output.ethLockbox, TEST_ETH_LOCKBOX_IMPL, "EthLockbox should match");
        assertEq(output.anchorStateRegistry, TEST_ANCHOR_STATE_REGISTRY_IMPL, "AnchorStateRegistry should match");
        assertEq(output.optimismPortalInterop, TEST_OPTIMISM_PORTAL_INTEROP_IMPL, "OptimismPortalInterop should match");
        assertEq(output.faultDisputeGame, TEST_FAULT_DISPUTE_GAME, "FaultDisputeGame should match");
        assertEq(output.permissionedDisputeGame, TEST_PERMISSIONED_DISPUTE_GAME, "PermissionedDisputeGame should match");
        assertEq(output.superFaultDisputeGame, TEST_SUPER_FAULT_DISPUTE_GAME, "SuperFaultDisputeGame should match");
        assertEq(
            output.superPermissionedDisputeGame,
            TEST_SUPER_PERMISSIONED_DISPUTE_GAME,
            "SuperPermissionedDisputeGame should match"
        );

        // Assert PreimageOracle
        assertEq(output.preimageOracleSingleton, TEST_PREIMAGE_ORACLE_SINGLETON, "PreimageOracle should match");
    }

    /// @notice Tests that ReadImplementationAddresses.run succeeds with OPCM V2.
    function test_run_withOPCMV2_succeeds() public {
        input = _setupMockProxies();

        // Setup OPCM V2
        MockOPContractsManagerV2 opcmV2 = new MockOPContractsManagerV2();
        input.opcm = address(opcmV2);

        // Setup MIPS with oracle
        MockMIPS mips = new MockMIPS(TEST_PREIMAGE_ORACLE_SINGLETON);
        vm.etch(TEST_MIPS_SINGLETON, address(mips).code);

        // Run the script
        ReadImplementationAddresses.Output memory output = script.run(input);

        // Assert EIP-1967 proxy implementations
        assertEq(output.delayedWETH, TEST_DELAYED_WETH_IMPL, "DelayedWETH should match");
        assertEq(output.optimismPortal, TEST_OPTIMISM_PORTAL_IMPL, "OptimismPortal should match");
        assertEq(output.systemConfig, TEST_SYSTEM_CONFIG_IMPL, "SystemConfig should match");
        assertEq(output.l1ERC721Bridge, TEST_L1_ERC721_BRIDGE_IMPL, "L1ERC721Bridge should match");
        assertEq(
            output.optimismMintableERC20Factory,
            TEST_OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL,
            "OptimismMintableERC20Factory should match"
        );
        assertEq(output.disputeGameFactory, TEST_DISPUTE_GAME_FACTORY_IMPL, "DisputeGameFactory should match");

        // Assert L1StandardBridge
        assertEq(output.l1StandardBridge, TEST_L1_STANDARD_BRIDGE_IMPL, "L1StandardBridge should match");

        // Assert L1CrossDomainMessenger from AddressManager
        assertEq(
            output.l1CrossDomainMessenger, TEST_L1_CROSS_DOMAIN_MESSENGER_IMPL, "L1CrossDomainMessenger should match"
        );

        // Assert OPCM V2 specific implementations
        assertEq(output.opcmGameTypeAdder, address(0), "OPCM GameTypeAdder should be zero in V2");
        assertEq(output.opcmDeployer, address(0), "OPCM Deployer should be zero in V2");
        assertEq(output.opcmUpgrader, address(0), "OPCM Upgrader should be zero in V2");
        assertEq(output.opcmInteropMigrator, TEST_OPCM_INTEROP_MIGRATOR, "OPCM InteropMigrator should match");
        assertEq(output.opcmStandardValidator, TEST_OPCM_STANDARD_VALIDATOR, "OPCM StandardValidator should match");

        // Assert implementations from OPCM
        assertEq(output.mipsSingleton, TEST_MIPS_SINGLETON, "MIPS singleton should match");
        assertEq(output.ethLockbox, TEST_ETH_LOCKBOX_IMPL, "EthLockbox should match");
        assertEq(output.anchorStateRegistry, TEST_ANCHOR_STATE_REGISTRY_IMPL, "AnchorStateRegistry should match");
        assertEq(output.optimismPortalInterop, TEST_OPTIMISM_PORTAL_INTEROP_IMPL, "OptimismPortalInterop should match");
        assertEq(output.faultDisputeGame, TEST_FAULT_DISPUTE_GAME, "FaultDisputeGame should match");
        assertEq(output.permissionedDisputeGame, TEST_PERMISSIONED_DISPUTE_GAME, "PermissionedDisputeGame should match");
        assertEq(output.superFaultDisputeGame, TEST_SUPER_FAULT_DISPUTE_GAME, "SuperFaultDisputeGame should match");
        assertEq(
            output.superPermissionedDisputeGame,
            TEST_SUPER_PERMISSIONED_DISPUTE_GAME,
            "SuperPermissionedDisputeGame should match"
        );

        // Assert PreimageOracle
        assertEq(output.preimageOracleSingleton, TEST_PREIMAGE_ORACLE_SINGLETON, "PreimageOracle should match");
    }

    /// @notice Tests that ReadImplementationAddresses.runWithBytes succeeds with OPCM V1.
    function test_runWithBytes_withOPCMV1_succeeds() public {
        input = _setupMockProxies();

        // Setup OPCM V1
        MockOPContractsManagerV1 opcmV1 = new MockOPContractsManagerV1();
        input.opcm = address(opcmV1);

        // Setup MIPS with oracle
        MockMIPS mips = new MockMIPS(TEST_PREIMAGE_ORACLE_SINGLETON);
        vm.etch(TEST_MIPS_SINGLETON, address(mips).code);

        // Encode input
        bytes memory inputBytes = abi.encode(input);

        // Run the script
        bytes memory outputBytes = script.runWithBytes(inputBytes);

        // Decode output
        ReadImplementationAddresses.Output memory output = abi.decode(outputBytes, (ReadImplementationAddresses.Output));

        // Assert key values
        assertEq(output.delayedWETH, TEST_DELAYED_WETH_IMPL, "DelayedWETH should match");
        assertEq(output.optimismPortal, TEST_OPTIMISM_PORTAL_IMPL, "OptimismPortal should match");
        assertEq(output.opcmDeployer, TEST_OPCM_DEPLOYER, "OPCM Deployer should match");
        assertEq(output.mipsSingleton, TEST_MIPS_SINGLETON, "MIPS singleton should match");
        assertEq(output.preimageOracleSingleton, TEST_PREIMAGE_ORACLE_SINGLETON, "PreimageOracle should match");
    }

    /// @notice Tests that ReadImplementationAddresses.runWithBytes succeeds with OPCM V2.
    function test_runWithBytes_withOPCMV2_succeeds() public {
        input = _setupMockProxies();

        // Setup OPCM V2
        MockOPContractsManagerV2 opcmV2 = new MockOPContractsManagerV2();
        input.opcm = address(opcmV2);

        // Setup MIPS with oracle
        MockMIPS mips = new MockMIPS(TEST_PREIMAGE_ORACLE_SINGLETON);
        vm.etch(TEST_MIPS_SINGLETON, address(mips).code);

        // Encode input
        bytes memory inputBytes = abi.encode(input);

        // Run the script
        bytes memory outputBytes = script.runWithBytes(inputBytes);

        // Decode output
        ReadImplementationAddresses.Output memory output = abi.decode(outputBytes, (ReadImplementationAddresses.Output));

        // Assert key values
        assertEq(output.delayedWETH, TEST_DELAYED_WETH_IMPL, "DelayedWETH should match");
        assertEq(output.optimismPortal, TEST_OPTIMISM_PORTAL_IMPL, "OptimismPortal should match");
        assertEq(output.opcmDeployer, address(0), "OPCM Deployer should be zero in V2");
        assertEq(output.mipsSingleton, TEST_MIPS_SINGLETON, "MIPS singleton should match");
        assertEq(output.preimageOracleSingleton, TEST_PREIMAGE_ORACLE_SINGLETON, "PreimageOracle should match");
    }

    function test_run_opcmCodeLengthZero_reverts() public {
        input = _setupMockProxies();
        input.opcm = address(0);
        vm.expectRevert("ReadImplementationAddresses: OPCM address has no code");
        script.run(input);
    }
}
