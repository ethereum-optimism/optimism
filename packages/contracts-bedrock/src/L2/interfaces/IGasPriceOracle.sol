// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IGasPriceOracle
/// @notice Interface for the GasPriceOracle contract.
interface IGasPriceOracle {
    function getL1Fee(bytes memory _data) external view returns (uint256);
    function getL1FeeUpperBound(uint256 _unsignedTxSize) external view returns (uint256);
    function setEcotone() external;
    function setFjord() external;
}
