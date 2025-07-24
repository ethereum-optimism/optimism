// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IL1BlockNumber } from "interfaces/legacy/IL1BlockNumber.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";

/// @title L1BlockNumber_TestInit
/// @notice Reusable test initialization for `L1BlockNumber` tests.
contract L1BlockNumber_TestInit is Test {
    IL1Block lb;
    IL1BlockNumber bn;

    uint64 constant number = 99;

    /// @notice Sets up the test suite.
    function setUp() external {
        vm.etch(Predeploys.L1_BLOCK_ATTRIBUTES, vm.getDeployedCode("L1Block.sol:L1Block"));
        lb = IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
        bn = IL1BlockNumber(
            DeployUtils.create1({
                _name: "L1BlockNumber",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IL1BlockNumber.__constructor__, ()))
            })
        );
        vm.prank(lb.DEPOSITOR_ACCOUNT());

        lb.setL1BlockValues({
            _number: number,
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: bytes32(uint256(10)),
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(uint256(0)),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });
    }
}

/// @title L1BlockNumber_Receive_Test
/// @notice Tests the `receive` function of the `L1BlockNumber` contract.
contract L1BlockNumber_Receive_Test is L1BlockNumber_TestInit {
    /// @notice Tests that `receive` is correctly dispatched.
    function test_receive_succeeds() external {
        (bool success, bytes memory ret) = address(bn).call{ value: 1 }(hex"");
        assertEq(success, true);
        assertEq(ret, abi.encode(number));
    }
}

/// @title L1BlockNumber_Fallback_Test
/// @notice Tests the `fallback` function of the `L1BlockNumber` contract.
contract L1BlockNumber_Fallback_Test is L1BlockNumber_TestInit {
    /// @notice Tests that `fallback` is correctly dispatched.
    function test_fallback_succeeds() external {
        (bool success, bytes memory ret) = address(bn).call(hex"11");
        assertEq(success, true);
        assertEq(ret, abi.encode(number));
    }
}

/// @title L1BlockNumber_GetL1BlockNumber_Test
/// @notice Tests the `getL1BlockNumber` function of the `L1BlockNumber` contract.
contract L1BlockNumber_GetL1BlockNumber_Test is L1BlockNumber_TestInit {
    /// @notice Tests that `getL1BlockNumber` returns the set block number.
    function test_getL1BlockNumber_succeeds() external view {
        assertEq(bn.getL1BlockNumber(), number);
    }
}
