// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IFeesDepositor } from "interfaces/L1/IFeesDepositor.sol";
import { FeesDepositor } from "src/L1/FeesDepositor.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { Features } from "src/libraries/Features.sol";

/// @title FeesDepositor_TestInit
/// @notice Base test contract with initialization for `FeesDepositor` tests.
contract FeesDepositor_TestInit is CommonTest {
    // Events
    event FeesDeposited(address indexed l2Recipient, uint256 amount);
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event MinDepositAmountUpdated(uint96 oldMinDepositAmount, uint96 newMinDepositAmount);
    event L2RecipientUpdated(address oldL2Recipient, address newL2Recipient);
    event GasLimitUpdated(uint32 oldGasLimit, uint32 newGasLimit);

    // Test state
    FeesDepositor feesDepositor;
    address l2Recipient = makeAddr("l2Recipient");
    uint96 minDepositAmount = 1 ether;
    uint32 gasLimit = 150_000;
    address depositFeesRecipient;

    /// @notice Test setup.
    function setUp() public virtual override {
        super.setUp();

        // Deploy FeesDepositor implementation
        address implementation = DeployUtils.create1({
            _name: "FeesDepositor",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IFeesDepositor.__constructor__, ()))
        });

        // Deploy proxy pointing to proxyAdmin
        address proxy = address(new Proxy(address(proxyAdmin)));

        // Set implementation
        vm.prank(address(proxyAdmin));
        Proxy(payable(proxy)).upgradeTo(implementation);

        // Cast proxy to FeesDepositor
        feesDepositor = FeesDepositor(payable(proxy));

        // Initialize through proxy
        vm.prank(proxyAdminOwner);
        feesDepositor.initialize(minDepositAmount, l2Recipient, l1CrossDomainMessenger, gasLimit);

        // Set depositFeesRecipient
        depositFeesRecipient =
            systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX) ? address(ethLockbox) : address(optimismPortal2);
    }
}

/// @title FeesDepositor_Initialize_Test
/// @notice Tests the initialization of the `FeesDepositor` contract.
contract FeesDepositor_Initialize_Test is FeesDepositor_TestInit {
    /// @notice This contract is excluded from the Initializable.t.sol test because it is not deployed as part of the
    /// standard deployment script and instead is deployed manually, that's why we have this test.
    function test_cannotReinitialize_succeeds() public {
        vm.expectRevert("Initializable: contract is already initialized");
        feesDepositor.initialize(minDepositAmount, l2Recipient, l1CrossDomainMessenger, gasLimit);
    }
}

/// @title FeesDepositor_Receive_Test
/// @notice Tests the receive function of the `FeesDepositor` contract.
contract FeesDepositor_Receive_Test is FeesDepositor_TestInit {
    function testFuzz_receive_belowThreshold_succeeds(uint256 _amount) external {
        // Handling the fork tests scenario
        uint256 depositFeesRecipientBalanceBefore = depositFeesRecipient.balance;
        _amount = bound(_amount, 0, minDepositAmount - 1);

        vm.deal(address(this), _amount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _amount, _amount);

        // Expect call to the messenger not to be done
        vm.expectCall(
            address(l1CrossDomainMessenger),
            _amount,
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (l2Recipient, hex"", gasLimit)),
            0
        );

        (bool success,) = address(feesDepositor).call{ value: _amount }("");

        assertTrue(success);
        assertEq(address(feesDepositor).balance, _amount);
        assertEq(address(depositFeesRecipient).balance, depositFeesRecipientBalanceBefore);
    }

    function testFuzz_receive_atOrAboveThreshold_succeeds(uint256 _sendAmount) external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        // Handling the fork tests scenario case for the fork tests
        uint256 depositFeesRecipientBalanceBefore = depositFeesRecipient.balance;
        _sendAmount = bound(_sendAmount, minDepositAmount, type(uint256).max - depositFeesRecipientBalanceBefore);

        vm.deal(address(this), _sendAmount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _sendAmount, _sendAmount);

        vm.expectEmit(address(feesDepositor));
        emit FeesDeposited(l2Recipient, _sendAmount);

        vm.expectCall(
            address(l1CrossDomainMessenger),
            _sendAmount,
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (l2Recipient, hex"", gasLimit))
        );

        (bool success,) = address(feesDepositor).call{ value: _sendAmount }("");

        assertTrue(success);
        assertEq(address(feesDepositor).balance, 0);
        assertEq(address(depositFeesRecipient).balance, depositFeesRecipientBalanceBefore + _sendAmount);
    }

    function testFuzz_receive_multipleDeposits_succeeds(uint256 _firstAmount, uint256 _secondAmount) external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        // Handling the fork tests scenario
        uint256 depositFeesRecipientBalanceBefore = depositFeesRecipient.balance;
        // First amount should not exceed minDepositAmount (so it doesn't trigger deposit)
        _firstAmount = bound(_firstAmount, 0, minDepositAmount - 1);

        // First deposit (should not trigger deposit)
        vm.deal(address(this), _firstAmount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _firstAmount, _firstAmount);

        (bool success1,) = address(feesDepositor).call{ value: _firstAmount }("");
        assertTrue(success1);
        assertEq(address(feesDepositor).balance, _firstAmount);
        assertEq(
            address(depositFeesRecipient).balance, depositFeesRecipientBalanceBefore, "depositFeesRecipient balance 1"
        );

        // Second amount should ensure total reaches threshold to trigger deposit
        _secondAmount = bound(
            _secondAmount,
            minDepositAmount - _firstAmount,
            type(uint256).max - depositFeesRecipient.balance - _firstAmount
        );

        uint256 totalAmount = _firstAmount + _secondAmount;

        // Second deposit (will trigger deposit since total >= minDepositAmount)
        vm.deal(address(this), _secondAmount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _secondAmount, totalAmount);

        vm.expectEmit(address(feesDepositor));
        emit FeesDeposited(l2Recipient, totalAmount);

        vm.expectCall(
            address(l1CrossDomainMessenger),
            totalAmount,
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (l2Recipient, hex"", gasLimit))
        );

        (bool success2,) = address(feesDepositor).call{ value: _secondAmount }("");
        assertTrue(success2);

        // Verify deposit occurred
        assertEq(address(feesDepositor).balance, 0);
        assertEq(
            address(depositFeesRecipient).balance,
            depositFeesRecipientBalanceBefore + totalAmount,
            "depositFeesRecipient balance 2"
        );
    }
}

