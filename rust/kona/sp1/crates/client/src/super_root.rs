//! Public IO and encoding helpers for SP1 super-root interop programs.

use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_sol_types::{SolValue, sol};
use serde::{Deserialize, Serialize};

pub use kona_interop::{OutputRootWithChain as SuperOutputRoot, SUPER_ROOT_VERSION, SuperRoot};

/// A decoded super-root proof preimage.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperRootProof {
    /// Encoding version. Must be [`SUPER_ROOT_VERSION`].
    pub version: u8,
    /// Superchain snapshot committed by this proof.
    pub super_root: SuperRoot,
}

impl SuperRootProof {
    /// Creates a versioned super-root proof.
    pub fn new(timestamp: u64, output_roots: Vec<SuperOutputRoot>) -> Self {
        Self { version: SUPER_ROOT_VERSION, super_root: SuperRoot::new(timestamp, output_roots) }
    }

    /// Validates the shape required by the Solidity encoder and aggregation coverage checks.
    pub fn validate(&self) -> Result<(), SuperRootError> {
        if self.version != SUPER_ROOT_VERSION {
            return Err(SuperRootError::InvalidVersion(self.version));
        }
        ensure_strictly_increasing_chains(&self.super_root.output_roots)
    }
}

/// Timestamp span for a super-range proof.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct TimestampSpan {
    /// First timestamp covered by the span.
    pub start: u64,
    /// Last timestamp covered by the span.
    pub end: u64,
}

impl TimestampSpan {
    /// Creates an inclusive timestamp span.
    pub const fn new(start: u64, end: u64) -> Result<Self, SuperRootError> {
        if start > end {
            return Err(SuperRootError::InvalidTimestampSpan { start, end });
        }
        Ok(Self { start, end })
    }

    /// Returns true when `timestamp` is inside the inclusive span.
    pub const fn contains(&self, timestamp: u64) -> bool {
        self.start <= timestamp && timestamp <= self.end
    }

    /// Validates that the inclusive span does not go backwards.
    pub const fn validate(&self) -> Result<(), SuperRootError> {
        if self.start > self.end {
            return Err(SuperRootError::InvalidTimestampSpan { start: self.start, end: self.end });
        }
        Ok(())
    }
}

/// Per-chain optimistic block commitment produced by a super-range proof.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperOptimisticBlock {
    /// The L2 chain ID.
    pub chain_id: U256,
    /// The local-safe block hash that backs the optimistic output root.
    pub block_hash: B256,
    /// The optimistic output root produced by the range program.
    pub output_root: B256,
}

/// Optimistic transition output for one `(timestamp, chain_id)` tuple.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperRangeTransition {
    /// Superchain timestamp covered by this tuple.
    pub timestamp: u64,
    /// Per-chain optimistic block commitment at `timestamp`.
    pub optimistic_block: SuperOptimisticBlock,
}

/// Inputs to a super-range proof over one or more chains and timestamps.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperRangeInputs {
    /// Inclusive timestamp span covered by the range.
    pub span: TimestampSpan,
    /// L1 head used to derive this range.
    pub l1_head: B256,
    /// Chain IDs covered by every timestamp in the range.
    pub chain_ids: Vec<U256>,
    /// Previous super-root proof for every timestamp in the span.
    ///
    /// Entry `i` must be the super-root proof for `(span.start + i) - 1`.
    pub previous_super_root_proofs: Vec<SuperRootProof>,
    /// Claimed optimistic transition outputs for every `(timestamp, chain_id)` tuple in the range.
    pub claimed_transitions: Vec<SuperRangeTransition>,
}

impl SuperRangeInputs {
    /// Validates range shape before execution.
    pub fn validate(&self) -> Result<(), SuperRootError> {
        self.span.validate()?;
        ensure_strictly_increasing_chain_ids(&self.chain_ids)?;
        ensure_range_previous_super_root_coverage(
            &self.span,
            &self.chain_ids,
            &self.previous_super_root_proofs,
        )?;
        ensure_range_transition_coverage(&self.span, &self.chain_ids, &self.claimed_transitions)
    }
}

/// Public output committed by the super-range program.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperRangeOutputs {
    /// Inclusive timestamp span proven by the range.
    pub span: TimestampSpan,
    /// L1 head used to derive this range.
    pub l1_head: B256,
    /// Previous super-root hash used for every timestamp in the span.
    ///
    /// Entry `i` is the super-root hash used to derive `span.start + i`.
    pub previous_super_roots: Vec<B256>,
    /// Optimistic transition outputs proven by the range.
    pub transitions: Vec<SuperRangeTransition>,
}

/// One timestamp of inputs to a span-shaped consolidation proof.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperConsolidationTransitionInput {
    /// Per-chain optimistic range outputs being consolidated at this timestamp.
    pub optimistic_blocks: Vec<SuperOptimisticBlock>,
    /// Claimed post-consolidation super-root proof for this timestamp.
    pub claimed_super_root_proof: SuperRootProof,
}

/// Inputs to the consolidation mode over one or more timestamps.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperConsolidationInputs {
    /// Inclusive timestamp span covered by the consolidation proof.
    pub span: TimestampSpan,
    /// Super root immediately before `span.start`.
    pub previous_super_root: B256,
    /// Ordered consolidation inputs for every timestamp in the span.
    pub transitions: Vec<SuperConsolidationTransitionInput>,
}

impl SuperConsolidationInputs {
    /// Validates consolidation shape before execution.
    pub fn validate(&self) -> Result<(), SuperRootError> {
        self.span.validate()?;
        ensure_consolidation_input_coverage(&self.span, &self.transitions)
    }
}

/// Consolidated output for one timestamp inside the consolidation mode.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperConsolidationTransition {
    /// Timestamp consolidated.
    pub timestamp: u64,
    /// Per-chain optimistic range outputs that were consolidated.
    pub optimistic_blocks: Vec<SuperOptimisticBlock>,
    /// Consolidated super root after this timestamp.
    pub super_root: B256,
}

/// Public output committed by the consolidation mode.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperConsolidationOutputs {
    /// Inclusive timestamp span proven by the consolidation program.
    pub span: TimestampSpan,
    /// Super root immediately before the span.
    pub previous_super_root: B256,
    /// Consolidated outputs for every timestamp in the span.
    pub transitions: Vec<SuperConsolidationTransition>,
}

/// Inputs to the unified super-root range program.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum SuperInteropInputs {
    /// Run the range mode.
    Range(SuperRangeInputs),
    /// Run the consolidation mode.
    Consolidation(SuperConsolidationInputs),
}

impl SuperInteropInputs {
    /// Validates the selected range program mode before execution.
    pub fn validate(&self) -> Result<(), SuperRootError> {
        match self {
            Self::Range(inputs) => inputs.validate(),
            Self::Consolidation(inputs) => inputs.validate(),
        }
    }
}

/// Outputs committed by the unified super-root range program.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum SuperInteropOutputs {
    /// Public output from range mode.
    Range(SuperRangeOutputs),
    /// Public output from consolidation mode.
    Consolidation(SuperConsolidationOutputs),
}

/// Inputs to the final super-aggregation proof.
///
/// Range program proofs are supplied separately through SP1 stdin with `write_proof`, in the same
/// order as these public range program outputs: first all range-mode proofs, then all
/// consolidation-mode proofs.
/// This typed input carries the range-program public outputs and vkey that the aggregation program
/// uses to verify those proof-stream entries.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperAggregationInputs {
    /// L1 head committed to by `ZKDisputeGame`.
    pub l1_head: B256,
    /// Starting super-root hash from the parent game or anchor state.
    pub starting_root_hash: B256,
    /// Starting super-root proof preimage.
    pub starting_super_root_proof: SuperRootProof,
    /// Claimed final super root.
    pub root_claim: B256,
    /// Claimed final timestamp.
    pub l2_sequence_number: u64,
    /// Prover address that must match `msg.sender` in `ZKDisputeGame.prove`.
    pub prover: Address,
    /// Public outputs from verified range-mode proofs, ordered to match the proof stream.
    pub range_outputs: Vec<SuperRangeOutputs>,
    /// Public outputs from verified consolidation-mode proofs, ordered to match the proof
    /// stream after the range-mode proofs.
    pub consolidation_outputs: Vec<SuperConsolidationOutputs>,
    /// Interim dynamically supplied range program verification key.
    ///
    /// This is development-only until the range vkey is embedded in the aggregation program
    /// See <https://github.com/ethereum-optimism/optimism/issues/21412/>
    pub range_vkey: [u32; 8],
}

