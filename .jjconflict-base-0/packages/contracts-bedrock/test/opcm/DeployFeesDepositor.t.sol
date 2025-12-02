// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { DeployFeesDepositor } from "scripts/deploy/DeployFeesDepositor.s.sol";

// Contracts
import { FeesDepositor } from "src/L1/FeesDepositor.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Interfaces
import { IFeesDepositor } from "interfaces/L1/IFeesDepositor.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title DeployFeesDepositor_Test
/// @notice This test is used to test the DeployFeesDepositor script.
contract DeployFeesDepositor_Test is Test {
    DeployFeesDepositor deployFeesDepositor;

    // Define default input variables for testing.
    address defaultProxyAdmin = makeAddr("defaultProxyAdmin");
    address defaultL2Recipient = makeAddr("defaultL2Recipient");
    IL1CrossDomainMessenger defaultMessenger = IL1CrossDomainMessenger(makeAddr("defaultMessenger"));
    uint96 defaultMinDepositAmount = 1 ether;
    uint32 defaultGasLimit = 200_000;

    /// @notice Sets up the test suite.
    function setUp() public {
        deployFeesDepositor = new DeployFeesDepositor();
    }

    /// @notice Tests that the DeployFeesDepositor script succeeds with valid fuzzed input values.
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
        (IFeesDepositor feesDepositorImpl, IProxy feesDepositorProxy) =
            deployFeesDepositor.run(_proxyAdmin, _minDepositAmount, _l2Recipient, _messenger, _gasLimit);

        // Verify the implementation is deployed correctly.
        FeesDepositor impl = new FeesDepositor();
        assertEq(address(feesDepositorImpl).code, address(impl).code, "Implementation code mismatch");

        // Verify the proxy is deployed correctly.
        Proxy proxy = new Proxy(_proxyAdmin);
        assertEq(address(feesDepositorProxy).code, address(proxy).code, "Proxy code mismatch");

        // Verify the proxy admin is set correctly.
        assertEq(EIP1967Helper.getAdmin(address(feesDepositorProxy)), _proxyAdmin, "Proxy admin mismatch");

        // Verify the proxy implementation is set correctly.
        assertEq(
            EIP1967Helper.getImplementation(address(feesDepositorProxy)),
            address(feesDepositorImpl),
            "Proxy implementation mismatch"
        );

        // Verify the FeesDepositor is initialized correctly.
        FeesDepositor feesDepositor = FeesDepositor(payable(address(feesDepositorProxy)));
        assertEq(feesDepositor.minDepositAmount(), _minDepositAmount, "MinDepositAmount mismatch");
        assertEq(feesDepositor.l2Recipient(), _l2Recipient, "L2Recipient mismatch");
        assertEq(address(feesDepositor.messenger()), _messenger, "Messenger mismatch");
        assertEq(feesDepositor.gasLimit(), _gasLimit, "GasLimit mismatch");
    }

    /// @notice Tests that the DeployFeesDepositor script reverts when called with zero input values.
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

    /// @notice Tests that the DeployFeesDepositor script succeeds when called with default input values.
    function test_run_defaultInput_succeeds() public {
        (IFeesDepositor feesDepositorImpl, IProxy feesDepositorProxy) = deployFeesDepositor.run(
            defaultProxyAdmin, defaultMinDepositAmount, defaultL2Recipient, address(defaultMessenger), defaultGasLimit
        );

        // Verify addresses are non-zero.
        assertNotEq(address(feesDepositorImpl), address(0), "Implementation address is zero");
        assertNotEq(address(feesDepositorProxy), address(0), "Proxy address is zero");

        // Verify contracts have code.
        assertGt(address(feesDepositorImpl).code.length, 0, "Implementation has no code");
        assertGt(address(feesDepositorProxy).code.length, 0, "Proxy has no code");

        // Verify the FeesDepositor is initialized correctly.
        IFeesDepositor feesDepositor = IFeesDepositor(payable(address(feesDepositorProxy)));
        assertEq(feesDepositor.minDepositAmount(), defaultMinDepositAmount, "MinDepositAmount mismatch");
        assertEq(feesDepositor.l2Recipient(), defaultL2Recipient, "L2Recipient mismatch");
        assertEq(address(feesDepositor.messenger()), address(defaultMessenger), "Messenger mismatch");
        assertEq(feesDepositor.gasLimit(), defaultGasLimit, "GasLimit mismatch");
    }
}
