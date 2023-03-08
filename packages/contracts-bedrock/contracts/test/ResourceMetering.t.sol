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

/**
 * @title MeterUserCustom
 * @notice A simple wrapper around `ResourceMetering` that allows the initial
 *         params to be set in the constructor.
 */
contract MeterUserCustom is ResourceMetering {
    uint256 public startGas;
    uint256 public endGas;

    constructor(
        uint128 _prevBaseFee,
        uint64 _prevBoughtGas,
        uint64 _prevBlockNum
    ) {
       params = ResourceMetering.ResourceParams({
           prevBaseFee: _prevBaseFee,
           prevBoughtGas: _prevBoughtGas,
           prevBlockNum: _prevBlockNum
       });
    }

    function use(uint64 _amount) public returns (uint256) {
        uint256 initialGas = gasleft();
        _metered(_amount, initialGas);
        return initialGas - gasleft();
    }
}

/**
 * @title ResourceMeteringCustom_Test
 * @notice A table test that sets the state of the ResourceParams and then requests
 *         various amounts of gas. This test ensures that a wide range of values
 *         can safely be used with the `ResourceMetering` contract.
 *         It also writes a CSV file to disk that includes useful information
 *         about how much gas is used and how expensive it is in USD terms to
 *         purchase the deposit gas.
 *         This contract is designed to have only a single test.
 */
contract ResourceMeteringCustom_Test is Test {
    MeterUser internal base;
    string internal outfile;

    // keccak256(abi.encodeWithSignature("Error(string)", "ResourceMetering: cannot buy more gas than available gas limit"))
    bytes32 internal cannotBuyMoreGas = 0x84edc668cfd5e050b8999f43ff87a1faaa93e5f935b20bc1dd4d3ff157ccf429;
    // keccak256(abi.encodeWithSignature("Panic(uint256)", 0x11))
    bytes32 internal overflowErr = 0x1ca389f2c8264faa4377de9ce8e14d6263ef29c68044a9272d405761bab2db27;

    /**
     * @notice Sets the initial block number to something sane for the
     *         deployment of MeterUser. Delete the CSV file if it exists
     *         then write the first line of the CSV.
     */
    function setUp() public {
        vm.roll(1_000_000);

        base = new MeterUser();
        outfile = string.concat(vm.projectRoot(), "/.resource-metering.csv");

        try vm.removeFile(outfile) {} catch {}
        vm.writeLine(outfile, "prevBaseFee,prevBoughtGas,prevBlockNumDiff,l1BaseFee,requestedGas,gasConsumed,ethPrice,usdCost,success");
    }

    /**
     * @notice Generate a CSV file. The call to `meter` should be called with at
     *         most the L1 block gas limit. Without specifying the amount of
     *         gas, it can take very long to execute.
     */
    function test_meter_generateArtifact_succeeds() external {
        // prevBaseFee value in ResourceParams
        uint128[] memory prevBaseFees = new uint128[](5);
        prevBaseFees[0] = uint128(uint256(base.MAXIMUM_BASE_FEE()));
        prevBaseFees[1] = uint128(uint256(base.MINIMUM_BASE_FEE()));
        prevBaseFees[2] = uint128(uint256(base.INITIAL_BASE_FEE()));
        prevBaseFees[3] = uint128(100_000);
        prevBaseFees[4] = uint128(500_000);

        // prevBoughtGas value in ResourceParams
        uint64[] memory prevBoughtGases = new uint64[](3);
        prevBoughtGases[0] = uint64(uint256(base.MAX_RESOURCE_LIMIT()));
        prevBoughtGases[1] = uint64(uint256(base.TARGET_RESOURCE_LIMIT()));
        prevBoughtGases[2] = uint64(0);

        // prevBlockNum diff, simulates blocks with no deposits when non zero
        uint64[] memory prevBlockNumDiffs = new uint64[](2);
        prevBlockNumDiffs[0] = 0;
        prevBlockNumDiffs[1] = 1;

        // The amount of L2 gas that a user requests
        uint64[] memory requestedGases = new uint64[](3);
        requestedGases[0] = uint64(uint256(base.MAX_RESOURCE_LIMIT()));
        requestedGases[1] = uint64(uint256(base.TARGET_RESOURCE_LIMIT()));
        requestedGases[2] = uint64(100_000);

        // The L1 base fee
        uint256[] memory l1BaseFees = new uint256[](4);
        l1BaseFees[0] = 1 gwei;
        l1BaseFees[1] = 50 gwei;
        l1BaseFees[2] = 75 gwei;
        l1BaseFees[3] = 100 gwei;

        // USD price of 1 ether
        uint256[] memory ethPrices = new uint256[](2);
        ethPrices[0] = 1600;
        ethPrices[1] = 3200;

        // Iterate over all of the test values and run a test
        for (uint256 i; i < prevBaseFees.length; i++) {
            for (uint256 j; j < prevBoughtGases.length; j++) {
                for (uint256 k; k < prevBlockNumDiffs.length; k++) {
                    for (uint256 l; l < requestedGases.length; l++) {
                        for (uint256 m; m < l1BaseFees.length; m++) {
                            for (uint256 n; n < ethPrices.length; n++) {
                                uint256 snapshotId = vm.snapshot();

                                uint128 prevBaseFee = prevBaseFees[i];
                                uint64 prevBoughtGas = prevBoughtGases[j];
                                uint64 prevBlockNumDiff = prevBlockNumDiffs[k];
                                uint64 requestedGas = requestedGases[l];
                                uint256 l1BaseFee = l1BaseFees[m];
                                uint256 ethPrice = ethPrices[n];
                                string memory result = "success";

                                vm.fee(l1BaseFee);

                                MeterUserCustom meter = new MeterUserCustom({
                                    _prevBaseFee: prevBaseFee,
                                    _prevBoughtGas: prevBoughtGas,
                                    _prevBlockNum: uint64(block.number)
                                });

                                vm.roll(block.number + prevBlockNumDiff);

                                uint256 gasConsumed = 0;
                                try meter.use{ gas: 30_000_000 }(requestedGas) returns (uint256 _gasConsumed) {
                                    gasConsumed = _gasConsumed;
                                } catch (bytes memory err) {
                                    bytes32 hash = keccak256(err);
                                    if (hash == cannotBuyMoreGas) {
                                        result = "ResourceMetering: cannot buy more gas than available gas limit";
                                    } else if (hash == overflowErr) {
                                        result = "arithmetic overflow/underflow";
                                    } else {
                                        result = "UNKNOWN ERROR";
                                    }
                                }

                                // Compute the USD cost of the gas used, don't
                                // worry too much about loss of precison under $1
                                uint256 usdCost = gasConsumed * l1BaseFee * ethPrice / 1 ether;

                                vm.writeLine(
                                    outfile,
                                    string.concat(
                                        vm.toString(prevBaseFee), ",",
                                        vm.toString(prevBoughtGas), ",",
                                        vm.toString(prevBlockNumDiff), ",",
                                        vm.toString(l1BaseFee), ",",
                                        vm.toString(requestedGas), ",",
                                        vm.toString(gasConsumed), ",",
                                        "$", vm.toString(ethPrice), ",",
                                        "$", vm.toString(usdCost), ",",
                                        result
                                    )
                                );

                                assertTrue(vm.revertTo(snapshotId));
                            }
                        }
                    }
                }
            }
        }
    }
}