sol! {
    /// Public values consumed by `ZKDisputeGame.prove`.
    #[derive(Debug, Serialize, Deserialize)]
    struct SuperAggregationPublicValues {
        bytes32 l1Head;
        bytes32 startingRootHash;
        bytes32 rootClaim;
        uint256 l2SequenceNumber;
        address prover;
    }
}

impl SuperAggregationInputs {
    /// Validates aggregation-public IO shape before recursive proof verification.
    pub fn validate(&self) -> Result<(), SuperRootError> {
        let starting_super_root = hash_super_root_proof(&self.starting_super_root_proof)?;
        if starting_super_root != self.starting_root_hash {
            return Err(SuperRootError::StartingSuperRootMismatch {
                expected: self.starting_root_hash,
                actual: starting_super_root,
            });
        }

        if self.range_outputs.is_empty() {
            return Err(SuperRootError::EmptyRangeOutputs);
        }
        if self.consolidation_outputs.is_empty() {
            return Err(SuperRootError::EmptyConsolidationOutputs);
        }

        let first_timestamp =
            self.starting_super_root_proof.super_root.timestamp.checked_add(1).ok_or(
                SuperRootError::AggregationTimestampOverflow {
                    timestamp: self.starting_super_root_proof.super_root.timestamp,
                },
            )?;
        if first_timestamp > self.l2_sequence_number {
            return Err(SuperRootError::InvalidTimestampSpan {
                start: first_timestamp,
                end: self.l2_sequence_number,
            });
        }

        // TODO(#21700): Bind the configured chain universe and timestamp-indexed activation
        // schedule, including an authenticated pre-activation output root for newly activated
        // chains.
        let chain_ids = self
            .starting_super_root_proof
            .super_root
            .output_roots
            .iter()
            .map(|output_root| U256::from(output_root.chain_id))
            .collect::<Vec<_>>();

        self.validate_consolidation_outputs(first_timestamp, &chain_ids)?;
        self.validate_range_outputs(first_timestamp, &chain_ids)
    }

    /// Converts aggregation inputs to the exact public-values tuple verified by `ZKDisputeGame`.
    pub fn public_values(&self) -> SuperAggregationPublicValues {
        SuperAggregationPublicValues {
            l1Head: self.l1_head,
            startingRootHash: self.starting_root_hash,
            rootClaim: self.root_claim,
            l2SequenceNumber: U256::from(self.l2_sequence_number),
            prover: self.prover,
        }
    }

    /// ABI-encodes the public values as `ZKDisputeGame.prove` constructs them.
    pub fn zk_dispute_game_public_values(&self) -> Vec<u8> {
        self.public_values().abi_encode()
    }

    fn validate_consolidation_outputs(
        &self,
        first_timestamp: u64,
        chain_ids: &[U256],
    ) -> Result<(), SuperRootError> {
        let mut expected_start = first_timestamp;
        let mut expected_previous_super_root = self.starting_root_hash;
        for output in &self.consolidation_outputs {
            output.span.validate()?;
            if output.span.start != expected_start {
                return Err(SuperRootError::ConsolidationSpanStartMismatch {
                    expected: expected_start,
                    actual: output.span.start,
                });
            }
            if output.span.end > self.l2_sequence_number {
                return Err(SuperRootError::ConsolidationSpanEndExceedsClaim {
                    end: output.span.end,
                    claim: self.l2_sequence_number,
                });
            }
            if output.previous_super_root != expected_previous_super_root {
                return Err(SuperRootError::PreviousSuperRootMismatch {
                    expected: expected_previous_super_root,
                    actual: output.previous_super_root,
                });
            }

            ensure_consolidation_transition_coverage(&output.span, chain_ids, &output.transitions)?;
            for transition in &output.transitions {
                expected_previous_super_root = transition.super_root;
            }

            expected_start = output.span.end.checked_add(1).ok_or(
                SuperRootError::AggregationTimestampOverflow { timestamp: output.span.end },
            )?;
        }

        let expected_after_final = self.l2_sequence_number.saturating_add(1);
        if expected_start != expected_after_final {
            return Err(SuperRootError::ConsolidationSpanStartMismatch {
                expected: expected_after_final,
                actual: expected_start,
            });
        }

        if expected_previous_super_root != self.root_claim {
            return Err(SuperRootError::RootClaimMismatch {
                expected: self.root_claim,
                actual: expected_previous_super_root,
            });
        }

        Ok(())
    }

    fn validate_range_outputs(
        &self,
        first_timestamp: u64,
        chain_ids: &[U256],
    ) -> Result<(), SuperRootError> {
        let mut expected_start = first_timestamp;
        for output in &self.range_outputs {
            if output.l1_head != self.l1_head {
                return Err(SuperRootError::RangeL1HeadMismatch {
                    expected: self.l1_head,
                    actual: output.l1_head,
                });
            }
            if output.span.start != expected_start {
                return Err(SuperRootError::RangeSpanStartMismatch {
                    expected: expected_start,
                    actual: output.span.start,
                });
            }
            if output.span.end > self.l2_sequence_number {
                return Err(SuperRootError::RangeSpanEndExceedsClaim {
                    end: output.span.end,
                    claim: self.l2_sequence_number,
                });
            }

            ensure_range_previous_super_root_outputs(&output.span, &output.previous_super_roots)?;
            ensure_range_transition_coverage(&output.span, chain_ids, &output.transitions)?;
            for (timestamp_offset, timestamp) in (output.span.start..=output.span.end).enumerate() {
                let expected_previous_super_root =
                    self.super_root_before_timestamp(timestamp, first_timestamp)?;
                let actual_previous_super_root = output.previous_super_roots[timestamp_offset];
                if actual_previous_super_root != expected_previous_super_root {
                    return Err(SuperRootError::PreviousSuperRootMismatch {
                        expected: expected_previous_super_root,
                        actual: actual_previous_super_root,
                    });
                }

                let transition_start = timestamp_offset * chain_ids.len();
                let transition_end = transition_start + chain_ids.len();
                let range_blocks = output
                    .transitions
                    .get(transition_start..transition_end)
                    .ok_or(SuperRootError::MissingRangeTimestamp { timestamp })?
                    .iter()
                    .map(|transition| transition.optimistic_block)
                    .collect::<Vec<_>>();
                let consolidation_blocks =
                    &self.consolidation_transition_at(timestamp)?.optimistic_blocks;
                if range_blocks != *consolidation_blocks {
                    return Err(SuperRootError::RangeConsolidationMismatch { timestamp });
                }
            }

            expected_start = output.span.end.checked_add(1).ok_or(
                SuperRootError::AggregationTimestampOverflow { timestamp: output.span.end },
            )?;
        }

        let expected_after_final = self.l2_sequence_number.saturating_add(1);
        if expected_start != expected_after_final {
            return Err(SuperRootError::RangeSpanStartMismatch {
                expected: expected_after_final,
                actual: expected_start,
            });
        }

        Ok(())
    }

    fn super_root_before_timestamp(
        &self,
        timestamp: u64,
        first_timestamp: u64,
    ) -> Result<B256, SuperRootError> {
        if timestamp == first_timestamp {
            Ok(self.starting_root_hash)
        } else {
            Ok(self.consolidation_transition_at(timestamp - 1)?.super_root)
        }
    }

