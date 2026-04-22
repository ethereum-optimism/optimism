// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @title ISP1PlonkAdapter
/// @notice Interface for the SP1PlonkAdapter contract.
interface ISP1PlonkAdapter is IZKVerifier {
    /// @notice Thrown when the provided verifier address has no code.
    error SP1PlonkAdapter_InvalidVerifier();

    /// @notice Returns the address of the underlying SP1 verifier.
    function sp1Verifier() external view returns (ISP1Verifier sp1Verifier_);

    /// @notice Constructor.
    ///
    /// @param _sp1Verifier The SP1 verifier contract.
    function __constructor__(ISP1Verifier _sp1Verifier) external;
}
