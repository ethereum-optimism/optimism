// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { GasPriceOracle } from "../L2/GasPriceOracle.sol";
import { L1Block } from "../L2/L1Block.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

contract GasPriceOracle_Test is CommonTest {
    event OverheadUpdated(uint256);
    event ScalarUpdated(uint256);
    event DecimalsUpdated(uint256);

    GasPriceOracle gasOracle;
    L1Block l1Block;
    address depositor;

    // set the initial L1 context values
    uint64 constant number = 10;
    uint64 constant timestamp = 11;
    uint256 constant basefee = 100;
    bytes32 constant hash = bytes32(uint256(64));
    uint64 constant sequenceNumber = 0;
    bytes32 constant batcherHash = bytes32(uint256(777));
    uint256 constant l1FeeOverhead = 310;
    uint256 constant l1FeeScalar = 10;

    function setUp() external {
        // place the L1Block contract at the predeploy address
        vm.etch(Predeploys.L1_BLOCK_ATTRIBUTES, address(new L1Block()).code);

        l1Block = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
        depositor = l1Block.DEPOSITOR_ACCOUNT();

        // We are not setting the gas oracle at its predeploy
        // address for simplicity purposes. Nothing in this test
        // requires it to be at a particular address
        gasOracle = new GasPriceOracle();

        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: number,
            _timestamp: timestamp,
            _basefee: basefee,
            _hash: hash,
            _sequenceNumber: sequenceNumber,
            _batcherHash: batcherHash,
            _l1FeeOverhead: l1FeeOverhead,
            _l1FeeScalar: l1FeeScalar
        });
    }

    function test_l1BaseFee() external {
        assertEq(gasOracle.l1BaseFee(), basefee);
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

    function test_scalar() external {
        assertEq(gasOracle.scalar(), l1FeeScalar);
    }

    function test_overhead() external {
        assertEq(gasOracle.overhead(), l1FeeOverhead);
    }

    function test_setGasPriceReverts() external {
        (bool success, bytes memory returndata) = address(gasOracle).call(
            abi.encodeWithSignature("setGasPrice(uint256)", 1)
        );

        assertEq(success, false);
        assertEq(returndata, hex"");
    }

    function test_setL1BaseFeeReverts() external {
        (bool success, bytes memory returndata) = address(gasOracle).call(
            abi.encodeWithSignature("setL1BaseFee(uint256)", 1)
        );

        assertEq(success, false);
        assertEq(returndata, hex"");
    }
}
