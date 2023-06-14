// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";

import { FeeVault } from "../universal/FeeVault.sol";
import { L1FeeVault } from "../L2/L1FeeVault.sol";
import { BaseFeeVault } from "../L2/BaseFeeVault.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

// Test the implementations of the FeeVault
contract FeeVault_Test is Bridge_Initializer {
    BaseFeeVault baseFeeVault = BaseFeeVault(payable(Predeploys.BASE_FEE_VAULT));
    L1FeeVault l1FeeVault = L1FeeVault(payable(Predeploys.L1_FEE_VAULT));

    uint256 constant otherMinimumWithdrawalAmount = 10 ether;

    function setUp() public override {
        super.setUp();
        vm.etch(
            Predeploys.BASE_FEE_VAULT,
            address(new BaseFeeVault(alice, NON_ZERO_VALUE, FeeVault.WithdrawalNetwork.L1)).code
        );
        vm.etch(
            Predeploys.L1_FEE_VAULT,
            address(
                new L1FeeVault(bob, otherMinimumWithdrawalAmount, FeeVault.WithdrawalNetwork.L2)
            ).code
        );

        vm.label(Predeploys.BASE_FEE_VAULT, "BaseFeeVault");
        vm.label(Predeploys.L1_FEE_VAULT, "L1FeeVault");
    }

    function test_constructor_succeeds() external {
        assertEq(baseFeeVault.RECIPIENT(), alice);
        assertEq(l1FeeVault.RECIPIENT(), bob);
        assertEq(baseFeeVault.MIN_WITHDRAWAL_AMOUNT(), NON_ZERO_VALUE);
        assertEq(l1FeeVault.MIN_WITHDRAWAL_AMOUNT(), otherMinimumWithdrawalAmount);
        assertEq(uint8(baseFeeVault.WITHDRAWAL_NETWORK()), uint8(FeeVault.WithdrawalNetwork.L1));
        assertEq(uint8(l1FeeVault.WITHDRAWAL_NETWORK()), uint8(FeeVault.WithdrawalNetwork.L2));
    }
}