    fn consolidation_transition_at(
        &self,
        timestamp: u64,
    ) -> Result<&SuperConsolidationTransition, SuperRootError> {
        for output in &self.consolidation_outputs {
            if output.span.contains(timestamp) {
                let index = (timestamp - output.span.start) as usize;
                return output
                    .transitions
                    .get(index)
                    .ok_or(SuperRootError::MissingConsolidationTimestamp { timestamp });
            }
        }

        Err(SuperRootError::MissingConsolidationTimestamp { timestamp })
    }
}

/// Errors returned by super-root public IO helpers.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SuperRootError {
    /// Super-root encoding version is unsupported.
    InvalidVersion(u8),
    /// Super-root proofs and chain coverage lists must contain at least one chain.
    EmptyOutputRoots,
    /// Chain IDs must be strictly increasing.
    ChainIdsNotStrictlyIncreasing {
        /// Previous chain ID in the ordered list.
        previous: U256,
        /// Current chain ID that failed the ordering check.
        current: U256,
    },
    /// Timestamp spans are inclusive and must not go backwards.
    InvalidTimestampSpan {
        /// First timestamp in the requested span.
        start: u64,
        /// Last timestamp in the requested span.
        end: u64,
    },
    /// Range spans must have a previous timestamp to bind the previous super root.
    InvalidTimestampSpanStart {
        /// First timestamp in the requested span.
        start: u64,
    },
    /// Range transition coverage is too large to represent on this host.
    RangeTransitionCountOverflow {
        /// Number of timestamps in the inclusive span.
        timestamps: u64,
        /// Number of chain IDs covered at each timestamp.
        chains: usize,
    },
    /// Range previous-root proofs must contain one entry per timestamp.
    InvalidRangePreviousRootCount {
        /// Expected number of previous-root proofs or hashes.
        expected: usize,
        /// Actual number supplied.
        actual: usize,
    },
    /// A range previous-root proof appeared at the wrong timestamp.
    RangePreviousRootTimestampMismatch {
        /// Expected previous timestamp at this position.
        expected: u64,
        /// Actual proof timestamp.
        actual: u64,
    },
    /// A range previous-root proof has the wrong chain count.
    MismatchedRangePreviousRootChainCount {
        /// Number of chain IDs expected by the range input.
        expected: usize,
        /// Number of output roots in the previous super-root proof.
        actual: usize,
    },
    /// A range previous-root proof disagrees with the expected chain ordering or coverage.
    MismatchedRangePreviousRootChainId {
        /// Expected chain ID at this position.
        expected: U256,
        /// Actual chain ID in the proof.
        actual: U256,
    },
    /// Range transition outputs must cover every `(timestamp, chain_id)` tuple exactly once.
    InvalidRangeTransitionCount {
        /// Expected number of transition outputs.
        expected: usize,
        /// Actual number of transition outputs.
        actual: usize,
    },
    /// A range transition output appeared at the wrong timestamp.
    RangeTransitionTimestampMismatch {
        /// Expected timestamp at this position.
        expected: u64,
        /// Actual transition timestamp.
        actual: u64,
    },
    /// A range transition output appeared for the wrong chain ID.
    RangeTransitionChainIdMismatch {
        /// Expected chain ID at this position.
        expected: U256,
        /// Actual transition chain ID.
        actual: U256,
    },
    /// Aggregation must include at least one range output.
    EmptyRangeOutputs,
    /// Aggregation must include at least one consolidation output.
    EmptyConsolidationOutputs,
    /// The starting super-root preimage does not match `startingRootHash`.
    StartingSuperRootMismatch {
        /// Starting root hash committed to by the dispute game.
        expected: B256,
        /// Hash of the supplied starting super-root proof.
        actual: B256,
    },
    /// Timestamp arithmetic overflowed while validating aggregation coverage.
    AggregationTimestampOverflow {
        /// Timestamp that could not be advanced.
        timestamp: u64,
    },
    /// A span-shaped consolidation input/output must contain one entry per timestamp.
    InvalidConsolidationInputCount {
        /// Expected number of timestamp entries in the consolidation span.
        expected: usize,
        /// Actual number of timestamp entries supplied.
        actual: usize,
    },
    /// A consolidation output appeared at the wrong timestamp.
    ConsolidationTimestampMismatch {
        /// Expected timestamp at this position.
        expected: u64,
        /// Actual consolidation timestamp.
        actual: u64,
    },
    /// A consolidation output span started at the wrong timestamp.
    ConsolidationSpanStartMismatch {
        /// Expected consolidation span start.
        expected: u64,
        /// Actual consolidation span start.
        actual: u64,
    },
    /// A consolidation output span extends past the claimed final timestamp.
    ConsolidationSpanEndExceedsClaim {
        /// Consolidation span end timestamp.
        end: u64,
        /// Claimed final timestamp.
        claim: u64,
    },
    /// Aggregation could not find a consolidation transition for a timestamp.
    MissingConsolidationTimestamp {
        /// Missing timestamp.
        timestamp: u64,
    },
    /// Aggregation could not find range transitions for a timestamp.
    MissingRangeTimestamp {
        /// Missing timestamp.
        timestamp: u64,
    },
    /// Range proof chaining used the wrong previous super root.
    PreviousSuperRootMismatch {
        /// Expected previous super root.
        expected: B256,
        /// Actual previous super root.
        actual: B256,
    },
    /// Final consolidated root does not match `rootClaim`.
    RootClaimMismatch {
        /// Root claim committed to by the dispute game.
        expected: B256,
        /// Root produced by the final consolidation output.
        actual: B256,
    },
    /// A range output started at the wrong timestamp.
    RangeSpanStartMismatch {
        /// Expected range start.
        expected: u64,
        /// Actual range start.
        actual: u64,
    },
    /// A range output extends past the claimed final timestamp.
    RangeSpanEndExceedsClaim {
        /// Range end timestamp.
        end: u64,
        /// Claimed final timestamp.
        claim: u64,
    },
    /// A range transition output does not match the corresponding consolidation input.
    RangeConsolidationMismatch {
        /// Timestamp where range and consolidation outputs disagree.
        timestamp: u64,
    },
    /// A range proof used a different L1 head than the dispute game.
    RangeL1HeadMismatch {
        /// L1 head committed to by the dispute game.
        expected: B256,
        /// L1 head committed by the range proof.
        actual: B256,
    },
    /// Consolidation inputs and claimed super-root proof cover different chain counts.
    MismatchedConsolidationChainCount {
        /// Number of optimistic blocks supplied to consolidation.
        optimistic_count: usize,
        /// Number of output roots in the claimed super-root proof.
        claimed_count: usize,
    },
    /// Consolidation inputs and claimed super-root proof disagree on chain ordering or coverage.
    MismatchedConsolidationChainId {
        /// Chain ID from the optimistic block list.
        optimistic_chain_id: U256,
        /// Chain ID from the claimed super-root proof.
        claimed_chain_id: U256,
    },
    /// Consolidation inputs and claimed super-root proof disagree on output root.
    MismatchedConsolidationOutputRoot {
        /// Chain ID whose output root mismatched.
        chain_id: U256,
        /// Optimistic output root supplied to consolidation.
        optimistic_output_root: B256,
        /// Output root committed by the claimed super-root proof.
        claimed_output_root: B256,
    },
}

