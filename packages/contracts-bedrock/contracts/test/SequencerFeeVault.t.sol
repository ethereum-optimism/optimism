// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { FeeVault_Initializer } from "./CommonTest.t.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";

// Libraries
import { Predeploys } from "../libraries/Predeploys.sol";

// Target contract dependencies
import { FeeVault } from "../universal/FeeVault.sol";

// Target contract
import { SequencerFeeVault } from "../L2/SequencerFeeVault.sol";

contract SequencerFeeVault_Test is FeeVault_Initializer {
    /// @dev Sets up the test suite.
    function setUp() public override {
        super.setUp();
        vm.etch(
            Predeploys.SEQUENCER_FEE_WALLET,
            address(new SequencerFeeVault(recipient, NON_ZERO_VALUE, FeeVault.WithdrawalNetwork.L1))
                .code
        );
        vm.label(Predeploys.SEQUENCER_FEE_WALLET, "SequencerFeeVault");
    }

    /// @dev Tests that the minimum withdrawal amount is correct.
    function test_minWithdrawalAmount_succeeds() external {
        assertEq(vault.MIN_WITHDRAWAL_AMOUNT(), NON_ZERO_VALUE);
    }

    /// @dev Tests that the l1 fee wallet is correct.
    function test_constructor_succeeds() external {
        assertEq(vault.l1FeeWallet(), recipient);
    }

    /// @dev Tests that the fee vault is able to receive ETH.
    function test_receive_succeeds() external {
        uint256 balance = address(vault).balance;

        vm.prank(alice);
        (bool success, ) = address(vault).call{ value: 100 }(hex"");

        assertEq(success, true);
        assertEq(address(vault).balance, balance + 100);
    }

    /// @dev Tests that `withdraw` reverts if the balance is less than the minimum
    ///      withdrawal amount.
    function test_withdraw_notEnough_reverts() external {
        assert(address(vault).balance < vault.MIN_WITHDRAWAL_AMOUNT());

        vm.expectRevert(
            "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );
        vault.withdraw();
    }

    /// @dev Tests that `withdraw` successfully initiates a withdrawal to L1.
    function test_withdraw_toL1_succeeds() external {
        uint256 amount = vault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(vault), amount);

        // No ether has been withdrawn yet
        assertEq(vault.totalProcessed(), 0);

        vm.expectEmit(true, true, true, true, address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(address(vault).balance, vault.RECIPIENT(), address(this));
        vm.expectEmit(true, true, true, true, address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(
            address(vault).balance,
            vault.RECIPIENT(),
            address(this),
            FeeVault.WithdrawalNetwork.L1
        );

        // The entire vault's balance is withdrawn
        vm.expectCall(
            Predeploys.L2_STANDARD_BRIDGE,
            address(vault).balance,
            abi.encodeWithSelector(
                StandardBridge.bridgeETHTo.selector,
                vault.l1FeeWallet(),
                35_000,
                bytes("")
            )
        );

        vault.withdraw();

        // The withdrawal was successful
        assertEq(vault.totalProcessed(), amount);
        assertEq(address(vault).balance, ZERO_VALUE);
        assertEq(Predeploys.L2_TO_L1_MESSAGE_PASSER.balance, amount);
    }
}

contract SequencerFeeVault_L2Withdrawal_Test is FeeVault_Initializer {
    /// @dev Sets up the test suite.
    function setUp() public override {
        super.setUp();
        vm.etch(
            Predeploys.SEQUENCER_FEE_WALLET,
            address(new SequencerFeeVault(recipient, NON_ZERO_VALUE, FeeVault.WithdrawalNetwork.L2))
                .code
        );
        vm.label(Predeploys.SEQUENCER_FEE_WALLET, "SequencerFeeVault");
    }

    /// @dev Tests that `withdraw` successfully initiates a withdrawal to L2.
    function test_withdraw_toL2_succeeds() external {
        uint256 amount = vault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(vault), amount);

        // No ether has been withdrawn yet
        assertEq(vault.totalProcessed(), 0);

        vm.expectEmit(true, true, true, true, address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(address(vault).balance, vault.RECIPIENT(), address(this));
        vm.expectEmit(true, true, true, true, address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(
            address(vault).balance,
            vault.RECIPIENT(),
            address(this),
            FeeVault.WithdrawalNetwork.L2
        );

        // The entire vault's balance is withdrawn
        vm.expectCall(recipient, address(vault).balance, bytes(""));

        vault.withdraw();

        // The withdrawal was successful
        assertEq(vault.totalProcessed(), amount);
        assertEq(address(vault).balance, ZERO_VALUE);
        assertEq(recipient.balance, amount);
    }
}
