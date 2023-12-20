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
    uint256 constant blobBasefee = 101;
    bytes32 constant hash = bytes32(uint256(64));
    uint64 constant sequenceNumber = 0;
    bytes32 constant batcherHash = bytes32(uint256(777));
    uint256 constant l1FeeOverhead = 310;
    uint256 constant l1FeeScalar = 10;
    uint32 constant blobBasefeeScalar = 15;
    uint32 constant basefeeScalar = 20;

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

contract GasPriceOracleEcotone_Test is GasPriceOracle_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        // Define the function signature
        bytes4 functionSignature = bytes4(keccak256("setL1BlockValuesEcotone()"));

        // Encode the function signature and extra data
        bytes memory callDataPacked = abi.encodePacked(
            basefeeScalar,
            blobBasefeeScalar,
            sequenceNumber,
            timestamp,
            number,
            basefee,
            blobBasefee,
            hash,
            batcherHash
        );
        bytes memory functionCallData = abi.encodePacked(functionSignature, callDataPacked);

        // Execute the function call
        vm.prank(depositor);
        (bool success, ) = address(l1Block).call(functionCallData);
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
        assertEq(gasPriceOracle.l1BaseFee(), basefee);
    }

    /// @dev Tests that `blobBasefee` is set correctly.
    function test_blobBasefee_succeeds() external {
        assertEq(gasPriceOracle.blobBasefee(), blobBasefee);
    }

    /// @dev Tests that `basefeeScalar` is set correctly.
    function test_basefeeScalar_succeeds() external {
        assertEq(gasPriceOracle.basefeeScalar(), basefeeScalar);
    }

    /// @dev Tests that `blobBasefeeScalar` is set correctly.
    function test_blobBasefeeScalar_succeeds() external {
        assertEq(gasPriceOracle.blobBasefeeScalar(), blobBasefeeScalar);
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