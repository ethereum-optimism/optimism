// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { ResourceMetering } from "../L1/ResourceMetering.sol";
import { Proxy } from "../universal/Proxy.sol";

contract MeterUser is ResourceMetering {
    constructor() {
        initialize();
    }

    function initialize() public initializer {
        __ResourceMetering_init();
    }

    function use(uint64 _amount) public metered(_amount) {}
}

contract ResourceMetering_TestInit is CommonTest {
    MeterUser internal meter;
    uint64 initialBlockNum;

    function setUp() external {
        _setUp();
        meter = new MeterUser();
        initialBlockNum = uint64(block.number);
    }
}

contract ResourceMetering_Getters_Test is ResourceMetering_TestInit {
    function test_initialResourceParams_succeeds() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, meter.INITIAL_BASE_FEE());
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum);
    }
}

contract ResourceMetering_Use_Test is ResourceMetering_TestInit {
    function test_metered_updateParamsNoChange_succeeds() external {
        meter.use(0); // equivalent to just updating the base fee and block number
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();
        meter.use(0);
        (uint128 postBaseFee, uint64 postBoughtGas, uint64 postBlockNum) = meter.params();

        assertEq(postBaseFee, prevBaseFee);
        assertEq(postBoughtGas, prevBoughtGas);
        assertEq(postBlockNum, prevBlockNum);
    }

    function test_metered_updateOneEmptyBlock_succeeds() external {
        vm.roll(initialBlockNum + 1);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        // Base fee decreases by 12.5%
        assertEq(prevBaseFee, 875000000);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 1);
    }

    function test_metered_updateTwoEmptyBlocks_succeeds() external {
        vm.roll(initialBlockNum + 2);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, 765624999);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 2);
    }

    function test_metered_updateTenEmptyBlocks_succeeds() external {
        vm.roll(initialBlockNum + 10);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, 263075576);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 10);
    }

    function test_metered_updateNoGasDelta_succeeds() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        meter.use(target);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, 1000000000);
        assertEq(prevBoughtGas, target);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_metered_useMaxSucceeds_succeeds() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        meter.use(target * elasticity);

        (, uint64 prevBoughtGas, ) = meter.params();
        assertEq(prevBoughtGas, target * elasticity);

        vm.roll(initialBlockNum + 1);
        meter.use(0);
        (uint128 postBaseFee, , ) = meter.params();
        // Base fee increases by 1/8 the difference
        assertEq(postBaseFee, 1375000000);
    }

    function testFuzz_metered_usingLessThanMax_succeeds(uint64 amount) external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        // todo: upgrade forge-std then use vm.bound.
        // vm.bound(amount, target * elasticity);
        vm.assume(amount <= target * elasticity);
        meter.use(amount);
    }
}

contract ResourceMetering_Use_TestFail is ResourceMetering_TestInit {
    function test_metered_usingMoreThanMax_reverts() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        vm.expectRevert("ResourceMetering: cannot buy more gas than available gas limit");
        meter.use(target * elasticity + 1);
    }
}
