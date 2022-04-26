//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Testing utilities */
import { DSTest } from "../../lib/ds-test/src/test.sol";
import { Vm } from "../../lib/forge-std/src/Vm.sol";

/* Target contract dependencies */
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";

/* Target contract */
import { OptimismPortal } from "../L1/OptimismPortal.sol";

contract OptimismPortal_Test is DSTest {
    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 mint,
        uint256 value,
        uint64 gasLimit,
        bool isCreation,
        bytes data
    );

    // Utilities
    Vm vm = Vm(HEVM_ADDRESS);
    bytes32 nonZeroHash = keccak256(abi.encode("NON_ZERO"));

    // Dependencies
    L2OutputOracle oracle;

    OptimismPortal op;

    function setUp() external {
        // Oracle value is zero, but this test does not depend on it.
        op = new OptimismPortal(oracle, 7 days);
    }

    function test_receive_withEthValueFromEOA() external {
        // EOA emulation
        vm.prank(address(this), address(this));

        vm.expectEmit(true, true, false, true);
        emit TransactionDeposited(address(this), address(this), 100, 100, 30_000, false, hex"");

        (bool s, ) = address(op).call{ value: 100 }(hex"");
        s; // Silence the compiler's "Return value of low-level calls not used" warning.

        assertEq(address(op).balance, 100);
    }
}
