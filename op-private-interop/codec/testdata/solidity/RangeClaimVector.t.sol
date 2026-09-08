// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// RangeClaimVector.t.sol pins op-private-interop/codec against solc's own abi.encode.
//
// The Go codec and the fixture corpus can only prove this package is self-consistent. Only a second
// compiler proves it is speaking ABI rather than a convincing dialect of it — and since the
// claim registry is LOG-LESS, the durable record is the postClaim CALLDATA these bytes make up —
// producer and reader meet here, with nothing in between to re-encode and normalise them.
//
// Self-contained on purpose: no forge-std, no imports, no remappings, so it runs anywhere. Drop it
// into any foundry project and `forge test`. Last run green under solc 0.8.30 / forge 1.4.4.
//
// The two GO_* constants are the Go encoder's output for the same two struct values; the Go side
// carries them in TestSolidityProducesTheSameBytes.

struct RangeClaim {
    uint8 version;
    uint64 firstBlock;
    uint64 lastBlock;
    bytes32 privateTerminalBlockHash;
    bytes32 privateTerminalParentHash;
    bytes32 l1Head;
    bytes32 rollupConfigHash;
    bytes32 depSetHash;
    bytes32 privateDataHash;
    bytes proof;
}

contract Encoder {
    // emptyProof mirrors the empty-proof vector in the Go TestSolidityProducesTheSameBytes.
    function emptyProof() external pure returns (bytes memory) {
        RangeClaim memory e = RangeClaim({
            version: 1,
            firstBlock: 1,
            lastBlock: 300,
            privateTerminalBlockHash: bytes32(uint256(1)),
            privateTerminalParentHash: bytes32(uint256(6)),
            l1Head: bytes32(uint256(2)),
            rollupConfigHash: bytes32(uint256(3)),
            depSetHash: bytes32(uint256(4)),
            privateDataHash: bytes32(uint256(5)),
            proof: hex""
        });
        return abi.encode(e);
    }

    // withProof mirrors the Go word-layout vector TestABIWordLayout (same values, same proof).
    function withProof() external pure returns (bytes memory) {
        RangeClaim memory e = RangeClaim({
            version: 1,
            firstBlock: 0x0102030405060708,
            lastBlock: 0x1112131415161718,
            privateTerminalBlockHash: bytes32(uint256(0x2222222222222222222222222222222222222222222222222222222222222222)),
            privateTerminalParentHash: bytes32(uint256(0x7777777777777777777777777777777777777777777777777777777777777777)),
            l1Head: bytes32(uint256(0x3333333333333333333333333333333333333333333333333333333333333333)),
            rollupConfigHash: bytes32(uint256(0x4444444444444444444444444444444444444444444444444444444444444444)),
            depSetHash: bytes32(uint256(0x5555555555555555555555555555555555555555555555555555555555555555)),
            privateDataHash: bytes32(uint256(0x6666666666666666666666666666666666666666666666666666666666666666)),
            proof: hex"aabbcc"
        });
        return abi.encode(e);
    }

    // reencode is decode-then-encode over a calldata struct. It is the check a registry would use
    // if it ever wanted to enforce canonical calldata on chain:
    // keccak256(reencode(claim)) == keccak256(claim-as-sent).
    function reencode(RangeClaim calldata _claim) external pure returns (bytes memory) {
        return abi.encode(_claim);
    }
}

contract RangeClaimVectorTest {
    Encoder private enc = new Encoder();

    bytes constant GO_EMPTY =
        hex"00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001"
        hex"0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000012c"
        hex"00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000006"
        hex"00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003"
        hex"00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000005"
        hex"00000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000";

    bytes constant GO_PROOF =
        hex"00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001"
        hex"00000000000000000000000000000000000000000000000001020304050607080000000000000000000000000000000000000000000000001112131415161718"
        hex"22222222222222222222222222222222222222222222222222222222222222227777777777777777777777777777777777777777777777777777777777777777"
        hex"33333333333333333333333333333333333333333333333333333333333333334444444444444444444444444444444444444444444444444444444444444444"
        hex"55555555555555555555555555555555555555555555555555555555555555556666666666666666666666666666666666666666666666666666666666666666"
        hex"00000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000003"
        hex"aabbcc0000000000000000000000000000000000000000000000000000000000";

    function testEmptyProofMatchesGo() public view {
        require(keccak256(enc.emptyProof()) == keccak256(GO_EMPTY), "empty-proof vector differs from Go");
    }

    function testWithProofMatchesGo() public view {
        require(keccak256(enc.withProof()) == keccak256(GO_PROOF), "with-proof vector differs from Go");
    }

    /// Decoding the Go bytes and re-encoding them must be a fixpoint — exactly the canonicality the
    /// Go decoder enforces, checked from the other side of the wire. With no event to launder the
    /// bytes, this is the property a calldata reader depends on: the encoding is the only encoding.
    function testReencodeIsAFixpoint() public view {
        RangeClaim memory e = abi.decode(GO_PROOF, (RangeClaim));
        require(keccak256(enc.reencode(e)) == keccak256(GO_PROOF), "re-encode is not a fixpoint");
        RangeClaim memory z = abi.decode(GO_EMPTY, (RangeClaim));
        require(keccak256(enc.reencode(z)) == keccak256(GO_EMPTY), "empty re-encode is not a fixpoint");
    }
}
