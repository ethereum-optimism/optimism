// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";
import { Constants } from "src/libraries/Constants.sol";

// Target contract
import { L1Block } from "src/L2/L1Block.sol";
import { NotDepositor, NotCrossL2Inbox } from "src/libraries/L1BlockErrors.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

contract L1BlockTest is CommonTest {
    address depositor;

    event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol);

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }
}

contract L1BlockBedrock_Test is L1BlockTest {
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

    /// @dev Tests that `setL1BlockValues` reverts if sender address is not the depositor
    function test_updatesValues_notDepositor_reverts() external {
        vm.expectRevert("L1Block: only the depositor account can set L1 block values");
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

contract L1BlockSetL1BlockValuesIsthmus_Test is L1BlockTest {
    /// @dev Tests that `setL1BlockValuesIsthmus` reverts if sender address is not the depositor
    function test_setL1BlockValuesIsthmus_notDepositor_reverts(address _caller) external {
        vm.assume(_caller != depositor);
        vm.prank(_caller);
        vm.expectRevert(NotDepositor.selector);
        l1Block.setL1BlockValuesIsthmus();
    }

    /// @dev Tests that `setL1BlockValuesIsthmus` succeeds if sender address is the depositor
    function test_setL1BlockValuesIsthmus_succeeds(
        uint32 baseFeeScalar,
        uint32 blobBaseFeeScalar,
        uint64 sequenceNumber,
        uint64 timestamp,
        uint64 number,
        uint256 baseFee,
        uint256 blobBaseFee,
        bytes32 hash,
        bytes32 batcherHash
    )
        external
    {
        // Ensure the `isDepositTransaction` flag is false before calling `setL1BlockValuesIsthmus`
        vm.prank(Predeploys.CROSS_L2_INBOX);
        assertEq(l1Block.isDeposit(), false);

        bytes memory setValuesEcotoneCalldata = abi.encodePacked(
            baseFeeScalar, blobBaseFeeScalar, sequenceNumber, timestamp, number, baseFee, blobBaseFee, hash, batcherHash
        );

        vm.prank(depositor);
        (bool success,) =
            address(l1Block).call(abi.encodePacked(L1Block.setL1BlockValuesIsthmus.selector, setValuesEcotoneCalldata));
        assertTrue(success, "function call failed");

        // Assert that the `isDepositTransaction` flag was properly set to true
        vm.prank(Predeploys.CROSS_L2_INBOX);
        assertEq(l1Block.isDeposit(), true);

        // Assert `setL1BlockValuesEcotone` was properly called, forwarding the calldata to it
        assertEq(l1Block.baseFeeScalar(), baseFeeScalar, "base fee scalar not properly set");
        assertEq(l1Block.blobBaseFeeScalar(), blobBaseFeeScalar, "blob base fee scalar not properly set");
        assertEq(l1Block.sequenceNumber(), sequenceNumber, "sequence number not properly set");
        assertEq(l1Block.timestamp(), timestamp, "timestamp not properly set");
        assertEq(l1Block.number(), number, "number not properly set");
        assertEq(l1Block.basefee(), baseFee, "base fee not properly set");
        assertEq(l1Block.blobBaseFee(), blobBaseFee, "blob base fee not properly set");
        assertEq(l1Block.hash(), hash, "hash not properly set");
        assertEq(l1Block.batcherHash(), batcherHash, "batcher hash not properly set");
    }
}

contract L1BlockEcotone_Test is L1BlockTest {
    /// @dev Tests that setL1BlockValuesEcotone updates the values appropriately.
    function testFuzz_setL1BlockValuesEcotone_succeeds(
        uint32 baseFeeScalar,
        uint32 blobBaseFeeScalar,
        uint64 sequenceNumber,
        uint64 timestamp,
        uint64 number,
        uint256 baseFee,
        uint256 blobBaseFee,
        bytes32 hash,
        bytes32 batcherHash
    )
        external
    {
        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesEcotone(
            baseFeeScalar, blobBaseFeeScalar, sequenceNumber, timestamp, number, baseFee, blobBaseFee, hash, batcherHash
        );

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "Function call failed");

        assertEq(l1Block.baseFeeScalar(), baseFeeScalar);
        assertEq(l1Block.blobBaseFeeScalar(), blobBaseFeeScalar);
        assertEq(l1Block.sequenceNumber(), sequenceNumber);
        assertEq(l1Block.timestamp(), timestamp);
        assertEq(l1Block.number(), number);
        assertEq(l1Block.basefee(), baseFee);
        assertEq(l1Block.blobBaseFee(), blobBaseFee);
        assertEq(l1Block.hash(), hash);
        assertEq(l1Block.batcherHash(), batcherHash);

        // ensure we didn't accidentally pollute the 128 bits of the sequencenum+scalars slot that
        // should be empty
        bytes32 scalarsSlot = vm.load(address(l1Block), bytes32(uint256(3)));
        bytes32 mask128 = hex"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000000000000000000000000000";

        assertEq(0, scalarsSlot & mask128);

        // ensure we didn't accidentally pollute the 128 bits of the number & timestamp slot that
        // should be empty
        bytes32 numberTimestampSlot = vm.load(address(l1Block), bytes32(uint256(0)));
        assertEq(0, numberTimestampSlot & mask128);
    }

    /// @dev Tests that `setL1BlockValuesEcotone` succeeds if sender address is the depositor
    function test_setL1BlockValuesEcotone_isDepositor_succeeds() external {
        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesEcotone(
            type(uint32).max,
            type(uint32).max,
            type(uint64).max,
            type(uint64).max,
            type(uint64).max,
            type(uint256).max,
            type(uint256).max,
            bytes32(type(uint256).max),
            bytes32(type(uint256).max)
        );

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "function call failed");
    }

    /// @dev Tests that `setL1BlockValuesEcotone` reverts if sender address is not the depositor
    function test_setL1BlockValuesEcotone_notDepositor_reverts() external {
        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesEcotone(
            type(uint32).max,
            type(uint32).max,
            type(uint64).max,
            type(uint64).max,
            type(uint64).max,
            type(uint256).max,
            type(uint256).max,
            bytes32(type(uint256).max),
            bytes32(type(uint256).max)
        );

        (bool success, bytes memory data) = address(l1Block).call(functionCallDataPacked);
        assertTrue(!success, "function call should have failed");
        // make sure return value is the expected function selector for "NotDepositor()"
        bytes memory expReturn = hex"3cc50b45";
        assertEq(data, expReturn);
    }
}

