// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";

import { L1FeeVault } from "../L2/L1FeeVault.sol";
import { BaseFeeVault } from "../L2/BaseFeeVault.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

// Test the implementations of the FeeVault
contract FeeVault_Test is Bridge_Initializer {
    BaseFeeVault baseFeeVault = BaseFeeVault(payable(Predeploys.BASE_FEE_VAULT));
    L1FeeVault l1FeeVault = L1FeeVault(payable(Predeploys.L1_FEE_VAULT));

    address constant recipient = address(0x10000);

    function setUp() public override {
        super.setUp();
        vm.etch(Predeploys.BASE_FEE_VAULT, address(new BaseFeeVault(recipient)).code);
        vm.etch(Predeploys.L1_FEE_VAULT, address(new L1FeeVault(recipient)).code);

        vm.label(Predeploys.BASE_FEE_VAULT, "BaseFeeVault");
        vm.label(Predeploys.L1_FEE_VAULT, "L1FeeVault");
    }

    function test_constructor_succeeds() external {
        assertEq(baseFeeVault.RECIPIENT(), recipient);
        assertEq(l1FeeVault.RECIPIENT(), recipient);
    }

    function test_minWithdrawalAmount_succeeds() external {
        assertEq(baseFeeVault.MIN_WITHDRAWAL_AMOUNT(), 10 ether);
        assertEq(l1FeeVault.MIN_WITHDRAWAL_AMOUNT(), 10 ether);
    }
}
