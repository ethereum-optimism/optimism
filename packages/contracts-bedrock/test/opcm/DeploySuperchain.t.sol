// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeploySuperchain } from "scripts/deploy/DeploySuperchain.s.sol";

// Contracts
import { Proxy } from "src/universal/Proxy.sol";

contract DeploySuperchain_Test is Test {
    DeploySuperchain deploySuperchain;

    // Define default input variables for testing.
    address defaultProxyAdminOwner = makeAddr("defaultProxyAdminOwner");
    address defaultGuardian = makeAddr("defaultGuardian");
    bool defaultPaused = false;

    function setUp() public {
        deploySuperchain = new DeploySuperchain();
    }

    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function testFuzz_run_memory_succeeds(address _superchainProxyAdminOwner, address _guardian, bool _paused) public {
        vm.assume(_superchainProxyAdminOwner != address(0));
        vm.assume(_guardian != address(0));

        DeploySuperchain.Input memory dsi = DeploySuperchain.Input(_guardian, _superchainProxyAdminOwner, _paused);

        // Run the deployment script.
        DeploySuperchain.Output memory dso = deploySuperchain.run(dsi);

        // Assert inputs were properly passed through to the contract initializers.
        assertEq(address(dso.superchainProxyAdmin.owner()), _superchainProxyAdminOwner, "100");
        assertEq(address(dso.superchainConfigProxy.guardian()), _guardian, "300");

        // Architecture assertions.
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        Proxy superchainConfigProxy = Proxy(payable(address(dso.superchainConfigProxy)));

        vm.startPrank(address(0));
        assertEq(superchainConfigProxy.implementation(), address(dso.superchainConfigImpl), "700");
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
        input.guardian = address(0);
        vm.expectRevert("DeploySuperchain: guardian not set");
        deploySuperchain.run(input);
    }

    function test_reuseAddresses_succeeds() public {
        DeploySuperchain.Input memory input = defaultInput();

        DeploySuperchain.Output memory output0 = deploySuperchain.run(input);
        DeploySuperchain.Output memory output1 = deploySuperchain.run(input);

        // We make sure that the implementation contracts are reused.
        assertEq(address(output0.superchainConfigImpl), address(output1.superchainConfigImpl), "100");

        // And we make sure that the proxy ones are redeployed
        assertNotEq(address(output0.superchainConfigProxy), address(output1.superchainConfigProxy), "300");
    }

    function defaultInput() internal view returns (DeploySuperchain.Input memory input_) {
        input_ = DeploySuperchain.Input(defaultGuardian, defaultProxyAdminOwner, defaultPaused);
    }
}
