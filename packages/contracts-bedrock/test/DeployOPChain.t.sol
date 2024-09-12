// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";
import {
    DeployImplementationsInput,
    DeployImplementations,
    DeployImplementationsInterop,
    DeployImplementationsOutput
} from "scripts/DeployImplementations.s.sol";
import { DeployOPChainInput, DeployOPChain, DeployOPChainOutput } from "scripts/DeployOPChain.s.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

import { AddressManager } from "src/legacy/AddressManager.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { OPStackManager } from "src/L1/OPStackManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

contract DeployOPChainInput_Test is Test {
    DeployOPChainInput doi;

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
        l2ChainId: 300,
        opsm: OPStackManager(makeAddr("opsm"))
    });

    function setUp() public {
        doi = new DeployOPChainInput();
    }

    function test_loadInput_succeeds() public {
        doi.loadInput(input);

        assertTrue(doi.inputSet(), "100");

        // Compare the test input struct to the getter methods.
        assertEq(input.roles.opChainProxyAdminOwner, doi.opChainProxyAdminOwner(), "200");
        assertEq(input.roles.systemConfigOwner, doi.systemConfigOwner(), "300");
        assertEq(input.roles.batcher, doi.batcher(), "400");
        assertEq(input.roles.unsafeBlockSigner, doi.unsafeBlockSigner(), "500");
        assertEq(input.roles.proposer, doi.proposer(), "600");
        assertEq(input.roles.challenger, doi.challenger(), "700");
        assertEq(input.basefeeScalar, doi.basefeeScalar(), "800");
        assertEq(input.blobBaseFeeScalar, doi.blobBaseFeeScalar(), "900");
        assertEq(input.l2ChainId, doi.l2ChainId(), "1000");
        assertEq(address(input.opsm), address(doi.opsm()), "1100");

        // Compare the test input struct to the `input` getter method.
        assertEq(keccak256(abi.encode(input)), keccak256(abi.encode(doi.input())), "1200");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployOPChainInput: input not set";

        vm.expectRevert(expectedErr);
        doi.opChainProxyAdminOwner();

        vm.expectRevert(expectedErr);
        doi.systemConfigOwner();

        vm.expectRevert(expectedErr);
        doi.batcher();

        vm.expectRevert(expectedErr);
        doi.unsafeBlockSigner();

        vm.expectRevert(expectedErr);
        doi.proposer();

        vm.expectRevert(expectedErr);
        doi.challenger();

        vm.expectRevert(expectedErr);
        doi.basefeeScalar();

        vm.expectRevert(expectedErr);
        doi.blobBaseFeeScalar();

        vm.expectRevert(expectedErr);
        doi.l2ChainId();
    }
}

