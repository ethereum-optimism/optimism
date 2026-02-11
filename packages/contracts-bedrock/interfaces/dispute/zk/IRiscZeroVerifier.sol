// Copyright 2025 RISC Zero, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

pragma solidity ^0.8.9;

/// @notice A receipt attesting to a claim using the RISC Zero proof system.
/// @dev A receipt contains two parts: a seal and a claim.
///
/// The seal is a zero-knowledge proof attesting to knowledge of a witness for the claim. The claim
/// is a set of public outputs, and for zkVM execution is the hash of a `ReceiptClaim` struct.
///
/// IMPORTANT: The `claimDigest` field must be a hash computed by the caller for verification to
/// have meaningful guarantees. Treat this similar to verifying an ECDSA signature, in that hashing
/// is a key operation in verification. The most common way to calculate this hash is to use the
/// `ReceiptClaimLib.ok(imageId, journalDigest).digest()` for successful executions.
struct Receipt {
    bytes seal;
    bytes32 claimDigest;
}

/// @notice Verifier interface for RISC Zero receipts of execution.
interface IRiscZeroVerifier {
    /// @notice Verify that the given seal is a valid RISC Zero proof of execution with the
    ///     given image ID and journal digest. Reverts on failure.
    /// @dev This method additionally ensures that the input hash is all-zeros (i.e. no
    /// committed input), the exit code is (Halted, 0), and there are no assumptions (i.e. the
    /// receipt is unconditional).
    /// @param seal The encoded cryptographic proof (i.e. SNARK).
    /// @param imageId The identifier for the guest program.
    /// @param journalDigest The SHA-256 digest of the journal bytes.
    function verify(bytes calldata seal, bytes32 imageId, bytes32 journalDigest) external view;

    /// @notice Verify that the given receipt is a valid RISC Zero receipt, ensuring the `seal` is
    /// valid a cryptographic proof of the execution with the given `claim`. Reverts on failure.
    /// @param receipt The receipt to be verified.
    function verifyIntegrity(Receipt calldata receipt) external view;
}
