// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { LegacyCrossDomainUtils } from "src/libraries/LegacyCrossDomainUtils.sol";

// Target contract
import { Encoding } from "src/libraries/Encoding.sol";

contract Encoding_Test is CommonTest {
    /// @dev Tests encoding and decoding a nonce and version.
    function testFuzz_nonceVersioning_succeeds(uint240 _nonce, uint16 _version) external pure {
        (uint240 nonce, uint16 version) = Encoding.decodeVersionedNonce(Encoding.encodeVersionedNonce(_nonce, _version));
        assertEq(version, _version);
        assertEq(nonce, _nonce);
    }

    /// @dev Tests decoding a versioned nonce.
    function testDiff_decodeVersionedNonce_succeeds(uint240 _nonce, uint16 _version) external {
        uint256 nonce = uint256(Encoding.encodeVersionedNonce(_nonce, _version));
        (uint256 decodedNonce, uint256 decodedVersion) = ffi.decodeVersionedNonce(nonce);

        assertEq(_version, uint16(decodedVersion));

        assertEq(_nonce, uint240(decodedNonce));
    }

    /// @dev Tests cross domain message encoding.
    function testDiff_encodeCrossDomainMessage_succeeds(
        uint240 _nonce,
        uint8 _version,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        uint8 version = _version % 2;
        uint256 nonce = Encoding.encodeVersionedNonce(_nonce, version);

        bytes memory encoding = Encoding.encodeCrossDomainMessage(nonce, _sender, _target, _value, _gasLimit, _data);

        bytes memory _encoding = ffi.encodeCrossDomainMessage(nonce, _sender, _target, _value, _gasLimit, _data);

        assertEq(encoding, _encoding);
    }

    /// @dev Tests legacy cross domain message encoding.
    function testFuzz_encodeCrossDomainMessageV0_matchesLegacy_succeeds(
        uint240 _nonce,
        address _sender,
        address _target,
        bytes memory _data
    )
        external
        pure
    {
        uint8 version = 0;
        uint256 nonce = Encoding.encodeVersionedNonce(_nonce, version);

        bytes memory legacyEncoding = LegacyCrossDomainUtils.encodeXDomainCalldata(_target, _sender, _data, nonce);

        bytes memory bedrockEncoding = Encoding.encodeCrossDomainMessageV0(_target, _sender, _data, nonce);

        assertEq(legacyEncoding, bedrockEncoding);
    }

    /// @dev Tests deposit transaction encoding.
    function testDiff_encodeDepositTransaction_succeeds(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gas,
        bool isCreate,
        bytes memory _data,
        uint64 _logIndex
    )
        external
    {
        Types.UserDepositTransaction memory t = Types.UserDepositTransaction(
            _from, _to, isCreate, _value, _mint, _gas, _data, bytes32(uint256(0)), _logIndex
        );

        bytes memory txn = Encoding.encodeDepositTransaction(t);
        bytes memory _txn = ffi.encodeDepositTransaction(t);

        assertEq(txn, _txn);
    }

    /// @dev Tests encodeSetL1BlockValuesInterop against the Go implementation.
    function testDiff_encodeSetL1BlockValuesInterop_succeeds(
        uint32 _baseFeeScalar,
        uint32 _blobBaseFeeScalar,
        uint64 _sequenceNumber,
        uint64 _timestamp,
        uint64 _number,
        uint256 _baseFee,
        uint256 _blobBaseFee,
        bytes32 _hash,
        bytes32 _batcherHash,
        uint256[] memory _dependencySet
    )
        external
    {
        vm.assume(_dependencySet.length <= type(uint8).max);
        vm.assume(uint160(uint256(_batcherHash)) == uint256(_batcherHash));

        bytes memory encoding = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: _baseFeeScalar,
            _blobBaseFeeScalar: _blobBaseFeeScalar,
            _sequenceNumber: _sequenceNumber,
            _timestamp: _timestamp,
            _number: _number,
            _baseFee: _baseFee,
            _blobBaseFee: _blobBaseFee,
            _hash: _hash,
            _batcherHash: _batcherHash,
            _dependencySet: _dependencySet
        });

        bytes memory _encoding = ffi.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: _baseFeeScalar,
            _blobBaseFeeScalar: _blobBaseFeeScalar,
            _sequenceNumber: _sequenceNumber,
            _timestamp: _timestamp,
            _number: _number,
            _baseFee: _baseFee,
            _blobBaseFee: _blobBaseFee,
            _hash: _hash,
            _batcherHash: _batcherHash,
            _dependencySet: _dependencySet
        });

        assertEq(encoding, _encoding);
    }

    /// @dev Tests that encodeSetL1BlockValuesInterop fails if the dependency set is too large.
    function test_encodeSetL1BlockValuesInterop_dependencySetTooLarge_fails() external {
        uint256[] memory dependencySet = new uint256[](256);

        vm.expectRevert("Encoding: dependency set length is too large");
        Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: type(uint32).max,
            _blobBaseFeeScalar: type(uint32).max,
            _sequenceNumber: type(uint64).max,
            _timestamp: type(uint64).max,
            _number: type(uint64).max,
            _baseFee: type(uint256).max,
            _blobBaseFee: type(uint256).max,
            _hash: bytes32(type(uint256).max),
            _batcherHash: bytes32(type(uint256).max),
            _dependencySet: dependencySet
        });
    }

    /// @dev Tests that encodeSetL1BlockValuesInterop fails if the batcher hash is invalid.
    function test_encodeSetL1BlockValuesInterop_invalidBatcherHash_fails() external {
        vm.expectRevert("Encoding: invalid batcher hash");
        Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: type(uint32).max,
            _blobBaseFeeScalar: type(uint32).max,
            _sequenceNumber: type(uint64).max,
            _timestamp: type(uint64).max,
            _number: type(uint64).max,
            _baseFee: type(uint256).max,
            _blobBaseFee: type(uint256).max,
            _hash: bytes32(type(uint256).max),
            _batcherHash: 0x1000000000000000000000005991a2df15a8f6a256d3ec51e99254cd3fb576a9,
            _dependencySet: new uint256[](0)
        });
    }
}
