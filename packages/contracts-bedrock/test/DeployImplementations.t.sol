// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, stdStorage, StdStorage } from "forge-std/Test.sol";
import { stdToml } from "forge-std/StdToml.sol";

import { DelayedWETH } from "src/dispute/DelayedWETH.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OPStackManager } from "src/L1/OPStackManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

import {
    DeployImplementationsInput,
    DeployImplementations,
    DeployImplementationsInterop,
    DeployImplementationsOutput
} from "scripts/DeployImplementations.s.sol";

contract DeployImplementationsInput_Test is Test {
    DeployImplementationsInput dii;

    bytes32 salt = bytes32(0);
    uint256 withdrawalDelaySeconds = 100;
    uint256 minProposalSizeBytes = 200;
    uint256 challengePeriodSeconds = 300;
    uint256 proofMaturityDelaySeconds = 400;
    uint256 disputeGameFinalityDelaySeconds = 500;
    string release = "op-contracts/latest";
    SuperchainConfig superchainConfigProxy = SuperchainConfig(makeAddr("superchainConfigProxy"));
    ProtocolVersions protocolVersionsProxy = ProtocolVersions(makeAddr("protocolVersionsProxy"));

    function setUp() public {
        dii = new DeployImplementationsInput();
    }

    function test_loadInputFile_succeeds() public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, "/test/fixtures/test-deploy-implementations-in.toml");

        dii.loadInputFile(path);

        // Compare the test inputs to the getter methods.
        assertEq(salt, dii.salt(), "100");
        assertEq(withdrawalDelaySeconds, dii.withdrawalDelaySeconds(), "200");
        assertEq(minProposalSizeBytes, dii.minProposalSizeBytes(), "300");
        assertEq(challengePeriodSeconds, dii.challengePeriodSeconds(), "400");
        assertEq(proofMaturityDelaySeconds, dii.proofMaturityDelaySeconds(), "500");
        assertEq(disputeGameFinalityDelaySeconds, dii.disputeGameFinalityDelaySeconds(), "600");
        assertEq(release, dii.release(), "700");

        assertEq(address(superchainConfigProxy), address(dii.superchainConfigProxy()), "800");
        assertEq(address(protocolVersionsProxy), address(dii.protocolVersionsProxy()), "900");
    }

    function test_getters_whenNotSet_revert() public {
        vm.expectRevert("DeployImplementationsInput: not set");
        dii.withdrawalDelaySeconds();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.minProposalSizeBytes();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.challengePeriodSeconds();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.proofMaturityDelaySeconds();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.disputeGameFinalityDelaySeconds();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.release();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.superchainConfigProxy();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.protocolVersionsProxy();

        vm.expectRevert("DeployImplementationsInput: not set");
        dii.superchainProxyAdmin();
    }

    function test_superchainProxyAdmin_whenNotSet_reverts() public {
        vm.expectRevert("DeployImplementationsInput: not set");
        dii.superchainProxyAdmin();

        dii.set(dii.superchainConfigProxy.selector, address(superchainConfigProxy));
        vm.expectRevert();
        dii.superchainProxyAdmin();

        Proxy noAdminProxy = new Proxy(address(0));
        dii.set(dii.superchainConfigProxy.selector, address(noAdminProxy));
        vm.expectRevert("DeployImplementationsInput: not set");
        dii.superchainProxyAdmin();
    }

    function test_superchainProxyAdmin_succeeds() public {
        Proxy proxyWithAdminSet = new Proxy(msg.sender);
        dii.set(dii.superchainConfigProxy.selector, address(proxyWithAdminSet));
        ProxyAdmin proxyAdmin = dii.superchainProxyAdmin();
        assertEq(address(msg.sender), address(proxyAdmin), "100");
    }
}

