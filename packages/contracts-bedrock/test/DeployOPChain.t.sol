// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import {
    DeployImplementationsInput,
    DeployImplementations,
    DeployImplementationsInterop,
    DeployImplementationsOutput
} from "scripts/DeployImplementations.s.sol";
import { DeployOPChainInput, DeployOPChain, DeployOPChainOutput } from "scripts/DeployOPChain.s.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";

import { IProxyAdmin } from "src/universal/interfaces/IProxyAdmin.sol";
import { IOptimismMintableERC20Factory } from "src/universal/interfaces/IOptimismMintableERC20Factory.sol";

import { IAddressManager } from "src/legacy/interfaces/IAddressManager.sol";

import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "src/dispute/interfaces/IPermissionedDisputeGame.sol";

import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";
import { IProtocolVersions, ProtocolVersion } from "src/L1/interfaces/IProtocolVersions.sol";
import { IOptimismPortal2 } from "src/L1/interfaces/IOptimismPortal2.sol";
import { ISystemConfig } from "src/L1/interfaces/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "src/L1/interfaces/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "src/L1/interfaces/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "src/L1/interfaces/IL1StandardBridge.sol";

contract DeployOPChainInput_Test is Test {
    DeployOPChainInput doi;

    // Define defaults.
    address opChainProxyAdminOwner = makeAddr("opChainProxyAdminOwner");
    address systemConfigOwner = makeAddr("systemConfigOwner");
    address batcher = makeAddr("batcher");
    address unsafeBlockSigner = makeAddr("unsafeBlockSigner");
    address proposer = makeAddr("proposer");
    address challenger = makeAddr("challenger");
    uint32 basefeeScalar = 100;
    uint32 blobBaseFeeScalar = 200;
    uint256 l2ChainId = 300;
    OPStackManager opsm = OPStackManager(makeAddr("opsm"));

    function setUp() public {
        doi = new DeployOPChainInput();
    }

    function test_set_succeeds() public {
        doi.set(doi.opChainProxyAdminOwner.selector, opChainProxyAdminOwner);
        doi.set(doi.systemConfigOwner.selector, systemConfigOwner);
        doi.set(doi.batcher.selector, batcher);
        doi.set(doi.unsafeBlockSigner.selector, unsafeBlockSigner);
        doi.set(doi.proposer.selector, proposer);
        doi.set(doi.challenger.selector, challenger);
        doi.set(doi.basefeeScalar.selector, basefeeScalar);
        doi.set(doi.blobBaseFeeScalar.selector, blobBaseFeeScalar);
        doi.set(doi.l2ChainId.selector, l2ChainId);
        doi.set(doi.opsmProxy.selector, address(opsm));
        // Compare the default inputs to the getter methods.
        assertEq(opChainProxyAdminOwner, doi.opChainProxyAdminOwner(), "200");
        assertEq(systemConfigOwner, doi.systemConfigOwner(), "300");
        assertEq(batcher, doi.batcher(), "400");
        assertEq(unsafeBlockSigner, doi.unsafeBlockSigner(), "500");
        assertEq(proposer, doi.proposer(), "600");
        assertEq(challenger, doi.challenger(), "700");
        assertEq(basefeeScalar, doi.basefeeScalar(), "800");
        assertEq(blobBaseFeeScalar, doi.blobBaseFeeScalar(), "900");
        assertEq(l2ChainId, doi.l2ChainId(), "1000");
        assertEq(address(opsm), address(doi.opsmProxy()), "1100");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployOPChainInput: not set";

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

    // Define default outputs to set.
    // We set these in storage because doing it locally in test_set_succeeds results in stack too deep.
    IProxyAdmin opChainProxyAdmin = IProxyAdmin(makeAddr("optimismPortal2Impl"));
    IAddressManager addressManager = IAddressManager(makeAddr("delayedWETHImpl"));
    IL1ERC721Bridge l1ERC721BridgeProxy = IL1ERC721Bridge(makeAddr("l1ERC721BridgeProxy"));
    ISystemConfig systemConfigProxy = ISystemConfig(makeAddr("systemConfigProxy"));
    IOptimismMintableERC20Factory optimismMintableERC20FactoryProxy =
        IOptimismMintableERC20Factory(makeAddr("optimismMintableERC20FactoryProxy"));
    IL1StandardBridge l1StandardBridgeProxy = IL1StandardBridge(payable(makeAddr("l1StandardBridgeProxy")));
    IL1CrossDomainMessenger l1CrossDomainMessengerProxy =
        IL1CrossDomainMessenger(makeAddr("l1CrossDomainMessengerProxy"));
    IOptimismPortal2 optimismPortalProxy = IOptimismPortal2(payable(makeAddr("optimismPortalProxy")));
    IDisputeGameFactory disputeGameFactoryProxy = IDisputeGameFactory(makeAddr("disputeGameFactoryProxy"));
    IAnchorStateRegistry anchorStateRegistryProxy = IAnchorStateRegistry(makeAddr("anchorStateRegistryProxy"));
    IAnchorStateRegistry anchorStateRegistryImpl = IAnchorStateRegistry(makeAddr("anchorStateRegistryImpl"));
    IFaultDisputeGame faultDisputeGame = IFaultDisputeGame(makeAddr("faultDisputeGame"));
    IPermissionedDisputeGame permissionedDisputeGame = IPermissionedDisputeGame(makeAddr("permissionedDisputeGame"));
    IDelayedWETH delayedWETHPermissionedGameProxy = IDelayedWETH(payable(makeAddr("delayedWETHPermissionedGameProxy")));
    IDelayedWETH delayedWETHPermissionlessGameProxy =
        IDelayedWETH(payable(makeAddr("delayedWETHPermissionlessGameProxy")));

    function setUp() public {
        doo = new DeployOPChainOutput();
    }

    function test_set_succeeds() public {
        vm.etch(address(opChainProxyAdmin), hex"01");
        vm.etch(address(addressManager), hex"01");
        vm.etch(address(l1ERC721BridgeProxy), hex"01");
        vm.etch(address(systemConfigProxy), hex"01");
        vm.etch(address(optimismMintableERC20FactoryProxy), hex"01");
        vm.etch(address(l1StandardBridgeProxy), hex"01");
        vm.etch(address(l1CrossDomainMessengerProxy), hex"01");
        vm.etch(address(optimismPortalProxy), hex"01");
        vm.etch(address(disputeGameFactoryProxy), hex"01");
        vm.etch(address(anchorStateRegistryProxy), hex"01");
        vm.etch(address(anchorStateRegistryImpl), hex"01");
        vm.etch(address(faultDisputeGame), hex"01");
        vm.etch(address(permissionedDisputeGame), hex"01");
        vm.etch(address(delayedWETHPermissionedGameProxy), hex"01");
        vm.etch(address(delayedWETHPermissionlessGameProxy), hex"01");

        doo.set(doo.opChainProxyAdmin.selector, address(opChainProxyAdmin));
        doo.set(doo.addressManager.selector, address(addressManager));
        doo.set(doo.l1ERC721BridgeProxy.selector, address(l1ERC721BridgeProxy));
        doo.set(doo.systemConfigProxy.selector, address(systemConfigProxy));
        doo.set(doo.optimismMintableERC20FactoryProxy.selector, address(optimismMintableERC20FactoryProxy));
        doo.set(doo.l1StandardBridgeProxy.selector, address(l1StandardBridgeProxy));
        doo.set(doo.l1CrossDomainMessengerProxy.selector, address(l1CrossDomainMessengerProxy));
        doo.set(doo.optimismPortalProxy.selector, address(optimismPortalProxy));
        doo.set(doo.disputeGameFactoryProxy.selector, address(disputeGameFactoryProxy));
        doo.set(doo.anchorStateRegistryProxy.selector, address(anchorStateRegistryProxy));
        doo.set(doo.anchorStateRegistryImpl.selector, address(anchorStateRegistryImpl));
        doo.set(doo.faultDisputeGame.selector, address(faultDisputeGame));
        doo.set(doo.permissionedDisputeGame.selector, address(permissionedDisputeGame));
        doo.set(doo.delayedWETHPermissionedGameProxy.selector, address(delayedWETHPermissionedGameProxy));
        doo.set(doo.delayedWETHPermissionlessGameProxy.selector, address(delayedWETHPermissionlessGameProxy));

        assertEq(address(opChainProxyAdmin), address(doo.opChainProxyAdmin()), "100");
        assertEq(address(addressManager), address(doo.addressManager()), "200");
        assertEq(address(l1ERC721BridgeProxy), address(doo.l1ERC721BridgeProxy()), "300");
        assertEq(address(systemConfigProxy), address(doo.systemConfigProxy()), "400");
        assertEq(address(optimismMintableERC20FactoryProxy), address(doo.optimismMintableERC20FactoryProxy()), "500");
        assertEq(address(l1StandardBridgeProxy), address(doo.l1StandardBridgeProxy()), "600");
        assertEq(address(l1CrossDomainMessengerProxy), address(doo.l1CrossDomainMessengerProxy()), "700");
        assertEq(address(optimismPortalProxy), address(doo.optimismPortalProxy()), "800");
        assertEq(address(disputeGameFactoryProxy), address(doo.disputeGameFactoryProxy()), "900");
        assertEq(address(anchorStateRegistryProxy), address(doo.anchorStateRegistryProxy()), "1100");
        assertEq(address(anchorStateRegistryImpl), address(doo.anchorStateRegistryImpl()), "1200");
        assertEq(address(faultDisputeGame), address(doo.faultDisputeGame()), "1300");
        assertEq(address(permissionedDisputeGame), address(doo.permissionedDisputeGame()), "1400");
        assertEq(address(delayedWETHPermissionedGameProxy), address(doo.delayedWETHPermissionedGameProxy()), "1500");
        assertEq(address(delayedWETHPermissionlessGameProxy), address(doo.delayedWETHPermissionlessGameProxy()), "1600");
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

    // Define default inputs for DeploySuperchain.
    address proxyAdminOwner = makeAddr("defaultProxyAdminOwner");
    address protocolVersionsOwner = makeAddr("defaultProtocolVersionsOwner");
    address guardian = makeAddr("defaultGuardian");
    bool paused = false;
    ProtocolVersion requiredProtocolVersion = ProtocolVersion.wrap(1);
    ProtocolVersion recommendedProtocolVersion = ProtocolVersion.wrap(2);

    // Define default inputs for DeployImplementations.
    // `superchainConfigProxy` and `protocolVersionsProxy` are set during `setUp` since they are
    // outputs of the previous step.
    uint256 withdrawalDelaySeconds = 100;
    uint256 minProposalSizeBytes = 200;
    uint256 challengePeriodSeconds = 300;
    uint256 proofMaturityDelaySeconds = 400;
    uint256 disputeGameFinalityDelaySeconds = 500;
    string release = "op-contracts/latest";
    ISuperchainConfig superchainConfigProxy;
    IProtocolVersions protocolVersionsProxy;

    // Define default inputs for DeployOPChain.
    // `opsm` is set during `setUp` since it is an output of the previous step.
    address opChainProxyAdminOwner = makeAddr("defaultOPChainProxyAdminOwner");
    address systemConfigOwner = makeAddr("defaultSystemConfigOwner");
    address batcher = makeAddr("defaultBatcher");
    address unsafeBlockSigner = makeAddr("defaultUnsafeBlockSigner");
    address proposer = makeAddr("defaultProposer");
    address challenger = makeAddr("defaultChallenger");
    uint32 basefeeScalar = 100;
    uint32 blobBaseFeeScalar = 200;
    uint256 l2ChainId = 300;
    OPStackManager opsm = OPStackManager(address(0));

    function setUp() public virtual {
        // Initialize deploy scripts.
        DeploySuperchain deploySuperchain = new DeploySuperchain();
        (DeploySuperchainInput dsi, DeploySuperchainOutput dso) = deploySuperchain.etchIOContracts();
        dsi.set(dsi.proxyAdminOwner.selector, proxyAdminOwner);
        dsi.set(dsi.protocolVersionsOwner.selector, protocolVersionsOwner);
        dsi.set(dsi.guardian.selector, guardian);
        dsi.set(dsi.paused.selector, paused);
        dsi.set(dsi.requiredProtocolVersion.selector, requiredProtocolVersion);
        dsi.set(dsi.recommendedProtocolVersion.selector, recommendedProtocolVersion);

        DeployImplementations deployImplementations = createDeployImplementationsContract();
        (DeployImplementationsInput dii, DeployImplementationsOutput dio) = deployImplementations.etchIOContracts();

        deployOPChain = new DeployOPChain();
        (doi, doo) = deployOPChain.etchIOContracts();

        // Deploy the superchain contracts.
        deploySuperchain.run(dsi, dso);

        // Populate the inputs for DeployImplementations based on the output of DeploySuperchain.
        superchainConfigProxy = dso.superchainConfigProxy();
        protocolVersionsProxy = dso.protocolVersionsProxy();

        // Deploy the implementations.
        dii.set(dii.withdrawalDelaySeconds.selector, withdrawalDelaySeconds);
        dii.set(dii.minProposalSizeBytes.selector, minProposalSizeBytes);
        dii.set(dii.challengePeriodSeconds.selector, challengePeriodSeconds);
        dii.set(dii.proofMaturityDelaySeconds.selector, proofMaturityDelaySeconds);
        dii.set(dii.disputeGameFinalityDelaySeconds.selector, disputeGameFinalityDelaySeconds);
        dii.set(dii.release.selector, release);
        dii.set(dii.superchainConfigProxy.selector, address(superchainConfigProxy));
        dii.set(dii.protocolVersionsProxy.selector, address(protocolVersionsProxy));
        deployImplementations.run(dii, dio);

        // Set the OPStackManager input for DeployOPChain.
        opsm = dio.opsmProxy();
    }

    // See the function of the same name in the `DeployImplementations_Test` contract of
    // `DeployImplementations.t.sol` for more details on why we use this method.
    function createDeployImplementationsContract() internal virtual returns (DeployImplementations) {
        return new DeployImplementations();
    }
}

contract DeployOPChain_Test is DeployOPChain_TestBase {
    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function testFuzz_run_memory_succeeds(bytes32 _seed) public {
        opChainProxyAdminOwner = address(uint160(uint256(hash(_seed, 0))));
        systemConfigOwner = address(uint160(uint256(hash(_seed, 1))));
        batcher = address(uint160(uint256(hash(_seed, 2))));
        unsafeBlockSigner = address(uint160(uint256(hash(_seed, 3))));
        proposer = address(uint160(uint256(hash(_seed, 4))));
        challenger = address(uint160(uint256(hash(_seed, 5))));
        basefeeScalar = uint32(uint256(hash(_seed, 6)));
        blobBaseFeeScalar = uint32(uint256(hash(_seed, 7)));
        l2ChainId = uint256(uint256(hash(_seed, 8)));

        doi.set(doi.opChainProxyAdminOwner.selector, opChainProxyAdminOwner);
        doi.set(doi.systemConfigOwner.selector, systemConfigOwner);
        doi.set(doi.batcher.selector, batcher);
        doi.set(doi.unsafeBlockSigner.selector, unsafeBlockSigner);
        doi.set(doi.proposer.selector, proposer);
        doi.set(doi.challenger.selector, challenger);
        doi.set(doi.basefeeScalar.selector, basefeeScalar);
        doi.set(doi.blobBaseFeeScalar.selector, blobBaseFeeScalar);
        doi.set(doi.l2ChainId.selector, l2ChainId);
        doi.set(doi.opsmProxy.selector, address(opsm)); // Not fuzzed since it must be an actual instance.

        deployOPChain.run(doi, doo);

        // TODO Add fault proof contract assertions below once OPSM fully supports them.

        // Assert that individual input fields were properly set based on the inputs.
        assertEq(opChainProxyAdminOwner, doi.opChainProxyAdminOwner(), "100");
        assertEq(systemConfigOwner, doi.systemConfigOwner(), "200");
        assertEq(batcher, doi.batcher(), "300");
        assertEq(unsafeBlockSigner, doi.unsafeBlockSigner(), "400");
        assertEq(proposer, doi.proposer(), "500");
        assertEq(challenger, doi.challenger(), "600");
        assertEq(basefeeScalar, doi.basefeeScalar(), "700");
        assertEq(blobBaseFeeScalar, doi.blobBaseFeeScalar(), "800");
        assertEq(l2ChainId, doi.l2ChainId(), "900");

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(doo.opChainProxyAdmin().owner()), opChainProxyAdminOwner, "2100");
        assertEq(address(doo.systemConfigProxy().owner()), systemConfigOwner, "2200");
        address batcherActual = address(uint160(uint256(doo.systemConfigProxy().batcherHash())));
        assertEq(batcherActual, batcher, "2300");
        assertEq(address(doo.systemConfigProxy().unsafeBlockSigner()), unsafeBlockSigner, "2400");
        // assertEq(address(...proposer()), proposer, "2500"); // TODO once we deploy dispute games.
        // assertEq(address(...challenger()), challenger, "2600"); // TODO once we deploy dispute games.

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
