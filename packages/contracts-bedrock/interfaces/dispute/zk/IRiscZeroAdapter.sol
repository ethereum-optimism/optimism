// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { IRiscZeroVerifier } from "interfaces/vendor/IRiscZeroVerifier.sol";

/// @title IRiscZeroAdapter
/// @notice Interface for the RiscZeroAdapter contract.
interface IRiscZeroAdapter is IZKVerifier {
    /// @notice Returns the address of the underlying RISC Zero verifier.
    function riscZeroVerifier() external view returns (IRiscZeroVerifier riscZeroVerifier_);

    /// @notice Constructor.
    ///
    /// @param _riscZeroVerifier The RISC Zero verifier contract.
    function __constructor__(IRiscZeroVerifier _riscZeroVerifier) external;
}
