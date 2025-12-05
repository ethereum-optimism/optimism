// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { Proxy } from "src/universal/Proxy.sol";
import { ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
import { DeploySuperchain } from "scripts/deploy/DeploySuperchain.s.sol";

contract DeploySuperchain_Test is Test {
    DeploySuperchain deploySuperchain;

    // Define default input variables for testing.
    address defaultProxyAdminOwner = makeAddr("defaultProxyAdminOwner");
    address defaultProtocolVersionsOwner = makeAddr("defaultProtocolVersionsOwner");
    address defaultGuardian = makeAddr("defaultGuardian");
    bool defaultPaused = false;
    bytes32 defaultRequiredProtocolVersion = bytes32(uint256(1));
    bytes32 defaultRecommendedProtocolVersion = bytes32(uint256(2));

    function setUp() public {
        deploySuperchain = new DeploySuperchain();
    }

    function unwrap(ProtocolVersion _pv) internal pure returns (bytes32) {
        return bytes32(ProtocolVersion.unwrap(_pv));
    }

    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function testFuzz_run_memory_succeeds(
        address _superchainProxyAdminOwner,
        address _protocolVersionsOwner,
        address _guardian,
        bool _paused,
        bytes32 _recommendedProtocolVersion,
        bytes32 _requiredProtocolVersion
    )
        public
    {
        vm.assume(_superchainProxyAdminOwner != address(0));
        vm.assume(_protocolVersionsOwner != address(0));
        vm.assume(_guardian != address(0));
        vm.assume(_recommendedProtocolVersion != bytes32(0));
        vm.assume(_requiredProtocolVersion != bytes32(0));

        DeploySuperchain.Input memory dsi = DeploySuperchain.Input(
            _guardian,
            _protocolVersionsOwner,
            _superchainProxyAdminOwner,
            _paused,
            _recommendedProtocolVersion,
            _requiredProtocolVersion
        );

        // Run the deployment script.
        DeploySuperchain.Output memory dso = deploySuperchain.run(dsi);

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(dso.superchainProxyAdmin.owner()), _superchainProxyAdminOwner, "100");
        assertEq(address(dso.protocolVersionsProxy.owner()), _protocolVersionsOwner, "200");
        assertEq(address(dso.superchainConfigProxy.guardian()), _guardian, "300");
        assertEq(unwrap(dso.protocolVersionsProxy.required()), _requiredProtocolVersion, "500");
        assertEq(unwrap(dso.protocolVersionsProxy.recommended()), _recommendedProtocolVersion, "600");

        // Architecture assertions.
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        Proxy superchainConfigProxy = Proxy(payable(address(dso.superchainConfigProxy)));
        Proxy protocolVersionsProxy = Proxy(payable(address(dso.protocolVersionsProxy)));

        vm.startPrank(address(0));
        assertEq(superchainConfigProxy.implementation(), address(dso.superchainConfigImpl), "700");
        assertEq(protocolVersionsProxy.implementation(), address(dso.protocolVersionsImpl), "800");
        assertEq(superchainConfigProxy.admin(), protocolVersionsProxy.admin(), "900");
        assertEq(superchainConfigProxy.admin(), address(dso.superchainProxyAdmin), "1000");
        vm.stopPrank();
    }

    function test_run_nullInput_reverts() public {
        DeploySuperchain.Input memory input;

        input = defaultInput();
        input.superchainProxyAdminOwner = address(0);
        vm.expectRevert("DeploySuperchain: superchainProxyAdminOwner not set");
        deploySuperchain.run(input);

        input = defaultInput();
        input.protocolVersionsOwner = address(0);
        vm.expectRevert("DeploySuperchain: protocolVersionsOwner not set");
        deploySuperchain.run(input);

        input = defaultInput();
        input.guardian = address(0);
        vm.expectRevert("DeploySuperchain: guardian not set");
        deploySuperchain.run(input);

        input = defaultInput();
        input.requiredProtocolVersion = bytes32(0);
        vm.expectRevert("DeploySuperchain: requiredProtocolVersion not set");
        deploySuperchain.run(input);

        input = defaultInput();
        input.recommendedProtocolVersion = bytes32(0);
        vm.expectRevert("DeploySuperchain: recommendedProtocolVersion not set");
        deploySuperchain.run(input);
    }

    function test_reuseAddresses_succeeds() public {
        DeploySuperchain.Input memory input = defaultInput();

        DeploySuperchain.Output memory output0 = deploySuperchain.run(input);
        DeploySuperchain.Output memory output1 = deploySuperchain.run(input);

        // We make sure that the implementation contracts are reused.
        assertEq(address(output0.superchainConfigImpl), address(output1.superchainConfigImpl), "100");
        assertEq(address(output0.protocolVersionsImpl), address(output1.protocolVersionsImpl), "200");

        // And we make sure that the proxy ones are redeployed
        assertNotEq(address(output0.superchainConfigProxy), address(output1.superchainConfigProxy), "300");
    }

    function defaultInput() internal view returns (DeploySuperchain.Input memory input_) {
        input_ = DeploySuperchain.Input(
            defaultGuardian,
            defaultProtocolVersionsOwner,
            defaultProxyAdminOwner,
            defaultPaused,
            defaultRecommendedProtocolVersion,
            defaultRequiredProtocolVersion
        );
    }
}