impl core::fmt::Display for SuperRootError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::InvalidVersion(version) => {
                write!(f, "invalid super-root version {version}")
            }
            Self::EmptyOutputRoots => f.write_str("super-root proof must contain output roots"),
            Self::ChainIdsNotStrictlyIncreasing { previous, current } => {
                write!(f, "chain IDs must be strictly increasing, got {current} after {previous}")
            }
            Self::InvalidTimestampSpan { start, end } => {
                write!(f, "invalid timestamp span {start}..={end}")
            }
            Self::InvalidTimestampSpanStart { start } => {
                write!(f, "range timestamp span start {start} has no previous timestamp")
            }
            Self::RangeTransitionCountOverflow { timestamps, chains } => write!(
                f,
                "range transition coverage is too large: {timestamps} timestamps across {chains} chains"
            ),
            Self::InvalidRangePreviousRootCount { expected, actual } => {
                write!(f, "range must contain {expected} previous roots, got {actual}")
            }
            Self::RangePreviousRootTimestampMismatch { expected, actual } => {
                write!(
                    f,
                    "range previous-root timestamp {actual} does not match expected {expected}"
                )
            }
            Self::MismatchedRangePreviousRootChainCount { expected, actual } => {
                write!(f, "range previous-root proof covers {actual} chains, expected {expected}")
            }
            Self::MismatchedRangePreviousRootChainId { expected, actual } => write!(
                f,
                "range previous-root chain ID {actual} does not match expected {expected}"
            ),
            Self::InvalidRangeTransitionCount { expected, actual } => {
                write!(f, "range must contain {expected} transitions, got {actual}")
            }
            Self::RangeTransitionTimestampMismatch { expected, actual } => {
                write!(f, "range transition timestamp {actual} does not match expected {expected}")
            }
            Self::RangeTransitionChainIdMismatch { expected, actual } => {
                write!(f, "range transition chain ID {actual} does not match expected {expected}")
            }
            Self::EmptyRangeOutputs => f.write_str("aggregation must contain range outputs"),
            Self::EmptyConsolidationOutputs => {
                f.write_str("aggregation must contain consolidation outputs")
            }
            Self::StartingSuperRootMismatch { expected, actual } => {
                write!(f, "starting super-root proof hashes to {actual}, expected {expected}")
            }
            Self::AggregationTimestampOverflow { timestamp } => {
                write!(f, "timestamp {timestamp} cannot be advanced")
            }
            Self::InvalidConsolidationInputCount { expected, actual } => {
                write!(f, "consolidation span must contain {expected} inputs, got {actual}")
            }
            Self::ConsolidationTimestampMismatch { expected, actual } => {
                write!(f, "consolidation timestamp {actual} does not match expected {expected}")
            }
            Self::ConsolidationSpanStartMismatch { expected, actual } => {
                write!(f, "consolidation span starts at {actual}, expected {expected}")
            }
            Self::ConsolidationSpanEndExceedsClaim { end, claim } => {
                write!(f, "consolidation span ends at {end}, past claimed final timestamp {claim}")
            }
            Self::MissingConsolidationTimestamp { timestamp } => {
                write!(f, "missing consolidation transition at timestamp {timestamp}")
            }
            Self::MissingRangeTimestamp { timestamp } => {
                write!(f, "missing range transitions at timestamp {timestamp}")
            }
            Self::PreviousSuperRootMismatch { expected, actual } => {
                write!(f, "previous super root {actual} does not match expected {expected}")
            }
            Self::RootClaimMismatch { expected, actual } => {
                write!(f, "final super root {actual} does not match root claim {expected}")
            }
            Self::RangeSpanStartMismatch { expected, actual } => {
                write!(f, "range span starts at {actual}, expected {expected}")
            }
            Self::RangeSpanEndExceedsClaim { end, claim } => {
                write!(f, "range span ends at {end}, past claimed final timestamp {claim}")
            }
            Self::RangeConsolidationMismatch { timestamp } => write!(
                f,
                "range transitions do not match consolidation optimistic blocks at timestamp {timestamp}"
            ),
            Self::RangeL1HeadMismatch { expected, actual } => {
                write!(f, "range proof L1 head {actual} does not match game L1 head {expected}")
            }
            Self::MismatchedConsolidationChainCount { optimistic_count, claimed_count } => write!(
                f,
                "consolidation covers {optimistic_count} optimistic blocks but {claimed_count} claimed output roots"
            ),
            Self::MismatchedConsolidationChainId { optimistic_chain_id, claimed_chain_id } => {
                write!(
                    f,
                    "consolidation optimistic chain ID {optimistic_chain_id} does not match claimed chain ID {claimed_chain_id}"
                )
            }
            Self::MismatchedConsolidationOutputRoot {
                chain_id,
                optimistic_output_root,
                claimed_output_root,
            } => write!(
                f,
                "consolidation optimistic output root {optimistic_output_root} for chain {chain_id} does not match claimed output root {claimed_output_root}"
            ),
        }
    }
}

impl std::error::Error for SuperRootError {}

/// Encodes a super-root proof exactly like `Encoding.encodeSuperRootProof`.
pub fn encode_super_root_proof(proof: &SuperRootProof) -> Result<Vec<u8>, SuperRootError> {
    proof.validate()?;

    let mut encoded = Vec::with_capacity(proof.super_root.encoded_length());
    proof.super_root.encode(&mut encoded);

    Ok(encoded)
}

/// Hashes a super-root proof exactly like `Hashing.hashSuperRootProof`.
pub fn hash_super_root_proof(proof: &SuperRootProof) -> Result<B256, SuperRootError> {
    Ok(B256::from(keccak256(encode_super_root_proof(proof)?)))
}

/// Ensures output roots are non-empty and ordered by strictly increasing chain ID.
pub fn ensure_strictly_increasing_chains(
    output_roots: &[SuperOutputRoot],
) -> Result<(), SuperRootError> {
    if output_roots.is_empty() {
        return Err(SuperRootError::EmptyOutputRoots);
    }

    for pair in output_roots.windows(2) {
        let previous = U256::from(pair[0].chain_id);
        let current = U256::from(pair[1].chain_id);
        if previous >= current {
            return Err(SuperRootError::ChainIdsNotStrictlyIncreasing { previous, current });
        }
    }

    Ok(())
}

fn ensure_strictly_increasing_chain_ids(chain_ids: &[U256]) -> Result<(), SuperRootError> {
    if chain_ids.is_empty() {
        return Err(SuperRootError::EmptyOutputRoots);
    }

    for pair in chain_ids.windows(2) {
        let previous = pair[0];
        let current = pair[1];
        if previous >= current {
            return Err(SuperRootError::ChainIdsNotStrictlyIncreasing { previous, current });
        }
    }

    Ok(())
}

fn ensure_range_transition_coverage(
    span: &TimestampSpan,
    chain_ids: &[U256],
    transitions: &[SuperRangeTransition],
) -> Result<(), SuperRootError> {
    let timestamp_count = inclusive_timestamp_count(span, chain_ids.len())?;
    let expected = usize::try_from(timestamp_count)
        .ok()
        .and_then(|timestamps| timestamps.checked_mul(chain_ids.len()))
        .ok_or(SuperRootError::RangeTransitionCountOverflow {
            timestamps: timestamp_count,
            chains: chain_ids.len(),
        })?;

    if transitions.len() != expected {
        return Err(SuperRootError::InvalidRangeTransitionCount {
            expected,
            actual: transitions.len(),
        });
    }

    let mut index = 0;
    for timestamp in span.start..=span.end {
        for &chain_id in chain_ids {
            let transition = transitions[index];
            if transition.timestamp != timestamp {
                return Err(SuperRootError::RangeTransitionTimestampMismatch {
                    expected: timestamp,
                    actual: transition.timestamp,
                });
            }
            if transition.optimistic_block.chain_id != chain_id {
                return Err(SuperRootError::RangeTransitionChainIdMismatch {
                    expected: chain_id,
                    actual: transition.optimistic_block.chain_id,
                });
            }
            index += 1;
        }
    }

    Ok(())
}

