// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { IKailuaTournament } from "interfaces/dispute/zk/IKailuaTournament.sol";
import { IKailuaTreasury } from "interfaces/dispute/zk/IKailuaTreasury.sol";
import { IRiscZeroVerifier } from "interfaces/dispute/zk/IRiscZeroVerifier.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Duration } from "src/dispute/lib/Types.sol";
import { KailuaLib } from "src/dispute/zk/KailuaLib.sol";
import {
    ClockNotExpired,
    IncorrectBondAmount,
    NoCreditToClaim,
    AlreadyEliminated,
    NotProven,
    ProvenFaulty,
    BadTarget
} from "src/dispute/lib/Errors.sol";

contract KailuaVerifier is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice The RISC Zero verifier contract
    IRiscZeroVerifier public immutable RISC_ZERO_VERIFIER;

    /// @notice The RISC Zero image id of the fault proof program
    bytes32 public immutable FPVM_IMAGE_ID;

    /// @notice The hash of the game configuration
    bytes32 public immutable ROLLUP_CONFIG_HASH;

    /// @notice The duration after which a permit expires
    Duration public immutable PERMIT_DURATION;

    constructor(IRiscZeroVerifier _verifierContract, bytes32 _imageId, bytes32 _configHash, Duration _permitDuration) {
        RISC_ZERO_VERIFIER = _verifierContract;
        FPVM_IMAGE_ID = _imageId;
        ROLLUP_CONFIG_HASH = _configHash;
        PERMIT_DURATION = _permitDuration;
    }

    /// @notice Maps parent-child to their fault proving permits
    mapping(bytes32 => FaultProofPermit[]) public faultProofPermits;

    /// @notice Describes a permit for fault proving
    /// @custom:field recipient             Address of the permit recipient
    /// @custom:field aggregateCollateral   Total collateral locked as of permit
    /// @custom:field timestamp             Timestamp of permit issuance
    /// @custom:field released              Flag for whether the collateral locked for this permit
    struct FaultProofPermit {
        uint256 aggregateCollateral;
        address recipient;
        uint64 timestamp;
        bool released;
    }

    /// @notice Returns the key for indexing fault proving permits
    function faultProofPermitKey(
        IKailuaTournament proposalParent,
        bytes32 proposalSignature
    )
        public
        pure
        returns (bytes32)
    {
        return sha256(abi.encodePacked(address(proposalParent), proposalSignature));
    }

    /// @notice Returns the earliest timestamp at which a fault proof permit can be released
    function faultProofPermitProvenAt(
        IKailuaTournament proposalParent,
        bytes32 proposalSignature
    )
        public
        view
        returns (uint64)
    {
        // INVARIANT: A validity proof for the same signature does not satisfy a fault proof permit.
        bytes32 validChildSignature = proposalParent.validChildSignature();
        if (proposalSignature == validChildSignature) {
            return 0;
        }
        // Fetch both fault and validity proof timestamps
        uint64 faultProofTimestamp = proposalParent.provenAt(proposalSignature).raw();
        uint64 validityProofTimestamp = proposalParent.provenAt(validChildSignature).raw();
        // Return the smaller timestamp if both proofs are present
        if (faultProofTimestamp > 0 && validityProofTimestamp > 0) {
            return faultProofTimestamp < validityProofTimestamp ? faultProofTimestamp : validityProofTimestamp;
        }
        // Return the larger timestamp otherwise
        return faultProofTimestamp > validityProofTimestamp ? faultProofTimestamp : validityProofTimestamp;
    }

    /// @notice Returns the exclusive beneficiary of a fault proof reward
    function faultProofPermitBeneficiary(
        IKailuaTournament proposalParent,
        bytes32 proposalSignature
    )
        public
        view
        returns (address)
    {
        // If the signature is still viable, there is no sole fault proof beneficiary
        if (proposalParent.isViableSignature(proposalSignature)) {
            return address(0x0);
        }
        // If there wasn't exactly one permit, then proving was not exclusive to one party
        FaultProofPermit[] storage proposalPermits =
            faultProofPermits[faultProofPermitKey(proposalParent, proposalSignature)];
        if (proposalPermits.length != 1) {
            return address(0x0);
        }
        // If there was no proof or the permit was expired as of proof submission, disqualify the beneficiary
        uint64 provingTime = faultProofPermitProvenAt(proposalParent, proposalSignature);
        if (provingTime == 0 || proposalPermits[0].timestamp + PERMIT_DURATION.raw() < provingTime) {
            return address(0x0);
        }
        // Return the successful sole beneficiary of the locked fault proof reward
        return proposalPermits[0].recipient;
    }

    /// @notice Given a reference timestamp, returns the number of expired permits, their total collateral, and the
    /// number of active permits
    function countExpiredPermits(
        bytes32 proposalKey,
        uint64 numExpiredPermits,
        uint64 timestamp
    )
        public
        view
        returns (uint64, uint256, uint64)
    {
        FaultProofPermit[] storage proposalPermits = faultProofPermits[proposalKey];
        uint256 expiredCollateral = 0;
        uint64 totalPermits = uint64(proposalPermits.length);
        // Increment numExpiredPermits if possible
        for (; numExpiredPermits < totalPermits; numExpiredPermits++) {
            if (proposalPermits[numExpiredPermits].timestamp + PERMIT_DURATION.raw() >= timestamp) {
                break;
            }
        }
        // Validate expiry
        if (numExpiredPermits > 0) {
            // If numExpiredPermits is invalid, revert
            if (proposalPermits[numExpiredPermits - 1].timestamp + PERMIT_DURATION.raw() >= timestamp) {
                revert BadTarget();
            }
            // Set expired collateral
            expiredCollateral = proposalPermits[numExpiredPermits - 1].aggregateCollateral;
        }
        return (numExpiredPermits, expiredCollateral, totalPermits - numExpiredPermits);
    }

    /// @notice Returns the collateral required to acquire a fault proof permit
    function faultProofPermitBond(IKailuaTreasury treasury) public view returns (uint256 bond) {
        bond = (treasury.participationBond() * 2 * treasury.ELIMINATION_SPLIT_PROVER_NUM())
            / treasury.ELIMINATION_SPLIT_DENOM();
    }

    /// @notice Locks the right to submit a fault proof for a given proposal signature
    /// @dev Do not call this function to acquire locks for faults that will not lead to elimination.
    function acquireFaultProofPermit(
        IKailuaTournament proposalParent,
        bytes32 proposalSignature,
        uint64 numExpiredPermits,
        address payoutRecipient
    )
        external
        payable
        returns (uint256 totalPermitsIssued_)
    {
        // INVARIANT: The child signature is still viable so no proof is submitted for/against it
        if (!proposalParent.isViableSignature(proposalSignature)) {
            revert ProvenFaulty();
        }
        // INVARIANT: The collateral submitted for the permit covers two times the proving reward
        IKailuaTreasury treasury = IKailuaTreasury(address(proposalParent.KAILUA_TREASURY()));
        if (msg.value < faultProofPermitBond(treasury)) {
            revert IncorrectBondAmount();
        }
        // INVARIANT: There are exactly numExpiredPermits expired permits as of block.timestamp
        bytes32 proposalKey = faultProofPermitKey(proposalParent, proposalSignature);
        (numExpiredPermits,,) = countExpiredPermits(proposalKey, numExpiredPermits, uint64(block.timestamp));
        // INVARIANT: There is at least one permit available
        FaultProofPermit[] storage proposalPermits = faultProofPermits[proposalKey];
        totalPermitsIssued_ = proposalPermits.length;
        if (totalPermitsIssued_ > 2 * numExpiredPermits) {
            revert ClockNotExpired();
        }
        // Calculate the aggregate collateral value
        uint256 aggregateCollateral = msg.value;
        if (totalPermitsIssued_ > 0) {
            aggregateCollateral += proposalPermits[totalPermitsIssued_ - 1].aggregateCollateral;
        }
        // Assign a new permit
        proposalPermits.push(
            FaultProofPermit({
                aggregateCollateral: aggregateCollateral,
                recipient: payoutRecipient,
                timestamp: uint64(block.timestamp),
                released: false
            })
        );
    }

    /// @notice Claims the total payout for a permit
    function releaseFaultProofPermit(
        IKailuaTournament proposalParent,
        bytes32 proposalSignature,
        uint64 numExpiredPermits,
        uint64 permitIndex
    )
        external
    {
        // INVARIANT: The child signature is proven faulty
        if (proposalParent.isViableSignature(proposalSignature)) {
            revert NotProven();
        }
        // INVARIANT: There are exactly numExpiredPermits expired permits as of proof submission
        uint64 proofTimestamp = faultProofPermitProvenAt(proposalParent, proposalSignature);
        bytes32 permitKey = faultProofPermitKey(proposalParent, proposalSignature);
        (, uint256 expiredCollateral, uint64 numActivePermits) =
            countExpiredPermits(permitKey, numExpiredPermits, proofTimestamp);
        // INVARIANT: The permit is not already released
        FaultProofPermit storage permit = faultProofPermits[permitKey][permitIndex];
        if (permit.released) {
            revert NoCreditToClaim();
        }
        // INVARIANT: The permit is not expired as of proof submission
        if (permit.timestamp + PERMIT_DURATION.raw() < proofTimestamp) {
            revert AlreadyEliminated();
        }
        // Calculate total payout
        uint256 payout = expiredCollateral / numActivePermits;
        // Add in recipient's own deposited collateral
        if (permitIndex > 0) {
            payout += permit.aggregateCollateral - faultProofPermits[permitKey][permitIndex - 1].aggregateCollateral;
        } else {
            payout += permit.aggregateCollateral;
        }
        // Pay out recipient
        permit.released = true;
        KailuaLib.pay(payout, payable(permit.recipient));
    }

    /// @notice Verifies a ZK proof
    function verify(
        address payoutRecipient,
        bytes32 preconditionHash,
        bytes32 l1Head,
        bytes32 agreedL2OutputRoot,
        bytes32 claimedL2OutputRoot,
        uint64 claimedL2BlockNumber,
        bytes calldata encodedSeal
    )
        external
        view
    {
        // Construct the expected journal
        bytes memory journal = abi.encodePacked(
            // The address of the recipient of the payout for this proof
            payoutRecipient,
            // The blob equivalence precondition hash
            preconditionHash,
            // The L1 head hash containing the safe L2 chain data that may reproduce the L2 head hash.
            l1Head,
            // The accepted output
            agreedL2OutputRoot,
            // The proposed output
            claimedL2OutputRoot,
            // The claim block number
            claimedL2BlockNumber,
            // The rollup configuration hash
            ROLLUP_CONFIG_HASH,
            // The FPVM Image ID
            FPVM_IMAGE_ID
        );

        // Revert on proof verification failure
        RISC_ZERO_VERIFIER.verify(encodedSeal, FPVM_IMAGE_ID, sha256(journal));
    }
}
