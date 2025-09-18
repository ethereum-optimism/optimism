// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IFeesDepositor } from "interfaces/L1/IFeesDepositor.sol";
import { FeesDepositor } from "src/L1/FeesDepositor.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { Features } from "src/libraries/Features.sol";

/// @title FeesDepositor_Uncategorized_Test
/// @notice Tests all functionality of FeesDepositor including receive, deposit, and setters.
contract FeesDepositor_Uncategorized_Test is CommonTest {
    FeesDepositor feesDepositor;

    address l2Recipient = makeAddr("l2Recipient");
    uint96 minDepositAmount = 1 ether;
    uint64 gasLimit = 150_000;
    bytes depositData = hex"1234";

    event FeesDeposited(address indexed l2Recipient, uint256 amount);
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event MinDepositAmountUpdated(uint96 oldminDepositAmount, uint96 newminDepositAmount);
    event L2RecipientUpdated(address oldL2Recipient, address newL2Recipient);
    event GasLimitUpdated(uint64 oldGasLimit, uint64 newGasLimit);
    event DepositDataUpdated(bytes oldDepositData, bytes newDepositData);

    function setUp() public override {
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
        feesDepositor.initialize(minDepositAmount, l2Recipient, optimismPortal2, gasLimit, depositData);
    }

    /// @notice This contract is excluded from the Initializable.t.sol test because it is not deployed as part of the
    /// standard deployment script and instead is deployed manually, that's why we have this test.
    function test_cannotReinitialize_succeeds() public {
        vm.expectRevert("Initializable: contract is already initialized");
        feesDepositor.initialize(minDepositAmount, l2Recipient, optimismPortal2, gasLimit, depositData);
    }

    function testFuzz_receive_belowThreshold_succeeds(uint256 _amount) external {
        // Handling the fork tests scenario
        address depositFeesRecipient =
            systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX) ? address(ethLockbox) : address(optimismPortal2);
        uint256 depositFeesRecipientBalanceBefore = depositFeesRecipient.balance;
        _amount = bound(_amount, 0, minDepositAmount - 1);

        vm.deal(address(this), _amount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _amount, _amount);

        // Expect call to the portal not to be done
        vm.expectCall(
            address(optimismPortal2),
            _amount,
            abi.encodeCall(IOptimismPortal.depositTransaction, (l2Recipient, _amount, gasLimit, false, depositData)),
            0
        );

        (bool success,) = address(feesDepositor).call{ value: _amount }("");

        assertTrue(success);
        assertEq(address(feesDepositor).balance, _amount);
        assertEq(address(depositFeesRecipient).balance, depositFeesRecipientBalanceBefore);
    }

    function testFuzz_receive_atOrAboveThreshold_succeeds(uint256 _sendAmount) external {
        // Handling the fork tests scenario case for the fork tests
        address depositFeesRecipient =
            systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX) ? address(ethLockbox) : address(optimismPortal2);
        uint256 depositFeesRecipientBalanceBefore = depositFeesRecipient.balance;
        _sendAmount = bound(_sendAmount, minDepositAmount, type(uint256).max - depositFeesRecipientBalanceBefore);

        vm.deal(address(this), _sendAmount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _sendAmount, _sendAmount);

        vm.expectEmit(address(feesDepositor));
        emit FeesDeposited(l2Recipient, _sendAmount);

        vm.expectCall(
            address(optimismPortal2),
            _sendAmount,
            abi.encodeCall(IOptimismPortal.depositTransaction, (l2Recipient, _sendAmount, gasLimit, false, depositData))
        );

        (bool success,) = address(feesDepositor).call{ value: _sendAmount }("");

        assertTrue(success);
        assertEq(address(feesDepositor).balance, 0);
        assertEq(address(depositFeesRecipient).balance, depositFeesRecipientBalanceBefore + _sendAmount);
    }

    function testFuzz_receive_multipleDeposits_succeeds(uint256 _firstAmount, uint256 _secondAmount) external {
        // Handling the fork tests scenario
        address depositFeesRecipient =
            systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX) ? address(ethLockbox) : address(optimismPortal2);
        uint256 depositFeesRecipientBalanceBefore = depositFeesRecipient.balance;
        // First amount should not exceed minDepositAmount (so it doesn't trigger deposit)
        _firstAmount = bound(_firstAmount, 0, minDepositAmount - 1);

        // First deposit (should not trigger portal deposit)
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

        // Second deposit (will trigger portal deposit since total >= minDepositAmount)
        vm.deal(address(this), _secondAmount);

        vm.expectEmit(address(feesDepositor));
        emit FundsReceived(address(this), _secondAmount, totalAmount);

        vm.expectEmit(address(feesDepositor));
        emit FeesDeposited(l2Recipient, totalAmount);

        vm.expectCall(
            address(optimismPortal2),
            totalAmount,
            abi.encodeCall(IOptimismPortal.depositTransaction, (l2Recipient, totalAmount, gasLimit, false, depositData))
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

    function testFuzz_setGasLimit_asOwner_succeeds(uint64 _newGasLimit) external {
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

        uint64 newGasLimit = 200_000;

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        feesDepositor.setGasLimit(newGasLimit);

        assertEq(feesDepositor.gasLimit(), gasLimit);
    }

    function testFuzz_setDepositData_asOwner_succeeds(bytes memory _newDepositData) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(feesDepositor));
        emit DepositDataUpdated(depositData, _newDepositData);

        vm.prank(owner);
        feesDepositor.setDepositData(_newDepositData);

        assertEq(feesDepositor.depositData(), _newDepositData);
    }

    function testFuzz_setDepositData_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        bytes memory newDepositData = hex"5678";

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        feesDepositor.setDepositData(newDepositData);

        assertEq(feesDepositor.depositData(), depositData);
    }
}
