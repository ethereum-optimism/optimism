// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { Types } from "../libraries/Types.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract Encoding_Test is CommonTest {
    function testFuzz_nonceVersioning_succeeds(uint240 _nonce, uint16 _version) external {
        (uint240 nonce, uint16 version) = Encoding.decodeVersionedNonce(
            Encoding.encodeVersionedNonce(_nonce, _version)
        );
        assertEq(version, _version);
        assertEq(nonce, _nonce);
    }

    function testDiff_decodeVersionedNonce_succeeds(uint240 _nonce, uint16 _version) external {
        uint256 nonce = uint256(Encoding.encodeVersionedNonce(_nonce, _version));
        (uint256 decodedNonce, uint256 decodedVersion) = ffi.decodeVersionedNonce(nonce);

        assertEq(_version, uint16(decodedVersion));

        assertEq(_nonce, uint240(decodedNonce));
    }

    function testDiff_encodeCrossDomainMessage_succeeds(
        uint240 _nonce,
        uint8 _version,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external {
        uint8 version = _version % 2;
        uint256 nonce = Encoding.encodeVersionedNonce(_nonce, version);

        bytes memory encoding = Encoding.encodeCrossDomainMessage(
            nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );

        bytes memory _encoding = ffi.encodeCrossDomainMessage(
            nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );

        assertEq(encoding, _encoding);
    }

    function testDiff_encodeDepositTransaction_succeeds(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gas,
        bool isCreate,
        bytes memory _data,
        uint256 _logIndex
    ) external {
        Types.UserDepositTransaction memory t = Types.UserDepositTransaction(
            _from,
            _to,
            isCreate,
            _value,
            _mint,
            _gas,
            _data,
            bytes32(uint256(0)),
            _logIndex
        );

        bytes memory txn = Encoding.encodeDepositTransaction(t);
        bytes memory _txn = ffi.encodeDepositTransaction(t);

        assertEq(txn, _txn);
    }
}
