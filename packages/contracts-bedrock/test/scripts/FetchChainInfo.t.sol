// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Test } from "forge-std/Test.sol";
import { FetchChainInfo, FetchChainInfoInput, FetchChainInfoOutput } from "scripts/FetchChainInfo.s.sol";
import { GameTypes, GameType } from "src/dispute/lib/Types.sol";
import { LibGameType } from "src/dispute/lib/LibUDT.sol";

address constant TEST_GUARDIAN = address(0xBEEF);
address constant TEST_PROPOSER = address(0xCAFE);
address constant TEST_CHALLENGER = address(0xAABB);
address constant TEST_PROXY_ADMIN_OWNER = address(0x8888);
address constant TEST_ADDRESS_MANAGER = address(0x123);

address constant WETH_PERMISSIONED = address(0x500);
address constant WETH_PERMISSIONLESS = address(0x600);

// Base mock contract with common functionality
contract BaseMockContract {
    address public owner;
    address public admin;
    bytes32 public batcherHash;
    address public unsafeBlockSigner;

    // Payable receive function to allow tests to work
    receive() external payable { }
}

// Legacy style contracts (e.g., L2OutputOracle era)
contract LegacyMockContract is BaseMockContract {
    address public GUARDIAN;
    address public SYSTEM_CONFIG;
    address public PORTAL;
    address public L2_ORACLE;
    address public PROPOSER;
    address public messenger; // For legacy contracts that might have modern names
    GameType public respectedGameType;

    function set_GUARDIAN(address _guardian) external {
        GUARDIAN = _guardian;
    }

    function set_SYSTEM_CONFIG(address _config) external {
        SYSTEM_CONFIG = _config;
    }

    function set_PORTAL(address _portal) external {
        PORTAL = _portal;
    }

    function set_L2_ORACLE(address _oracle) external {
        L2_ORACLE = _oracle;
    }

    function set_PROPOSER(address _proposer) external {
        PROPOSER = _proposer;
    }

    function set_messenger(address _messenger) external {
        messenger = _messenger;
    }

    function set_respectedGameType(GameType _type) external {
        respectedGameType = _type;
    }

    // This function will intentionally revert to test fallback paths
    function disputeGameFactory() external pure {
        revert("Function does not exist");
    }
}

// Modern style contracts without fault proof
contract ModernMockContract is BaseMockContract {
    address public guardian;
    address public systemConfig;
    address public portal;
    address public messenger;
    address public superchainConfig;
    address public l1ERC721Bridge;
    address public optimismMintableERC20Factory;
    address public disputeGameFactory; // returns address(0) by default
    GameType public respectedGameType;

    function set_guardian(address _guardian) external {
        guardian = _guardian;
    }

    function set_systemConfig(address _config) external {
        systemConfig = _config;
    }

    function set_portal(address _portal) external {
        portal = _portal;
    }

    function set_messenger(address _messenger) external {
        messenger = _messenger;
    }

    function set_superchainConfig(address _config) external {
        superchainConfig = _config;
    }

    function set_l1ERC721Bridge(address _bridge) external {
        l1ERC721Bridge = _bridge;
    }

    function set_optimismMintableERC20Factory(address _factory) external {
        optimismMintableERC20Factory = _factory;
    }

    function set_disputeGameFactory(address _factory) external {
        disputeGameFactory = _factory;
    }

    function set_respectedGameType(GameType _type) external {
        respectedGameType = _type;
    }
}

contract DisputeGameFactoryMock is ModernMockContract {
    mapping(GameType => address) public gameImpls;

    function set_gameImpl(GameType _type, address _impl) external {
        gameImpls[_type] = _impl;
    }
}

contract PermissionedDisputeGameMock is ModernMockContract {
    address public challenger;
    address public proposer;
    address public vm;
    address public anchorStateRegistry;

    function set_challenger(address _challenger) external {
        challenger = _challenger;
    }

    function set_proposer(address _proposer) external {
        proposer = _proposer;
    }

    function set_vm(address _vm) external {
        vm = _vm;
    }

    function set_anchorStateRegistry(address _registry) external {
        anchorStateRegistry = _registry;
    }

    function weth() external pure returns (address) {
        return WETH_PERMISSIONED;
    }
}

