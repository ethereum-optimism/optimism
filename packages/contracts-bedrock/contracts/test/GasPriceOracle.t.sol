// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { GasPriceOracle } from "../L2/GasPriceOracle.sol";
import { L1Block } from "../L2/L1Block.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";

contract GasPriceOracle_Test is CommonTest {

    event OverheadUpdated(uint256);
    event ScalarUpdated(uint256);
    event DecimalsUpdated(uint256);

    GasPriceOracle internal gasOracle;
    L1Block internal l1Block;
    address internal depositor;

    function setUp() external {
        // place the L1Block contract at the predeploy address
        vm.etch(
            Predeploys.L1_BLOCK_ATTRIBUTES,
            address(new L1Block()).code
        );

        l1Block = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
        depositor = l1Block.DEPOSITOR_ACCOUNT();

        // We are not setting the gas oracle at its predeploy
        // address for simplicity purposes. Nothing in this test
        // requires it to be at a particular address
        gasOracle = new GasPriceOracle(alice);

        // set the initial L1 context values
        uint64 number = 10;
        uint64 timestamp = 11;
        uint256 basefee = 100;
        bytes32 hash = bytes32(uint256(64));
        uint64 sequenceNumber = 0;

        vm.prank(depositor);
        l1Block.setL1BlockValues(
            number,
            timestamp,
            basefee,
            hash,
            sequenceNumber
        );
    }

    function test_owner() external {
        address owner = gasOracle.owner();
        // alice is passed into the constructor of the gasOracle
        assertEq(owner, alice);
    }

    function test_storageLayout() external {
        address owner = gasOracle.owner();

        // the overhead is at slot 3
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        gasOracle.setOverhead(456);
        assertEq(
            456,
            uint256(vm.load(address(gasOracle), bytes32(uint256(3))))
        );

        // scalar is at slot 4
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        gasOracle.setScalar(333);
        assertEq(
            333,
            uint256(vm.load(address(gasOracle), bytes32(uint256(4))))
        );

        // decimals is at slot 5
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        assertEq(
            6,
            uint256(vm.load(address(gasOracle), bytes32(uint256(5))))
        );
    }

    function test_l1BaseFee() external {
        uint256 l1BaseFee = gasOracle.l1BaseFee();
        assertEq(l1BaseFee, 100);
    }

    function test_gasPrice() external {
        vm.fee(100);
        uint256 gasPrice = gasOracle.gasPrice();
        assertEq(gasPrice, 100);
    }

    function test_baseFee() external {
        vm.fee(64);
        uint256 gasPrice = gasOracle.baseFee();
        assertEq(gasPrice, 64);
    }

    function test_setGasPriceReverts() external {
        address owner = gasOracle.owner();
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        (bool success, bytes memory returndata) = address(gasOracle).call(
            abi.encodeWithSignature(
                "setGasPrice(uint256)",
                1
            )
        );

        assertEq(success, false);
        assertEq(returndata, hex"");
    }

    function test_setL1BaseFeeReverts() external {
        address owner = gasOracle.owner();
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        (bool success, bytes memory returndata) = address(gasOracle).call(
            abi.encodeWithSignature(
                "setL1BaseFee(uint256)",
                1
            )
        );

        assertEq(success, false);
        assertEq(returndata, hex"");
    }

    function test_setOverhead() external {
        vm.expectEmit(true, true, true, true);
        emit OverheadUpdated(1234);

        address owner = gasOracle.owner();
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        gasOracle.setOverhead(1234);
        assertEq(gasOracle.overhead(), 1234);
    }

    function test_onlyOwnerSetOverhead() external {
        vm.expectRevert("CrossDomainOwnable: caller is not the owner");
        gasOracle.setOverhead(0);
    }

    function test_setScalar() external {
        vm.expectEmit(true, true, true, true);
        emit ScalarUpdated(666);

        address owner = gasOracle.owner();
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        gasOracle.setScalar(666);
        assertEq(gasOracle.scalar(), 666);
    }

    function test_onlyOwnerSetScalar() external {
        vm.expectRevert("CrossDomainOwnable: caller is not the owner");
        gasOracle.setScalar(0);
    }

    function test_setDecimals() external {
        vm.expectEmit(true, true, true, true);
        emit DecimalsUpdated(18);

        address owner = gasOracle.owner();
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(owner));
        gasOracle.setDecimals(18);
        assertEq(gasOracle.decimals(), 18);
    }

     function test_onlyOwnerSetDecimals() external {
         vm.expectRevert("CrossDomainOwnable: caller is not the owner");
         gasOracle.setDecimals(0);
     }
}