/// @title FeesDepositor_SetMinDepositAmount_Test
/// @notice Tests the setMinDepositAmount function of the `FeesDepositor` contract.
contract FeesDepositor_SetMinDepositAmount_Test is FeesDepositor_TestInit {
    function testFuzz_setMinDepositAmount_asOwner_succeeds(uint96 _newMinDepositAmount) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(feesDepositor));
        emit MinDepositAmountUpdated(minDepositAmount, _newMinDepositAmount);

        vm.prank(owner);
        feesDepositor.setMinDepositAmount(_newMinDepositAmount);

        assertEq(feesDepositor.minDepositAmount(), _newMinDepositAmount);
    }

    function testFuzz_setMinDepositAmount_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        uint96 newMinDepositAmount = 2 ether;

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        feesDepositor.setMinDepositAmount(newMinDepositAmount);

        assertEq(feesDepositor.minDepositAmount(), minDepositAmount);
    }
}

/// @title FeesDepositor_SetL2Recipient_Test
/// @notice Tests the setL2Recipient function of the `FeesDepositor` contract.
contract FeesDepositor_SetL2Recipient_Test is FeesDepositor_TestInit {
    function testFuzz_setL2Recipient_asOwner_succeeds(address _newL2Recipient) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(feesDepositor));
        emit L2RecipientUpdated(l2Recipient, _newL2Recipient);

        vm.prank(owner);
        feesDepositor.setL2Recipient(_newL2Recipient);

        assertEq(feesDepositor.l2Recipient(), _newL2Recipient);
    }

    function testFuzz_setL2Recipient_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        address newL2Recipient = makeAddr("newL2Recipient");

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        feesDepositor.setL2Recipient(newL2Recipient);

        assertEq(feesDepositor.l2Recipient(), l2Recipient);
    }
}

/// @title FeesDepositor_SetGasLimit_Test
/// @notice Tests the setGasLimit function of the `FeesDepositor` contract.
contract FeesDepositor_SetGasLimit_Test is FeesDepositor_TestInit {
    function testFuzz_setGasLimit_asOwner_succeeds(uint32 _newGasLimit) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(feesDepositor));
        emit GasLimitUpdated(gasLimit, _newGasLimit);

        vm.prank(owner);
        feesDepositor.setGasLimit(_newGasLimit);

        assertEq(feesDepositor.gasLimit(), _newGasLimit);
    }

    function testFuzz_setGasLimit_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        uint32 newGasLimit = 200_000;

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        feesDepositor.setGasLimit(newGasLimit);

        assertEq(feesDepositor.gasLimit(), gasLimit);
    }
}
