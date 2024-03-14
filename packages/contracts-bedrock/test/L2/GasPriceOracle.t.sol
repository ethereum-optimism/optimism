// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";

contract GasPriceOracle_Test is CommonTest {
    event OverheadUpdated(uint256);
    event ScalarUpdated(uint256);
    event DecimalsUpdated(uint256);

    address depositor;

    // The initial L1 context values
    uint64 constant number = 10;
    uint64 constant timestamp = 11;
    uint256 constant baseFee = 2 * (10 ** 6);
    uint256 constant blobBaseFee = 3 * (10 ** 6);
    bytes32 constant hash = bytes32(uint256(64));
    uint64 constant sequenceNumber = 0;
    bytes32 constant batcherHash = bytes32(uint256(777));
    uint256 constant l1FeeOverhead = 310;
    uint256 constant l1FeeScalar = 10;
    uint32 constant blobBaseFeeScalar = 15;
    uint32 constant baseFeeScalar = 20;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }
}

contract GasPriceOracleBedrock_Test is GasPriceOracle_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: number,
            _timestamp: timestamp,
            _basefee: baseFee,
            _hash: hash,
            _sequenceNumber: sequenceNumber,
            _batcherHash: batcherHash,
            _l1FeeOverhead: l1FeeOverhead,
            _l1FeeScalar: l1FeeScalar
        });
    }

    /// @dev Tests that `l1BaseFee` is set correctly.
    function test_l1BaseFee_succeeds() external {
        assertEq(gasPriceOracle.l1BaseFee(), baseFee);
    }

    /// @dev Tests that `gasPrice` is set correctly.
    function test_gasPrice_succeeds() external {
        vm.fee(100);
        uint256 gasPrice = gasPriceOracle.gasPrice();
        assertEq(gasPrice, 100);
    }

    /// @dev Tests that `baseFee` is set correctly.
    function test_baseFee_succeeds() external {
        vm.fee(64);
        uint256 gasPrice = gasPriceOracle.baseFee();
        assertEq(gasPrice, 64);
    }

    /// @dev Tests that `scalar` is set correctly.
    function test_scalar_succeeds() external {
        assertEq(gasPriceOracle.scalar(), l1FeeScalar);
    }

    /// @dev Tests that `overhead` is set correctly.
    function test_overhead_succeeds() external {
        assertEq(gasPriceOracle.overhead(), l1FeeOverhead);
    }

    /// @dev Tests that `decimals` is set correctly.
    function test_decimals_succeeds() external {
        assertEq(gasPriceOracle.decimals(), 6);
        assertEq(gasPriceOracle.DECIMALS(), 6);
    }

    /// @dev Tests that `setGasPrice` reverts since it was removed in bedrock.
    function test_setGasPrice_doesNotExist_reverts() external {
        (bool success, bytes memory returndata) =
            address(gasPriceOracle).call(abi.encodeWithSignature("setGasPrice(uint256)", 1));

        assertEq(success, false);
        assertEq(returndata, hex"");
    }

    /// @dev Tests that `setL1BaseFee` reverts since it was removed in bedrock.
    function test_setL1BaseFee_doesNotExist_reverts() external {
        (bool success, bytes memory returndata) =
            address(gasPriceOracle).call(abi.encodeWithSignature("setL1BaseFee(uint256)", 1));

        assertEq(success, false);
        assertEq(returndata, hex"");
    }
}

contract GasPriceOracleEcotone_Test is GasPriceOracle_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        bytes memory calldataPacked = Encoding.encodeSetL1BlockValuesEcotone(
            baseFeeScalar, blobBaseFeeScalar, sequenceNumber, timestamp, number, baseFee, blobBaseFee, hash, batcherHash
        );

        // Execute the function call
        vm.prank(depositor);
        (bool success,) = address(l1Block).call(calldataPacked);
        require(success, "Function call failed");

        vm.prank(depositor);
        gasPriceOracle.setEcotone();
    }

    /// @dev Tests that `setEcotone` is only callable by the depositor.
    function test_setEcotone_wrongCaller_reverts() external {
        vm.expectRevert("GasPriceOracle: only the depositor account can set isEcotone flag");
        gasPriceOracle.setEcotone();
    }

    /// @dev Tests that `gasPrice` is set correctly.
    function test_gasPrice_succeeds() external {
        vm.fee(100);
        uint256 gasPrice = gasPriceOracle.gasPrice();
        assertEq(gasPrice, 100);
    }

    /// @dev Tests that `baseFee` is set correctly.
    function test_baseFee_succeeds() external {
        vm.fee(64);
        uint256 gasPrice = gasPriceOracle.baseFee();
        assertEq(gasPrice, 64);
    }

    /// @dev Tests that `overhead` reverts since it was removed in ecotone.
    function test_overhead_legacyFunction_reverts() external {
        vm.expectRevert("GasPriceOracle: overhead() is deprecated");
        gasPriceOracle.overhead();
    }

    /// @dev Tests that `scalar` reverts since it was removed in ecotone.
    function test_scalar_legacyFunction_reverts() external {
        vm.expectRevert("GasPriceOracle: scalar() is deprecated");
        gasPriceOracle.scalar();
    }

    /// @dev Tests that `l1BaseFee` is set correctly.
    function test_l1BaseFee_succeeds() external {
        assertEq(gasPriceOracle.l1BaseFee(), baseFee);
    }

    /// @dev Tests that `blobBaseFee` is set correctly.
    function test_blobBaseFee_succeeds() external {
        assertEq(gasPriceOracle.blobBaseFee(), blobBaseFee);
    }

    /// @dev Tests that `baseFeeScalar` is set correctly.
    function test_baseFeeScalar_succeeds() external {
        assertEq(gasPriceOracle.baseFeeScalar(), baseFeeScalar);
    }

    /// @dev Tests that `blobBaseFeeScalar` is set correctly.
    function test_blobBaseFeeScalar_succeeds() external {
        assertEq(gasPriceOracle.blobBaseFeeScalar(), blobBaseFeeScalar);
    }

    /// @dev Tests that `decimals` is set correctly.
    function test_decimals_succeeds() external {
        assertEq(gasPriceOracle.decimals(), 6);
        assertEq(gasPriceOracle.DECIMALS(), 6);
    }

    /// @dev Tests that `getL1GasUsed` and `getL1Fee` return expected values
    function test_getL1Fee_succeeds() external {
        bytes memory data = hex"0000010203"; // 2 zero bytes, 3 non-zero bytes
        // (2*4) + (3*16) + (68*16) == 1144
        uint256 gas = gasPriceOracle.getL1GasUsed(data);
        assertEq(gas, 1144);
        uint256 price = gasPriceOracle.getL1Fee(data);
        // gas * (2M*16*20 + 3M*15) / 16M == 48977.5
        assertEq(price, 48977);
    }
}
