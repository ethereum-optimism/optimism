// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { DeploySuperchainInput, DeploySuperchain, DeploySuperchainOutput } from "scripts/DeploySuperchain.s.sol";

contract DeploySuperchainInput_Test is Test {
    DeploySuperchainInput dsi;

    DeploySuperchainInput.Input input = DeploySuperchainInput.Input({
        roles: DeploySuperchainInput.Roles({
            proxyAdminOwner: makeAddr("defaultProxyAdminOwner"),
            protocolVersionsOwner: makeAddr("defaultProtocolVersionsOwner"),
            guardian: makeAddr("defaultGuardian")
        }),
        paused: false,
        requiredProtocolVersion: ProtocolVersion.wrap(1),
        recommendedProtocolVersion: ProtocolVersion.wrap(2)
    });

    function setUp() public {
        dsi = new DeploySuperchainInput();
    }

    function test_loadInput_succeeds() public {
        // We avoid using a fuzz test here because we'd need to modify the inputs of multiple
        // parameters to e.g. avoid the zero address. Therefore we hardcode a concrete test case
        // which is simpler and still sufficient.
        dsi.loadInput(input);
        assertLoadInput();
    }

    function test_loadInputFile_succeeds() public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, "/test/fixtures/test-deploy-superchain-in.toml");

        dsi.loadInputFile(path);
        assertLoadInput();
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

    function assertLoadInput() internal view {
        assertTrue(dsi.inputSet(), "100");

        // Compare the test input struct to the getter methods.
        assertEq(input.roles.proxyAdminOwner, dsi.proxyAdminOwner(), "200");
        assertEq(input.roles.protocolVersionsOwner, dsi.protocolVersionsOwner(), "300");
        assertEq(input.roles.guardian, dsi.guardian(), "400");
        assertEq(input.paused, dsi.paused(), "500");
        assertEq(
            ProtocolVersion.unwrap(input.requiredProtocolVersion),
            ProtocolVersion.unwrap(dsi.requiredProtocolVersion()),
            "600"
        );
        assertEq(
            ProtocolVersion.unwrap(input.recommendedProtocolVersion),
            ProtocolVersion.unwrap(dsi.recommendedProtocolVersion()),
            "700"
        );

        // Compare the test input struct to the `input` getter method.
        assertEq(keccak256(abi.encode(input)), keccak256(abi.encode(dsi.input())), "800");
    }
}

