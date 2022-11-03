// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { Types } from "../libraries/Types.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract Hashing_Test is CommonTest {
    function setUp() external {
        _setUp();
    }

    function test_hashDepositSource() external {
        bytes32 sourceHash = Hashing.hashDepositSource(
            0xd25df7858efc1778118fb133ac561b138845361626dfb976699c5287ed0f4959,
            0x1
        );

        assertEq(sourceHash, 0xf923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc);
    }

    function test_hashCrossDomainMessage_differential(
        uint240 _nonce,
        uint16 _version,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external {
        // Ensure the version is valid
        uint16 version = uint16(bound(uint256(_version), 0, 1));
        uint256 nonce = Encoding.encodeVersionedNonce(_nonce, version);

        bytes32 _hash = ffi.hashCrossDomainMessage(
            nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );

        bytes32 hash = Hashing.hashCrossDomainMessage(
            nonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );

        assertEq(hash, _hash);
    }

    function test_hashWithdrawal_differential(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) external {
        bytes32 hash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction(_nonce, _sender, _target, _value, _gasLimit, _data)
        );

        bytes32 _hash = ffi.hashWithdrawal(_nonce, _sender, _target, _value, _gasLimit, _data);

        assertEq(hash, _hash);
    }

    function test_hashOutputRootProof_differential(
        bytes32 _version,
        bytes32 _stateRoot,
        bytes32 _messagePasserStorageRoot,
        bytes32 _latestBlockhash
    ) external {
        Types.OutputRootProof memory proof = Types.OutputRootProof({
            version: _version,
            stateRoot: _stateRoot,
            messagePasserStorageRoot: _messagePasserStorageRoot,
            latestBlockhash: _latestBlockhash
        });

        bytes32 hash = Hashing.hashOutputRootProof(proof);

        bytes32 _hash = ffi.hashOutputRootProof(
            _version,
            _stateRoot,
            _messagePasserStorageRoot,
            _latestBlockhash
        );

        assertEq(hash, _hash);
    }

    // TODO(tynes): foundry bug cannot serialize
    // bytes32 as strings with vm.toString
    function test_hashDepositTransaction_differential(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gas,
        bytes memory _data,
        uint256 _logIndex
    ) external {
        bytes32 hash = Hashing.hashDepositTransaction(
            Types.UserDepositTransaction(
                _from,
                _to,
                false, // isCreate
                _value,
                _mint,
                _gas,
                _data,
                bytes32(uint256(0)),
                _logIndex
            )
        );

        bytes32 _hash = ffi.hashDepositTransaction(
            _from,
            _to,
            _mint,
            _value,
            _gas,
            _data,
            _logIndex
        );

        assertEq(hash, _hash);
    }
}
