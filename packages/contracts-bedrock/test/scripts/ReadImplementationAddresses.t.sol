// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { ReadImplementationAddresses } from "scripts/deploy/ReadImplementationAddresses.s.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2, IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { Constants } from "src/libraries/Constants.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IStaticL1ChugSplashProxy } from "interfaces/legacy/IL1ChugSplashProxy.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

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
address constant TEST_OPCM_IMPL = address(0x2000);
address constant TEST_OPCM_DEPLOYER = address(0x2001);
address constant TEST_OPCM_UPGRADER = address(0x2002);
address constant TEST_OPCM_GAME_TYPE_ADDER = address(0x2003);
address constant TEST_OPCM_STANDARD_VALIDATOR = address(0x2004);
address constant TEST_OPCM_INTEROP_MIGRATOR = address(0x2005);

/// @title ReadImplementationAddressesTest
/// @notice Tests that ReadImplementationAddresses.run and ReadImplementationAddresses.runWithBytes succeed with OPCM V1
/// and OPCM V2. Also tests that all proxy types and OPCM implementations are correctly read.
contract ReadImplementationAddressesTest is Test {
    ReadImplementationAddresses script;
    ReadImplementationAddresses.Input input;

    function setUp() public {
        script = new ReadImplementationAddresses();
        // Setup proxies
        input = _setUpProxies();
    }

    /// @notice Internal helper function to setup proxies.
    /// @dev This function assigns mock addresses to the input struct for the following proxies:
    /// - optimismPortalProxy
    /// - systemConfigProxy
    /// - l1ERC721BridgeProxy
    /// - optimismMintableERC20FactoryProxy
    /// - disputeGameFactoryProxy
    /// - l1StandardBridgeProxy
    /// and to the AddressManager.
    /// @return input_ The input struct.
    function _setUpProxies() internal returns (ReadImplementationAddresses.Input memory input_) {
        input_.optimismPortalProxy = makeAddr("optimismPortalProxy");
        input_.systemConfigProxy = makeAddr("systemConfigProxy");
        input_.l1ERC721BridgeProxy = makeAddr("l1ERC721BridgeProxy");
        input_.optimismMintableERC20FactoryProxy = makeAddr("optimismMintableERC20FactoryProxy");
        input_.disputeGameFactoryProxy = makeAddr("disputeGameFactoryProxy");
        input_.l1StandardBridgeProxy = makeAddr("l1StandardBridgeProxy");
        input_.addressManager = makeAddr("addressManager");

        return input_;
    }

    /// @notice Tests that ReadImplementationAddresses.run succeeds with OPCM V1.
    function test_run_withOPCMV1_succeeds() public {
        _mockCommon();

        _mockOPCMV1();

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
        _mockCommon();
        _mockOPCMV2();

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
        _mockCommon();
        _mockOPCMV1();

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
        _mockCommon();
        _mockOPCMV2();

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
        // Mock get EIP-1967 Proxy implementations
        _mockGetEIP1967Impls(input);
        // Mock get L1StandardBridge implementation
        _mockExpect(
            address(input.l1StandardBridgeProxy),
            abi.encodeCall(IStaticL1ChugSplashProxy.getImplementation, ()),
            abi.encode(TEST_L1_STANDARD_BRIDGE_IMPL)
        );

        input.opcm = address(0);
        vm.expectRevert("ReadImplementationAddresses: OPCM address has no code");
        script.run(input);
    }

    /// @notice Internal helper function to mock common functionality for both OPCM V1 and OPCM V2.
    /// @dev This function mocks the getters for EIP-1967 Proxy implementations, the L1StandardBridge implementation,
    /// the L1CrossDomainMessenger from AddressManager and the PreimageOracle from MIPS singleton.
    function _mockCommon() internal {
        // Mock get EIP-1967 Proxy implementations
        _mockGetEIP1967Impls(input);

        // Mock get L1StandardBridge implementation
        _mockExpect(
            address(input.l1StandardBridgeProxy),
            abi.encodeCall(IStaticL1ChugSplashProxy.getImplementation, ()),
            abi.encode(TEST_L1_STANDARD_BRIDGE_IMPL)
        );

        // Mock getter for the L1CrossDomainMessenger from AddressManager
        _mockExpect(
            input.addressManager,
            abi.encodeCall(IAddressManager.getAddress, ("OVM_L1CrossDomainMessenger")),
            abi.encode(TEST_L1_CROSS_DOMAIN_MESSENGER_IMPL)
        );

        // Mock getter for the PreimageOracle from MIPS singleton
        _mockExpect(TEST_MIPS_SINGLETON, abi.encodeCall(IMIPS64.oracle, ()), abi.encode(TEST_PREIMAGE_ORACLE_SINGLETON));
    }

    /// @notice Internal helper function to mock and expect calls for EIP-1967 implementation
    /// @dev Mocks the getters for all EIP-1967 proxies in the input struct
    /// @param _input The input struct containing the proxy addresses
    function _mockGetEIP1967Impls(ReadImplementationAddresses.Input memory _input) internal {
        _mockGetEIP1967Impl(_input.optimismPortalProxy, TEST_OPTIMISM_PORTAL_IMPL);
        _mockGetEIP1967Impl(_input.systemConfigProxy, TEST_SYSTEM_CONFIG_IMPL);
        _mockGetEIP1967Impl(_input.l1ERC721BridgeProxy, TEST_L1_ERC721_BRIDGE_IMPL);
        _mockGetEIP1967Impl(_input.optimismMintableERC20FactoryProxy, TEST_OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL);
        _mockGetEIP1967Impl(_input.disputeGameFactoryProxy, TEST_DISPUTE_GAME_FACTORY_IMPL);
    }

    /// @notice Internal helper function to mock and expect calls for EIP-1967 implementation
    /// @param _proxy The proxy address to mock and expect calls on
    /// @param _impl The implementation address to mock and expect calls on
    function _mockGetEIP1967Impl(address _proxy, address _impl) internal {
        _mockExpect(address(_proxy), abi.encodeCall(IProxy.implementation, ()), abi.encode(_impl));
    }

    /// @notice Internal helper function to mock OPCM V1 functionality
    /// @dev Etches code to opcm implementation, sets input opcm argument and mocks opcm v1 getters
    function _mockOPCMV1() internal {
        // Set the OPCM address to the OPCM V1 implementation
        input.opcm = TEST_OPCM_IMPL;

        // Etch code to the OPCM V1 implementation
        vm.etch(TEST_OPCM_IMPL, "0x01");

        // Mock the OPCM version
        _mockExpect(TEST_OPCM_IMPL, abi.encodeCall(IOPContractsManager.version, ()), abi.encode("6.0.0"));

        // Mock the OPCM GameTypeAdder
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManager.opcmGameTypeAdder, ()),
            abi.encode(TEST_OPCM_GAME_TYPE_ADDER)
        );

        // Mock the OPCM Deployer
        _mockExpect(
            TEST_OPCM_IMPL, abi.encodeCall(IOPContractsManager.opcmDeployer, ()), abi.encode(TEST_OPCM_DEPLOYER)
        );

        // Mock the OPCM Upgrader
        _mockExpect(
            TEST_OPCM_IMPL, abi.encodeCall(IOPContractsManager.opcmUpgrader, ()), abi.encode(TEST_OPCM_UPGRADER)
        );

        // Mock the OPCM InteropMigrator
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManager.opcmInteropMigrator, ()),
            abi.encode(TEST_OPCM_INTEROP_MIGRATOR)
        );

        // Mock the OPCM StandardValidator
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManager.opcmStandardValidator, ()),
            abi.encode(TEST_OPCM_STANDARD_VALIDATOR)
        );

        // Mock OPCMV1.implementations()
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManager.implementations, ()),
            abi.encode(
                IOPContractsManager.Implementations({
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
                })
            )
        );
    }

    /// @notice Internal helper function to mock OPCM V2 functionality
    /// @dev Etches code to opcm implementation, sets input opcm argument and mocks opcm v1 getters
    function _mockOPCMV2() internal {
        // Set the OPCM address to the OPCM V2 implementation
        input.opcm = TEST_OPCM_IMPL;

        // Etch code to the OPCM V2 implementation
        vm.etch(TEST_OPCM_IMPL, "0x01");

        // Mock the OPCM version
        _mockExpect(
            TEST_OPCM_IMPL, abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode(Constants.OPCM_V2_MIN_VERSION)
        );

        // Mock the OPCM StandardValidator
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManagerV2.opcmStandardValidator, ()),
            abi.encode(TEST_OPCM_STANDARD_VALIDATOR)
        );

        // Mock the OPCM InteropMigrator
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManagerV2.opcmMigrator, ()),
            abi.encode(TEST_OPCM_INTEROP_MIGRATOR)
        );

        // Mock OPCMV2.implementations()
        _mockExpect(
            TEST_OPCM_IMPL,
            abi.encodeCall(IOPContractsManagerV2.implementations, ()),
            abi.encode(
                IOPContractsManagerContainer.Implementations({
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
                })
            )
        );
    }

    /// @notice Internal helper function to mock and expect calls
    /// @param _target The target address to mock and expect calls on
    /// @param _callData The calldata to mock and expect calls on
    /// @param _returnData The return data to mock and expect calls on
    function _mockExpect(address _target, bytes memory _callData, bytes memory _returnData) internal {
        vm.mockCall(_target, _callData, _returnData);
        vm.expectCall(_target, _callData);
    }
}
