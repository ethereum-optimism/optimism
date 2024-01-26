// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { DeploymentSummary } from "../proofs/utils/DeploymentSummary.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { LegacyMintableERC20 } from "src/legacy/LegacyMintableERC20.sol";

// Tests
import { L1CrossDomainMessenger_Test } from "test/L1/L1CrossDomainMessenger.t.sol";
import { OptimismPortal_Test } from "test/L1/OptimismPortal.t.sol";
import { L1ERC721Bridge_Test, TestERC721 } from "test/L1/L1ERC721Bridge.t.sol";
import {
    L1StandardBridge_Getter_Test,
    L1StandardBridge_Initialize_Test,
    L1StandardBridge_Pause_Test
} from "test/L1/L1StandardBridge.t.sol";

/// @dev Contract testing the deployment summary correctness
contract DeploymentSummary_TestOptimismPortal is DeploymentSummary, OptimismPortal_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(l2OutputOracleProxyAddress);
        systemConfig = SystemConfig(systemConfigProxyAddress);

        // Set up utilized addresses
        depositor = makeAddr("depositor");
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);
    }

    /// @dev Skips the first line of `super.test_constructor_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_constructor_succeeds() external override {
        // OptimismPortal opImpl = OptimismPortal(payable(deploy.mustGetAddress("OptimismPortal")));
        OptimismPortal opImpl = OptimismPortal(payable(optimismPortalAddress));
        assertEq(address(opImpl.L2_ORACLE()), address(0));
        assertEq(address(opImpl.l2Oracle()), address(0));
        assertEq(address(opImpl.SYSTEM_CONFIG()), address(0));
        assertEq(address(opImpl.systemConfig()), address(0));
        assertEq(address(opImpl.superchainConfig()), address(0));
        assertEq(opImpl.l2Sender(), Constants.DEFAULT_L2_SENDER);
    }

    /// @dev Skips the first line of `super.test_initialize_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_initialize_succeeds() external override {
        // address guardian = deploy.cfg().superchainConfigGuardian();
        address guardian = superchainConfig.guardian();
        assertEq(address(optimismPortal.L2_ORACLE()), address(l2OutputOracle));
        assertEq(address(optimismPortal.l2Oracle()), address(l2OutputOracle));
        assertEq(address(optimismPortal.SYSTEM_CONFIG()), address(systemConfig));
        assertEq(address(optimismPortal.systemConfig()), address(systemConfig));
        assertEq(optimismPortal.GUARDIAN(), guardian);
        assertEq(optimismPortal.guardian(), guardian);
        assertEq(address(optimismPortal.superchainConfig()), address(superchainConfig));
        assertEq(optimismPortal.l2Sender(), Constants.DEFAULT_L2_SENDER);
        assertEq(optimismPortal.paused(), false);
    }

    /// @notice This test is overridden because `KontrolDeployment` doesn't initialize
    ///         the L2OutputOracle, which is needed in this test
    function test_simple_isOutputFinalized_succeeds() external override { }

    /// @notice This test is overridden because `KontrolDeployment` doesn't initialize
    ///         the L2OutputOracle, which is needed in this test
    function test_isOutputFinalized_succeeds() external override { }
}