contract DeployImplementationsOutput_Test is Test {
    using stdToml for string;

    DeployImplementationsInput dii;
    DeployImplementationsOutput dio;

    function setUp() public {
        dii = new DeployImplementationsInput();
        dio = new DeployImplementationsOutput();
    }

    function test_set_succeeds() public {
        Proxy proxy = new Proxy(address(0));
        address opsmImpl = address(makeAddr("opsmImpl"));
        vm.prank(address(0));
        proxy.upgradeTo(opsmImpl);

        OPStackManager opsmProxy = OPStackManager(address(proxy));
        OptimismPortal2 optimismPortalImpl = OptimismPortal2(payable(makeAddr("optimismPortalImpl")));
        DelayedWETH delayedWETHImpl = DelayedWETH(payable(makeAddr("delayedWETHImpl")));
        PreimageOracle preimageOracleSingleton = PreimageOracle(makeAddr("preimageOracleSingleton"));
        MIPS mipsSingleton = MIPS(makeAddr("mipsSingleton"));
        SystemConfig systemConfigImpl = SystemConfig(makeAddr("systemConfigImpl"));
        L1CrossDomainMessenger l1CrossDomainMessengerImpl =
            L1CrossDomainMessenger(makeAddr("l1CrossDomainMessengerImpl"));
        L1ERC721Bridge l1ERC721BridgeImpl = L1ERC721Bridge(makeAddr("l1ERC721BridgeImpl"));
        L1StandardBridge l1StandardBridgeImpl = L1StandardBridge(payable(makeAddr("l1StandardBridgeImpl")));
        OptimismMintableERC20Factory optimismMintableERC20FactoryImpl =
            OptimismMintableERC20Factory(makeAddr("optimismMintableERC20FactoryImpl"));
        DisputeGameFactory disputeGameFactoryImpl = DisputeGameFactory(makeAddr("disputeGameFactoryImpl"));

        vm.etch(address(opsmProxy), address(opsmProxy).code);
        vm.etch(address(opsmImpl), hex"01");
        vm.etch(address(optimismPortalImpl), hex"01");
        vm.etch(address(delayedWETHImpl), hex"01");
        vm.etch(address(preimageOracleSingleton), hex"01");
        vm.etch(address(mipsSingleton), hex"01");
        vm.etch(address(systemConfigImpl), hex"01");
        vm.etch(address(l1CrossDomainMessengerImpl), hex"01");
        vm.etch(address(l1ERC721BridgeImpl), hex"01");
        vm.etch(address(l1StandardBridgeImpl), hex"01");
        vm.etch(address(optimismMintableERC20FactoryImpl), hex"01");
        vm.etch(address(disputeGameFactoryImpl), hex"01");

        dio.set(dio.opsmProxy.selector, address(opsmProxy));
        dio.set(dio.optimismPortalImpl.selector, address(optimismPortalImpl));
        dio.set(dio.delayedWETHImpl.selector, address(delayedWETHImpl));
        dio.set(dio.preimageOracleSingleton.selector, address(preimageOracleSingleton));
        dio.set(dio.mipsSingleton.selector, address(mipsSingleton));
        dio.set(dio.systemConfigImpl.selector, address(systemConfigImpl));
        dio.set(dio.l1CrossDomainMessengerImpl.selector, address(l1CrossDomainMessengerImpl));
        dio.set(dio.l1ERC721BridgeImpl.selector, address(l1ERC721BridgeImpl));
        dio.set(dio.l1StandardBridgeImpl.selector, address(l1StandardBridgeImpl));
        dio.set(dio.optimismMintableERC20FactoryImpl.selector, address(optimismMintableERC20FactoryImpl));
        dio.set(dio.disputeGameFactoryImpl.selector, address(disputeGameFactoryImpl));

        assertEq(address(opsmProxy), address(dio.opsmProxy()), "50");
        assertEq(address(optimismPortalImpl), address(dio.optimismPortalImpl()), "100");
        assertEq(address(delayedWETHImpl), address(dio.delayedWETHImpl()), "200");
        assertEq(address(preimageOracleSingleton), address(dio.preimageOracleSingleton()), "300");
        assertEq(address(mipsSingleton), address(dio.mipsSingleton()), "400");
        assertEq(address(systemConfigImpl), address(dio.systemConfigImpl()), "500");
        assertEq(address(l1CrossDomainMessengerImpl), address(dio.l1CrossDomainMessengerImpl()), "600");
        assertEq(address(l1ERC721BridgeImpl), address(dio.l1ERC721BridgeImpl()), "700");
        assertEq(address(l1StandardBridgeImpl), address(dio.l1StandardBridgeImpl()), "800");
        assertEq(address(optimismMintableERC20FactoryImpl), address(dio.optimismMintableERC20FactoryImpl()), "900");
        assertEq(address(disputeGameFactoryImpl), address(dio.disputeGameFactoryImpl()), "950");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployUtils: zero address";

        vm.expectRevert(expectedErr);
        dio.optimismPortalImpl();

        vm.expectRevert(expectedErr);
        dio.delayedWETHImpl();

        vm.expectRevert(expectedErr);
        dio.preimageOracleSingleton();

        vm.expectRevert(expectedErr);
        dio.mipsSingleton();

        vm.expectRevert(expectedErr);
        dio.systemConfigImpl();

        vm.expectRevert(expectedErr);
        dio.l1CrossDomainMessengerImpl();

        vm.expectRevert(expectedErr);
        dio.l1ERC721BridgeImpl();

        vm.expectRevert(expectedErr);
        dio.l1StandardBridgeImpl();

        vm.expectRevert(expectedErr);
        dio.optimismMintableERC20FactoryImpl();

        vm.expectRevert(expectedErr);
        dio.disputeGameFactoryImpl();
    }

    function test_getters_whenAddrHasNoCode_reverts() public {
        address emptyAddr = makeAddr("emptyAddr");
        bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

        dio.set(dio.optimismPortalImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.optimismPortalImpl();

        dio.set(dio.delayedWETHImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.delayedWETHImpl();

        dio.set(dio.preimageOracleSingleton.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.preimageOracleSingleton();

        dio.set(dio.mipsSingleton.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.mipsSingleton();

        dio.set(dio.systemConfigImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.systemConfigImpl();

        dio.set(dio.l1CrossDomainMessengerImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.l1CrossDomainMessengerImpl();

        dio.set(dio.l1ERC721BridgeImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.l1ERC721BridgeImpl();

        dio.set(dio.l1StandardBridgeImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.l1StandardBridgeImpl();

        dio.set(dio.optimismMintableERC20FactoryImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dio.optimismMintableERC20FactoryImpl();
    }

    function test_outputFileInputDataPrepended_succeeds() public {
        string memory root = vm.projectRoot();

        // Use the expected data from the test fixture.
        string memory expOutPath = string.concat(root, "/test/fixtures/test-deploy-implementations-out.toml");
        string memory expOutToml = vm.readFile(expOutPath);

        // Parse each dii field of expOutToml
        dii.loadInputFile(string.concat(root, "/test/fixtures/test-deploy-implementations-in.toml"));

        address protocolVersionsProxy = expOutToml.readAddress(".protocolVersionsProxy");
        assertEq(protocolVersionsProxy, address(dii.protocolVersionsProxy()), "100");
        string memory release = expOutToml.readString(".release");
        assertEq(release, dii.release(), "200");
        bytes32 salt = expOutToml.readBytes32(".salt");
        assertEq(salt, dii.salt(), "300");
        address superchainConfigProxy = expOutToml.readAddress(".superchainConfigProxy");
        assertEq(superchainConfigProxy, address(dii.superchainConfigProxy()), "400");
        uint256 challengePeriodSeconds = expOutToml.readUint(".faultProofs.challengePeriodSeconds");
        assertEq(challengePeriodSeconds, dii.challengePeriodSeconds(), "500");
        uint256 disputeGameFinalityDelaySeconds = expOutToml.readUint(".faultProofs.disputeGameFinalityDelaySeconds");
        assertEq(disputeGameFinalityDelaySeconds, dii.disputeGameFinalityDelaySeconds(), "600");
        uint256 minProposalSizeBytes = expOutToml.readUint(".faultProofs.minProposalSizeBytes");
        assertEq(minProposalSizeBytes, dii.minProposalSizeBytes(), "700");
        uint256 proofMaturityDelaySeconds = expOutToml.readUint(".faultProofs.proofMaturityDelaySeconds");
        assertEq(proofMaturityDelaySeconds, dii.proofMaturityDelaySeconds(), "800");
        uint256 withdrawalDelaySeconds = expOutToml.readUint(".faultProofs.withdrawalDelaySeconds");
        assertEq(withdrawalDelaySeconds, dii.withdrawalDelaySeconds(), "900");
    }

    function test_writeOutputFile_succeeds() public {
        string memory root = vm.projectRoot();

        // Use the expected data from the test fixture.
        string memory expOutPath = string.concat(root, "/test/fixtures/test-deploy-implementations-out.toml");
        string memory expOutToml = vm.readFile(expOutPath);

        // Load the input file to use later when writing new output file.
        dii.loadInputFile(string.concat(root, "/test/fixtures/test-deploy-implementations-in.toml"));

        // Parse outputs
        OPStackManager opsmProxy = OPStackManager(address(Proxy(payable(expOutToml.readAddress(".outputs.opsmProxy")))));
        DelayedWETH delayedWETHImpl = DelayedWETH(payable(expOutToml.readAddress(".outputs.delayedWETHImpl")));
        OptimismPortal2 optimismPortalImpl =
            OptimismPortal2(payable(expOutToml.readAddress(".outputs.optimismPortalImpl")));
        PreimageOracle preimageOracleSingleton =
            PreimageOracle(expOutToml.readAddress(".outputs.preimageOracleSingleton"));
        MIPS mipsSingleton = MIPS(expOutToml.readAddress(".outputs.mipsSingleton"));
        SystemConfig systemConfigImpl = SystemConfig(expOutToml.readAddress(".outputs.systemConfigImpl"));
        L1CrossDomainMessenger l1CrossDomainMessengerImpl =
            L1CrossDomainMessenger(expOutToml.readAddress(".outputs.l1CrossDomainMessengerImpl"));
        L1ERC721Bridge l1ERC721BridgeImpl = L1ERC721Bridge(expOutToml.readAddress(".outputs.l1ERC721BridgeImpl"));
        L1StandardBridge l1StandardBridgeImpl =
            L1StandardBridge(payable(expOutToml.readAddress(".outputs.l1StandardBridgeImpl")));
        OptimismMintableERC20Factory optimismMintableERC20FactoryImpl =
            OptimismMintableERC20Factory(expOutToml.readAddress(".outputs.optimismMintableERC20FactoryImpl"));
        DisputeGameFactory disputeGameFactoryImpl =
            DisputeGameFactory(expOutToml.readAddress(".outputs.disputeGameFactoryImpl"));

        // Etch code at each address so the code checks pass when settings values.
        vm.etch(address(opsmProxy), address(new Proxy(address(0))).code);
        address opsmImpl = address(makeAddr("opsmImpl"));
        vm.prank(address(0));
        Proxy(payable(address(opsmProxy))).upgradeTo(opsmImpl);
        vm.etch(address(opsmImpl), hex"01");
        vm.etch(address(delayedWETHImpl), hex"01");
        vm.etch(address(optimismPortalImpl), hex"01");
        vm.etch(address(preimageOracleSingleton), hex"01");
        vm.etch(address(mipsSingleton), hex"01");
        vm.etch(address(systemConfigImpl), hex"01");
        vm.etch(address(l1CrossDomainMessengerImpl), hex"01");
        vm.etch(address(l1ERC721BridgeImpl), hex"01");
        vm.etch(address(l1StandardBridgeImpl), hex"01");
        vm.etch(address(optimismMintableERC20FactoryImpl), hex"01");
        vm.etch(address(disputeGameFactoryImpl), hex"01");

        // Manually set the values in the DeployImplementationsOutput contract.
        dio.set(dio.opsmProxy.selector, address(opsmProxy));
        dio.set(dio.delayedWETHImpl.selector, address(delayedWETHImpl));
        dio.set(dio.optimismPortalImpl.selector, address(optimismPortalImpl));
        dio.set(dio.preimageOracleSingleton.selector, address(preimageOracleSingleton));
        dio.set(dio.mipsSingleton.selector, address(mipsSingleton));
        dio.set(dio.systemConfigImpl.selector, address(systemConfigImpl));
        dio.set(dio.l1CrossDomainMessengerImpl.selector, address(l1CrossDomainMessengerImpl));
        dio.set(dio.l1ERC721BridgeImpl.selector, address(l1ERC721BridgeImpl));
        dio.set(dio.l1StandardBridgeImpl.selector, address(l1StandardBridgeImpl));
        dio.set(dio.optimismMintableERC20FactoryImpl.selector, address(optimismMintableERC20FactoryImpl));
        dio.set(dio.disputeGameFactoryImpl.selector, address(disputeGameFactoryImpl));

        // File must end with a new line because Foundry adds a new line at the end of the file.
        string memory actOutPath = string.concat(root, "/.testdata/test-deploy-implementations-output.toml");
        // Foundry serializes TOML key-value pairs in alphabetical order when writing to a TOML file.
        dio.writeOutputFile(dii, actOutPath);
        string memory actOutToml = vm.readFile(actOutPath);

        // Clean up before asserting so that we don't leave any files behind.
        vm.removeFile(actOutPath);

        assertEq(expOutToml, actOutToml);
    }
}

contract DeployImplementations_Test is Test {
    using stdStorage for StdStorage;

    DeployImplementations deployImplementations;
    DeployImplementationsInput dii;
    DeployImplementationsOutput dio;

    // Define default inputs for testing.
    uint256 withdrawalDelaySeconds = 100;
    uint256 minProposalSizeBytes = 200;
    uint256 challengePeriodSeconds = 300;
    uint256 proofMaturityDelaySeconds = 400;
    uint256 disputeGameFinalityDelaySeconds = 500;
    string release = "op-contracts/latest";
    SuperchainConfig superchainConfigProxy = SuperchainConfig(makeAddr("superchainConfigProxy"));
    ProtocolVersions protocolVersionsProxy = ProtocolVersions(makeAddr("protocolVersionsProxy"));

    function setUp() public virtual {
        deployImplementations = new DeployImplementations();
        (dii, dio) = deployImplementations.etchIOContracts();
    }

    // By deploying the `DeployImplementations` contract with this virtual function, we provide a
    // hook that child contracts can override to return a different implementation of the contract.
    // This lets us test e.g. the `DeployImplementationsInterop` contract without duplicating test code.
    function createDeployImplementationsContract() internal virtual returns (DeployImplementations) {
        return new DeployImplementations();
    }

    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function testFuzz_run_memory_succeeds(bytes32 _seed) public {
        withdrawalDelaySeconds = uint256(hash(_seed, 0));
        minProposalSizeBytes = uint256(hash(_seed, 1));
        challengePeriodSeconds = bound(uint256(hash(_seed, 2)), 0, type(uint64).max);
        proofMaturityDelaySeconds = uint256(hash(_seed, 3));
        disputeGameFinalityDelaySeconds = uint256(hash(_seed, 4));
        release = string(bytes.concat(hash(_seed, 5)));
        protocolVersionsProxy = ProtocolVersions(address(uint160(uint256(hash(_seed, 7)))));

        // Must configure the ProxyAdmin contract which is used to upgrade the OPSM's proxy contract.
        ProxyAdmin superchainProxyAdmin = new ProxyAdmin(msg.sender);
        superchainConfigProxy = SuperchainConfig(address(new Proxy(payable(address(superchainProxyAdmin)))));

        SuperchainConfig superchainConfigImpl = SuperchainConfig(address(uint160(uint256(hash(_seed, 6)))));
        vm.prank(address(superchainProxyAdmin));
        Proxy(payable(address(superchainConfigProxy))).upgradeTo(address(superchainConfigImpl));

        vm.etch(address(superchainProxyAdmin), address(superchainProxyAdmin).code);
        vm.etch(address(superchainConfigProxy), address(superchainConfigProxy).code);
        vm.etch(address(protocolVersionsProxy), hex"01");

        dii.set(dii.withdrawalDelaySeconds.selector, withdrawalDelaySeconds);
        dii.set(dii.minProposalSizeBytes.selector, minProposalSizeBytes);
        dii.set(dii.challengePeriodSeconds.selector, challengePeriodSeconds);
        dii.set(dii.proofMaturityDelaySeconds.selector, proofMaturityDelaySeconds);
        dii.set(dii.disputeGameFinalityDelaySeconds.selector, disputeGameFinalityDelaySeconds);
        dii.set(dii.release.selector, release);
        dii.set(dii.superchainConfigProxy.selector, address(superchainConfigProxy));
        dii.set(dii.protocolVersionsProxy.selector, address(protocolVersionsProxy));

        deployImplementations.run(dii, dio);

        // Assert that individual input fields were properly set based on the inputs.
        assertEq(withdrawalDelaySeconds, dii.withdrawalDelaySeconds(), "100");
        assertEq(minProposalSizeBytes, dii.minProposalSizeBytes(), "200");
        assertEq(challengePeriodSeconds, dii.challengePeriodSeconds(), "300");
        assertEq(proofMaturityDelaySeconds, dii.proofMaturityDelaySeconds(), "400");
        assertEq(disputeGameFinalityDelaySeconds, dii.disputeGameFinalityDelaySeconds(), "500");
        assertEq(release, dii.release(), "525");
        assertEq(address(superchainConfigProxy), address(dii.superchainConfigProxy()), "550");
        assertEq(address(protocolVersionsProxy), address(dii.protocolVersionsProxy()), "575");
        assertEq(address(superchainProxyAdmin), address(dii.superchainProxyAdmin()), "580");

        // Architecture assertions.
        assertEq(address(dio.mipsSingleton().oracle()), address(dio.preimageOracleSingleton()), "600");

        // Ensure that `checkOutput` passes. This is called by the `run` function during execution,
        // so this just acts as a sanity check. It reverts on failure.
        dio.checkOutput(dii);
    }

    function getOutPathForFileIOTest() internal pure virtual returns (string memory) {
        return "/.testdata/test-deploy-implementations-out.toml";
    }

    function test_run_io_succeeds() public {
        string memory root = vm.projectRoot();
        string memory inpath = string.concat(root, "/test/fixtures/test-deploy-implementations-in.toml");
        string memory outpath = string.concat(root, getOutPathForFileIOTest());

        vm.etch(address(superchainConfigProxy), address(new Proxy(address(0))).code);
        vm.prank(address(0));
        Proxy(payable(address(superchainConfigProxy))).changeAdmin(msg.sender);
        vm.etch(address(protocolVersionsProxy), hex"01");

        deployImplementations.run(inpath, outpath);

        string memory actOutToml = vm.readFile(outpath);
        string memory expOutToml =
            vm.readFile(string.concat(root, "/test/fixtures/test-deploy-implementations-out.toml"));

        // Clean up before asserting so that we don't leave any files behind.
        vm.removeFile(outpath);

        assertEq(expOutToml, actOutToml);
    }

    function testFuzz_run_largeChallengePeriodSeconds_reverts(uint256 _challengePeriodSeconds) public {
        // Set the defaults.
        dii.set(dii.withdrawalDelaySeconds.selector, withdrawalDelaySeconds);
        dii.set(dii.minProposalSizeBytes.selector, minProposalSizeBytes);
        dii.set(dii.challengePeriodSeconds.selector, challengePeriodSeconds);
        dii.set(dii.proofMaturityDelaySeconds.selector, proofMaturityDelaySeconds);
        dii.set(dii.disputeGameFinalityDelaySeconds.selector, disputeGameFinalityDelaySeconds);
        dii.set(dii.release.selector, release);
        dii.set(dii.superchainConfigProxy.selector, address(superchainConfigProxy));
        dii.set(dii.protocolVersionsProxy.selector, address(protocolVersionsProxy));

        // Set the challenge period to a value that is too large, using vm.store because the setter
        // method won't allow it.
        challengePeriodSeconds = bound(_challengePeriodSeconds, uint256(type(uint64).max) + 1, type(uint256).max);
        uint256 slot =
            stdstore.enable_packed_slots().target(address(dii)).sig(dii.challengePeriodSeconds.selector).find();
        vm.store(address(dii), bytes32(slot), bytes32(challengePeriodSeconds));

        vm.expectRevert("DeployImplementationsInput: challengePeriodSeconds too large");
        deployImplementations.run(dii, dio);
    }
}

contract DeployImplementationsInterop_Test is DeployImplementations_Test {
    function createDeployImplementationsContract() internal override returns (DeployImplementations) {
        return new DeployImplementationsInterop();
    }

    // Avoiding unwanted interactions with file I/O tests.
    function getOutPathForFileIOTest() internal pure override returns (string memory) {
        return "/.testdata/test-deploy-implementations-interop-out.toml";
    }
}
