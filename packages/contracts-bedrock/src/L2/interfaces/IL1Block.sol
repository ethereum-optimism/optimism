// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL1Block
/// @notice Interface for the L1Block contract.
interface IL1Block {
    function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external;
    function setL1BlockValues(
        uint64 _number,
        uint64 _timestamp,
        uint256 _basefee,
        bytes32 _hash,
        uint64 _sequenceNumber,
        bytes32 _batcherHash,
        uint256 _l1FeeOverhead,
        uint256 _l1FeeScalar
    )
        external;
    function setL1BlockValuesEcotone() external;
}