contract DeploymentSummary_TestL1CrossDomainMessenger is DeploymentSummary, L1CrossDomainMessenger_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(l2OutputOracleProxyAddress);
        systemConfig = SystemConfig(systemConfigProxyAddress);
        l1CrossDomainMessenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxyAddress);

        // Set up utilized addresses
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);
    }

    /// @dev Skips the first line of `super.test_constructor_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_constructor_succeeds() external override {
        // L1CrossDomainMessenger impl = L1CrossDomainMessenger(deploy.mustGetAddress("L1CrossDomainMessenger"));
        L1CrossDomainMessenger impl = L1CrossDomainMessenger(l1CrossDomainMessengerAddress);
        assertEq(address(impl.superchainConfig()), address(0));
        assertEq(address(impl.PORTAL()), address(0));
        assertEq(address(impl.portal()), address(0));
        assertEq(address(impl.OTHER_MESSENGER()), Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        assertEq(address(impl.otherMessenger()), Predeploys.L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @notice This test is overridden because `KontrolDeployment` doesn't deploy
    ///         L2CrossDomainMessenger, which is needed in this test
    function test_relayMessage_v2_reverts() external override { }
}

contract DeploymentSummary_TestL1ERC721Bridge is DeploymentSummary, L1ERC721Bridge_Test {
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(l2OutputOracleProxyAddress);
        systemConfig = SystemConfig(systemConfigProxyAddress);
        l1CrossDomainMessenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxyAddress);
        l1ERC721Bridge = L1ERC721Bridge(l1ERC721BridgeProxyAddress);

        // Set up utilized addresses
        alice = makeAddr("alice");
        bob = makeAddr("bob");
        vm.deal(alice, 10000 ether);
        vm.deal(bob, 10000 ether);

        // Bridge_Initializer setUp
        L1Token = new ERC20("Native L1 Token", "L1T");

        LegacyL2Token = new LegacyMintableERC20({
            _l2Bridge: address(l2StandardBridge),
            _l1Token: address(L1Token),
            _name: string.concat("LegacyL2-", L1Token.name()),
            _symbol: string.concat("LegacyL2-", L1Token.symbol())
        });
        vm.label(address(LegacyL2Token), "LegacyMintableERC20");

        // Deploy the L2 ERC20 now
        // L2Token = OptimismMintableERC20(
        //     l2OptimismMintableERC20Factory.createStandardL2Token(
        //         address(L1Token),
        //         string(abi.encodePacked("L2-", L1Token.name())),
        //         string(abi.encodePacked("L2-", L1Token.symbol()))
        //     )
        // );

        // BadL2Token = OptimismMintableERC20(
        //     l2OptimismMintableERC20Factory.createStandardL2Token(
        //         address(1),
        //         string(abi.encodePacked("L2-", L1Token.name())),
        //         string(abi.encodePacked("L2-", L1Token.symbol()))
        //     )
        // );

        NativeL2Token = new ERC20("Native L2 Token", "L2T");

        // RemoteL1Token = OptimismMintableERC20(
        //     l1OptimismMintableERC20Factory.createStandardL2Token(
        //         address(NativeL2Token),
        //         string(abi.encodePacked("L1-", NativeL2Token.name())),
        //         string(abi.encodePacked("L1-", NativeL2Token.symbol()))
        //     )
        // );

        // BadL1Token = OptimismMintableERC20(
        //     l1OptimismMintableERC20Factory.createStandardL2Token(
        //         address(1),
        //         string(abi.encodePacked("L1-", NativeL2Token.name())),
        //         string(abi.encodePacked("L1-", NativeL2Token.symbol()))
        //     )
        // );

        // L1ERC721Bridge_Test setUp
        localToken = new TestERC721();
        remoteToken = new TestERC721();

        // Mint alice a token.
        localToken.mint(alice, tokenId);

        // Approve the bridge to transfer the token.
        vm.prank(alice);
        localToken.approve(address(l1ERC721Bridge), tokenId);
    }

    /// @dev Skips the first line of `super.test_constructor_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_constructor_succeeds() public override {
        // L1ERC721Bridge impl = L1ERC721Bridge(deploy.mustGetAddress("L1ERC721Bridge"));
        L1ERC721Bridge impl = L1ERC721Bridge(l1ERC721BridgeAddress);
        assertEq(address(impl.MESSENGER()), address(0));
        assertEq(address(impl.messenger()), address(0));
        assertEq(address(impl.OTHER_BRIDGE()), Predeploys.L2_ERC721_BRIDGE);
        assertEq(address(impl.otherBridge()), Predeploys.L2_ERC721_BRIDGE);
        assertEq(address(impl.superchainConfig()), address(0));
    }
}

contract DeploymentSummary_TestL1StandardBridge is
    DeploymentSummary,
    L1StandardBridge_Getter_Test,
    L1StandardBridge_Initialize_Test,
    L1StandardBridge_Pause_Test
{
    /// @notice super.setUp is not called on purpose
    function setUp() public override {
        // Recreate Deployment Summary state changes
        DeploymentSummary deploymentSummary = new DeploymentSummary();
        deploymentSummary.recreateDeployment();

        // Set summary addresses
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        l2OutputOracle = L2OutputOracle(l2OutputOracleProxyAddress);
        systemConfig = SystemConfig(systemConfigProxyAddress);
        l1CrossDomainMessenger = L1CrossDomainMessenger(l1CrossDomainMessengerProxyAddress);
        l1ERC721Bridge = L1ERC721Bridge(l1ERC721BridgeProxyAddress);
        l1StandardBridge = L1StandardBridge(payable(l1StandardBridgeProxyAddress));
    }

    /// @dev Skips the first line of `super.test_constructor_succeeds` because
    ///      we're not exercising the `Deploy` logic in these tests. However,
    ///      the remaining assertions of the test are important to check
    function test_constructor_succeeds() external override {
        // L1StandardBridge impl = L1StandardBridge(deploy.mustGetAddress("L1StandardBridge"));
        L1StandardBridge impl = L1StandardBridge(payable(l1StandardBridgeAddress));
        assertEq(address(impl.superchainConfig()), address(0));
        assertEq(address(impl.MESSENGER()), address(0));
        assertEq(address(impl.messenger()), address(0));
        assertEq(address(impl.OTHER_BRIDGE()), Predeploys.L2_STANDARD_BRIDGE);
        assertEq(address(impl.otherBridge()), Predeploys.L2_STANDARD_BRIDGE);
        assertEq(address(l2StandardBridge), Predeploys.L2_STANDARD_BRIDGE);
    }
}
