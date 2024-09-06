// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { stdToml } from "forge-std/StdToml.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";

contract DeploySuperchainInput_Test is Test {
    DeploySuperchainInput dsi;

    address proxyAdminOwner = makeAddr("defaultProxyAdminOwner");
    address protocolVersionsOwner = makeAddr("defaultProtocolVersionsOwner");
    address guardian = makeAddr("defaultGuardian");
    bool paused = false;
    ProtocolVersion requiredProtocolVersion = ProtocolVersion.wrap(1);
    ProtocolVersion recommendedProtocolVersion = ProtocolVersion.wrap(2);

    function setUp() public {
        dsi = new DeploySuperchainInput();
    }

    function test_loadInputFile_succeeds() public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, "/test/fixtures/test-deploy-superchain-in.toml");

        dsi.loadInputFile(path);

        // Compare the test input struct to the getter methods.
        assertEq(proxyAdminOwner, dsi.proxyAdminOwner(), "100");
        assertEq(protocolVersionsOwner, dsi.protocolVersionsOwner(), "200");
        assertEq(guardian, dsi.guardian(), "300");
        assertEq(paused, dsi.paused(), "400");
        assertEq(
            ProtocolVersion.unwrap(requiredProtocolVersion),
            ProtocolVersion.unwrap(dsi.requiredProtocolVersion()),
            "500"
        );
        assertEq(
            ProtocolVersion.unwrap(recommendedProtocolVersion),
            ProtocolVersion.unwrap(dsi.recommendedProtocolVersion()),
            "600"
        );
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeploySuperchainInput: input not set";

        vm.expectRevert(expectedErr);
        dsi.proxyAdminOwner();

        vm.expectRevert(expectedErr);
        dsi.protocolVersionsOwner();

        vm.expectRevert(expectedErr);
        dsi.guardian();

        vm.expectRevert(expectedErr);
        dsi.paused();

        vm.expectRevert(expectedErr);
        dsi.requiredProtocolVersion();

        vm.expectRevert(expectedErr);
        dsi.recommendedProtocolVersion();
    }
}

