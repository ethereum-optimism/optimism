// Copyright 2024, 2025 RISC Zero, Inc.
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
pragma solidity 0.8.24;

import "./IKailuaTreasury.sol";
import "src/dispute/lib/Types.sol";

/// @notice Denotes the proven status of the game
/// @custom:value NONE indicates that no proof has been submitted yet.
enum ProofStatus {
    NONE,
    FAULT,
    VALIDITY
}

/// @notice Emitted when a proof is submitted.
/// @param signature The proposal signature
/// @param status The proven status
event Proven(bytes32 indexed signature, ProofStatus indexed status);

interface IKailuaTournament {
    /// @notice Returns the KailuaTreasury of this tournament
    function KAILUA_TREASURY() external view returns (IKailuaTreasury);
    /// @notice The timestamp of when the first proof for a proposal signature was made
    function provenAt(bytes32) external view returns (Timestamp);
    /// @notice Returns the hash of the output claim and all blob hashes associated with this proposal
    function signature() external view returns (bytes32);
    /// @notice Returns whether a child can be considered valid
    function isViableSignature(bytes32 childSignature) external view returns (bool);
    /// @notice Returns the signature of the child proven valid
    function validChildSignature() external view returns (bytes32);
}
