// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL2StandardBridge
/// @notice Interface for the L2StandardBridge contract.
interface IL2StandardBridge {
    function withdraw(
        address _l2Token,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    )
        external
        payable;
    function withdrawTo(
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    )
        external
        payable;
    function l1TokenBridge() external view returns (address);
}
