// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @title ISP1Adapter
/// @notice Interface for the SP1Adapter contract.
interface ISP1Adapter is IZKVerifier {
    /// @notice Returns the address of the underlying SP1 verifier.
    function sp1Verifier() external view returns (ISP1Verifier sp1Verifier_);

    /// @notice Constructor.
    ///
    /// @param _sp1Verifier The SP1 verifier contract.
    function __constructor__(ISP1Verifier _sp1Verifier) external;
}
