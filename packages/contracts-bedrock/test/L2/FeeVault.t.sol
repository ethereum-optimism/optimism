// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { Reverter } from "test/mocks/Callers.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";
import { IL2ToL1MessagePasserCGT } from "interfaces/L2/IL2ToL1MessagePasserCGT.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Features } from "src/libraries/Features.sol";

/// @title FeeVault_Uncategorized_Test
/// @notice Abstract test contract for fee feeVault testing.
///         Subclasses can override the feeVault-specific variables.
abstract contract FeeVault_Uncategorized_Test is CommonTest {
    // Variables that can be overridden by concrete test contracts
    address recipient;
    IFeeVault feeVault;
    string feeVaultName;
    uint256 minWithdrawalAmount;
    Types.WithdrawalNetwork withdrawalNetwork;

    /// @notice Helper function to set up L2 withdrawal configuration.
    function _setupL2Withdrawal() internal {
        // Set the withdrawal network to L2
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setWithdrawalNetwork(Types.WithdrawalNetwork.L2);
    }

    /// @notice Tests that the initialize function succeeds.
    function test_initialize_succeeds() external view {
        assertEq(feeVault.recipient(), recipient);
        assertEq(feeVault.minWithdrawalAmount(), minWithdrawalAmount);
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(withdrawalNetwork));
    }

    /// @notice Tests that the initialize function reverts if the contract is already initialized.
    function test_initialize_reinitialization_reverts() external {
        _setupL2Withdrawal();

        vm.expectRevert(IFeeVault.InvalidInitialization.selector);
        feeVault.initialize(recipient, minWithdrawalAmount, Types.WithdrawalNetwork.L1);
    }

    /// @notice Tests that the immutable values match the storage getters.
    function test_immutableMatchesStorageVariables_succeeds() external view {
        assertEq(feeVault.RECIPIENT(), feeVault.recipient());
        assertEq(feeVault.MIN_WITHDRAWAL_AMOUNT(), feeVault.minWithdrawalAmount());
        assertEq(uint8(feeVault.WITHDRAWAL_NETWORK()), uint8(feeVault.withdrawalNetwork()));
    }

    /// @notice Tests that the fee feeVault is able to receive ETH.
    function test_receive_succeeds() external {
        uint256 balance = address(feeVault).balance;

        vm.prank(alice);
        (bool success,) = address(feeVault).call{ value: 100 }(hex"");

        assertEq(success, true);
        assertEq(address(feeVault).balance, balance + 100);
    }

    /// @notice Tests that `withdraw` reverts if the balance is less than the minimum withdrawal
    ///         amount.
    function testFuzz_withdraw_notEnough_reverts(uint256 _minWithdrawalAmount) external {
        // Set the minimum withdrawal amount
        _minWithdrawalAmount = bound(_minWithdrawalAmount, 1, type(uint256).max);
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setMinWithdrawalAmount(_minWithdrawalAmount);

        // Set the balance to be less than the minimum withdrawal amount
        vm.deal(address(feeVault), _minWithdrawalAmount - 1);

        vm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        feeVault.withdraw();
    }

    /// @notice Tests that `withdraw` successfully initiates a withdrawal to L1.
    function test_withdraw_toL1_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        // Setup L1 withdrawal
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setWithdrawalNetwork(Types.WithdrawalNetwork.L1);

        // Set recipient
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setRecipient(recipient);

        // Set minimum withdrawal amount
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setMinWithdrawalAmount(minWithdrawalAmount);

        // Set the balance to be greater than the minimum withdrawal amount
        uint256 amount = feeVault.minWithdrawalAmount() + 1;
        vm.deal(address(feeVault), amount);

        // No ether has been withdrawn yet
        assertEq(feeVault.totalProcessed(), 0);

        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, recipient, address(this));
        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, recipient, address(this), Types.WithdrawalNetwork.L1);

        // The entire feeVault's balance is withdrawn
        vm.expectCall(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            address(feeVault).balance,
            abi.encodeCall(IL2ToL1MessagePasser.initiateWithdrawal, (recipient, 400_000, hex""))
        );

        // The message is passed to the correct recipient
        vm.expectEmit(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        emit MessagePassed(
            l2ToL1MessagePasser.messageNonce(),
            address(feeVault),
            recipient,
            amount,
            400_000,
            hex"",
            Hashing.hashWithdrawal(
                Types.WithdrawalTransaction({
                    nonce: l2ToL1MessagePasser.messageNonce(),
                    sender: address(feeVault),
                    target: recipient,
                    value: amount,
                    gasLimit: 400_000,
                    data: hex""
                })
            )
        );

        feeVault.withdraw();

        // The withdrawal was successful
        assertEq(feeVault.totalProcessed(), amount);
        assertEq(address(feeVault).balance, 0);
        assertEq(Predeploys.L2_TO_L1_MESSAGE_PASSER.balance, amount);
    }

    /// @notice Tests that withdraw to L1 reverts when custom gas token is enabled and value is sent.
    function testFuzz_withdraw_toL1WithCustomGasToken_reverts(uint256 _amount) external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        // Setup L1 withdrawal
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setWithdrawalNetwork(Types.WithdrawalNetwork.L1);

        // Set recipient
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setRecipient(recipient);

        // Set minimum withdrawal amount
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        feeVault.setMinWithdrawalAmount(minWithdrawalAmount);

        // Set the balance to be greater than the minimum withdrawal amount
        _amount = bound(_amount, feeVault.minWithdrawalAmount() + 1, type(uint128).max);
        vm.deal(address(feeVault), _amount);

        // Withdrawal should revert due to CGT mode
        vm.expectRevert(IL2ToL1MessagePasserCGT.L2ToL1MessagePasserCGT_NotAllowedOnCGTMode.selector);
        feeVault.withdraw();
    }

    /// @notice Tests that `withdraw` successfully initiates a withdrawal to L2.
    function test_withdraw_toL2_succeeds() public {
        _setupL2Withdrawal();

        uint256 amount = feeVault.minWithdrawalAmount() + 1;
        vm.deal(address(feeVault), amount);

        // No ether has been withdrawn yet
        assertEq(feeVault.totalProcessed(), 0);

        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, feeVault.RECIPIENT(), address(this));
        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, feeVault.RECIPIENT(), address(this), Types.WithdrawalNetwork.L2);

        // The entire feeVault's balance is withdrawn
        vm.expectCall(recipient, address(feeVault).balance, bytes(""));

        uint256 withdrawnAmount = feeVault.withdraw();

        // The withdrawal was successful
        assertEq(withdrawnAmount, amount);
        assertEq(feeVault.totalProcessed(), amount);
        assertEq(address(feeVault).balance, 0);
        assertEq(recipient.balance, amount);
    }

    /// @notice Tests that `withdraw` fails if the Recipient reverts. This also serves to simulate
    ///         a situation where insufficient gas is provided to the RECIPIENT.
    function test_withdraw_toL2recipientReverts_fails() external {
        _setupL2Withdrawal();

        uint256 amount = feeVault.minWithdrawalAmount();

        vm.deal(address(feeVault), amount);
        // No ether has been withdrawn yet
        assertEq(feeVault.totalProcessed(), 0);

        // Ensure the RECIPIENT reverts
        vm.etch(feeVault.RECIPIENT(), type(Reverter).runtimeCode);

        // The entire feeVault's balance is withdrawn
        vm.expectCall(recipient, address(feeVault).balance, bytes(""));
        vm.expectRevert("FeeVault: failed to send ETH to L2 fee recipient");
        feeVault.withdraw();
        assertEq(feeVault.totalProcessed(), 0);
    }

    /// @notice Tests that the owner can successfully set minimum withdrawal amount with fuzz testing.
    function testFuzz_setMinWithdrawalAmount_succeeds(uint256 _newMinWithdrawalAmount) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setMinWithdrawalAmount(_newMinWithdrawalAmount);

        // Verify the value was updated
        assertEq(feeVault.minWithdrawalAmount(), _newMinWithdrawalAmount);
    }

    /// @notice Tests that non-owner cannot set minimum withdrawal amount with fuzz testing.
    function testFuzz_setMinWithdrawalAmount_onlyOwner_reverts(address _caller, uint256 _newAmount) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
        vm.assume(_caller != owner);

        uint256 initialAmount = feeVault.minWithdrawalAmount();

        vm.prank(_caller);
        vm.expectRevert(IFeeVault.FeeVault_OnlyProxyAdminOwner.selector);
        IFeeVault(payable(address(feeVault))).setMinWithdrawalAmount(_newAmount);

        // Verify the value and boolean flag were NOT changed
        assertEq(feeVault.minWithdrawalAmount(), initialAmount);
    }

    /// @notice Tests that the owner can successfully set recipient with fuzz testing.
    function testFuzz_setRecipient_succeeds(address _newRecipient) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setRecipient(_newRecipient);

        // Verify the value was updated
        assertEq(feeVault.recipient(), _newRecipient);
    }

    /// @notice Tests that non-owner cannot set recipient with fuzz testing.
    function testFuzz_setRecipient_onlyOwner_reverts(address _caller, address _newRecipient) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
        vm.assume(_caller != owner);

        address initialRecipient = feeVault.recipient();

        vm.prank(_caller);
        vm.expectRevert(IFeeVault.FeeVault_OnlyProxyAdminOwner.selector);
        IFeeVault(payable(address(feeVault))).setRecipient(_newRecipient);

        // Verify the value and boolean flag were NOT changed
        assertEq(feeVault.recipient(), initialRecipient);
    }

    /// @notice Tests that the owner can successfully set withdrawal network with fuzz testing.
    function testFuzz_setWithdrawalNetwork_succeeds(uint8 _networkValue) external {
        // Bound to valid enum values (0 = L1, 1 = L2)
        _networkValue = uint8(bound(_networkValue, 0, 1));
        Types.WithdrawalNetwork newNetwork = Types.WithdrawalNetwork(_networkValue);

        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setWithdrawalNetwork(newNetwork);

        // Verify the value was updated
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(newNetwork));
    }

    /// @notice Tests that non-owner cannot set withdrawal network with fuzz testing.
    function testFuzz_setWithdrawalNetwork_onlyOwner_reverts(address _caller, uint8 _networkValue) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
        vm.assume(_caller != owner);

        // Bound to valid enum values
        _networkValue = uint8(bound(_networkValue, 0, 1));
        Types.WithdrawalNetwork newNetwork = Types.WithdrawalNetwork(_networkValue);

        Types.WithdrawalNetwork initialNetwork = feeVault.withdrawalNetwork();

        vm.prank(_caller);
        vm.expectRevert(IFeeVault.FeeVault_OnlyProxyAdminOwner.selector);
        IFeeVault(payable(address(feeVault))).setWithdrawalNetwork(newNetwork);

        // Verify the value and boolean flag were NOT changed
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(initialNetwork));
    }
}