contract PermissionlessDisputeGameMock is ModernMockContract {
    function weth() external pure returns (address) {
        return WETH_PERMISSIONLESS;
    }
}

contract OracleMock is ModernMockContract {
    address public oracle;

    function set_oracle(address _oracle) external {
        oracle = _oracle;
    }
}

contract ProxyAdminMock {
    address public owner;

    constructor(address _owner) {
        owner = _owner;
    }
}

contract FetchChainInfoTest is Test {
    FetchChainInfo fetchChainInfo;

    // Struct to avoid stack too deep errors
    struct TestContext {
        FetchChainInfoInput input;
        FetchChainInfoOutput output;
        address systemConfigProxy;
        address l1StandardBridgeProxy;
        address l1CrossDomainMessenger;
        address optimismPortal;
        address disputeGameFactory;
        address superchainConfig;
        address permissionedGame;
        address permissionlessGame;
        address l2OutputOracle;
        address mips;
        address preimageOracle;
        address anchorStateRegistry;
        address proxyAdmin;
        address proxyAdminOwner;
    }

    function _setupAddressManagerSlot(address messenger, address managerAddress) internal {
        uint256 ADDRESS_MANAGER_SLOT = 1;
        bytes32 slot = keccak256(abi.encode(messenger, ADDRESS_MANAGER_SLOT));
        vm.store(messenger, slot, bytes32(uint256(uint160(managerAddress))));
    }

    function _setupProxyAdmin(address systemConfigProxy, address proxyAdmin) internal {
        vm.mockCall(systemConfigProxy, abi.encodeWithSignature("admin()"), abi.encode(proxyAdmin));
    }

    function _prepareTestContext() internal returns (TestContext memory ctx) {
        ctx.input = new FetchChainInfoInput();
        ctx.output = new FetchChainInfoOutput();

        ctx.proxyAdminOwner = TEST_PROXY_ADMIN_OWNER;
        ctx.proxyAdmin = address(new ProxyAdminMock(ctx.proxyAdminOwner));

        return ctx;
    }

    function _prepareLegacyTestContext() internal returns (TestContext memory ctx) {
        ctx = _prepareTestContext();

        ctx.systemConfigProxy = address(new LegacyMockContract());
        ctx.l1StandardBridgeProxy = address(new LegacyMockContract());
        ctx.l1CrossDomainMessenger = address(new LegacyMockContract());
        ctx.optimismPortal = address(new LegacyMockContract());
        ctx.l2OutputOracle = address(new LegacyMockContract());

        ctx.input.set(ctx.input.systemConfigProxy.selector, ctx.systemConfigProxy);
        ctx.input.set(ctx.input.l1StandardBridgeProxy.selector, ctx.l1StandardBridgeProxy);

        _setupProxyAdmin(ctx.systemConfigProxy, ctx.proxyAdmin);

        return ctx;
    }

    function _prepareModernTestContext() internal returns (TestContext memory ctx) {
        ctx = _prepareTestContext();

        ctx.systemConfigProxy = address(new ModernMockContract());
        ctx.l1StandardBridgeProxy = address(new ModernMockContract());
        ctx.l1CrossDomainMessenger = address(new ModernMockContract());
        ctx.optimismPortal = address(new ModernMockContract());

        ctx.input.set(ctx.input.systemConfigProxy.selector, ctx.systemConfigProxy);
        ctx.input.set(ctx.input.l1StandardBridgeProxy.selector, ctx.l1StandardBridgeProxy);

        _setupProxyAdmin(ctx.systemConfigProxy, ctx.proxyAdmin);

        return ctx;
    }

    function test_legacy_contract_l2_output_oracle() public {
        TestContext memory ctx = _prepareLegacyTestContext();

        LegacyMockContract(payable(ctx.l1StandardBridgeProxy)).set_messenger(ctx.l1CrossDomainMessenger);
        LegacyMockContract(payable(ctx.l1CrossDomainMessenger)).set_PORTAL(ctx.optimismPortal);
        LegacyMockContract(payable(ctx.optimismPortal)).set_L2_ORACLE(ctx.l2OutputOracle);
        LegacyMockContract(payable(ctx.optimismPortal)).set_GUARDIAN(TEST_GUARDIAN);
        LegacyMockContract(payable(ctx.l2OutputOracle)).set_PROPOSER(TEST_PROPOSER);

        _setupAddressManagerSlot(ctx.l1CrossDomainMessenger, TEST_ADDRESS_MANAGER);

        // Mock a system config without disputeGameFactory (force L2_ORACLE path)
        vm.mockCall(ctx.systemConfigProxy, abi.encodeWithSignature("disputeGameFactory()"), abi.encode(address(0)));

        fetchChainInfo = new FetchChainInfo();
        fetchChainInfo.run(ctx.input, ctx.output);

        assertEq(ctx.output.l2OutputOracleProxy(), ctx.l2OutputOracle, "L2OutputOracle should match");
        assertEq(ctx.output.guardian(), TEST_GUARDIAN, "Guardian should match");
        assertEq(ctx.output.proposer(), TEST_PROPOSER, "Proposer should match");

        assertFalse(ctx.output.faultProofPermissioned(), "Permissioned proofs should be disabled");
        assertFalse(ctx.output.faultProofPermissionless(), "Permissionless proofs should be disabled");
    }

    function test_modern_contract_fp_permissioned() public {
        TestContext memory ctx = _prepareModernTestContext();

        ctx.disputeGameFactory = address(new DisputeGameFactoryMock());
        ctx.superchainConfig = address(new ModernMockContract());

        ModernMockContract(payable(ctx.l1StandardBridgeProxy)).set_messenger(ctx.l1CrossDomainMessenger);
        ModernMockContract(payable(ctx.l1CrossDomainMessenger)).set_portal(ctx.optimismPortal);
        ModernMockContract(payable(ctx.systemConfigProxy)).set_disputeGameFactory(ctx.disputeGameFactory);
        ModernMockContract(payable(ctx.optimismPortal)).set_superchainConfig(ctx.superchainConfig);
        ModernMockContract(payable(ctx.optimismPortal)).set_guardian(TEST_GUARDIAN);

        _setupAddressManagerSlot(ctx.l1CrossDomainMessenger, TEST_ADDRESS_MANAGER);

        fetchChainInfo = new FetchChainInfo();
        fetchChainInfo.run(ctx.input, ctx.output);

        assertEq(ctx.output.systemConfigProxy(), ctx.systemConfigProxy, "SystemConfig should match");
        assertEq(ctx.output.disputeGameFactoryProxy(), ctx.disputeGameFactory, "DisputeGameFactory should match");
        assertEq(ctx.output.guardian(), TEST_GUARDIAN, "Guardian should match");

        assertTrue(ctx.output.faultProofPermissioned(), "Permissioned proofs should be enabled");
        assertFalse(ctx.output.faultProofPermissionless(), "Permissionless proofs should be disabled");
        assertTrue(LibGameType.raw(ctx.output.respectedGameType()) == LibGameType.raw(GameTypes.CANNON));
    }

    function test_modern_contract_fp_permissionless() public {
        TestContext memory ctx = _prepareModernTestContext();

        ctx.disputeGameFactory = address(new DisputeGameFactoryMock());
        ctx.permissionedGame = address(new PermissionedDisputeGameMock());
        ctx.permissionlessGame = address(new PermissionlessDisputeGameMock());
        ctx.superchainConfig = address(new ModernMockContract());
        ctx.mips = address(new OracleMock());
        ctx.preimageOracle = address(new ModernMockContract());
        ctx.anchorStateRegistry = address(new ModernMockContract());

        ModernMockContract(payable(ctx.l1StandardBridgeProxy)).set_messenger(ctx.l1CrossDomainMessenger);
        ModernMockContract(payable(ctx.l1CrossDomainMessenger)).set_portal(ctx.optimismPortal);
        ModernMockContract(payable(ctx.systemConfigProxy)).set_disputeGameFactory(ctx.disputeGameFactory);
        ModernMockContract(payable(ctx.optimismPortal)).set_superchainConfig(ctx.superchainConfig);
        ModernMockContract(payable(ctx.optimismPortal)).set_guardian(TEST_GUARDIAN);
        ModernMockContract(payable(ctx.optimismPortal)).set_systemConfig(ctx.systemConfigProxy);

        DisputeGameFactoryMock(payable(ctx.disputeGameFactory)).set_gameImpl(GameTypes.CANNON, ctx.permissionlessGame);
        DisputeGameFactoryMock(payable(ctx.disputeGameFactory)).set_gameImpl(
            GameTypes.PERMISSIONED_CANNON, ctx.permissionedGame
        );

        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_challenger(TEST_CHALLENGER);
        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_proposer(TEST_PROPOSER);
        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_vm(ctx.mips);
        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_anchorStateRegistry(ctx.anchorStateRegistry);

        OracleMock(payable(ctx.mips)).set_oracle(ctx.preimageOracle);

        _setupAddressManagerSlot(ctx.l1CrossDomainMessenger, TEST_ADDRESS_MANAGER);

        fetchChainInfo = new FetchChainInfo();
        fetchChainInfo.run(ctx.input, ctx.output);

        assertEq(ctx.output.systemConfigProxy(), ctx.systemConfigProxy, "SystemConfig should match");
        assertEq(ctx.output.disputeGameFactoryProxy(), ctx.disputeGameFactory, "DisputeGameFactory should match");
        assertEq(ctx.output.guardian(), TEST_GUARDIAN, "Guardian should match");
        assertEq(ctx.output.permissionedDisputeGame(), ctx.permissionedGame, "PermissionedDisputeGame should match");
        assertEq(ctx.output.faultDisputeGame(), ctx.permissionlessGame, "FaultDisputeGame should match");
        assertEq(ctx.output.challenger(), TEST_CHALLENGER, "Challenger should match");
        assertEq(ctx.output.proposer(), TEST_PROPOSER, "Proposer should match");
        assertEq(ctx.output.mips(), ctx.mips, "MIPS should match");
        assertEq(ctx.output.preimageOracle(), ctx.preimageOracle, "PreimageOracle should match");
        assertEq(ctx.output.anchorStateRegistryProxy(), ctx.anchorStateRegistry, "AnchorStateRegistry should match");

        assertTrue(ctx.output.faultProofPermissioned(), "Permissioned proofs should be enabled");
        assertTrue(ctx.output.faultProofPermissionless(), "Permissionless proofs should be enabled");
    }

    // Test to verify fallback mechanism for guardian() to GUARDIAN()
    function test_guardian_fallback() public {
        TestContext memory ctx = _prepareTestContext();

        // Create mixed mock contracts to test fallback
        ctx.systemConfigProxy = address(new ModernMockContract());
        ctx.l1StandardBridgeProxy = address(new ModernMockContract());
        ctx.l1CrossDomainMessenger = address(new ModernMockContract());
        ctx.optimismPortal = address(new LegacyMockContract()); // Legacy with GUARDIAN

        // Explicitly mock getDisputeGameFactoryProxy to return zero address
        vm.mockCall(ctx.systemConfigProxy, abi.encodeWithSignature("disputeGameFactory()"), abi.encode(address(0)));

        ctx.input.set(ctx.input.systemConfigProxy.selector, ctx.systemConfigProxy);
        ctx.input.set(ctx.input.l1StandardBridgeProxy.selector, ctx.l1StandardBridgeProxy);

        ModernMockContract(payable(ctx.l1StandardBridgeProxy)).set_messenger(ctx.l1CrossDomainMessenger);
        ModernMockContract(payable(ctx.l1CrossDomainMessenger)).set_portal(ctx.optimismPortal);
        // Set "GUARDIAN" but not "guardian" to test fallback
        LegacyMockContract(payable(ctx.optimismPortal)).set_GUARDIAN(TEST_GUARDIAN);
        LegacyMockContract(payable(ctx.optimismPortal)).set_L2_ORACLE(address(new LegacyMockContract()));

        _setupAddressManagerSlot(ctx.l1CrossDomainMessenger, TEST_ADDRESS_MANAGER);
        _setupProxyAdmin(ctx.systemConfigProxy, ctx.proxyAdmin);

        fetchChainInfo = new FetchChainInfo();
        fetchChainInfo.run(ctx.input, ctx.output);

        assertEq(ctx.output.guardian(), TEST_GUARDIAN, "Guardian should match GUARDIAN");
    }

    // Test to verify the fallback mechanism for portal() to PORTAL()
    function test_portal_fallback() public {
        TestContext memory ctx = _prepareTestContext();

        // Create mixed mock contracts to test fallback
        ctx.systemConfigProxy = address(new ModernMockContract());
        ctx.l1StandardBridgeProxy = address(new ModernMockContract());
        ctx.l1CrossDomainMessenger = address(new LegacyMockContract());
        ctx.optimismPortal = address(new LegacyMockContract());

        // Explicitly mock getDisputeGameFactoryProxy to return zero address
        vm.mockCall(ctx.systemConfigProxy, abi.encodeWithSignature("disputeGameFactory()"), abi.encode(address(0)));

        ctx.input.set(ctx.input.systemConfigProxy.selector, ctx.systemConfigProxy);
        ctx.input.set(ctx.input.l1StandardBridgeProxy.selector, ctx.l1StandardBridgeProxy);

        ModernMockContract(payable(ctx.l1StandardBridgeProxy)).set_messenger(ctx.l1CrossDomainMessenger);
        // Set "PORTAL" but not "portal" to test fallback
        LegacyMockContract(payable(ctx.l1CrossDomainMessenger)).set_PORTAL(ctx.optimismPortal);
        LegacyMockContract(payable(ctx.optimismPortal)).set_GUARDIAN(TEST_GUARDIAN);
        LegacyMockContract(payable(ctx.optimismPortal)).set_L2_ORACLE(address(new LegacyMockContract()));

        _setupAddressManagerSlot(ctx.l1CrossDomainMessenger, TEST_ADDRESS_MANAGER);
        _setupProxyAdmin(ctx.systemConfigProxy, ctx.proxyAdmin);

        fetchChainInfo = new FetchChainInfo();
        fetchChainInfo.run(ctx.input, ctx.output);

        assertEq(ctx.output.optimismPortalProxy(), ctx.optimismPortal, "OptimismPortal should match PORTAL");
    }

    // Test delayedWETH mechanism for permissioned and permissionless games
    function test_delayed_weth() public {
        TestContext memory ctx = _prepareModernTestContext();

        // Setup dispute game factory with both game types
        ctx.disputeGameFactory = address(new DisputeGameFactoryMock());
        ctx.permissionedGame = address(new PermissionedDisputeGameMock());
        ctx.permissionlessGame = address(new PermissionlessDisputeGameMock());
        ctx.mips = address(new OracleMock());

        ModernMockContract(payable(ctx.l1StandardBridgeProxy)).set_messenger(ctx.l1CrossDomainMessenger);
        ModernMockContract(payable(ctx.l1CrossDomainMessenger)).set_portal(ctx.optimismPortal);
        ModernMockContract(payable(ctx.systemConfigProxy)).set_disputeGameFactory(ctx.disputeGameFactory);

        DisputeGameFactoryMock(payable(ctx.disputeGameFactory)).set_gameImpl(GameTypes.CANNON, ctx.permissionlessGame);
        DisputeGameFactoryMock(payable(ctx.disputeGameFactory)).set_gameImpl(
            GameTypes.PERMISSIONED_CANNON, ctx.permissionedGame
        );

        // Set up required properties on permissioned game
        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_challenger(TEST_CHALLENGER);
        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_proposer(TEST_PROPOSER);
        PermissionedDisputeGameMock(payable(ctx.permissionedGame)).set_vm(ctx.mips);

        // Setup MIPS oracle to avoid oracle() call on null address
        OracleMock(payable(ctx.mips)).set_oracle(address(0xBEEF));

        _setupAddressManagerSlot(ctx.l1CrossDomainMessenger, TEST_ADDRESS_MANAGER);

        fetchChainInfo = new FetchChainInfo();
        fetchChainInfo.run(ctx.input, ctx.output);

        assertEq(ctx.output.delayedWETHPermissionedGameProxy(), WETH_PERMISSIONED, "PermissionedGame WETH should match");
        assertEq(
            ctx.output.delayedWETHPermissionlessGameProxy(), WETH_PERMISSIONLESS, "PermissionlessGame WETH should match"
        );
    }
}
