// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import { IRiscZeroVerifier } from "interfaces/dispute/zk/IRiscZeroVerifier.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import {
    AlreadyEliminated,
    BadTarget,
    BondTransferFailed,
    ClockNotExpired,
    IncorrectBondAmount,
    NoCreditToClaim,
    NotProven,
    ProvenFaulty
} from "src/dispute/lib/Errors.sol";
import { Duration } from "src/dispute/lib/Types.sol";

interface IKailuaVerifier is ISemver {
    function FPVM_IMAGE_ID() external view returns (bytes32);
    function PERMIT_DURATION() external view returns (Duration);
    function RISC_ZERO_VERIFIER() external view returns (IRiscZeroVerifier);
    function ROLLUP_CONFIG_HASH() external view returns (bytes32);
    function acquireFaultProofPermit(
        address proposalParent,
        bytes32 proposalSignature,
        uint64 numExpiredPermits,
        address payoutRecipient
    )
        external
        payable
        returns (uint256 totalPermitsIssued_);
    function countExpiredPermits(
        bytes32 proposalKey,
        uint64 numExpiredPermits,
        uint64 timestamp
    )
        external
        view
        returns (uint64, uint256, uint64);
    function faultProofPermitBeneficiary(
        address proposalParent,
        bytes32 proposalSignature
    )
        external
        view
        returns (address);
    function faultProofPermitBond(address treasury) external view returns (uint256 bond);
    function faultProofPermitKey(address proposalParent, bytes32 proposalSignature) external pure returns (bytes32);
    function faultProofPermitProvenAt(address proposalParent, bytes32 proposalSignature) external view returns (uint64);
    function faultProofPermits(
        bytes32,
        uint256
    )
        external
        view
        returns (uint256 aggregateCollateral, address recipient, uint64 timestamp, bool released);
    function releaseFaultProofPermit(
        address proposalParent,
        bytes32 proposalSignature,
        uint64 numExpiredPermits,
        uint64 permitIndex
    )
        external;
    function verify(
        address payoutRecipient,
        bytes32 preconditionHash,
        bytes32 l1Head,
        bytes32 agreedL2OutputRoot,
        bytes32 claimedL2OutputRoot,
        uint64 claimedL2BlockNumber,
        bytes memory encodedSeal
    )
        external
        view;
    function version() external view returns (string memory);
}
