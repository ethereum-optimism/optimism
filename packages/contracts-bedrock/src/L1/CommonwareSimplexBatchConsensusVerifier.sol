// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title CommonwareSimplexBatchConsensusVerifier
/// @notice Verifies the fixed Commonware Simplex secp256r1 batch-consensus proof envelope used by the POC.
/// @dev The fallback accepts raw proof bytes and returns ABI-encoded bool so op-node can eth_call the verifier
///      directly with the blob transaction calldata.
contract CommonwareSimplexBatchConsensusVerifier {
    bytes9 internal constant SIMPLEX_PREFIX = "CWSIMPLX1";

    uint256 internal constant PROOF_SIZE = 0x01e3;
    uint256 internal constant OUTER_DIGEST_OFFSET = 0x0009;
    uint256 internal constant OUTER_MARKER_OFFSET = 0x0029;
    uint256 internal constant CERTIFICATE_OFFSET = 0x00ed;
    uint256 internal constant PROPOSAL_OFFSET = CERTIFICATE_OFFSET + 0x0009;
    uint256 internal constant PAYLOAD_OFFSET = PROPOSAL_OFFSET + 0x0003;
    uint256 internal constant SIGNERS_LEN_OFFSET = PROPOSAL_OFFSET + 0x0023;
    uint256 internal constant BITMAP_OFFSET = SIGNERS_LEN_OFFSET + 0x0008;
    uint256 internal constant SIGNATURE_LEN_OFFSET = BITMAP_OFFSET + 0x0001;
    uint256 internal constant SIGNATURE_OFFSET = SIGNATURE_LEN_OFFSET + 0x0001;
    uint256 internal constant PROPOSAL_LEN = 0x0023;
    uint256 internal constant HASH_INPUT_LEN = 0x004d;
    uint256 internal constant P256_INPUT_LEN = 0x00a0;

    address internal constant P256_PRECOMPILE = 0x0000000000000000000000000000000000000100;

    bytes32 internal constant VALIDATOR_0_X = 0xb94425908ddd66d4dc8fc0e1516fe888c7558e5945da5c0b3df3d9a12cc7d991;
    bytes32 internal constant VALIDATOR_0_Y = 0x30a50e74eb9cf19dbe8545f55f25774d035431ddd8f60996f6be020ab9932cff;
    bytes32 internal constant VALIDATOR_1_X = 0x6348f24d507e944cc8a2c73c88f05cc1224b4873bd7b12fb75d48be495b1a3c6;
    bytes32 internal constant VALIDATOR_1_Y = 0x674fe8b903c5d7f11d5f6261851f0e0e5cd835dfea422f0a59a0c43a56d8482b;
    bytes32 internal constant VALIDATOR_2_X = 0xc2d3f77d431b8d1b66f73bff503fd3f82dc4172992e7bebd9f836dbc8da7efa8;
    bytes32 internal constant VALIDATOR_2_Y = 0xea311f669388c70096d166207adbe142f4ae09384a66d9aa73b9e6646ab8ad7f;

    /// @notice Raw-call entrypoint used by op-node derivation.
    fallback(bytes calldata _proof) external returns (bytes memory) {
        return abi.encode(_verify(_proof));
    }

    /// @notice Named verifier entrypoint for tests and tooling.
    function verifyBatchConsensus(bytes calldata _proof) external view returns (bool) {
        return _verify(_proof);
    }

    function _verify(bytes calldata _proof) internal view returns (bool) {
        if (_proof.length != PROOF_SIZE) return false;
        if (_readBytes9(_proof, 0) != SIMPLEX_PREFIX) return false;
        if (_readBytes9(_proof, CERTIFICATE_OFFSET) != SIMPLEX_PREFIX) return false;
        if (_byteAt(_proof, OUTER_MARKER_OFFSET) != 0x01) return false;
        if (_byteAt(_proof, BITMAP_OFFSET) != 0x0e) return false;
        if (_byteAt(_proof, SIGNATURE_LEN_OFFSET) != 0x03) return false;
        if (_readBytes32(_proof, OUTER_DIGEST_OFFSET) != _readBytes32(_proof, PAYLOAD_OFFSET)) return false;
        if (_readUint64(_proof, SIGNERS_LEN_OFFSET) != 4) return false;

        bytes32 digest = _simplexSigningDigest(_proof);
        return _verifyValidator(_proof, digest, 0) && _verifyValidator(_proof, digest, 1)
            && _verifyValidator(_proof, digest, 2);
    }

    function _simplexSigningDigest(bytes calldata _proof) internal pure returns (bytes32 digest_) {
        bytes memory input = new bytes(HASH_INPUT_LEN);
        uint256 proposalOffset = PROPOSAL_OFFSET;
        assembly {
            let ptr := add(input, 0x20)
            mstore(ptr, 0x296f702d626174636865722d636f6e73656e7375732d706f632f73696d706c65)
            mstore(add(ptr, 0x20), 0x785f46494e414c495a4500000000000000000000000000000000000000000000)
            calldatacopy(add(ptr, 0x2a), add(_proof.offset, proposalOffset), 0x23)
        }
        digest_ = sha256(input);
    }

    function _verifyValidator(
        bytes calldata _proof,
        bytes32 _digest,
        uint256 _validatorIndex
    )
        internal
        view
        returns (bool)
    {
        uint256 sigOffset = SIGNATURE_OFFSET + (_validatorIndex * 64);
        bytes32 r = _readBytes32(_proof, sigOffset);
        bytes32 s = _readBytes32(_proof, sigOffset + 32);
        (bytes32 x, bytes32 y) = _validatorKey(_validatorIndex);
        return _verifyP256(_digest, r, s, x, y);
    }

    function _verifyP256(bytes32 _digest, bytes32 _r, bytes32 _s, bytes32 _x, bytes32 _y) internal view returns (bool) {
        bytes memory input = new bytes(P256_INPUT_LEN);
        assembly {
            let ptr := add(input, 0x20)
            mstore(ptr, _digest)
            mstore(add(ptr, 0x20), _r)
            mstore(add(ptr, 0x40), _s)
            mstore(add(ptr, 0x60), _x)
            mstore(add(ptr, 0x80), _y)
        }
        (bool success, bytes memory output) = P256_PRECOMPILE.staticcall(input);
        return success && output.length == 32 && abi.decode(output, (uint256)) == 1;
    }

    function _validatorKey(uint256 _validatorIndex) internal pure returns (bytes32 x_, bytes32 y_) {
        if (_validatorIndex == 0) return (VALIDATOR_0_X, VALIDATOR_0_Y);
        if (_validatorIndex == 1) return (VALIDATOR_1_X, VALIDATOR_1_Y);
        if (_validatorIndex == 2) return (VALIDATOR_2_X, VALIDATOR_2_Y);
        revert("invalid validator index");
    }

    function _readBytes9(bytes calldata _data, uint256 _offset) internal pure returns (bytes9 out_) {
        assembly {
            out_ := calldataload(add(_data.offset, _offset))
        }
    }

    function _readBytes32(bytes calldata _data, uint256 _offset) internal pure returns (bytes32 out_) {
        assembly {
            out_ := calldataload(add(_data.offset, _offset))
        }
    }

    function _readUint64(bytes calldata _data, uint256 _offset) internal pure returns (uint64 out_) {
        assembly {
            out_ := shr(192, calldataload(add(_data.offset, _offset)))
        }
    }

    function _byteAt(bytes calldata _data, uint256 _offset) internal pure returns (uint8 out_) {
        out_ = uint8(_data[_offset]);
    }
}
