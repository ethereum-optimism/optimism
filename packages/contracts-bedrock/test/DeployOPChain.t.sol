// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

import { AddressManager } from "src/legacy/AddressManager.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";

import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

import { DeployOPChainInput, DeployOPChain, DeployOPChainOutput } from "scripts/DeployOPChain.s.sol";

contract DeployOPChainInput_Test is Test {
    DeployOPChainInput dsi;

    DeployOPChainInput.Input input = DeployOPChainInput.Input({
        roles: DeployOPChainInput.Roles({
            opChainProxyAdminOwner: makeAddr("opChainProxyAdminOwner"),
            systemConfigOwner: makeAddr("systemConfigOwner"),
            batcher: makeAddr("batcher"),
            unsafeBlockSigner: makeAddr("unsafeBlockSigner"),
            proposer: makeAddr("proposer"),
            challenger: makeAddr("challenger")
        }),
        basefeeScalar: 100,
        blobBaseFeeScalar: 200,
        l2ChainId: 300
    });

    function setUp() public {
        dsi = new DeployOPChainInput();
    }

    function test_loadInput_succeeds() public {
        dsi.loadInput(input);

        assertTrue(dsi.inputSet(), "100");

        // Compare the test input struct to the getter methods.
        assertEq(input.roles.opChainProxyAdminOwner, dsi.opChainProxyAdminOwner(), "200");
        assertEq(input.roles.systemConfigOwner, dsi.systemConfigOwner(), "300");
        assertEq(input.roles.batcher, dsi.batcher(), "400");
        assertEq(input.roles.unsafeBlockSigner, dsi.unsafeBlockSigner(), "500");
        assertEq(input.roles.proposer, dsi.proposer(), "600");
        assertEq(input.roles.challenger, dsi.challenger(), "700");
        assertEq(input.basefeeScalar, dsi.basefeeScalar(), "800");
        assertEq(input.blobBaseFeeScalar, dsi.blobBaseFeeScalar(), "900");
        assertEq(input.l2ChainId, dsi.l2ChainId(), "1000");

        // Compare the test input struct to the `input` getter method.
        assertEq(keccak256(abi.encode(input)), keccak256(abi.encode(dsi.input())), "1100");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployOPChainInput: input not set";

        vm.expectRevert(expectedErr);
        dsi.opChainProxyAdminOwner();

        vm.expectRevert(expectedErr);
        dsi.systemConfigOwner();

        vm.expectRevert(expectedErr);
        dsi.batcher();

        vm.expectRevert(expectedErr);
        dsi.unsafeBlockSigner();

        vm.expectRevert(expectedErr);
        dsi.proposer();

        vm.expectRevert(expectedErr);
        dsi.challenger();

        vm.expectRevert(expectedErr);
        dsi.basefeeScalar();

        vm.expectRevert(expectedErr);
        dsi.blobBaseFeeScalar();

        vm.expectRevert(expectedErr);
        dsi.l2ChainId();
    }
}

