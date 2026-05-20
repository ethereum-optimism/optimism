// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title CommonwareSimplexBatchConsensusVerifier
/// @notice Verifies a Commonware Simplex secp256r1 batch-consensus proof envelope.
/// @dev The fallback accepts raw proof bytes and returns ABI-encoded bool so op-node can eth_call the verifier
///      directly with the blob transaction calldata.
contract CommonwareSimplexBatchConsensusVerifier {
    bytes9 internal constant SIMPLEX_PREFIX = "CWSIMPLX1";

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
    uint256 internal constant MAX_BITMAP_SIGNERS = 8;

    address internal constant P256_PRECOMPILE = 0x0000000000000000000000000000000000000100;

    uint256 public committeeSize;
    uint256 public quorum;
    mapping(uint256 => bytes32) public validatorX;
    mapping(uint256 => bytes32) public validatorY;

    constructor(bytes32[] memory _validatorX, bytes32[] memory _validatorY, uint256 _quorum) {
        require(_validatorX.length == _validatorY.length, "CommonwareSimplex: length mismatch");
        require(_validConfig(_validatorX.length, _quorum), "CommonwareSimplex: invalid config");
        committeeSize = _validatorX.length;
        quorum = _quorum;
        for (uint256 i = 0; i < _validatorX.length; i++) {
            require(_validatorX[i] != bytes32(0) && _validatorY[i] != bytes32(0), "CommonwareSimplex: empty validator");
            for (uint256 j = 0; j < i; j++) {
                require(
                    _validatorX[i] != _validatorX[j] || _validatorY[i] != _validatorY[j],
                    "CommonwareSimplex: duplicate validator"
                );
            }
            validatorX[i] = _validatorX[i];
            validatorY[i] = _validatorY[i];
        }
    }

    /// @notice Raw-call entrypoint used by op-node derivation.
    fallback(bytes calldata _proof) external returns (bytes memory) {
        return abi.encode(_verify(_proof));
    }

    /// @notice Named verifier entrypoint for tests and tooling.
    function verifyBatchConsensus(bytes calldata _proof) external view returns (bool) {
        return _verify(_proof);
    }

    function _verify(bytes calldata _proof) internal view returns (bool) {
        uint256 size = committeeSize;
        uint256 threshold = quorum;
        if (!_validConfig(size, threshold)) return false;
        if (_proof.length < SIGNATURE_OFFSET) return false;
        if (_readBytes9(_proof, 0) != SIMPLEX_PREFIX) return false;
        if (_readBytes9(_proof, CERTIFICATE_OFFSET) != SIMPLEX_PREFIX) return false;
        if (_byteAt(_proof, OUTER_MARKER_OFFSET) != 0x01) return false;
        if (_readBytes32(_proof, OUTER_DIGEST_OFFSET) != _readBytes32(_proof, PAYLOAD_OFFSET)) return false;
        if (_readUint64(_proof, SIGNERS_LEN_OFFSET) != size) return false;

        uint8 bitmap = _byteAt(_proof, BITMAP_OFFSET);
        uint256 signatureCount = _byteAt(_proof, SIGNATURE_LEN_OFFSET);
        if (signatureCount < threshold) return false;
        if (_proof.length != SIGNATURE_OFFSET + (signatureCount * 64)) return false;
        if (_countBits(bitmap) != signatureCount) return false;

        bytes32 digest = _simplexSigningDigest(_proof);
        return _verifyQuorum(_proof, digest, bitmap, signatureCount, size, threshold);
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

    function _verifyQuorum(
        bytes calldata _proof,
        bytes32 _digest,
        uint8 _bitmap,
        uint256 _signatureCount,
        uint256 _committeeSize,
        uint256 _quorum
    )
        internal
        view
        returns (bool)
    {
        uint256 verified = 0;
        uint256 bitmap = uint256(_bitmap);
        for (uint256 validatorIndex = 0; validatorIndex < _committeeSize; validatorIndex++) {
            if ((bitmap & (uint256(1) << validatorIndex)) == 0) continue;
            if (!_verifyValidator(_proof, _digest, validatorIndex, verified)) return false;
            verified++;
        }
        return verified == _signatureCount && verified >= _quorum;
    }

    function _verifyValidator(
        bytes calldata _proof,
        bytes32 _digest,
        uint256 _validatorIndex,
        uint256 _signatureIndex
    )
        internal
        view
        returns (bool)
    {
        uint256 sigOffset = SIGNATURE_OFFSET + (_signatureIndex * 64);
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

    function _validatorKey(uint256 _validatorIndex) internal view returns (bytes32 x_, bytes32 y_) {
        x_ = validatorX[_validatorIndex];
        y_ = validatorY[_validatorIndex];
    }

    function _validConfig(uint256 _committeeSize, uint256 _quorum) internal pure returns (bool) {
        return _committeeSize > 0 && _committeeSize <= MAX_BITMAP_SIGNERS && _quorum > 0 && _quorum <= _committeeSize;
    }

    function _countBits(uint8 _bitmap) internal pure returns (uint256 count_) {
        uint256 bitmap = uint256(_bitmap);
        for (uint256 i = 0; i < MAX_BITMAP_SIGNERS; i++) {
            if ((bitmap & (uint256(1) << i)) != 0) count_++;
        }
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