contract DeploySuperchainOutput_Test is Test {
    DeploySuperchainOutput dso;

    function setUp() public {
        dso = new DeploySuperchainOutput();
    }

    function test_set_succeeds() public {
        // We don't fuzz, because we need code at the address, and we can't etch code if the fuzzer
        // provides precompiles, so we end up with a lot of boilerplate logic to get valid inputs.
        // Hardcoding a concrete set of valid addresses is simpler and still sufficient.
        DeploySuperchainOutput.Output memory output = DeploySuperchainOutput.Output({
            superchainProxyAdmin: ProxyAdmin(makeAddr("superchainProxyAdmin")),
            superchainConfigImpl: SuperchainConfig(makeAddr("superchainConfigImpl")),
            superchainConfigProxy: SuperchainConfig(makeAddr("superchainConfigProxy")),
            protocolVersionsImpl: ProtocolVersions(makeAddr("protocolVersionsImpl")),
            protocolVersionsProxy: ProtocolVersions(makeAddr("protocolVersionsProxy"))
        });

        // Ensure each address has code, since these are expected to be contracts.
        vm.etch(address(output.superchainProxyAdmin), hex"01");
        vm.etch(address(output.superchainConfigImpl), hex"01");
        vm.etch(address(output.superchainConfigProxy), hex"01");
        vm.etch(address(output.protocolVersionsImpl), hex"01");
        vm.etch(address(output.protocolVersionsProxy), hex"01");

        // Set the output data.
        dso.set(dso.superchainProxyAdmin.selector, address(output.superchainProxyAdmin));
        dso.set(dso.superchainConfigImpl.selector, address(output.superchainConfigImpl));
        dso.set(dso.superchainConfigProxy.selector, address(output.superchainConfigProxy));
        dso.set(dso.protocolVersionsImpl.selector, address(output.protocolVersionsImpl));
        dso.set(dso.protocolVersionsProxy.selector, address(output.protocolVersionsProxy));

        // Compare the test output struct to the getter methods.
        assertEq(address(output.superchainProxyAdmin), address(dso.superchainProxyAdmin()), "100");
        assertEq(address(output.superchainConfigImpl), address(dso.superchainConfigImpl()), "200");
        assertEq(address(output.superchainConfigProxy), address(dso.superchainConfigProxy()), "300");
        assertEq(address(output.protocolVersionsImpl), address(dso.protocolVersionsImpl()), "400");
        assertEq(address(output.protocolVersionsProxy), address(dso.protocolVersionsProxy()), "500");

        // Compare the test output struct to the `output` getter method.
        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(dso.output())), "600");
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
        bytes memory expOutData = vm.parseToml(expOutToml);
        DeploySuperchainOutput.Output memory expOutput = abi.decode(expOutData, (DeploySuperchainOutput.Output));

        dso.set(dso.superchainProxyAdmin.selector, address(expOutput.superchainProxyAdmin));
        dso.set(dso.superchainConfigImpl.selector, address(expOutput.superchainConfigImpl));
        dso.set(dso.superchainConfigProxy.selector, address(expOutput.superchainConfigProxy));
        dso.set(dso.protocolVersionsImpl.selector, address(expOutput.protocolVersionsImpl));
        dso.set(dso.protocolVersionsProxy.selector, address(expOutput.protocolVersionsProxy));

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

    // Define a default input struct for testing.
    DeploySuperchainInput.Input input = DeploySuperchainInput.Input({
        roles: DeploySuperchainInput.Roles({
            proxyAdminOwner: makeAddr("defaultProxyAdminOwner"),
            protocolVersionsOwner: makeAddr("defaultProtocolVersionsOwner"),
            guardian: makeAddr("defaultGuardian")
        }),
        paused: false,
        requiredProtocolVersion: ProtocolVersion.wrap(1),
        recommendedProtocolVersion: ProtocolVersion.wrap(2)
    });

    function setUp() public {
        deploySuperchain = new DeploySuperchain();
        (dsi, dso) = deploySuperchain.getIOContracts();
    }

    function unwrap(ProtocolVersion _pv) internal pure returns (uint256) {
        return ProtocolVersion.unwrap(_pv);
    }

    function test_run_memory_succeeds(DeploySuperchainInput.Input memory _input) public {
        vm.assume(_input.roles.proxyAdminOwner != address(0));
        vm.assume(_input.roles.protocolVersionsOwner != address(0));
        vm.assume(_input.roles.guardian != address(0));

        DeploySuperchainOutput.Output memory output = deploySuperchain.run(_input);

        // Assert that individual input fields were properly set based on the input struct.
        assertEq(_input.roles.proxyAdminOwner, dsi.proxyAdminOwner(), "100");
        assertEq(_input.roles.protocolVersionsOwner, dsi.protocolVersionsOwner(), "200");
        assertEq(_input.roles.guardian, dsi.guardian(), "300");
        assertEq(_input.paused, dsi.paused(), "400");
        assertEq(unwrap(_input.requiredProtocolVersion), unwrap(dsi.requiredProtocolVersion()), "500");
        assertEq(unwrap(_input.recommendedProtocolVersion), unwrap(dsi.recommendedProtocolVersion()), "600");

        // Assert that individual output fields were properly set based on the output struct.
        assertEq(address(output.superchainProxyAdmin), address(dso.superchainProxyAdmin()), "700");
        assertEq(address(output.superchainConfigImpl), address(dso.superchainConfigImpl()), "800");
        assertEq(address(output.superchainConfigProxy), address(dso.superchainConfigProxy()), "900");
        assertEq(address(output.protocolVersionsImpl), address(dso.protocolVersionsImpl()), "1000");
        assertEq(address(output.protocolVersionsProxy), address(dso.protocolVersionsProxy()), "1100");

        // Assert that the full input and output structs were properly set.
        assertEq(keccak256(abi.encode(_input)), keccak256(abi.encode(DeploySuperchainInput(dsi).input())), "1200");
        assertEq(keccak256(abi.encode(output)), keccak256(abi.encode(DeploySuperchainOutput(dso).output())), "1300");

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(output.superchainProxyAdmin.owner()), _input.roles.proxyAdminOwner, "1400");
        assertEq(address(output.protocolVersionsProxy.owner()), _input.roles.protocolVersionsOwner, "1500");
        assertEq(address(output.superchainConfigProxy.guardian()), _input.roles.guardian, "1600");
        assertEq(output.superchainConfigProxy.paused(), _input.paused, "1700");
        assertEq(unwrap(output.protocolVersionsProxy.required()), unwrap(_input.requiredProtocolVersion), "1800");
        assertEq(unwrap(output.protocolVersionsProxy.recommended()), unwrap(_input.recommendedProtocolVersion), "1900");

        // Architecture assertions.
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        Proxy superchainConfigProxy = Proxy(payable(address(output.superchainConfigProxy)));
        Proxy protocolVersionsProxy = Proxy(payable(address(output.protocolVersionsProxy)));

        vm.startPrank(address(0));
        assertEq(superchainConfigProxy.implementation(), address(output.superchainConfigImpl), "900");
        assertEq(protocolVersionsProxy.implementation(), address(output.protocolVersionsImpl), "1000");
        assertEq(superchainConfigProxy.admin(), protocolVersionsProxy.admin(), "1100");
        assertEq(superchainConfigProxy.admin(), address(output.superchainProxyAdmin), "1200");
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
        input.roles.proxyAdminOwner = address(0);
        vm.expectRevert("DeploySuperchainInput: null proxyAdminOwner");
        deploySuperchain.run(input);

        vm.revertTo(snapshotId);
        input.roles.protocolVersionsOwner = address(0);
        vm.expectRevert("DeploySuperchainInput: null protocolVersionsOwner");
        deploySuperchain.run(input);

        vm.revertTo(snapshotId);
        input.roles.guardian = address(0);
        vm.expectRevert("DeploySuperchainInput: null guardian");
        deploySuperchain.run(input);
    }
}