fn ensure_range_previous_super_root_coverage(
    span: &TimestampSpan,
    chain_ids: &[U256],
    previous_super_root_proofs: &[SuperRootProof],
) -> Result<(), SuperRootError> {
    let expected = consolidation_timestamp_count(span)?;
    if previous_super_root_proofs.len() != expected {
        return Err(SuperRootError::InvalidRangePreviousRootCount {
            expected,
            actual: previous_super_root_proofs.len(),
        });
    }

    for (offset, proof) in previous_super_root_proofs.iter().enumerate() {
        let transition_timestamp = span.start + offset as u64;
        let expected_timestamp = transition_timestamp
            .checked_sub(1)
            .ok_or(SuperRootError::InvalidTimestampSpanStart { start: transition_timestamp })?;
        if proof.super_root.timestamp != expected_timestamp {
            return Err(SuperRootError::RangePreviousRootTimestampMismatch {
                expected: expected_timestamp,
                actual: proof.super_root.timestamp,
            });
        }

        proof.validate()?;
        ensure_previous_super_root_chains_match(chain_ids, &proof.super_root.output_roots)?;
    }

    Ok(())
}

fn ensure_range_previous_super_root_outputs(
    span: &TimestampSpan,
    previous_super_roots: &[B256],
) -> Result<(), SuperRootError> {
    let expected = consolidation_timestamp_count(span)?;
    if previous_super_roots.len() != expected {
        return Err(SuperRootError::InvalidRangePreviousRootCount {
            expected,
            actual: previous_super_roots.len(),
        });
    }
    Ok(())
}

fn ensure_previous_super_root_chains_match(
    chain_ids: &[U256],
    output_roots: &[SuperOutputRoot],
) -> Result<(), SuperRootError> {
    if output_roots.len() != chain_ids.len() {
        return Err(SuperRootError::MismatchedRangePreviousRootChainCount {
            expected: chain_ids.len(),
            actual: output_roots.len(),
        });
    }

    for (&expected, output_root) in chain_ids.iter().zip(output_roots) {
        let actual = U256::from(output_root.chain_id);
        if actual != expected {
            return Err(SuperRootError::MismatchedRangePreviousRootChainId { expected, actual });
        }
    }

    Ok(())
}

fn ensure_consolidation_input_coverage(
    span: &TimestampSpan,
    inputs: &[SuperConsolidationTransitionInput],
) -> Result<(), SuperRootError> {
    let expected = consolidation_timestamp_count(span)?;
    if inputs.len() != expected {
        return Err(SuperRootError::InvalidConsolidationInputCount {
            expected,
            actual: inputs.len(),
        });
    }

    let mut chain_ids = Vec::new();
    for (offset, input) in inputs.iter().enumerate() {
        let expected_timestamp = span.start + offset as u64;
        if input.claimed_super_root_proof.super_root.timestamp != expected_timestamp {
            return Err(SuperRootError::ConsolidationTimestampMismatch {
                expected: expected_timestamp,
                actual: input.claimed_super_root_proof.super_root.timestamp,
            });
        }

        input.claimed_super_root_proof.validate()?;
        ensure_strictly_increasing_optimistic_blocks(&input.optimistic_blocks)?;
        ensure_matching_consolidation_chains(
            &input.optimistic_blocks,
            &input.claimed_super_root_proof.super_root.output_roots,
        )?;

        if offset == 0 {
            chain_ids = input
                .optimistic_blocks
                .iter()
                .map(|optimistic_block| optimistic_block.chain_id)
                .collect();
        } else {
            ensure_optimistic_blocks_match_chain_ids(&input.optimistic_blocks, &chain_ids)?;
        }
    }

    Ok(())
}

fn ensure_consolidation_transition_coverage(
    span: &TimestampSpan,
    chain_ids: &[U256],
    transitions: &[SuperConsolidationTransition],
) -> Result<(), SuperRootError> {
    let expected = consolidation_timestamp_count(span)?;
    if transitions.len() != expected {
        return Err(SuperRootError::InvalidConsolidationInputCount {
            expected,
            actual: transitions.len(),
        });
    }

    for (offset, transition) in transitions.iter().enumerate() {
        let expected_timestamp = span.start + offset as u64;
        if transition.timestamp != expected_timestamp {
            return Err(SuperRootError::ConsolidationTimestampMismatch {
                expected: expected_timestamp,
                actual: transition.timestamp,
            });
        }

        ensure_optimistic_blocks_match_chain_ids(&transition.optimistic_blocks, chain_ids)?;
    }

    Ok(())
}

fn consolidation_timestamp_count(span: &TimestampSpan) -> Result<usize, SuperRootError> {
    let timestamp_count = inclusive_timestamp_count(span, 0)?;
    usize::try_from(timestamp_count).map_err(|_| SuperRootError::RangeTransitionCountOverflow {
        timestamps: timestamp_count,
        chains: 0,
    })
}

fn inclusive_timestamp_count(span: &TimestampSpan, chains: usize) -> Result<u64, SuperRootError> {
    span.end
        .checked_sub(span.start)
        .and_then(|distance| distance.checked_add(1))
        .ok_or(SuperRootError::RangeTransitionCountOverflow { timestamps: u64::MAX, chains })
}

fn ensure_optimistic_blocks_match_chain_ids(
    optimistic_blocks: &[SuperOptimisticBlock],
    chain_ids: &[U256],
) -> Result<(), SuperRootError> {
    if optimistic_blocks.len() != chain_ids.len() {
        return Err(SuperRootError::InvalidRangeTransitionCount {
            expected: chain_ids.len(),
            actual: optimistic_blocks.len(),
        });
    }

    for (optimistic_block, &chain_id) in optimistic_blocks.iter().zip(chain_ids) {
        if optimistic_block.chain_id != chain_id {
            return Err(SuperRootError::RangeTransitionChainIdMismatch {
                expected: chain_id,
                actual: optimistic_block.chain_id,
            });
        }
    }

    Ok(())
}

fn ensure_strictly_increasing_optimistic_blocks(
    optimistic_blocks: &[SuperOptimisticBlock],
) -> Result<(), SuperRootError> {
    if optimistic_blocks.is_empty() {
        return Err(SuperRootError::EmptyOutputRoots);
    }

    for pair in optimistic_blocks.windows(2) {
        let previous = pair[0].chain_id;
        let current = pair[1].chain_id;
        if previous >= current {
            return Err(SuperRootError::ChainIdsNotStrictlyIncreasing { previous, current });
        }
    }

    Ok(())
}

