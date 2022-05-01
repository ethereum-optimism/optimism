//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { DSTest } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { Withdrawer } from "../L2/Withdrawer.sol";

import {
    AddressAliasHelper
} from "@eth-optimism/contracts/standards/AddressAliasHelper.sol";
import {
    Lib_RLPWriter
} from "@eth-optimism/contracts/libraries/rlp/Lib_RLPWriter.sol";
import {
    Lib_Bytes32Utils
} from "@eth-optimism/contracts/libraries/utils/Lib_Bytes32Utils.sol";


contract WithdrawerTestCommon is DSTest {
    Vm vm = Vm(HEVM_ADDRESS);
    address immutable ZERO_ADDRESS = address(0);
    address immutable NON_ZERO_ADDRESS = address(1);
    uint256 immutable NON_ZERO_VALUE = 100;
    uint256 immutable ZERO_VALUE = 0;
    uint256 immutable NON_ZERO_GASLIMIT = 50000;
    bytes NON_ZERO_DATA = hex"1111";

    event WithdrawalInitiated(
        uint256 indexed nonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );

    Withdrawer wd;

    function setUp() public virtual {
        wd = new Withdrawer();
    }
}

contract WithdrawerTestInitiateWithdrawal is WithdrawerTestCommon {

    // Test: initiateWithdrawal should emit the correct log when called by a contract
    function test_initiateWithdrawal_fromContract() external {
        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(
            0,
            AddressAliasHelper.undoL1ToL2Alias(address(this)),
            NON_ZERO_ADDRESS,
            NON_ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            NON_ZERO_DATA
        );

        wd.initiateWithdrawal{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            NON_ZERO_GASLIMIT,
            NON_ZERO_DATA
        );
    }

    // Test: initiateWithdrawal should emit the correct log when called by an EOA
    function test_initiateWithdrawal_fromEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));
        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(
            0,
            address(this),
            NON_ZERO_ADDRESS,
            NON_ZERO_VALUE,
            NON_ZERO_GASLIMIT,
            NON_ZERO_DATA
        );

        wd.initiateWithdrawal{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            NON_ZERO_GASLIMIT,
            NON_ZERO_DATA
        );
    }
}

contract WithdawerBurnTest is WithdrawerTestCommon {

    event WithdrawerBalanceBurnt(uint256 indexed amount);

    function setUp() public override {
        // fund a new withdrawer
        super.setUp();
        wd.initiateWithdrawal{ value: NON_ZERO_VALUE }(
            NON_ZERO_ADDRESS,
            NON_ZERO_GASLIMIT,
            NON_ZERO_DATA
        );
    }

    // Test: burn should destroy the ETH held in the contract
    function test_burn() external {
        assertEq(address(wd).balance, NON_ZERO_VALUE);
        vm.expectEmit(true, false, false, false);
        emit WithdrawerBalanceBurnt(NON_ZERO_VALUE);
        wd.burn();

        // The Withdrawer should have no balance
        assertEq(address(wd).balance, 0);
    }
}
