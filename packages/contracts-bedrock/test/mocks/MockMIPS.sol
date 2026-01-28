// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title MockMIPS
/// @dev Mock implementation of an MIPS.
contract MockMIPS {
    /// @notice Address of the oracle.
    address public immutable oracleAddr;

    constructor(address _oracle) {
        oracleAddr = _oracle;
    }

    /// @notice Returns the address of the oracle.
    function oracle() external view returns (address) {
        return oracleAddr;
    }
}
