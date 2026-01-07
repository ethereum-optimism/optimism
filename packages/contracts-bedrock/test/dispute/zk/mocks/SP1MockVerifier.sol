// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "src/dispute/zk/ISP1Verifier.sol";

contract SP1MockVerifier is ISP1Verifier {
    /// @notice Verifies a mock proof with given public values and vkey.
    /// @param proofBytes The proof of the program execution the SP1 zkVM encoded as bytes.
    function verifyProof(bytes32, bytes calldata, bytes calldata proofBytes) external pure {
        assert(proofBytes.length == 0);
    }
}