contract DeployOPChainOutput_Test is Test {
    DeployOPChainOutput doo;

    function setUp() public {
        doo = new DeployOPChainOutput();
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

        doo.set(doo.opChainProxyAdmin.selector, address(output.opChainProxyAdmin));
        doo.set(doo.addressManager.selector, address(output.addressManager));
        doo.set(doo.l1ERC721BridgeProxy.selector, address(output.l1ERC721BridgeProxy));
        doo.set(doo.systemConfigProxy.selector, address(output.systemConfigProxy));
        doo.set(doo.optimismMintableERC20FactoryProxy.selector, address(output.optimismMintableERC20FactoryProxy));
        doo.set(doo.l1StandardBridgeProxy.selector, address(output.l1StandardBridgeProxy));
        doo.set(doo.l1CrossDomainMessengerProxy.selector, address(output.l1CrossDomainMessengerProxy));
        doo.set(doo.optimismPortalProxy.selector, address(output.optimismPortalProxy));
        doo.set(doo.disputeGameFactoryProxy.selector, address(output.disputeGameFactoryProxy));
        doo.set(doo.disputeGameFactoryImpl.selector, address(output.disputeGameFactoryImpl));
        doo.set(doo.anchorStateRegistryProxy.selector, address(output.anchorStateRegistryProxy));
        doo.set(doo.anchorStateRegistryImpl.selector, address(output.anchorStateRegistryImpl));
        doo.set(doo.faultDisputeGame.selector, address(output.faultDisputeGame));
        doo.set(doo.permissionedDisputeGame.selector, address(output.permissionedDisputeGame));
        doo.set(doo.delayedWETHPermissionedGameProxy.selector, address(output.delayedWETHPermissionedGameProxy));
        doo.set(doo.delayedWETHPermissionlessGameProxy.selector, address(output.delayedWETHPermissionlessGameProxy));

        assertEq(address(output.opChainProxyAdmin), address(doo.opChainProxyAdmin()), "100");
        assertEq(address(output.addressManager), address(doo.addressManager()), "200");
        assertEq(address(output.l1ERC721BridgeProxy), address(doo.l1ERC721BridgeProxy()), "300");
        assertEq(address(output.systemConfigProxy), address(doo.systemConfigProxy()), "400");
        assertEq(
            address(output.optimismMintableERC20FactoryProxy), address(doo.optimismMintableERC20FactoryProxy()), "500"
        );
        assertEq(address(output.l1StandardBridgeProxy), address(doo.l1StandardBridgeProxy()), "600");
        assertEq(address(output.l1CrossDomainMessengerProxy), address(doo.l1CrossDomainMessengerProxy()), "700");
        assertEq(address(output.optimismPortalProxy), address(doo.optimismPortalProxy()), "800");
        assertEq(address(output.disputeGameFactoryProxy), address(doo.disputeGameFactoryProxy()), "900");
        assertEq(address(output.disputeGameFactoryImpl), address(doo.disputeGameFactoryImpl()), "1000");
        assertEq(address(output.anchorStateRegistryProxy), address(doo.anchorStateRegistryProxy()), "1100");
        assertEq(address(output.anchorStateRegistryImpl), address(doo.anchorStateRegistryImpl()), "1200");
        assertEq(address(output.faultDisputeGame), address(doo.faultDisputeGame()), "1300");
        assertEq(address(output.permissionedDisputeGame), address(doo.permissionedDisputeGame()), "1400");
        assertEq(
            address(output.delayedWETHPermissionedGameProxy), address(doo.delayedWETHPermissionedGameProxy()), "1500"
        );
        assertEq(
            address(output.delayedWETHPermissionlessGameProxy),
            address(doo.delayedWETHPermissionlessGameProxy()),
            "1600"
        );

        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(doo.output())), "1700");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployUtils: zero address";

        vm.expectRevert(expectedErr);
        doo.opChainProxyAdmin();

        vm.expectRevert(expectedErr);
        doo.addressManager();

        vm.expectRevert(expectedErr);
        doo.l1ERC721BridgeProxy();

        vm.expectRevert(expectedErr);
        doo.systemConfigProxy();

        vm.expectRevert(expectedErr);
        doo.optimismMintableERC20FactoryProxy();

        vm.expectRevert(expectedErr);
        doo.l1StandardBridgeProxy();

        vm.expectRevert(expectedErr);
        doo.l1CrossDomainMessengerProxy();

        vm.expectRevert(expectedErr);
        doo.optimismPortalProxy();

        vm.expectRevert(expectedErr);
        doo.disputeGameFactoryProxy();

        vm.expectRevert(expectedErr);
        doo.disputeGameFactoryImpl();

        vm.expectRevert(expectedErr);
        doo.anchorStateRegistryProxy();

        vm.expectRevert(expectedErr);
        doo.anchorStateRegistryImpl();

        vm.expectRevert(expectedErr);
        doo.faultDisputeGame();

        vm.expectRevert(expectedErr);
        doo.permissionedDisputeGame();

        vm.expectRevert(expectedErr);
        doo.delayedWETHPermissionedGameProxy();

        vm.expectRevert(expectedErr);
        doo.delayedWETHPermissionlessGameProxy();
    }

    function test_getters_whenAddrHasNoCode_reverts() public {
        address emptyAddr = makeAddr("emptyAddr");
        bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

        doo.set(doo.opChainProxyAdmin.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.opChainProxyAdmin();

        doo.set(doo.addressManager.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.addressManager();

        doo.set(doo.l1ERC721BridgeProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.l1ERC721BridgeProxy();

        doo.set(doo.systemConfigProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.systemConfigProxy();

        doo.set(doo.optimismMintableERC20FactoryProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.optimismMintableERC20FactoryProxy();

        doo.set(doo.l1StandardBridgeProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.l1StandardBridgeProxy();

        doo.set(doo.l1CrossDomainMessengerProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.l1CrossDomainMessengerProxy();

        doo.set(doo.optimismPortalProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.optimismPortalProxy();

        doo.set(doo.disputeGameFactoryProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.disputeGameFactoryProxy();

        doo.set(doo.disputeGameFactoryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.disputeGameFactoryImpl();

        doo.set(doo.anchorStateRegistryProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.anchorStateRegistryProxy();

        doo.set(doo.anchorStateRegistryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.anchorStateRegistryImpl();

        doo.set(doo.faultDisputeGame.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.faultDisputeGame();

        doo.set(doo.permissionedDisputeGame.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.permissionedDisputeGame();

        doo.set(doo.delayedWETHPermissionedGameProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.delayedWETHPermissionedGameProxy();

        doo.set(doo.delayedWETHPermissionlessGameProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        doo.delayedWETHPermissionlessGameProxy();
    }
}

// To mimic a production environment, we default to integration tests here that actually run the
// DeploySuperchain and DeployImplementations scripts.
contract DeployOPChain_TestBase is Test {
    DeployOPChain deployOPChain;
    DeployOPChainInput doi;
    DeployOPChainOutput doo;

    // We define a default initial input set for DeploySuperchain. The other inputs are dependent
    // on the outputs of the previous scripts, so we initialize them in the `setUp` method.
    address proxyAdminOwner = makeAddr("defaultProxyAdminOwner");
    address protocolVersionsOwner = makeAddr("defaultProtocolVersionsOwner");
    address guardian = makeAddr("defaultGuardian");
    bool paused = false;
    ProtocolVersion requiredProtocolVersion = ProtocolVersion.wrap(1);
    ProtocolVersion recommendedProtocolVersion = ProtocolVersion.wrap(2);

    DeployImplementationsInput.Input deployImplementationsInput = DeployImplementationsInput.Input({
        withdrawalDelaySeconds: 100,
        minProposalSizeBytes: 200,
        challengePeriodSeconds: 300,
        proofMaturityDelaySeconds: 400,
        disputeGameFinalityDelaySeconds: 500,
        release: "op-contracts/latest",
        // These are set during `setUp` since they are outputs of the previous step.
        superchainConfigProxy: SuperchainConfig(address(0)),
        protocolVersionsProxy: ProtocolVersions(address(0))
    });

    DeployOPChainInput.Input deployOPChainInput = DeployOPChainInput.Input({
        roles: DeployOPChainInput.Roles({
            opChainProxyAdminOwner: makeAddr("defaultOPChainProxyAdminOwner"),
            systemConfigOwner: makeAddr("defaultSystemConfigOwner"),
            batcher: makeAddr("defaultBatcher"),
            unsafeBlockSigner: makeAddr("defaultUnsafeBlockSigner"),
            proposer: makeAddr("defaultProposer"),
            challenger: makeAddr("defaultChallenger")
        }),
        basefeeScalar: 100,
        blobBaseFeeScalar: 200,
        l2ChainId: 300,
        // This is set during `setUp` since it is an output of the previous step.
        opsm: OPStackManager(address(0))
    });

    // Set during `setUp`.
    DeployImplementationsOutput.Output deployImplementationsOutput;

    function setUp() public {
        // Initialize deploy scripts.
        DeploySuperchain deploySuperchain = new DeploySuperchain();
        (DeploySuperchainInput dsi, DeploySuperchainOutput dso) = deploySuperchain.etchIOContracts();
        dsi.set(dsi.proxyAdminOwner.selector, proxyAdminOwner);
        dsi.set(dsi.protocolVersionsOwner.selector, protocolVersionsOwner);
        dsi.set(dsi.guardian.selector, guardian);
        dsi.set(dsi.paused.selector, paused);
        dsi.set(dsi.requiredProtocolVersion.selector, requiredProtocolVersion);
        dsi.set(dsi.recommendedProtocolVersion.selector, recommendedProtocolVersion);

        DeployImplementations deployImplementations = new DeployImplementations();
        deployOPChain = new DeployOPChain();
        (doi, doo) = deployOPChain.getIOContracts();

        // Deploy the superchain contracts.
        deploySuperchain.run(dsi, dso);

        // Populate the input struct for DeployImplementations based on the output of DeploySuperchain.
        deployImplementationsInput.superchainConfigProxy = dso.superchainConfigProxy();
        deployImplementationsInput.protocolVersionsProxy = dso.protocolVersionsProxy();

        // Deploy the implementations using the updated DeployImplementations input struct.
        deployImplementationsOutput = deployImplementations.run(deployImplementationsInput);

        // Set the OPStackManager on the input struct for DeployOPChain.
        deployOPChainInput.opsm = deployImplementationsOutput.opsm;
    }

    // See the function of the same name in the `DeployImplementations_Test` contract of
    // `DeployImplementations.t.sol` for more details on why we use this method.
    function createDeployImplementationsContract() internal virtual returns (DeployImplementations) {
        return new DeployImplementations();
    }
}

contract DeployOPChain_Test is DeployOPChain_TestBase {
    function testFuzz_run_succeeds(DeployOPChainInput.Input memory _input) public {
        vm.assume(_input.roles.opChainProxyAdminOwner != address(0));
        vm.assume(_input.roles.systemConfigOwner != address(0));
        vm.assume(_input.roles.batcher != address(0));
        vm.assume(_input.roles.unsafeBlockSigner != address(0));
        vm.assume(_input.roles.proposer != address(0));
        vm.assume(_input.roles.challenger != address(0));
        vm.assume(_input.l2ChainId != 0 && _input.l2ChainId != block.chainid);

        _input.opsm = deployOPChainInput.opsm;

        DeployOPChainOutput.Output memory output = deployOPChain.run(_input);

        // TODO Add fault proof contract assertions below once OPSM fully supports them.

        // Assert that individual input fields were properly set based on the input struct.
        assertEq(_input.roles.opChainProxyAdminOwner, doi.opChainProxyAdminOwner(), "100");
        assertEq(_input.roles.systemConfigOwner, doi.systemConfigOwner(), "200");
        assertEq(_input.roles.batcher, doi.batcher(), "300");
        assertEq(_input.roles.unsafeBlockSigner, doi.unsafeBlockSigner(), "400");
        assertEq(_input.roles.proposer, doi.proposer(), "500");
        assertEq(_input.roles.challenger, doi.challenger(), "600");
        assertEq(_input.basefeeScalar, doi.basefeeScalar(), "700");
        assertEq(_input.blobBaseFeeScalar, doi.blobBaseFeeScalar(), "800");
        assertEq(_input.l2ChainId, doi.l2ChainId(), "900");

        // Assert that individual output fields were properly set based on the output struct.
        assertEq(address(output.opChainProxyAdmin), address(doo.opChainProxyAdmin()), "1100");
        assertEq(address(output.addressManager), address(doo.addressManager()), "1200");
        assertEq(address(output.l1ERC721BridgeProxy), address(doo.l1ERC721BridgeProxy()), "1300");
        assertEq(address(output.systemConfigProxy), address(doo.systemConfigProxy()), "1400");
        assertEq(
            address(output.optimismMintableERC20FactoryProxy), address(doo.optimismMintableERC20FactoryProxy()), "1500"
        );
        assertEq(address(output.l1StandardBridgeProxy), address(doo.l1StandardBridgeProxy()), "1600");
        assertEq(address(output.l1CrossDomainMessengerProxy), address(doo.l1CrossDomainMessengerProxy()), "1700");
        assertEq(address(output.optimismPortalProxy), address(doo.optimismPortalProxy()), "1800");

        // Assert that the full input and output structs were properly set.
        assertEq(keccak256(abi.encode(_input)), keccak256(abi.encode(DeployOPChainInput(doi).input())), "1900");
        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(DeployOPChainOutput(doo).output())), "2000");

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(output.opChainProxyAdmin.owner()), _input.roles.opChainProxyAdminOwner, "2100");
        assertEq(address(output.systemConfigProxy.owner()), _input.roles.systemConfigOwner, "2200");
        address batcher = address(uint160(uint256(output.systemConfigProxy.batcherHash())));
        assertEq(batcher, _input.roles.batcher, "2300");
        assertEq(address(output.systemConfigProxy.unsafeBlockSigner()), _input.roles.unsafeBlockSigner, "2400");
        // assertEq(address(...proposer()), _input.roles.proposer, "2500"); // TODO once we deploy dispute games.
        // assertEq(address(...challenger()), _input.roles.challenger, "2600"); // TODO once we deploy dispute games.

        // Most architecture assertions are handled within the OP Stack Manager itself and therefore
        // we only assert on the things that are not visible onchain.
        // TODO add these assertions: AddressManager, Proxy, ProxyAdmin, etc.
    }
}

contract DeployOPChain_Test_Interop is DeployOPChain_Test {
    function createDeployImplementationsContract() internal override returns (DeployImplementations) {
        return new DeployImplementationsInterop();
    }
}
