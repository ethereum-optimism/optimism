// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
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

    function set(
        uint128 _prevBaseFee,
        uint64 _prevBoughtGas,
        uint64 _prevBlockNum
    ) public {
        params = ResourceMetering.ResourceParams({
            prevBaseFee: _prevBaseFee,
            prevBoughtGas: _prevBoughtGas,
            prevBlockNum: _prevBlockNum
        });
    }
}

contract ResourceMetering_Test is Test {
    MeterUser internal meter;
    uint64 initialBlockNum;

    function setUp() public {
        meter = new MeterUser();
        initialBlockNum = uint64(block.number);
    }

    /**
     * @notice The INITIAL_BASE_FEE must be less than the MAXIMUM_BASE_FEE
     *         and greater than the MINIMUM_BASE_FEE.
     */
    function test_meter_initialBaseFee_succeeds() external {
        uint256 max = uint256(meter.MAXIMUM_BASE_FEE());
        uint256 min = uint256(meter.MINIMUM_BASE_FEE());
        uint256 initial = uint256(meter.INITIAL_BASE_FEE());
        assertTrue(max > initial);
        assertTrue(min < initial);
    }

    /**
     * @notice The MINIMUM_BASE_FEE must be less than the MAXIMUM_BASE_FEE.
     */
    function test_meter_minBaseFeeLessThanMaxBaseFee_succeeds() external {
        uint256 max = uint256(meter.MAXIMUM_BASE_FEE());
        uint256 min = uint256(meter.MINIMUM_BASE_FEE());
        assertTrue(max > min);
    }

    function test_meter_initialResourceParams_succeeds() external {
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, meter.INITIAL_BASE_FEE());
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_meter_updateParamsNoChange_succeeds() external {
        meter.use(0); // equivalent to just updating the base fee and block number
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();
        meter.use(0);
        (uint128 postBaseFee, uint64 postBoughtGas, uint64 postBlockNum) = meter.params();

        assertEq(postBaseFee, prevBaseFee);
        assertEq(postBoughtGas, prevBoughtGas);
        assertEq(postBlockNum, prevBlockNum);
    }

    function test_meter_updateOneEmptyBlock_succeeds() external {
        vm.roll(initialBlockNum + 1);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        // Base fee decreases by 12.5%
        assertEq(prevBaseFee, 875000000);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 1);
    }

    function test_meter_updateTwoEmptyBlocks_succeeds() external {
        vm.roll(initialBlockNum + 2);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, 765624999);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 2);
    }

    function test_meter_updateTenEmptyBlocks_succeeds() external {
        vm.roll(initialBlockNum + 10);
        meter.use(0);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, 263075576);
        assertEq(prevBoughtGas, 0);
        assertEq(prevBlockNum, initialBlockNum + 10);
    }

    function test_meter_updateNoGasDelta_succeeds() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        meter.use(target);
        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();

        assertEq(prevBaseFee, 1000000000);
        assertEq(prevBoughtGas, target);
        assertEq(prevBlockNum, initialBlockNum);
    }

    function test_meter_useMax_succeeds() external {
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

    function test_meter_useMoreThanMax_reverts() external {
        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        vm.expectRevert("ResourceMetering: cannot buy more gas than available gas limit");
        meter.use(target * elasticity + 1);
    }

    /**
     * @notice The max resource limit should be able to be used when the L1
     *         deposit base fee is at its max value. This previously would
     *         revert because prevBaseFee is a uint128 and checked math when
     *         multiplying against a uint64 _amount can result in an overflow
     *         even though its assigning to a uint256. The values MUST be casted
     *         to uint256 when doing the multiplication to prevent overflows.
     *         The function is called with the L1 block gas limit to ensure that
     *         the MAX_RESOURCE_LIMIT can be consumed at the MAXIMUM_BASE_FEE.
     */
    function test_meter_useMaxWithMaxBaseFee_succeeds() external {
        uint128 _prevBaseFee = uint128(uint256(meter.MAXIMUM_BASE_FEE()));
        uint64 _prevBoughtGas = 0;
        uint64 _prevBlockNum = uint64(block.number);

        meter.set({
            _prevBaseFee: _prevBaseFee,
            _prevBoughtGas: _prevBoughtGas,
            _prevBlockNum: _prevBlockNum
        });

        (uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum) = meter.params();
        assertEq(prevBaseFee, _prevBaseFee);
        assertEq(prevBoughtGas, _prevBoughtGas);
        assertEq(prevBlockNum, _prevBlockNum);

        uint64 gasRequested = uint64(uint256(meter.MAX_RESOURCE_LIMIT()));
        meter.use{ gas: 30_000_000 }(gasRequested);
    }

    // Demonstrates that the resource metering arithmetic can tolerate very large gaps between
    // deposits.
    function testFuzz_meter_largeBlockDiff_succeeds(uint64 _amount, uint256 _blockDiff) external {
        // This test fails if the following line is commented out.
        // At 12 seconds per block, this number is effectively unreachable.
        vm.assume(_blockDiff < 433576281058164217753225238677900874458691);

        uint64 target = uint64(uint256(meter.TARGET_RESOURCE_LIMIT()));
        uint64 elasticity = uint64(uint256(meter.ELASTICITY_MULTIPLIER()));
        vm.assume(_amount < target * elasticity);
        vm.roll(initialBlockNum + _blockDiff);
        meter.use(_amount);
    }
}