contract DeployOPChainOutput_Test is Test {
    DeployOPChainOutput dso;

    function setUp() public {
        dso = new DeployOPChainOutput();
    }

    function test_set_succeeds() public {
        DeployOPChainOutput.Output memory output = DeployOPChainOutput.Output({
            opChainProxyAdmin: ProxyAdmin(makeAddr("optimismPortal2Impl")),
            addressManager: AddressManager(makeAddr("delayedWETHImpl")),
            l1ERC721BridgeProxy: L1ERC721Bridge(makeAddr("l1ERC721BridgeProxy")),
            systemConfigProxy: SystemConfig(makeAddr("systemConfigProxy")),
            optimismMintableERC20FactoryProxy: OptimismMintableERC20Factory(makeAddr("optimismMintableERC20FactoryProxy")),
            l1StandardBridgeProxy: L1StandardBridge(payable(makeAddr("l1StandardBridgeProxy"))),
            l1CrossDomainMessengerProxy: L1CrossDomainMessenger(makeAddr("l1CrossDomainMessengerProxy")),
            optimismPortalProxy: OptimismPortal2(payable(makeAddr("optimismPortalProxy"))),
            disputeGameFactoryProxy: DisputeGameFactory(makeAddr("disputeGameFactoryProxy")),
            disputeGameFactoryImpl: DisputeGameFactory(makeAddr("disputeGameFactoryImpl")),
            anchorStateRegistryProxy: AnchorStateRegistry(makeAddr("anchorStateRegistryProxy")),
            anchorStateRegistryImpl: AnchorStateRegistry(makeAddr("anchorStateRegistryImpl")),
            faultDisputeGame: FaultDisputeGame(makeAddr("faultDisputeGame")),
            permissionedDisputeGame: PermissionedDisputeGame(makeAddr("permissionedDisputeGame")),
            delayedWETHPermissionedGameProxy: DelayedWETH(payable(makeAddr("delayedWETHPermissionedGameProxy"))),
            delayedWETHPermissionlessGameProxy: DelayedWETH(payable(makeAddr("delayedWETHPermissionlessGameProxy")))
        });

        vm.etch(address(output.opChainProxyAdmin), hex"01");
        vm.etch(address(output.addressManager), hex"01");
        vm.etch(address(output.l1ERC721BridgeProxy), hex"01");
        vm.etch(address(output.systemConfigProxy), hex"01");
        vm.etch(address(output.optimismMintableERC20FactoryProxy), hex"01");
        vm.etch(address(output.l1StandardBridgeProxy), hex"01");
        vm.etch(address(output.l1CrossDomainMessengerProxy), hex"01");
        vm.etch(address(output.optimismPortalProxy), hex"01");
        vm.etch(address(output.disputeGameFactoryProxy), hex"01");
        vm.etch(address(output.disputeGameFactoryImpl), hex"01");
        vm.etch(address(output.anchorStateRegistryProxy), hex"01");
        vm.etch(address(output.anchorStateRegistryImpl), hex"01");
        vm.etch(address(output.faultDisputeGame), hex"01");
        vm.etch(address(output.permissionedDisputeGame), hex"01");
        vm.etch(address(output.delayedWETHPermissionedGameProxy), hex"01");
        vm.etch(address(output.delayedWETHPermissionlessGameProxy), hex"01");

        dso.set(dso.opChainProxyAdmin.selector, address(output.opChainProxyAdmin));
        dso.set(dso.addressManager.selector, address(output.addressManager));
        dso.set(dso.l1ERC721BridgeProxy.selector, address(output.l1ERC721BridgeProxy));
        dso.set(dso.systemConfigProxy.selector, address(output.systemConfigProxy));
        dso.set(dso.optimismMintableERC20FactoryProxy.selector, address(output.optimismMintableERC20FactoryProxy));
        dso.set(dso.l1StandardBridgeProxy.selector, address(output.l1StandardBridgeProxy));
        dso.set(dso.l1CrossDomainMessengerProxy.selector, address(output.l1CrossDomainMessengerProxy));
        dso.set(dso.optimismPortalProxy.selector, address(output.optimismPortalProxy));
        dso.set(dso.disputeGameFactoryProxy.selector, address(output.disputeGameFactoryProxy));
        dso.set(dso.disputeGameFactoryImpl.selector, address(output.disputeGameFactoryImpl));
        dso.set(dso.anchorStateRegistryProxy.selector, address(output.anchorStateRegistryProxy));
        dso.set(dso.anchorStateRegistryImpl.selector, address(output.anchorStateRegistryImpl));
        dso.set(dso.faultDisputeGame.selector, address(output.faultDisputeGame));
        dso.set(dso.permissionedDisputeGame.selector, address(output.permissionedDisputeGame));
        dso.set(dso.delayedWETHPermissionedGameProxy.selector, address(output.delayedWETHPermissionedGameProxy));
        dso.set(dso.delayedWETHPermissionlessGameProxy.selector, address(output.delayedWETHPermissionlessGameProxy));

        assertEq(address(output.opChainProxyAdmin), address(dso.opChainProxyAdmin()), "100");
        assertEq(address(output.addressManager), address(dso.addressManager()), "200");
        assertEq(address(output.l1ERC721BridgeProxy), address(dso.l1ERC721BridgeProxy()), "300");
        assertEq(address(output.systemConfigProxy), address(dso.systemConfigProxy()), "400");
        assertEq(
            address(output.optimismMintableERC20FactoryProxy), address(dso.optimismMintableERC20FactoryProxy()), "500"
        );
        assertEq(address(output.l1StandardBridgeProxy), address(dso.l1StandardBridgeProxy()), "600");
        assertEq(address(output.l1CrossDomainMessengerProxy), address(dso.l1CrossDomainMessengerProxy()), "700");
        assertEq(address(output.optimismPortalProxy), address(dso.optimismPortalProxy()), "800");
        assertEq(address(output.disputeGameFactoryProxy), address(dso.disputeGameFactoryProxy()), "900");
        assertEq(address(output.disputeGameFactoryImpl), address(dso.disputeGameFactoryImpl()), "1000");
        assertEq(address(output.anchorStateRegistryProxy), address(dso.anchorStateRegistryProxy()), "1100");
        assertEq(address(output.anchorStateRegistryImpl), address(dso.anchorStateRegistryImpl()), "1200");
        assertEq(address(output.faultDisputeGame), address(dso.faultDisputeGame()), "1300");
        assertEq(address(output.permissionedDisputeGame), address(dso.permissionedDisputeGame()), "1400");
        assertEq(
            address(output.delayedWETHPermissionedGameProxy), address(dso.delayedWETHPermissionedGameProxy()), "1500"
        );
        assertEq(
            address(output.delayedWETHPermissionlessGameProxy),
            address(dso.delayedWETHPermissionlessGameProxy()),
            "1600"
        );

        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(dso.output())), "1700");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployUtils: zero address";

        vm.expectRevert(expectedErr);
        dso.opChainProxyAdmin();

        vm.expectRevert(expectedErr);
        dso.addressManager();

        vm.expectRevert(expectedErr);
        dso.l1ERC721BridgeProxy();

        vm.expectRevert(expectedErr);
        dso.systemConfigProxy();

        vm.expectRevert(expectedErr);
        dso.optimismMintableERC20FactoryProxy();

        vm.expectRevert(expectedErr);
        dso.l1StandardBridgeProxy();

        vm.expectRevert(expectedErr);
        dso.l1CrossDomainMessengerProxy();

        vm.expectRevert(expectedErr);
        dso.optimismPortalProxy();

        vm.expectRevert(expectedErr);
        dso.disputeGameFactoryProxy();

        vm.expectRevert(expectedErr);
        dso.disputeGameFactoryImpl();

        vm.expectRevert(expectedErr);
        dso.anchorStateRegistryProxy();

        vm.expectRevert(expectedErr);
        dso.anchorStateRegistryImpl();

        vm.expectRevert(expectedErr);
        dso.faultDisputeGame();

        vm.expectRevert(expectedErr);
        dso.permissionedDisputeGame();

        vm.expectRevert(expectedErr);
        dso.delayedWETHPermissionedGameProxy();

        vm.expectRevert(expectedErr);
        dso.delayedWETHPermissionlessGameProxy();
    }

    function test_getters_whenAddrHasNoCode_reverts() public {
        address emptyAddr = makeAddr("emptyAddr");
        bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

        dso.set(dso.opChainProxyAdmin.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.opChainProxyAdmin();

        dso.set(dso.addressManager.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.addressManager();

        dso.set(dso.l1ERC721BridgeProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.l1ERC721BridgeProxy();

        dso.set(dso.systemConfigProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.systemConfigProxy();

        dso.set(dso.optimismMintableERC20FactoryProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.optimismMintableERC20FactoryProxy();

        dso.set(dso.l1StandardBridgeProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.l1StandardBridgeProxy();

        dso.set(dso.l1CrossDomainMessengerProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.l1CrossDomainMessengerProxy();

        dso.set(dso.optimismPortalProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.optimismPortalProxy();

        dso.set(dso.disputeGameFactoryProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.disputeGameFactoryProxy();

        dso.set(dso.disputeGameFactoryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.disputeGameFactoryImpl();

        dso.set(dso.anchorStateRegistryProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.anchorStateRegistryProxy();

        dso.set(dso.anchorStateRegistryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.anchorStateRegistryImpl();

        dso.set(dso.faultDisputeGame.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.faultDisputeGame();

        dso.set(dso.permissionedDisputeGame.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.permissionedDisputeGame();

        dso.set(dso.delayedWETHPermissionedGameProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.delayedWETHPermissionedGameProxy();

        dso.set(dso.delayedWETHPermissionlessGameProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.delayedWETHPermissionlessGameProxy();
    }
}
