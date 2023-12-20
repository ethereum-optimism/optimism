// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import "forge-std/console.sol";

// Target contract
import { L1Block } from "src/L2/L1Block.sol";

contract L1BlockTest is CommonTest {
    address depositor;

    // The initial L1 context values
    uint64 constant number = 123;
    uint64 constant timestamp = 456;
    uint256 constant basefee = 789;
    uint256 constant blobBasefee = 1011;
    bytes32 constant hash = bytes32(uint256(1213));
    uint64 constant sequenceNumber = 14;
    bytes32 constant batcherHash = bytes32(uint256(1516));
    uint256 constant l1FeeOverhead = 1718;
    uint256 constant l1FeeScalar = 1920;
    uint32 constant basefeeScalar = 21;
    uint32 constant blobBasefeeScalar = 22;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }
}

contract L1BlockBedrock_Test is L1BlockTest {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

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

    // @dev Tests that `setL1BlockValues` updates the values correctly.
    function testFuzz_updatesValues_succeeds(
        uint64 n,
        uint64 t,
        uint256 b,
        bytes32 h,
        uint64 s,
        bytes32 bt,
        uint256 fo,
        uint256 fs
    )
        external
    {
        vm.prank(depositor);
        l1Block.setL1BlockValues(n, t, b, h, s, bt, fo, fs);
        assertEq(l1Block.number(), n);
        assertEq(l1Block.timestamp(), t);
        assertEq(l1Block.basefee(), b);
        assertEq(l1Block.hash(), h);
        assertEq(l1Block.sequenceNumber(), s);
        assertEq(l1Block.batcherHash(), bt);
        assertEq(l1Block.l1FeeOverhead(), fo);
        assertEq(l1Block.l1FeeScalar(), fs);
    }

    /// @dev Tests that `number` returns the correct value.
    function test_number_succeeds() external {
        assertEq(l1Block.number(), number);
    }

    /// @dev Tests that `timestamp` returns the correct value.
    function test_timestamp_succeeds() external {
        assertEq(l1Block.timestamp(), timestamp);
    }

    /// @dev Tests that `basefee` returns the correct value.
    function test_basefee_succeeds() external {
        assertEq(l1Block.basefee(), basefee);
    }

    /// @dev Tests that `hash` returns the correct value.
    function test_hash_succeeds() external {
        assertEq(l1Block.hash(), hash);
    }

    /// @dev Tests that `sequenceNumber` returns the correct value.
    function test_sequenceNumber_succeeds() external {
        assertEq(l1Block.sequenceNumber(), sequenceNumber);
    }

    /// @dev Tests that `setL1BlockValues` can set max values.
    function test_updateValues_succeeds() external {
        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: type(uint64).max,
            _timestamp: type(uint64).max,
            _basefee: type(uint256).max,
            _hash: keccak256(abi.encode(1)),
            _sequenceNumber: type(uint64).max,
            _batcherHash: bytes32(type(uint256).max),
            _l1FeeOverhead: type(uint256).max,
            _l1FeeScalar: type(uint256).max
        });
    }
}

contract L1BlockEcotone_Test is L1BlockTest {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        bytes4 functionSignature = bytes4(keccak256("setL1BlockValuesEcotone()"));
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

        bytes memory functionCallDataPacked = abi.encodePacked(functionSignature, callDataPacked);

        vm.prank(depositor);
        (bool success, ) = address(l1Block).call(functionCallDataPacked);
        require(success, "Function call failed");
    }

    /// @dev Tests that `number` returns the correct value.
    function test_number_succeeds() external {
        assertEq(l1Block.number(), number);
    }

    /// @dev Tests that `timestamp` returns the correct value.
    function test_timestamp_succeeds() external {
        assertEq(l1Block.timestamp(), timestamp);
    }

    /// @dev Tests that `basefee` returns the correct value.
    function test_basefee_succeeds() external {
        assertEq(l1Block.basefee(), basefee);
    }

    /// @dev Tests that `blobBasefee` returns the correct value.
    function test_blobBaseFee_succeeds() external {
        assertEq(l1Block.blobBasefee(), blobBasefee);
    }

    /// @dev Tests that `hash` returns the correct value.
    function test_hash_succeeds() external {
        assertEq(l1Block.hash(), hash);
    }

    /// @dev Tests that `sequenceNumber` returns the correct value.
    function test_sequenceNumber_succeeds() external {
        assertEq(l1Block.sequenceNumber(), sequenceNumber);
    }

    /// @dev Tests that `batcherHash` returns the correct value.
    function test_batcherHash_succeeds() external {
        assertEq(l1Block.batcherHash(), batcherHash);
    }

    /// @dev Tests that `basefeeScalar` returns the correct value.
    function test_baseFeeScalar_succeeds() external {
        assertEq(l1Block.basefeeScalar(), basefeeScalar);
    }

    /// @dev Tests that `blobBasefeeScalar` returns the correct value.
    function test_blobBaseFeeScalar_succeeds() external {
        assertEq(l1Block.blobBasefeeScalar(), blobBasefeeScalar);
    }
}