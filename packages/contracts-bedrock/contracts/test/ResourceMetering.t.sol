//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";

import { ResourceMetering } from "../L1/ResourceMetering.sol";

contract MeterUser is ResourceMetering {
    function use(uint64 _amount) public metered(_amount) {}
}

contract ResourceMetering_Test is CommonTest {
    MeterUser internal meter;
    uint64 initialBlockNum;

    function setUp() external {
        meter = new MeterUser();
        initialBlockNum = uint64(block.number);
        vm.fee(meter.INITIAL_BASE_FEE());
    }

    function test_initialResourceParams() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();

        assertEq(prevBaseFee, meter.INITIAL_BASE_FEE());
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_updateParamsNoChange() external {
        meter.use(0); // equivalent to just updating the base fee and block number
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();
        meter.use(0);
        (uint128 postBaseFee, uint64 postBoughtGas, uint64 postBlockNum) = meter.getResourceParams();

        assertEq(postBaseFee, prevBaseFee);
        assertEq(postBoughtGas, prevBoughtGas);
        assertEq(postBlockNum, prevBlockNum);
    }

    function test_updateOneEmptyBlock() external {
        vm.roll(initialBlockNum + 1);
        vm.fee(meter.INITIAL_BASE_FEE());
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();

        // Base fee decreases by 12.5%
        assertEq(prevBaseFee, 875000000);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 1);
    }

    function test_updateTwoEmptyBlocks() external {
        vm.roll(initialBlockNum + 2);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();

        assertEq(prevBaseFee, 765625000);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 2);
    }

    function test_updateTenEmptyBlocks() external {
        vm.roll(initialBlockNum + 10);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();

        assertEq(prevBaseFee, 263075576);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 10);
    }

    function test_updateNoGasDelta() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        meter.use(target);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();

        assertEq(prevBaseFee, 1000000000);
        assertEq(prevBoughtGas, target);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_useMaxSucceeds() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        meter.use(target * elasticity);

        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.getResourceParams();
        assertEq(prevBoughtGas, target * elasticity);

        // Base fee increases by 12.5%
        vm.roll(initialBlockNum + 1);
        meter.use(0);
        (uint128 postBaseFee, uint64 postBoughtGas, uint64 postBlockNum) = meter.getResourceParams();
        assertEq(postBaseFee, 1125000000);
    }

    function test_useMoreThanMaxReverts() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        vm.expectRevert("OptimismPortal: cannot buy more gas than available gas limit");
        meter.use(target * elasticity + 1);
    }
}
