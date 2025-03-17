// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Vm } from "forge-std/Vm.sol";

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Encoding } from "src/libraries/Encoding.sol";

// Interfaces

// Free function for setting the prevBaseFee param in the OptimismPortal.
function setPrevBaseFee(Vm _vm, address _op, uint128 _prevBaseFee) {
    _vm.store(address(_op), bytes32(uint256(1)), bytes32((block.number << 192) | _prevBaseFee));
}

contract SetPrevBaseFee_Test is CommonTest {
    function test_setPrevBaseFee_succeeds() external {
        setPrevBaseFee(vm, address(optimismPortal2), 100 gwei);
        (uint128 prevBaseFee,, uint64 prevBlockNum) = optimismPortal2.params();
        assertEq(uint256(prevBaseFee), 100 gwei);
        assertEq(uint256(prevBlockNum), block.number);
    }
}

contract GasBenchMark_L1Block is CommonTest {
    address depositor;
    bytes setValuesCalldata;

    function setUp() public virtual override {
        super.setUp();

        // Get the address of the depositor.
        depositor = l1Block.DEPOSITOR_ACCOUNT();

        // Set up the calldata for setting the values.
        setValuesCalldata = Encoding.encodeSetL1BlockValuesEcotone(
            type(uint32).max,
            type(uint32).max,
            type(uint64).max,
            type(uint64).max,
            type(uint64).max,
            type(uint256).max,
            type(uint256).max,
            keccak256(abi.encode(1)),
            bytes32(type(uint256).max)
        );

        // Start pranking the depositor account.
        vm.startPrank(depositor);
    }
}

contract GasBenchMark_L1Block_SetValuesEcotone is GasBenchMark_L1Block {
    function test_setL1BlockValuesEcotone_benchmark() external {
        // Skip if the test is running in coverage.
        skipIfCoverage();

        // Test
        SafeCall.call({ _target: address(l1Block), _calldata: setValuesCalldata });

        // Assert
        assertLt(vm.lastCallGas().gasTotalUsed, 160_000);
    }
}

contract GasBenchMark_L1Block_SetValuesEcotone_Warm is GasBenchMark_L1Block {
    function test_setL1BlockValuesEcotone_benchmark() external {
        // Skip if the test is running in coverage.
        skipIfCoverage();

        // Setup
        // Trigger so storage is warm.
        SafeCall.call({ _target: address(l1Block), _calldata: setValuesCalldata });

        // Test
        SafeCall.call({ _target: address(l1Block), _calldata: setValuesCalldata });

        // Assert
        // setL1BlockValuesEcotone system tx ONLY gets 1m gas.
        // 200k is a safe boundary to prevent hitting the limit.
        assertLt(vm.lastCallGas().gasTotalUsed, 200_000);
    }
}
