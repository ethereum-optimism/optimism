// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title IPreimageOracle
/// @notice Interface for a preimage oracle.
interface IPreimageOracle {
    /// @notice Reads a preimage from the oracle.
    /// @param _key The key of the preimage to read.
    /// @param _offset The offset of the preimage to read.
    /// @return dat_ The preimage data.
    /// @return datLen_ The length of the preimage data.
    function readPreimage(bytes32 _key, uint256 _offset) external view returns (bytes32 dat_, uint256 datLen_);

    /// @notice Loads of local data part into the preimage oracle.
    /// @param _ident The identifier of the local data.
    /// @param _localContext The local key context for the preimage oracle. Optionally, can be set as a constant
    ///                      if the caller only requires one set of local keys.
    /// @param _word The local data word.
    /// @param _size The number of bytes in `_word` to load.
    /// @param _partOffset The offset of the local data part to write to the oracle.
    /// @dev The local data parts are loaded into the preimage oracle under the context
    ///      of the caller - no other account can write to the caller's context
    ///      specific data.
    ///
    ///      There are 5 local data identifiers:
    ///      ┌────────────┬────────────────────────┐
    ///      │ Identifier │      Data              │
    ///      ├────────────┼────────────────────────┤
    ///      │          1 │ L1 Head Hash (bytes32) │
    ///      │          2 │ Output Root (bytes32)  │
    ///      │          3 │ Root Claim (bytes32)   │
    ///      │          4 │ L2 Block Number (u64)  │
    ///      │          5 │ Chain ID (u64)         │
    ///      └────────────┴────────────────────────┘
    function loadLocalData(
        uint256 _ident,
        bytes32 _localContext,
        bytes32 _word,
        uint256 _size,
        uint256 _partOffset
    )
        external
        returns (bytes32 key_);

    /// @notice Prepares a preimage to be read by keccak256 key, starting at the given offset and up to 32 bytes
    ///         (clipped at preimage length, if out of data).
    /// @param _partOffset The offset of the preimage to read.
    /// @param _preimage The preimage data.
    function loadKeccak256PreimagePart(uint256 _partOffset, bytes calldata _preimage) external;

    /// @notice Prepares a preimage to be read by sha256 key, starting at the given offset and up to 32 bytes
    ///         (clipped at preimage length, if out of data).
    /// @param _partOffset The offset of the preimage to read.
    /// @param _preimage The preimage data.
    function loadSha256PreimagePart(uint256 _partOffset, bytes calldata _preimage) external;

    /// @notice Verifies that `p(_z) = _y` given `_commitment` that corresponds to the polynomial `p(x)` and a KZG
    //          proof. The value `y` is the pre-image, and the preimage key is `5 ++ keccak256(_commitment ++ z)[1:]`.
    /// @param _z Big endian point value. Part of the preimage key.
    /// @param _y Big endian point value. The preimage for the key.
    /// @param _commitment The commitment to the polynomial. 48 bytes, part of the preimage key.
    /// @param _proof The KZG proof, part of the preimage key.
    /// @param _partOffset The offset of the preimage to store.
    function loadBlobPreimagePart(
        uint256 _z,
        uint256 _y,
        bytes calldata _commitment,
        bytes calldata _proof,
        uint256 _partOffset
    )
        external;
}
