// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";

import { SequencerFeeVault } from "../L2/SequencerFeeVault.sol";
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

contract SequencerFeeVault_Test is Bridge_Initializer {
    SequencerFeeVault vault =
        SequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET));
    address constant recipient = address(256);

    function setUp() public override {
        super.setUp();

        vm.etch(
            Predeploys.SEQUENCER_FEE_WALLET,
            address(new SequencerFeeVault()).code
        );

        vm.store(
            Predeploys.SEQUENCER_FEE_WALLET,
            bytes32(uint256(0)),
            bytes32(uint256(uint160(recipient)))
        );
    }

    function test_minWithdrawalAmount() external {
        assertEq(
            vault.MIN_WITHDRAWAL_AMOUNT(),
            15 ether
        );
    }

    function test_constructor() external {
        assertEq(
            vault.l1FeeWallet(),
            recipient
        );
    }

    function test_receive() external {
        assertEq(
            address(vault).balance,
            0
        );

        vm.prank(alice);
        (bool success,) = address(vault).call{ value: 100 }(hex"");

        assertEq(success, true);
        assertEq(
            address(vault).balance,
            100
        );
    }

    function test_revertWithdraw() external {
        assert(address(vault).balance < vault.MIN_WITHDRAWAL_AMOUNT());

        vm.expectRevert(
            "SequencerFeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );
        vault.withdraw();
    }

    function test_withdraw() external {
        vm.deal(address(vault), vault.MIN_WITHDRAWAL_AMOUNT() + 1);

        vm.expectCall(
            Predeploys.L2_STANDARD_BRIDGE,
            abi.encodeWithSelector(
                L2StandardBridge.withdrawTo.selector,
                Predeploys.LEGACY_ERC20_ETH,
                vault.l1FeeWallet(),
                address(vault).balance,
                0,
                bytes("")
            )
        );

        vault.withdraw();
    }
}