contract L1BlockDepositsComplete_Test is L1BlockTest {
    // @dev Tests that `depositsComplete` reverts if the caller is not the depositor.
    function test_deposits_is_depositor_reverts(address _caller) external {
        vm.assume(_caller != depositor);
        vm.expectRevert(NotDepositor.selector);
        l1Block.depositsComplete();
    }

    // @dev Tests that `depositsComplete` succeeds if the caller is the depositor.
    function test_depositsComplete_succeeds() external {
        // Set the `isDeposit` flag to true
        vm.prank(depositor);
        l1Block.setL1BlockValuesIsthmus();

        // Assert that the `isDeposit` flag was properly set to true
        vm.prank(Predeploys.CROSS_L2_INBOX);
        assertTrue(l1Block.isDeposit());

        // Call `depositsComplete`
        vm.prank(depositor);
        l1Block.depositsComplete();

        // Assert that the `isDeposit` flag was properly set to false
        /// @dev Assuming that `isDeposit()` wil return the proper value. That function is tested as well
        vm.prank(Predeploys.CROSS_L2_INBOX);
        assertEq(l1Block.isDeposit(), false);
    }
}

contract L1BlockCustomGasToken_Test is L1BlockTest {
    function testFuzz_setGasPayingToken_succeeds(
        address _token,
        uint8 _decimals,
        string memory _name,
        string memory _symbol
    )
        external
    {
        vm.assume(_token != address(0));
        vm.assume(_token != Constants.ETHER);
        vm.assume(bytes(_name).length <= 32);
        vm.assume(bytes(_symbol).length <= 32);

        bytes32 name = bytes32(abi.encodePacked(_name));
        bytes32 symbol = bytes32(abi.encodePacked(_symbol));

        vm.expectEmit(address(l1Block));
        emit GasPayingTokenSet({ token: _token, decimals: _decimals, name: name, symbol: symbol });

        vm.prank(depositor);
        l1Block.setGasPayingToken({ _token: _token, _decimals: _decimals, _name: name, _symbol: symbol });

        (address token, uint8 decimals) = l1Block.gasPayingToken();
        assertEq(token, _token);
        assertEq(decimals, _decimals);

        assertEq(_name, l1Block.gasPayingTokenName());
        assertEq(_symbol, l1Block.gasPayingTokenSymbol());
        assertTrue(l1Block.isCustomGasToken());
    }

    function test_setGasPayingToken_isDepositor_reverts() external {
        vm.expectRevert(NotDepositor.selector);
        l1Block.setGasPayingToken(address(this), 18, "Test", "TST");
    }
}

contract L1BlockIsDeposit_Test is L1BlockTest {
    /// @dev Tests that `isDeposit` reverts if the caller is not the cross L2 inbox.
    function test_isDeposit_notCrossL2Inbox_reverts(address _caller) external {
        vm.assume(_caller != Predeploys.CROSS_L2_INBOX);
        vm.expectRevert(NotCrossL2Inbox.selector);
        l1Block.isDeposit();
    }

    /// @dev Tests that `isDeposit` always returns the correct value.
    function test_isDeposit_succeeds() external {
        // Assert is false if the value is not updated
        vm.prank(Predeploys.CROSS_L2_INBOX);
        assertEq(l1Block.isDeposit(), false);

        /// @dev Assuming that `setL1BlockValuesIsthmus` will set the proper value. That function is tested as well
        vm.prank(depositor);
        l1Block.setL1BlockValuesIsthmus();

        // Assert is true if the value is updated
        vm.prank(Predeploys.CROSS_L2_INBOX);
        assertEq(l1Block.isDeposit(), true);
    }
}
