// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

contract GasPriceOracle_Test is CommonTest {
    event OverheadUpdated(uint256);
    event ScalarUpdated(uint256);
    event DecimalsUpdated(uint256);

    address depositor;

    // The initial L1 context values
    uint64 constant number = 10;
    uint64 constant timestamp = 11;
    uint256 constant basefee = 100;
    bytes32 constant hash = bytes32(uint256(64));
    uint64 constant sequenceNumber = 0;
    bytes32 constant batcherHash = bytes32(uint256(777));
    uint256 constant l1FeeOverhead = 310;
    uint256 constant l1FeeScalar = 10;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        depositor = l1Block.DEPOSITOR_ACCOUNT();

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

    /// @dev Tests that `l1BaseFee` is set correctly.
    function test_l1BaseFee_succeeds() external {
        assertEq(gasPriceOracle.l1BaseFee(), basefee);
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