fn ensure_matching_consolidation_chains(
    optimistic_blocks: &[SuperOptimisticBlock],
    claimed_output_roots: &[SuperOutputRoot],
) -> Result<(), SuperRootError> {
    if optimistic_blocks.len() != claimed_output_roots.len() {
        return Err(SuperRootError::MismatchedConsolidationChainCount {
            optimistic_count: optimistic_blocks.len(),
            claimed_count: claimed_output_roots.len(),
        });
    }

    for (optimistic_block, claimed_output_root) in
        optimistic_blocks.iter().zip(claimed_output_roots)
    {
        let claimed_chain_id = U256::from(claimed_output_root.chain_id);
        if optimistic_block.chain_id != claimed_chain_id {
            return Err(SuperRootError::MismatchedConsolidationChainId {
                optimistic_chain_id: optimistic_block.chain_id,
                claimed_chain_id,
            });
        }
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{Address, B256, U256, address, b256, hex};

    use super::{
        SuperAggregationInputs, SuperConsolidationInputs, SuperConsolidationOutputs,
        SuperConsolidationTransition, SuperConsolidationTransitionInput, SuperInteropInputs,
        SuperOptimisticBlock, SuperOutputRoot, SuperRangeInputs, SuperRangeOutputs,
        SuperRangeTransition, SuperRootError, SuperRootProof, TimestampSpan,
        encode_super_root_proof, ensure_strictly_increasing_chains, hash_super_root_proof,
    };

    fn output(chain_id: u64, fill: u8) -> SuperOutputRoot {
        SuperOutputRoot { chain_id, output_root: B256::from([fill; 32]) }
    }

    fn optimistic(chain_id: u64, block_fill: u8, output_fill: u8) -> SuperOptimisticBlock {
        SuperOptimisticBlock {
            chain_id: U256::from(chain_id),
            block_hash: B256::from([block_fill; 32]),
            output_root: B256::from([output_fill; 32]),
        }
    }

    fn previous_proof(timestamp: u64) -> SuperRootProof {
        SuperRootProof::new(timestamp, vec![output(10, 0x01), output(20, 0x02)])
    }

    fn transition(
        timestamp: u64,
        chain_id: u64,
        block_fill: u8,
        output_fill: u8,
    ) -> SuperRangeTransition {
        SuperRangeTransition {
            timestamp,
            optimistic_block: optimistic(chain_id, block_fill, output_fill),
        }
    }

    fn consolidation_transition(
        timestamp: u64,
        optimistic_blocks: Vec<SuperOptimisticBlock>,
        super_root_fill: u8,
    ) -> SuperConsolidationTransition {
        SuperConsolidationTransition {
            timestamp,
            optimistic_blocks,
            super_root: B256::from([super_root_fill; 32]),
        }
    }

    fn range_vkey() -> [u32; 8] {
        [1, 2, 3, 4, 5, 6, 7, 8]
    }

    fn valid_aggregation_inputs() -> SuperAggregationInputs {
        let starting_super_root_proof =
            SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]);
        let starting_root_hash =
            hash_super_root_proof(&starting_super_root_proof).expect("starting root hashes");
        let final_super_root = B256::from([0x55; 32]);
        let timestamp_100 = vec![optimistic(10, 0x11, 0x12), optimistic(20, 0x21, 0x22)];
        let timestamp_101 = vec![optimistic(10, 0x31, 0x32), optimistic(20, 0x41, 0x42)];

        SuperAggregationInputs {
            l1_head: B256::from([0x99; 32]),
            starting_root_hash,
            starting_super_root_proof,
            root_claim: final_super_root,
            l2_sequence_number: 101,
            prover: address!("0x1234567890123456789012345678901234567890"),
            range_outputs: vec![
                SuperRangeOutputs {
                    span: TimestampSpan::new(100, 100).expect("valid span"),
                    l1_head: B256::from([0x99; 32]),
                    previous_super_roots: vec![starting_root_hash],
                    transitions: vec![
                        SuperRangeTransition { timestamp: 100, optimistic_block: timestamp_100[0] },
                        SuperRangeTransition { timestamp: 100, optimistic_block: timestamp_100[1] },
                    ],
                },
                SuperRangeOutputs {
                    span: TimestampSpan::new(101, 101).expect("valid span"),
                    l1_head: B256::from([0x99; 32]),
                    previous_super_roots: vec![B256::from([0x44; 32])],
                    transitions: vec![
                        SuperRangeTransition { timestamp: 101, optimistic_block: timestamp_101[0] },
                        SuperRangeTransition { timestamp: 101, optimistic_block: timestamp_101[1] },
                    ],
                },
            ],
            consolidation_outputs: vec![
                SuperConsolidationOutputs {
                    span: TimestampSpan::new(100, 100).expect("valid span"),
                    previous_super_root: starting_root_hash,
                    transitions: vec![consolidation_transition(100, timestamp_100, 0x44)],
                },
                SuperConsolidationOutputs {
                    span: TimestampSpan::new(101, 101).expect("valid span"),
                    previous_super_root: B256::from([0x44; 32]),
                    transitions: vec![consolidation_transition(101, timestamp_101, 0x55)],
                },
            ],
            range_vkey: range_vkey(),
        }
    }

    #[test]
    fn super_root_hash_matches_solidity_encoding() {
        let proof =
            SuperRootProof::new(0x0102_0304_0506_0708, vec![output(10, 0x11), output(20, 0x22)]);

        let encoded = encode_super_root_proof(&proof).expect("proof encodes");
        let expected = hex!(
            "01"
            "0102030405060708"
            "000000000000000000000000000000000000000000000000000000000000000a"
            "1111111111111111111111111111111111111111111111111111111111111111"
            "0000000000000000000000000000000000000000000000000000000000000014"
            "2222222222222222222222222222222222222222222222222222222222222222"
        );
        assert_eq!(encoded, expected);

        let hash = hash_super_root_proof(&proof).expect("proof hashes");
        assert_eq!(
            hash,
            b256!("0x83f2565a4b43b7d2f60fccdb5e05664202ca4dc491174fbeb83a5a5a87144542")
        );
    }

    #[test]
    fn final_public_values_match_zk_dispute_game_abi_encoding() {
        let mut inputs = valid_aggregation_inputs();
        inputs.l1_head = B256::from([0x11; 32]);
        inputs.starting_root_hash = B256::from([0x22; 32]);
        inputs.root_claim = B256::from([0x33; 32]);
        inputs.l2_sequence_number = 0x0102_0304_0506_0708;

        let encoded = inputs.zk_dispute_game_public_values();
        let expected = hex!(
            "1111111111111111111111111111111111111111111111111111111111111111"
            "2222222222222222222222222222222222222222222222222222222222222222"
            "3333333333333333333333333333333333333333333333333333333333333333"
            "0000000000000000000000000000000000000000000000000102030405060708"
            "0000000000000000000000001234567890123456789012345678901234567890"
        );

        assert_eq!(encoded, expected);
    }

    #[test]
    fn aggregation_inputs_bind_range_outputs_to_final_claim() {
        let inputs = valid_aggregation_inputs();
        inputs.validate().expect("matching aggregation coverage is valid");
    }

    #[test]
    fn aggregation_inputs_reject_range_l1_head_mismatch() {
        let mut inputs = valid_aggregation_inputs();
        let expected_l1_head = inputs.l1_head;
        let actual_l1_head = B256::from([0xee; 32]);
        inputs.range_outputs[0].l1_head = actual_l1_head;

        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::RangeL1HeadMismatch {
                expected: expected_l1_head,
                actual: actual_l1_head,
            })
        );
    }

    #[test]
    fn aggregation_inputs_reject_root_claim_mismatch() {
        let mut inputs = valid_aggregation_inputs();
        inputs.root_claim = B256::from([0x66; 32]);

        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::RootClaimMismatch {
                expected: B256::from([0x66; 32]),
                actual: B256::from([0x55; 32]),
            })
        );
    }

    #[test]
    fn aggregation_inputs_reject_range_consolidation_mismatch() {
        let mut inputs = valid_aggregation_inputs();
        inputs.range_outputs[0].transitions[1].optimistic_block.output_root =
            B256::from([0xee; 32]);

        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::RangeConsolidationMismatch { timestamp: 100 })
        );
    }

    #[test]
    fn aggregation_inputs_reject_intermediate_range_previous_root_mismatch() {
        let mut inputs = valid_aggregation_inputs();
        inputs.range_outputs[1].previous_super_roots[0] = B256::from([0xee; 32]);

        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::PreviousSuperRootMismatch {
                expected: B256::from([0x44; 32]),
                actual: B256::from([0xee; 32]),
            })
        );
    }

    #[test]
    fn aggregation_inputs_reject_timestamp_coverage_gaps() {
        let mut missing_consolidation_timestamp = valid_aggregation_inputs();
        missing_consolidation_timestamp.consolidation_outputs.pop();
        assert_eq!(
            missing_consolidation_timestamp.validate(),
            Err(SuperRootError::ConsolidationSpanStartMismatch { expected: 102, actual: 101 })
        );

        let mut missing_range_timestamp = valid_aggregation_inputs();
        missing_range_timestamp.range_outputs.pop();
        assert_eq!(
            missing_range_timestamp.validate(),
            Err(SuperRootError::RangeSpanStartMismatch { expected: 102, actual: 101 })
        );
    }

    #[test]
    fn aggregation_inputs_reject_chain_coverage_gaps() {
        let mut missing_consolidation_chain = valid_aggregation_inputs();
        missing_consolidation_chain.consolidation_outputs[0].transitions[0].optimistic_blocks.pop();
        assert_eq!(
            missing_consolidation_chain.validate(),
            Err(SuperRootError::InvalidRangeTransitionCount { expected: 2, actual: 1 })
        );

        let mut missing_range_chain = valid_aggregation_inputs();
        missing_range_chain.range_outputs[0].transitions.pop();
        assert_eq!(
            missing_range_chain.validate(),
            Err(SuperRootError::InvalidRangeTransitionCount { expected: 2, actual: 1 })
        );
    }

    #[test]
    fn aggregation_inputs_reject_consolidation_previous_root_mismatch() {
        let mut inputs = valid_aggregation_inputs();
        let expected = inputs.consolidation_outputs[1].previous_super_root;
        let actual = B256::from([0xee; 32]);
        inputs.consolidation_outputs[1].previous_super_root = actual;

        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::PreviousSuperRootMismatch { expected, actual })
        );
    }

    #[test]
    fn chain_ordering_requires_non_empty_strictly_increasing_ids() {
        assert_eq!(ensure_strictly_increasing_chains(&[]), Err(SuperRootError::EmptyOutputRoots));

        let duplicate = [output(10, 0x11), output(10, 0x22)];
        assert_eq!(
            ensure_strictly_increasing_chains(&duplicate),
            Err(SuperRootError::ChainIdsNotStrictlyIncreasing {
                previous: U256::from(10),
                current: U256::from(10),
            })
        );

        let reversed = [output(20, 0x11), output(10, 0x22)];
        assert_eq!(
            ensure_strictly_increasing_chains(&reversed),
            Err(SuperRootError::ChainIdsNotStrictlyIncreasing {
                previous: U256::from(20),
                current: U256::from(10),
            })
        );

        ensure_strictly_increasing_chains(&[output(10, 0x11), output(20, 0x22)])
            .expect("ordered chain IDs are valid");
    }

    #[test]
    fn timestamp_span_boundaries_are_inclusive() {
        let span = TimestampSpan::new(100, 102).expect("span is valid");
        assert!(!span.contains(99));
        assert!(span.contains(100));
        assert!(span.contains(101));
        assert!(span.contains(102));
        assert!(!span.contains(103));

        assert_eq!(
            TimestampSpan::new(102, 100),
            Err(SuperRootError::InvalidTimestampSpan { start: 102, end: 100 })
        );

        let inputs = SuperRangeInputs {
            span: TimestampSpan { start: 102, end: 100 },
            l1_head: B256::ZERO,
            chain_ids: vec![U256::from(10)],
            previous_super_root_proofs: vec![],
            claimed_transitions: vec![transition(102, 10, 0x11, 0x12)],
        };
        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::InvalidTimestampSpan { start: 102, end: 100 })
        );
    }

    #[test]
    fn range_inputs_bind_full_timestamp_chain_coverage() {
        let inputs = SuperRangeInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            l1_head: B256::from([0x99; 32]),
            chain_ids: vec![U256::from(10), U256::from(20)],
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]),
                SuperRootProof::new(100, vec![output(10, 0x03), output(20, 0x04)]),
            ],
            claimed_transitions: vec![
                transition(100, 10, 0x11, 0x12),
                transition(100, 20, 0x21, 0x22),
                transition(101, 10, 0x31, 0x32),
                transition(101, 20, 0x41, 0x42),
            ],
        };

        inputs.validate().expect("complete ordered range coverage is valid");
    }

    #[test]
    fn range_inputs_reject_previous_root_count_timestamp_and_chain_mismatches() {
        let base = SuperRangeInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            l1_head: B256::from([0x99; 32]),
            chain_ids: vec![U256::from(10), U256::from(20)],
            previous_super_root_proofs: vec![previous_proof(99), previous_proof(100)],
            claimed_transitions: vec![
                transition(100, 10, 0x11, 0x12),
                transition(100, 20, 0x21, 0x22),
                transition(101, 10, 0x31, 0x32),
                transition(101, 20, 0x41, 0x42),
            ],
        };

        let missing_previous = SuperRangeInputs {
            previous_super_root_proofs: vec![previous_proof(99)],
            ..base.clone()
        };
        assert_eq!(
            missing_previous.validate(),
            Err(SuperRootError::InvalidRangePreviousRootCount { expected: 2, actual: 1 })
        );

        let wrong_timestamp = SuperRangeInputs {
            previous_super_root_proofs: vec![previous_proof(98), previous_proof(100)],
            ..base.clone()
        };
        assert_eq!(
            wrong_timestamp.validate(),
            Err(SuperRootError::RangePreviousRootTimestampMismatch { expected: 99, actual: 98 })
        );

        let missing_chain = SuperRangeInputs {
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01)]),
                previous_proof(100),
            ],
            ..base.clone()
        };
        assert_eq!(
            missing_chain.validate(),
            Err(SuperRootError::MismatchedRangePreviousRootChainCount { expected: 2, actual: 1 })
        );

        let wrong_chain = SuperRangeInputs {
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(30, 0x02)]),
                previous_proof(100),
            ],
            ..base
        };
        assert_eq!(
            wrong_chain.validate(),
            Err(SuperRootError::MismatchedRangePreviousRootChainId {
                expected: U256::from(20),
                actual: U256::from(30),
            })
        );
    }

    #[test]
    fn range_inputs_reject_incomplete_or_misordered_transition_coverage() {
        let base = SuperRangeInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            l1_head: B256::from([0x99; 32]),
            chain_ids: vec![U256::from(10), U256::from(20)],
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]),
                SuperRootProof::new(100, vec![output(10, 0x03), output(20, 0x04)]),
            ],
            claimed_transitions: vec![
                transition(100, 10, 0x11, 0x12),
                transition(100, 20, 0x21, 0x22),
                transition(101, 10, 0x31, 0x32),
                transition(101, 20, 0x41, 0x42),
            ],
        };

        let missing = SuperRangeInputs {
            claimed_transitions: base.claimed_transitions[..3].to_vec(),
            ..base.clone()
        };
        assert_eq!(
            missing.validate(),
            Err(SuperRootError::InvalidRangeTransitionCount { expected: 4, actual: 3 })
        );

        let wrong_timestamp = SuperRangeInputs {
            claimed_transitions: vec![
                transition(100, 10, 0x11, 0x12),
                transition(101, 20, 0x21, 0x22),
                transition(101, 10, 0x31, 0x32),
                transition(101, 20, 0x41, 0x42),
            ],
            ..base.clone()
        };
        assert_eq!(
            wrong_timestamp.validate(),
            Err(SuperRootError::RangeTransitionTimestampMismatch { expected: 100, actual: 101 })
        );

        let wrong_chain = SuperRangeInputs {
            claimed_transitions: vec![
                transition(100, 10, 0x11, 0x12),
                transition(100, 30, 0x21, 0x22),
                transition(101, 10, 0x31, 0x32),
                transition(101, 20, 0x41, 0x42),
            ],
            ..base
        };
        assert_eq!(
            wrong_chain.validate(),
            Err(SuperRootError::RangeTransitionChainIdMismatch {
                expected: U256::from(20),
                actual: U256::from(30),
            })
        );
    }

    #[test]
    fn range_inputs_bind_previous_super_roots_per_timestamp() {
        let base = SuperRangeInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            l1_head: B256::from([0x99; 32]),
            chain_ids: vec![U256::from(10), U256::from(20)],
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]),
                SuperRootProof::new(100, vec![output(10, 0x03), output(20, 0x04)]),
            ],
            claimed_transitions: vec![
                transition(100, 10, 0x11, 0x12),
                transition(100, 20, 0x21, 0x22),
                transition(101, 10, 0x31, 0x32),
                transition(101, 20, 0x41, 0x42),
            ],
        };

        base.validate().expect("per-timestamp previous roots are valid");

        let missing_previous_root = SuperRangeInputs {
            previous_super_root_proofs: base.previous_super_root_proofs[..1].to_vec(),
            ..base.clone()
        };
        assert_eq!(
            missing_previous_root.validate(),
            Err(SuperRootError::InvalidRangePreviousRootCount { expected: 2, actual: 1 })
        );

        let wrong_previous_timestamp = SuperRangeInputs {
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]),
                SuperRootProof::new(101, vec![output(10, 0x03), output(20, 0x04)]),
            ],
            ..base.clone()
        };
        assert_eq!(
            wrong_previous_timestamp.validate(),
            Err(SuperRootError::RangePreviousRootTimestampMismatch { expected: 100, actual: 101 })
        );

        let zero_start = SuperRangeInputs {
            span: TimestampSpan::new(0, 0).expect("valid inclusive span"),
            l1_head: B256::from([0x99; 32]),
            chain_ids: vec![U256::from(10), U256::from(20)],
            previous_super_root_proofs: vec![SuperRootProof::new(
                0,
                vec![output(10, 0x01), output(20, 0x02)],
            )],
            claimed_transitions: vec![transition(0, 10, 0x11, 0x12), transition(0, 20, 0x21, 0x22)],
        };
        assert_eq!(
            zero_start.validate(),
            Err(SuperRootError::InvalidTimestampSpanStart { start: 0 })
        );

        let missing_previous_chain = SuperRangeInputs {
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]),
                SuperRootProof::new(100, vec![output(10, 0x03)]),
            ],
            ..base.clone()
        };
        assert_eq!(
            missing_previous_chain.validate(),
            Err(SuperRootError::MismatchedRangePreviousRootChainCount { expected: 2, actual: 1 })
        );

        let wrong_previous_chain = SuperRangeInputs {
            previous_super_root_proofs: vec![
                SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]),
                SuperRootProof::new(100, vec![output(10, 0x03), output(30, 0x04)]),
            ],
            ..base
        };
        assert_eq!(
            wrong_previous_chain.validate(),
            Err(SuperRootError::MismatchedRangePreviousRootChainId {
                expected: U256::from(20),
                actual: U256::from(30),
            })
        );
    }

    #[test]
    fn consolidation_inputs_bind_transition_boundary() {
        let inputs = SuperConsolidationInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            previous_super_root: B256::from([0xaa; 32]),
            transitions: vec![
                SuperConsolidationTransitionInput {
                    optimistic_blocks: vec![optimistic(10, 0x11, 0x12), optimistic(20, 0x21, 0x22)],
                    claimed_super_root_proof: SuperRootProof::new(
                        100,
                        vec![output(10, 0x31), output(20, 0x32)],
                    ),
                },
                SuperConsolidationTransitionInput {
                    optimistic_blocks: vec![optimistic(10, 0x41, 0x42), optimistic(20, 0x51, 0x52)],
                    claimed_super_root_proof: SuperRootProof::new(
                        101,
                        vec![output(10, 0x61), output(20, 0x62)],
                    ),
                },
            ],
        };

        inputs.validate().expect("matching consolidation coverage is valid");
    }

    #[test]
    fn consolidation_inputs_reject_mismatched_chain_coverage() {
        let inputs = SuperConsolidationInputs {
            span: TimestampSpan::new(100, 100).expect("valid span"),
            previous_super_root: B256::from([0xaa; 32]),
            transitions: vec![SuperConsolidationTransitionInput {
                optimistic_blocks: vec![optimistic(10, 0x11, 0x12), optimistic(20, 0x21, 0x22)],
                claimed_super_root_proof: SuperRootProof::new(
                    100,
                    vec![output(10, 0x31), output(30, 0x32)],
                ),
            }],
        };

        assert_eq!(
            inputs.validate(),
            Err(SuperRootError::MismatchedConsolidationChainId {
                optimistic_chain_id: U256::from(20),
                claimed_chain_id: U256::from(30),
            })
        );

        let count_mismatch = SuperConsolidationInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            transitions: vec![SuperConsolidationTransitionInput {
                optimistic_blocks: vec![optimistic(10, 0x11, 0x12), optimistic(20, 0x21, 0x22)],
                claimed_super_root_proof: SuperRootProof::new(
                    100,
                    vec![output(10, 0x31), output(20, 0x32)],
                ),
            }],
            ..inputs
        };
        assert_eq!(
            count_mismatch.validate(),
            Err(SuperRootError::InvalidConsolidationInputCount { expected: 2, actual: 1 })
        );

        let timestamp_mismatch = SuperConsolidationInputs {
            transitions: vec![SuperConsolidationTransitionInput {
                optimistic_blocks: vec![optimistic(10, 0x11, 0x12), optimistic(20, 0x21, 0x22)],
                claimed_super_root_proof: SuperRootProof::new(
                    101,
                    vec![output(10, 0x31), output(20, 0x32)],
                ),
            }],
            ..inputs
        };
        assert_eq!(
            timestamp_mismatch.validate(),
            Err(SuperRootError::ConsolidationTimestampMismatch { expected: 100, actual: 101 })
        );
    }

    #[test]
    fn aggregation_inputs_serialize_with_dynamic_range_vkey() {
        let mut inputs = valid_aggregation_inputs();
        inputs.range_vkey = [0, 1, 2, 3, 4, 5, 6, 7];

        let encoded = serde_json::to_vec(&inputs).expect("aggregation inputs serialize");
        let decoded: SuperAggregationInputs =
            serde_json::from_slice(&encoded).expect("aggregation inputs deserialize");

        assert_eq!(decoded, inputs);
        assert_eq!(decoded.range_vkey, [0, 1, 2, 3, 4, 5, 6, 7]);
    }

    #[test]
    fn super_interop_inputs_serialize_mode_discriminant() {
        let range = SuperInteropInputs::Range(SuperRangeInputs {
            span: TimestampSpan::new(100, 100).expect("valid span"),
            l1_head: B256::from([0x99; 32]),
            chain_ids: vec![U256::from(10)],
            previous_super_root_proofs: vec![SuperRootProof::new(99, vec![output(10, 0x01)])],
            claimed_transitions: vec![transition(100, 10, 0x11, 0x12)],
        });
        let consolidation = SuperInteropInputs::Consolidation(SuperConsolidationInputs {
            span: TimestampSpan::new(100, 100).expect("valid span"),
            previous_super_root: B256::from([0xaa; 32]),
            transitions: vec![SuperConsolidationTransitionInput {
                optimistic_blocks: vec![optimistic(10, 0x11, 0x12)],
                claimed_super_root_proof: SuperRootProof::new(100, vec![output(10, 0x31)]),
            }],
        });

        let range_encoded = serde_json::to_vec(&range).expect("range mode serializes");
        let consolidation_encoded =
            serde_json::to_vec(&consolidation).expect("consolidation mode serializes");

        assert_ne!(range_encoded, consolidation_encoded);
        let decoded_range: SuperInteropInputs =
            serde_json::from_slice(&range_encoded).expect("range mode deserializes");
        let decoded_consolidation: SuperInteropInputs =
            serde_json::from_slice(&consolidation_encoded)
                .expect("consolidation mode deserializes");
        assert_eq!(decoded_range, range);
        assert_eq!(decoded_consolidation, consolidation);
    }

    #[test]
    fn public_values_bind_the_prover_address() {
        let prover = Address::from([0x44; 20]);
        let mut inputs = valid_aggregation_inputs();
        inputs.prover = prover;

        assert_eq!(inputs.public_values().prover, prover);
    }
}
