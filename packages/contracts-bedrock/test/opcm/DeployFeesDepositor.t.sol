// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

// Interfaces
import { IFeesDepositor } from "interfaces/L1/IFeesDepositor.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

import { DeployFeesDepositor } from "scripts/deploy/DeployFeesDepositor.s.sol";
import { FeesDepositor } from "src/L1/FeesDepositor.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

contract DeployFeesDepositor_Test is Test {
    DeployFeesDepositor deployFeesDepositor;

    // Define default input variables for testing.
    address defaultProxyAdmin = makeAddr("defaultProxyAdmin");
    address defaultL2Recipient = makeAddr("defaultL2Recipient");
    IL1CrossDomainMessenger defaultMessenger = IL1CrossDomainMessenger(makeAddr("defaultMessenger"));
    uint96 defaultMinDepositAmount = 1 ether;
    uint32 defaultGasLimit = 200_000;

    function setUp() public {
        deployFeesDepositor = new DeployFeesDepositor();
    }

    function testFuzz_run_succeeds(
        address _proxyAdmin,
        uint96 _minDepositAmount,
        address _l2Recipient,
        address _messenger,
        uint32 _gasLimit
    )
        public
    {
        vm.assume(_proxyAdmin != address(0));
        vm.assume(_l2Recipient != address(0));
        vm.assume(_messenger != address(0));
        vm.assume(_minDepositAmount > 0);
        vm.assume(_gasLimit > 0);

        // Run the deployment script.
        DeployFeesDepositor.Output memory output1 =
            deployFeesDepositor.run(_proxyAdmin, _minDepositAmount, _l2Recipient, _messenger, _gasLimit);

        // Verify the implementation is deployed correctly.
        FeesDepositor impl = new FeesDepositor();
        assertEq(output1.feesDepositorImpl.code, address(impl).code, "Implementation code mismatch");

        // Verify the proxy is deployed correctly.
        Proxy proxy = new Proxy(_proxyAdmin);
        assertEq(output1.feesDepositorProxy.code, address(proxy).code, "Proxy code mismatch");

        // Verify the proxy admin is set correctly.
        assertEq(EIP1967Helper.getAdmin(output1.feesDepositorProxy), _proxyAdmin, "Proxy admin mismatch");

        // Verify the proxy implementation is set correctly.
        assertEq(
            EIP1967Helper.getImplementation(output1.feesDepositorProxy),
            output1.feesDepositorImpl,
            "Proxy implementation mismatch"
        );

        // Verify the FeesDepositor is initialized correctly.
        FeesDepositor feesDepositor = FeesDepositor(payable(output1.feesDepositorProxy));
        assertEq(feesDepositor.minDepositAmount(), _minDepositAmount, "MinDepositAmount mismatch");
        assertEq(feesDepositor.l2Recipient(), _l2Recipient, "L2Recipient mismatch");
        assertEq(address(feesDepositor.messenger()), _messenger, "Messenger mismatch");
        assertEq(feesDepositor.gasLimit(), _gasLimit, "GasLimit mismatch");
    }

    function test_run_nullInput_reverts() public {
        // Test zero proxyAdmin
        vm.expectRevert("DeployFeesDepositor: proxyAdmin cannot be zero address");
        deployFeesDepositor.run(
            address(0), defaultMinDepositAmount, defaultL2Recipient, address(defaultMessenger), defaultGasLimit
        );

        // Test zero l2Recipient
        vm.expectRevert("DeployFeesDepositor: l2Recipient cannot be zero address");
        deployFeesDepositor.run(
            defaultProxyAdmin, defaultMinDepositAmount, address(0), address(defaultMessenger), defaultGasLimit
        );

        // Test zero messenger
        vm.expectRevert("DeployFeesDepositor: messenger cannot be zero address");
        deployFeesDepositor.run(
            defaultProxyAdmin, defaultMinDepositAmount, defaultL2Recipient, address(0), defaultGasLimit
        );

        // Test zero minDepositAmount
        vm.expectRevert("DeployFeesDepositor: minDepositAmount must be greater than zero");
        deployFeesDepositor.run(defaultProxyAdmin, 0, defaultL2Recipient, address(defaultMessenger), defaultGasLimit);

        // Test zero gasLimit
        vm.expectRevert("DeployFeesDepositor: gasLimit must be greater than zero");
        deployFeesDepositor.run(
            defaultProxyAdmin, defaultMinDepositAmount, defaultL2Recipient, address(defaultMessenger), 0
        );
    }

    function test_run_defaultInput_succeeds() public {
        DeployFeesDepositor.Output memory output = deployFeesDepositor.run(
            defaultProxyAdmin, defaultMinDepositAmount, defaultL2Recipient, address(defaultMessenger), defaultGasLimit
        );

        // Verify addresses are non-zero.
        assertNotEq(output.feesDepositorImpl, address(0), "Implementation address is zero");
        assertNotEq(output.feesDepositorProxy, address(0), "Proxy address is zero");

        // Verify contracts have code.
        assertGt(output.feesDepositorImpl.code.length, 0, "Implementation has no code");
        assertGt(output.feesDepositorProxy.code.length, 0, "Proxy has no code");

        // Verify the FeesDepositor is initialized correctly.
        FeesDepositor feesDepositor = FeesDepositor(payable(output.feesDepositorProxy));
        assertEq(feesDepositor.minDepositAmount(), defaultMinDepositAmount, "MinDepositAmount mismatch");
        assertEq(feesDepositor.l2Recipient(), defaultL2Recipient, "L2Recipient mismatch");
        assertEq(address(feesDepositor.messenger()), address(defaultMessenger), "Messenger mismatch");
        assertEq(feesDepositor.gasLimit(), defaultGasLimit, "GasLimit mismatch");
    }
}