contract DeploySuperchainOutput_Test is Test {
    using stdToml for string;

    DeploySuperchainOutput dso;

    function setUp() public {
        dso = new DeploySuperchainOutput();
    }

    function test_set_succeeds() public {
        // We don't fuzz, because we need code at the address, and we can't etch code if the fuzzer
        // provides precompiles, so we end up with a lot of boilerplate logic to get valid inputs.
        // Hardcoding a concrete set of valid addresses is simpler and still sufficient.
        ProxyAdmin superchainProxyAdmin = ProxyAdmin(makeAddr("superchainProxyAdmin"));
        SuperchainConfig superchainConfigImpl = SuperchainConfig(makeAddr("superchainConfigImpl"));
        SuperchainConfig superchainConfigProxy = SuperchainConfig(makeAddr("superchainConfigProxy"));
        ProtocolVersions protocolVersionsImpl = ProtocolVersions(makeAddr("protocolVersionsImpl"));
        ProtocolVersions protocolVersionsProxy = ProtocolVersions(makeAddr("protocolVersionsProxy"));

        // Ensure each address has code, since these are expected to be contracts.
        vm.etch(address(superchainProxyAdmin), hex"01");
        vm.etch(address(superchainConfigImpl), hex"01");
        vm.etch(address(superchainConfigProxy), hex"01");
        vm.etch(address(protocolVersionsImpl), hex"01");
        vm.etch(address(protocolVersionsProxy), hex"01");

        // Set the output data.
        dso.set(dso.superchainProxyAdmin.selector, address(superchainProxyAdmin));
        dso.set(dso.superchainConfigImpl.selector, address(superchainConfigImpl));
        dso.set(dso.superchainConfigProxy.selector, address(superchainConfigProxy));
        dso.set(dso.protocolVersionsImpl.selector, address(protocolVersionsImpl));
        dso.set(dso.protocolVersionsProxy.selector, address(protocolVersionsProxy));

        // Compare the test output struct to the getter methods.
        assertEq(address(superchainProxyAdmin), address(dso.superchainProxyAdmin()), "100");
        assertEq(address(superchainConfigImpl), address(dso.superchainConfigImpl()), "200");
        assertEq(address(superchainConfigProxy), address(dso.superchainConfigProxy()), "300");
        assertEq(address(protocolVersionsImpl), address(dso.protocolVersionsImpl()), "400");
        assertEq(address(protocolVersionsProxy), address(dso.protocolVersionsProxy()), "500");
    }

    function test_getters_whenNotSet_revert() public {
        bytes memory expectedErr = "DeployUtils: zero address";

        vm.expectRevert(expectedErr);
        dso.superchainProxyAdmin();

        vm.expectRevert(expectedErr);
        dso.superchainConfigImpl();

        vm.expectRevert(expectedErr);
        dso.superchainConfigProxy();

        vm.expectRevert(expectedErr);
        dso.protocolVersionsImpl();

        vm.expectRevert(expectedErr);
        dso.protocolVersionsProxy();
    }

    function test_getters_whenAddrHasNoCode_reverts() public {
        address emptyAddr = makeAddr("emptyAddr");
        bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

        dso.set(dso.superchainProxyAdmin.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.superchainProxyAdmin();

        dso.set(dso.superchainConfigImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.superchainConfigImpl();

        dso.set(dso.superchainConfigProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.superchainConfigProxy();

        dso.set(dso.protocolVersionsImpl.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.protocolVersionsImpl();

        dso.set(dso.protocolVersionsProxy.selector, emptyAddr);
        vm.expectRevert(expectedErr);
        dso.protocolVersionsProxy();
    }

    function test_writeOutputFile_succeeds() public {
        string memory root = vm.projectRoot();

        // Use the expected data from the test fixture.
        string memory expOutPath = string.concat(root, "/test/fixtures/test-deploy-superchain-out.toml");
        string memory expOutToml = vm.readFile(expOutPath);

        // Parse each field of expOutToml individually.
        ProxyAdmin expSuperchainProxyAdmin = ProxyAdmin(expOutToml.readAddress("superchainProxyAdmin"));
        SuperchainConfig expSuperchainConfigImpl = SuperchainConfig(expOutToml.readAddress("superchainConfigImpl"));
        SuperchainConfig expSuperchainConfigProxy = SuperchainConfig(expOutToml.readAddress("superchainConfigProxy"));
        ProtocolVersions expProtocolVersionsImpl = ProtocolVersions(expOutToml.readAddress("protocolVersionsImpl"));
        ProtocolVersions expProtocolVersionsProxy = ProtocolVersions(expOutToml.readAddress("protocolVersionsProxy"));

        dso.set(dso.superchainProxyAdmin.selector, address(expSuperchainProxyAdmin));
        dso.set(dso.superchainConfigImpl.selector, address(expSuperchainConfigImpl));
        dso.set(dso.superchainConfigProxy.selector, address(expSuperchainConfigProxy));
        dso.set(dso.protocolVersionsImpl.selector, address(expProtocolVersionsImpl));
        dso.set(dso.protocolVersionsProxy.selector, address(expProtocolVersionsProxy));

        string memory actOutPath = string.concat(root, "/.testdata/test-deploy-superchain-output.toml");
        dso.writeOutputFile(actOutPath);
        string memory actOutToml = vm.readFile(actOutPath);

        // Clean up before asserting so that we don't leave any files behind.
        vm.removeFile(actOutPath);

        assertEq(expOutToml, actOutToml);
    }
}

contract DeploySuperchain_Test is Test {
    DeploySuperchain deploySuperchain;
    DeploySuperchainInput dsi;
    DeploySuperchainOutput dso;

    // Define default input variables for testing.
    address defaultProxyAdminOwner = makeAddr("defaultProxyAdminOwner");
    address defaultProtocolVersionsOwner = makeAddr("defaultProtocolVersionsOwner");
    address defaultGuardian = makeAddr("defaultGuardian");
    bool defaultPaused = false;
    ProtocolVersion defaultRequiredProtocolVersion = ProtocolVersion.wrap(1);
    ProtocolVersion defaultRecommendedProtocolVersion = ProtocolVersion.wrap(2);

    function setUp() public {
        deploySuperchain = new DeploySuperchain();
        (dsi, dso) = deploySuperchain.getIOContracts();
    }

    function unwrap(ProtocolVersion _pv) internal pure returns (uint256) {
        return ProtocolVersion.unwrap(_pv);
    }

    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function test_run_memory_succeeds(bytes32 _seed) public {
        // Generate random input values from the seed. This doesn't give us the benefit of the forge
        // fuzzer's dictionary, but that's ok because we are just testing that values are set and
        // passed correctly.
        address proxyAdminOwner = address(uint160(uint256(hash(_seed, 0))));
        address protocolVersionsOwner = address(uint160(uint256(hash(_seed, 1))));
        address guardian = address(uint160(uint256(hash(_seed, 2))));
        bool paused = bool(uint8(uint256(hash(_seed, 3))) % 2 == 0);
        ProtocolVersion requiredProtocolVersion = ProtocolVersion.wrap(uint256(hash(_seed, 4)));
        ProtocolVersion recommendedProtocolVersion = ProtocolVersion.wrap(uint256(hash(_seed, 5)));

        // Set the input values on the input contract.
        dsi.set(dsi.proxyAdminOwner.selector, proxyAdminOwner);
        dsi.set(dsi.protocolVersionsOwner.selector, protocolVersionsOwner);
        dsi.set(dsi.guardian.selector, guardian);
        dsi.set(dsi.paused.selector, paused);
        dsi.set(dsi.requiredProtocolVersion.selector, requiredProtocolVersion);
        dsi.set(dsi.recommendedProtocolVersion.selector, recommendedProtocolVersion);

        // Run the deployment script.
        deploySuperchain.run(dsi, dso);

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(dso.superchainProxyAdmin().owner()), proxyAdminOwner, "100");
        assertEq(address(dso.protocolVersionsProxy().owner()), protocolVersionsOwner, "200");
        assertEq(address(dso.superchainConfigProxy().guardian()), guardian, "300");
        assertEq(dso.superchainConfigProxy().paused(), paused, "400");
        assertEq(unwrap(dso.protocolVersionsProxy().required()), unwrap(requiredProtocolVersion), "500");
        assertEq(unwrap(dso.protocolVersionsProxy().recommended()), unwrap(recommendedProtocolVersion), "600");

        // Architecture assertions.
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        Proxy superchainConfigProxy = Proxy(payable(address(dso.superchainConfigProxy())));
        Proxy protocolVersionsProxy = Proxy(payable(address(dso.protocolVersionsProxy())));

        vm.startPrank(address(0));
        assertEq(superchainConfigProxy.implementation(), address(dso.superchainConfigImpl()), "700");
        assertEq(protocolVersionsProxy.implementation(), address(dso.protocolVersionsImpl()), "800");
        assertEq(superchainConfigProxy.admin(), protocolVersionsProxy.admin(), "900");
        assertEq(superchainConfigProxy.admin(), address(dso.superchainProxyAdmin()), "1000");
        vm.stopPrank();

        // Ensure that `checkOutput` passes. This is called by the `run` function during execution,
        // so this just acts as a sanity check. It reverts on failure.
        dso.checkOutput();
    }

    function test_run_io_succeeds() public {
        string memory root = vm.projectRoot();
        string memory inpath = string.concat(root, "/test/fixtures/test-deploy-superchain-in.toml");
        string memory outpath = string.concat(root, "/.testdata/test-deploy-superchain-out.toml");

        deploySuperchain.run(inpath, outpath);

        string memory actOutToml = vm.readFile(outpath);
        string memory expOutToml = vm.readFile(string.concat(root, "/test/fixtures/test-deploy-superchain-out.toml"));

        // Clean up before asserting so that we don't leave any files behind.
        vm.removeFile(outpath);
        assertEq(expOutToml, actOutToml);
    }

    function test_run_ZeroAddressRoleInput_reverts() public {
        // Snapshot the state so we can revert to the default `input` struct between assertions.
        uint256 snapshotId = vm.snapshot();

        // Assert over each role being set to the zero address.
        // input.roles.proxyAdminOwner = address(0);
        vm.expectRevert("DeploySuperchainInput: null proxyAdminOwner");
        deploySuperchain.run(dsi, dso);

        vm.revertTo(snapshotId);
        // input.roles.protocolVersionsOwner = address(0);
        vm.expectRevert("DeploySuperchainInput: null protocolVersionsOwner");
        deploySuperchain.run(dsi, dso);

        vm.revertTo(snapshotId);
        // input.roles.guardian = address(0);
        vm.expectRevert("DeploySuperchainInput: null guardian");
        deploySuperchain.run(dsi, dso);
    }
}
