// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { Reverter } from "test/mocks/Callers.sol";
import { StandardBridge } from "src/universal/StandardBridge.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { FeeVault } from "src/universal/FeeVault.sol";

// Target contract
import { SequencerFeeVault } from "src/L2/SequencerFeeVault.sol";

contract SequencerFeeVault_Test is CommonTest {
    address recipient;

    /// @dev Sets up the test suite.
    function setUp() public override {
        super.setUp();
        recipient = deploy.cfg().sequencerFeeVaultRecipient();
    }

    /// @dev Tests that the minimum withdrawal amount is correct.
    function test_minWithdrawalAmount_succeeds() external {
        assertEq(sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT(), deploy.cfg().sequencerFeeVaultMinimumWithdrawalAmount());
    }

    /// @dev Tests that the l1 fee wallet is correct.
    function test_constructor_succeeds() external {
        assertEq(sequencerFeeVault.l1FeeWallet(), recipient);
    }

    /// @dev Tests that the fee vault is able to receive ETH.
    function test_receive_succeeds() external {
        uint256 balance = address(sequencerFeeVault).balance;

        vm.prank(alice);
        (bool success,) = address(sequencerFeeVault).call{ value: 100 }(hex"");

        assertEq(success, true);
        assertEq(address(sequencerFeeVault).balance, balance + 100);
    }

    /// @dev Tests that `withdraw` reverts if the balance is less than the minimum
    ///      withdrawal amount.
    function test_withdraw_notEnough_reverts() external {
        assert(address(sequencerFeeVault).balance < sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT());

        vm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        sequencerFeeVault.withdraw();
    }

    /// @dev Tests that `withdraw` successfully initiates a withdrawal to L1.
    function test_withdraw_toL1_succeeds() external {
        uint256 amount = sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(sequencerFeeVault), amount);

        // No ether has been withdrawn yet
        assertEq(sequencerFeeVault.totalProcessed(), 0);

        vm.expectEmit(address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(address(sequencerFeeVault).balance, sequencerFeeVault.RECIPIENT(), address(this));
        vm.expectEmit(address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(
            address(sequencerFeeVault).balance,
            sequencerFeeVault.RECIPIENT(),
            address(this),
            FeeVault.WithdrawalNetwork.L1
        );

        // The entire vault's balance is withdrawn
        vm.expectCall(
            Predeploys.L2_STANDARD_BRIDGE,
            address(sequencerFeeVault).balance,
            abi.encodeWithSelector(
                StandardBridge.bridgeETHTo.selector, sequencerFeeVault.l1FeeWallet(), 35_000, bytes("")
            )
        );

        sequencerFeeVault.withdraw();

        // The withdrawal was successful
        assertEq(sequencerFeeVault.totalProcessed(), amount);
        assertEq(address(sequencerFeeVault).balance, 0);
        assertEq(Predeploys.L2_TO_L1_MESSAGE_PASSER.balance, amount);
    }
}

contract SequencerFeeVault_L2Withdrawal_Test is CommonTest {
    /// @dev a cache for the config fee recipient
    address recipient;

    /// @dev Sets up the test suite.
    function setUp() public override {
        super.setUp();

        // Alter the deployment to use WithdrawalNetwork.L2
        vm.etch(
            EIP1967Helper.getImplementation(Predeploys.SEQUENCER_FEE_WALLET),
            address(
                new SequencerFeeVault(
                    deploy.cfg().sequencerFeeVaultRecipient(),
                    deploy.cfg().sequencerFeeVaultMinimumWithdrawalAmount(),
                    FeeVault.WithdrawalNetwork.L2
                )
            ).code
        );

        recipient = deploy.cfg().sequencerFeeVaultRecipient();
    }

    /// @dev Tests that `withdraw` successfully initiates a withdrawal to L2.
    function test_withdraw_toL2_succeeds() external {
        uint256 amount = sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(sequencerFeeVault), amount);

        // No ether has been withdrawn yet
        assertEq(sequencerFeeVault.totalProcessed(), 0);

        vm.expectEmit(address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(address(sequencerFeeVault).balance, sequencerFeeVault.RECIPIENT(), address(this));
        vm.expectEmit(address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(
            address(sequencerFeeVault).balance,
            sequencerFeeVault.RECIPIENT(),
            address(this),
            FeeVault.WithdrawalNetwork.L2
        );

        // The entire vault's balance is withdrawn
        vm.expectCall(recipient, address(sequencerFeeVault).balance, bytes(""));

        sequencerFeeVault.withdraw();

        // The withdrawal was successful
        assertEq(sequencerFeeVault.totalProcessed(), amount);
        assertEq(address(sequencerFeeVault).balance, 0);
        assertEq(recipient.balance, amount);
    }

    /// @dev Tests that `withdraw` fails if the Recipient reverts. This also serves to simulate
    ///     a situation where insufficient gas is provided to the RECIPIENT.
    function test_withdraw_toL2recipientReverts_fails() external {
        uint256 amount = sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT();

        vm.deal(address(sequencerFeeVault), amount);
        // No ether has been withdrawn yet
        assertEq(sequencerFeeVault.totalProcessed(), 0);

        // Ensure the RECIPIENT reverts
        vm.etch(sequencerFeeVault.RECIPIENT(), type(Reverter).runtimeCode);

        // The entire vault's balance is withdrawn
        vm.expectCall(recipient, address(sequencerFeeVault).balance, bytes(""));
        vm.expectRevert("FeeVault: failed to send ETH to L2 fee recipient");
        sequencerFeeVault.withdraw();
        assertEq(sequencerFeeVault.totalProcessed(), 0);
    }
}
