use crate::{
    CrossSafetyError,
    safety_checker::{ValidationError, ValidationError::InitiatingMessageNotFound},
};
use alloy_primitives::{BlockHash, ChainId};
use derive_more::Constructor;
use kona_interop::InteropValidator;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::{CrossChainSafetyProvider, StorageError};
use kona_supervisor_types::ExecutingMessage;
use op_alloy_consensus::interop::SafetyLevel;
use std::collections::HashSet;

/// Uses a [`CrossChainSafetyProvider`] to verify the safety of cross-chain message dependencies.
#[derive(Debug, Constructor)]
pub struct CrossSafetyChecker<'a, P, V> {
    chain_id: ChainId,
    validator: &'a V,
    provider: &'a P,
    required_level: SafetyLevel,
}

impl<P, V> CrossSafetyChecker<'_, P, V>
where
    P: CrossChainSafetyProvider,
    V: InteropValidator,
{
    /// Verifies that all executing messages in the given block are valid based on the validity
    /// checks
    pub fn validate_block(&self, block: BlockInfo) -> Result<(), CrossSafetyError> {
        self.map_dependent_block(&block, self.chain_id, |message, initiating_block_fetcher| {
            // Step 1: Validate interop timestamps before any dependency checks
            self.validator
                .validate_interop_timestamps(
                    message.chain_id,  // initiating chain id
                    message.timestamp, // initiating block timestamp
                    self.chain_id,     // executing chain id
                    block.timestamp,   // executing block timestamp
                    None,
                )
                .map_err(ValidationError::InteropValidationError)?;

            // Step 2: Verify message dependency without fetching the initiating block.
            // This avoids unnecessary I/O and ensures we skip validation when:
            //  - The current target head of the chain is behind the initiating block (must wait for
            //    that chain to process further)
            // Only if the target head is ahead but the initiating block is missing, we return a
            // validation error.
            self.verify_message_dependency(&message)?;

            // Step 3: Lazily fetch the initiating block only after dependency checks pass.
            let initiating_block = initiating_block_fetcher()?;

            // Step 4: Validate message existence and integrity.
            self.validate_executing_message(initiating_block, &message)?;

            // Step 5: Perform cyclic dependency detection starting from the dependent block.
            self.check_cyclic_dependency(
                &block,
                &initiating_block,
                message.chain_id,
                &mut HashSet::new(),
            )
        })?;

        Ok(())
    }

    /// Ensures that the block a message depends on satisfies the given safety level.
    fn verify_message_dependency(
        &self,
        message: &ExecutingMessage,
    ) -> Result<(), CrossSafetyError> {
        let head = self.provider.get_safety_head_ref(message.chain_id, self.required_level)?;

        if head.number < message.block_number {
            return Err(CrossSafetyError::DependencyNotSafe {
                chain_id: message.chain_id,
                block_number: message.block_number,
            });
        }

        Ok(())
    }

    /// Recursively checks for a cyclic dependency in cross-chain messages.
    ///
    /// # Purpose
    /// This function walks backwards through message dependencies starting from a candidate block.
    /// If any dependency chain leads back to the candidate itself (with the same timestamp), it is
    /// considered a **cycle**, which would make the candidate block invalid for cross-safe
    /// promotion.
    ///
    /// # How It Works
    /// - It stops recursion if the block is already at required level (cannot be part of a new
    ///   cycle).
    /// - It only follows dependencies that occur at the same timestamp as the candidate.
    /// - It uses a `visited` set to avoid re-processing blocks or getting stuck in infinite loops.
    ///   It doesn't care about cycle that is created excluding the candidate block as that will be
    ///   detected by the specific chain's safety checker
    ///
    /// Example:
    /// - A (candidate) → B → C → A → ❌ cycle detected (includes candidate)
    /// - A → B → C → D (no return to A) → ✅ safe
    /// - B → C → D → B → ❌ ignored, since candidate is not involved
    fn check_cyclic_dependency(
        &self,
        candidate: &BlockInfo,
        current: &BlockInfo,
        chain_id: ChainId,
        visited: &mut HashSet<(ChainId, BlockHash)>,
    ) -> Result<(), CrossSafetyError> {
        // Skipping different timestamps
        if candidate.timestamp != current.timestamp {
            return Ok(());
        }

        // Already visited, avoid infinite loop
        let current_id = (chain_id, current.hash);
        if !visited.insert(current_id) {
            return Ok(());
        }

        // Reached back to candidate - cycle detected
        if candidate.hash == current.hash && self.chain_id == chain_id {
            return Err(ValidationError::CyclicDependency { block: *candidate }.into());
        }

        let head = self.provider.get_safety_head_ref(chain_id, self.required_level)?;
        if head.number >= current.number {
            return Ok(()); // Already at target safety level - cannot form a cycle
        }

        self.map_dependent_block(current, chain_id, |message, origin_block_fetcher| {
            let origin_block = origin_block_fetcher()?;
            self.check_cyclic_dependency(candidate, &origin_block, message.chain_id, visited)
        })
    }

    fn validate_executing_message(
        &self,
        init_block: BlockInfo,
        message: &ExecutingMessage,
    ) -> Result<(), CrossSafetyError> {
        // Ensure timestamp invariant
        if init_block.timestamp != message.timestamp {
            return Err(ValidationError::TimestampInvariantViolation {
                expected_timestamp: init_block.timestamp,
                actual_timestamp: message.timestamp,
            }
            .into());
        }

        // Try to fetch the original log from storage
        let init_msg = self
            .provider
            .get_log(message.chain_id, message.block_number, message.log_index)
            .map_err(|err| match err {
                StorageError::EntryNotFound(_) => {
                    CrossSafetyError::ValidationError(InitiatingMessageNotFound)
                }
                other => other.into(),
            })?;

        // Verify the hash of the message against the original
        // Don't need to verify the checksum as we're already verifying all the individual fields.
        if init_msg.hash != message.hash {
            return Err(ValidationError::InvalidMessageHash {
                message_hash: message.hash,
                original_hash: init_msg.hash,
            }
            .into());
        }

        Ok(())
    }

    /// For each executing log in the block, provide a lazy fetcher for the initiating block.
    /// The callback decides if/when to fetch the initiating block.
    fn map_dependent_block<F>(
        &self,
        exec_block: &BlockInfo,
        chain_id: ChainId,
        mut f: F,
    ) -> Result<(), CrossSafetyError>
    where
        F: for<'a> FnMut(
            ExecutingMessage,
            &'a dyn Fn() -> Result<BlockInfo, CrossSafetyError>,
        ) -> Result<(), CrossSafetyError>,
    {
        let logs = self.provider.get_block_logs(chain_id, exec_block.number)?;
        for log in logs {
            if let Some(msg) = log.executing_message {
                // Capture what we need for a lazy fetch.
                let provider = &self.provider;
                let chain = msg.chain_id;
                let number = msg.block_number;

                // Zero-arg closure that fetches the initiating block on demand.
                let fetcher =
                    || provider.get_block(chain, number).map_err(CrossSafetyError::Storage);

                // Pass the message and the reference to the fetcher.
                f(msg, &fetcher)?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use kona_interop::{DerivedRefPair, InteropValidationError};
    use kona_supervisor_storage::{EntryNotFoundError, StorageError};
    use kona_supervisor_types::Log;
    use mockall::mock;
    use op_alloy_consensus::interop::SafetyLevel;

    mock! (
        #[derive(Debug)]
        pub Provider {}

        impl CrossChainSafetyProvider for Provider {
            fn get_block(&self, chain_id: ChainId, block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_log(&self, chain_id: ChainId, block_number: u64, log_index: u32) -> Result<Log, StorageError>;
            fn get_block_logs(&self, chain_id: ChainId, block_number: u64) -> Result<Vec<Log>, StorageError>;
            fn get_safety_head_ref(&self, chain_id: ChainId, level: SafetyLevel) -> Result<BlockInfo, StorageError>;
            fn update_current_cross_unsafe(&self, chain_id: ChainId, block: &BlockInfo) -> Result<(), StorageError>;
            fn update_current_cross_safe(&self, chain_id: ChainId, block: &BlockInfo) -> Result<DerivedRefPair, StorageError>;
        }
    );

    mock! (
        #[derive(Debug)]
        pub Validator {}

        impl InteropValidator for Validator {
            fn validate_interop_timestamps(
                &self,
                initiating_chain_id: ChainId,
                initiating_timestamp: u64,
                executing_chain_id: ChainId,
                executing_timestamp: u64,
                timeout: Option<u64>,
            ) -> Result<(), InteropValidationError>;

            fn is_post_interop(&self, chain_id: ChainId, timestamp: u64) -> bool;

            fn is_interop_activation_block(&self, chain_id: ChainId, block: BlockInfo) -> bool;
        }
    );

    fn b256(n: u64) -> B256 {
        let mut bytes = [0u8; 32];
        bytes[24..].copy_from_slice(&n.to_be_bytes());
        B256::from(bytes)
    }

    #[test]
    fn verify_message_dependency_success() {
        let chain_id = 1;
        let msg = ExecutingMessage {
            chain_id,
            block_number: 100,
            log_index: 0,
            timestamp: 0,
            hash: b256(0),
        };

        let head_info =
            BlockInfo { number: 101, hash: b256(101), parent_hash: b256(100), timestamp: 0 };

        let mut provider = MockProvider::default();
        let validator = MockValidator::default();

        provider
            .expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::CrossSafe)
            .returning(move |_, _| Ok(head_info));

        let checker = CrossSafetyChecker::new(1, &validator, &provider, SafetyLevel::CrossSafe);
        let result = checker.verify_message_dependency(&msg);
        assert!(result.is_ok());
    }

    #[test]
    fn verify_message_dependency_failed() {
        let chain_id = 1;
        let msg = ExecutingMessage {
            chain_id,
            block_number: 105, // dependency is ahead of safety head
            log_index: 0,
            timestamp: 0,
            hash: b256(123),
        };

        let head_block = BlockInfo {
            number: 100, // safety head is behind the message dependency
            hash: b256(100),
            parent_hash: b256(99),
            timestamp: 0,
        };

        let mut provider = MockProvider::default();
        let validator = MockValidator::default();

        provider
            .expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::CrossSafe)
            .returning(move |_, _| Ok(head_block));

        let checker = CrossSafetyChecker::new(1, &validator, &provider, SafetyLevel::CrossSafe);
        let result = checker.verify_message_dependency(&msg);

        assert!(
            matches!(result, Err(CrossSafetyError::DependencyNotSafe { .. })),
            "Expected DependencyNotSafe error"
        );
    }

    #[test]
    fn validate_block_success() {
        let init_chain_id = 1;
        let exec_chain_id = 2;

        let block =
            BlockInfo { number: 101, hash: b256(101), parent_hash: b256(100), timestamp: 200 };

        let dep_block =
            BlockInfo { number: 100, hash: b256(100), parent_hash: b256(99), timestamp: 195 };

        let exec_msg = ExecutingMessage {
            chain_id: init_chain_id,
            block_number: 100,
            log_index: 0,
            timestamp: 195,
            hash: b256(999),
        };

        let init_log = Log {
            index: 0,
            hash: b256(999), // Matches msg.hash → passes checksum
            executing_message: None,
        };

        let exec_log = Log { index: 0, hash: b256(999), executing_message: Some(exec_msg) };

        let head =
            BlockInfo { number: 101, hash: b256(101), parent_hash: b256(100), timestamp: 200 };

        let mut provider = MockProvider::default();
        let mut validator = MockValidator::default();

        provider
            .expect_get_block_logs()
            .withf(move |cid, num| *cid == exec_chain_id && *num == 101)
            .returning(move |_, _| Ok(vec![exec_log.clone()]));

        provider
            .expect_get_block()
            .withf(move |cid, num| *cid == init_chain_id && *num == 100)
            .returning(move |_, _| Ok(dep_block));

        provider
            .expect_get_log()
            .withf(move |cid, blk, idx| *cid == init_chain_id && *blk == 100 && *idx == 0)
            .returning(move |_, _, _| Ok(init_log.clone()));

        provider
            .expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == init_chain_id && *lvl == SafetyLevel::CrossSafe)
            .returning(move |_, _| Ok(head));

        validator.expect_validate_interop_timestamps().returning(move |_, _, _, _, _| Ok(()));

        let checker =
            CrossSafetyChecker::new(exec_chain_id, &validator, &provider, SafetyLevel::CrossSafe);
        let result = checker.validate_block(block);
        assert!(result.is_ok());
    }

    #[test]
    fn validate_executing_message_timestamp_violation() {
        let chain_id = 1;
        let msg = ExecutingMessage {
            chain_id,
            block_number: 100,
            log_index: 0,
            timestamp: 1234,
            hash: b256(999),
        };

        let init_block = BlockInfo {
            number: 100,
            hash: b256(100),
            parent_hash: b256(99),
            timestamp: 9999, // Different timestamp to trigger invariant violation
        };

        let provider = MockProvider::default();
        let validator = MockValidator::default();
        let checker =
            CrossSafetyChecker::new(chain_id, &validator, &provider, SafetyLevel::CrossSafe);

        let result = checker.validate_executing_message(init_block, &msg);
        assert!(matches!(
            result,
            Err(CrossSafetyError::ValidationError(
                ValidationError::TimestampInvariantViolation { .. }
            ))
        ));
    }

    #[test]
    fn validate_executing_message_initiating_message_not_found() {
        let chain_id = 1;
        let msg = ExecutingMessage {
            chain_id,
            block_number: 100,
            log_index: 0,
            timestamp: 1234,
            hash: b256(999),
        };

        let init_block =
            BlockInfo { number: 100, hash: b256(100), parent_hash: b256(99), timestamp: 1234 };

        let mut provider = MockProvider::default();
        provider
            .expect_get_log()
            .withf(move |cid, blk, idx| *cid == chain_id && *blk == 100 && *idx == 0)
            .returning(|_, _, _| {
                Err(StorageError::EntryNotFound(EntryNotFoundError::LogNotFound {
                    block_number: 100,
                    log_index: 0,
                }))
            });

        let validator = MockValidator::default();

        let checker =
            CrossSafetyChecker::new(chain_id, &validator, &provider, SafetyLevel::CrossSafe);
        let result = checker.validate_executing_message(init_block, &msg);

        assert!(matches!(
            result,
            Err(CrossSafetyError::ValidationError(InitiatingMessageNotFound))
        ));
    }

    #[test]
    fn validate_executing_message_hash_mismatch() {
        let chain_id = 1;
        let msg = ExecutingMessage {
            chain_id,
            block_number: 100,
            log_index: 0,
            timestamp: 1234,
            hash: b256(123),
        };

        let init_block =
            BlockInfo { number: 100, hash: b256(100), parent_hash: b256(99), timestamp: 1234 };

        let init_log = Log {
            index: 0,
            hash: b256(990), // Checksum mismatch
            executing_message: None,
        };

        let mut provider = MockProvider::default();
        provider
            .expect_get_log()
            .withf(move |cid, blk, idx| *cid == chain_id && *blk == 100 && *idx == 0)
            .returning(move |_, _, _| Ok(init_log.clone()));

        let validator = MockValidator::default();
        let checker =
            CrossSafetyChecker::new(chain_id, &validator, &provider, SafetyLevel::CrossSafe);
        let result = checker.validate_executing_message(init_block, &msg);

        assert!(matches!(
            result,
            Err(CrossSafetyError::ValidationError(ValidationError::InvalidMessageHash {
                message_hash: _,
                original_hash: _
            }))
        ));
    }

    #[test]
    fn validate_executing_message_success() {
        let chain_id = 1;
        let timestamp = 1234;

        let init_block = BlockInfo {
            number: 100,
            hash: b256(100),
            parent_hash: b256(99),
            timestamp, // Matches msg.timestamp
        };

        let init_log = Log {
            index: 0,
            hash: b256(999), // Matches msg.hash → passes checksum
            executing_message: None,
        };

        let msg = ExecutingMessage {
            chain_id,
            block_number: 100,
            log_index: 0,
            timestamp,
            hash: b256(999),
        };

        let mut provider = MockProvider::default();
        provider
            .expect_get_log()
            .withf(move |cid, blk, idx| *cid == chain_id && *blk == 100 && *idx == 0)
            .returning(move |_, _, _| Ok(init_log.clone()));

        let validator = MockValidator::default();
        let checker =
            CrossSafetyChecker::new(chain_id, &validator, &provider, SafetyLevel::CrossSafe);

        let result = checker.validate_executing_message(init_block, &msg);
        assert!(result.is_ok(), "Expected successful validation");
    }

    #[test]
    fn detect_cycle_when_it_loops_back_to_candidate() {
        // Scenario:
        // candidate: (chain 1, block 10)
        // → depends on (chain 2, block 11)
        // → depends on (chain 3, block 20)
        // → depends on (chain 1, block 10) ← back to candidate!
        // Expected result: cyclic dependency detected.

        let ts = 100;
        let candidate =
            BlockInfo { number: 10, hash: b256(10), parent_hash: b256(9), timestamp: ts };

        let block11 =
            BlockInfo { number: 11, hash: b256(11), parent_hash: b256(10), timestamp: ts };

        let block20 =
            BlockInfo { number: 20, hash: b256(20), parent_hash: b256(19), timestamp: ts };

        let mut provider = MockProvider::default();
        let validator = MockValidator::default();

        // All blocks are below safety head (to allow traversal)
        provider.expect_get_safety_head_ref().returning(|_, _| {
            Ok(BlockInfo { number: 0, hash: b256(0), parent_hash: b256(0), timestamp: 0 })
        });

        // Define log dependencies
        provider.expect_get_block_logs().returning(move |chain, number| {
            match (chain.to_string().as_str(), number) {
                ("1", 10) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1010),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 2,
                        block_number: 11,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(222),
                    }),
                }]),
                ("2", 11) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1020),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 3,
                        block_number: 20,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(333),
                    }),
                }]),
                ("3", 20) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1030),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 1,
                        block_number: 10,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(444),
                    }),
                }]),
                _ => Ok(vec![]),
            }
        });

        // Define block fetch behavior
        provider.expect_get_block().returning(move |chain, number| {
            match (chain.to_string().as_str(), number) {
                ("2", 11) => Ok(block11),
                ("3", 20) => Ok(block20),
                ("1", 10) => Ok(candidate),
                _ => panic!("unexpected block lookup: chain={chain} num={number}"),
            }
        });

        let checker = CrossSafetyChecker::new(1, &validator, &provider, SafetyLevel::CrossSafe);

        let result = checker.check_cyclic_dependency(&candidate, &block11, 2, &mut HashSet::new());

        assert!(
            matches!(
                result,
                Err(CrossSafetyError::ValidationError(ValidationError::CyclicDependency { .. }))
            ),
            "Expected cyclic dependency error"
        );
    }

    #[test]
    fn no_cycle_if_dependency_path_does_not_reach_candidate() {
        // Scenario:
        // candidate: (chain 1, block 10)
        // → depends on (chain 2, block 11)
        // → depends on (chain 3, block 20)
        // But no further dependency → path ends safely without cycling back to candidate.
        // Expected result: no cycle detected.

        let ts = 100;
        let candidate =
            BlockInfo { number: 10, hash: b256(10), parent_hash: b256(9), timestamp: ts };

        let block11 =
            BlockInfo { number: 11, hash: b256(11), parent_hash: b256(10), timestamp: ts };

        let block20 =
            BlockInfo { number: 20, hash: b256(20), parent_hash: b256(19), timestamp: ts };

        let mut provider = MockProvider::default();
        let validator = MockValidator::default();

        // All blocks are below safety head (to allow traversal)
        provider.expect_get_safety_head_ref().returning(|_, _| {
            Ok(BlockInfo { number: 0, hash: b256(0), parent_hash: b256(0), timestamp: 0 })
        });

        // Define log dependencies
        provider.expect_get_block_logs().returning(move |chain, number| {
            match (chain.to_string().as_str(), number) {
                ("1", 10) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1010),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 2,
                        block_number: 11,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(222),
                    }),
                }]),
                ("2", 11) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1020),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 3,
                        block_number: 20,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(333),
                    }),
                }]),
                ("3", 20) => Ok(vec![]), // No further dependency — traversal ends here
                _ => Ok(vec![]),
            }
        });

        // Define block fetch behavior
        provider.expect_get_block().returning(move |chain, number| {
            match (chain.to_string().as_str(), number) {
                ("2", 11) => Ok(block11),
                ("3", 20) => Ok(block20),
                _ => panic!("unexpected block lookup: chain={chain} num={number}"),
            }
        });

        let checker = CrossSafetyChecker::new(1, &validator, &provider, SafetyLevel::CrossSafe);

        let result = checker.check_cyclic_dependency(&candidate, &block11, 2, &mut HashSet::new());

        assert!(result.is_ok(), "Expected no cycle when dependency path does not reach candidate");
    }

    #[test]
    fn ignores_cycle_that_does_not_include_candidate() {
        // Scenario:
        // There is a cycle between blocks:
        // Chain2 block 11 → Chain3 block 20 → Chain2 block 11 (forms a cycle)
        // But candidate block (Chain1 block 10) is not in the cycle.
        // Expected result: cycle is ignored since it doesn't involve the candidate.

        let ts = 100;
        let candidate =
            BlockInfo { number: 10, hash: b256(10), parent_hash: b256(9), timestamp: ts };

        let block11 =
            BlockInfo { number: 11, hash: b256(11), parent_hash: b256(10), timestamp: ts };

        let block20 =
            BlockInfo { number: 20, hash: b256(20), parent_hash: b256(19), timestamp: ts };

        let mut provider = MockProvider::default();
        let validator = MockValidator::default();

        // All blocks are below safety head (so we traverse them)
        provider.expect_get_safety_head_ref().returning(|_, _| {
            Ok(BlockInfo { number: 0, hash: b256(0), parent_hash: b256(0), timestamp: 0 })
        });

        // Block logs setup:
        // Chain1 block 10 → Chain2 block 11
        // Chain2 block 11 → Chain3 block 20
        // Chain3 block 20 → Chain2 block 11 (cycle here, but no candidate involvement)
        provider.expect_get_block_logs().returning(move |chain, number| {
            match (chain.to_string().as_str(), number) {
                ("1", 10) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1010),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 2,
                        block_number: 11,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(222),
                    }),
                }]),
                ("2", 11) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1020),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 3,
                        block_number: 20,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(333),
                    }),
                }]),
                ("3", 20) => Ok(vec![Log {
                    index: 0,
                    hash: b256(1030),
                    executing_message: Some(ExecutingMessage {
                        chain_id: 2,
                        block_number: 11,
                        log_index: 0,
                        timestamp: ts,
                        hash: b256(444),
                    }),
                }]),
                _ => Ok(vec![]),
            }
        });

        // Block fetches
        provider.expect_get_block().returning(move |chain, number| {
            match (chain.to_string().as_str(), number) {
                ("2", 11) => Ok(block11),
                ("3", 20) => Ok(block20),
                _ => panic!("unexpected block lookup"),
            }
        });

        let checker = CrossSafetyChecker::new(1, &validator, &provider, SafetyLevel::CrossSafe);

        // Start traversal from chain2: block11 is a dependency of candidate
        let result = checker.check_cyclic_dependency(&candidate, &block11, 2, &mut HashSet::new());

        assert!(
            result.is_ok(),
            "Expected no cycle error because candidate is not part of the cycle"
        );
    }

    #[test]
    fn stops_traversal_if_timestamp_differs() {
        // Scenario:
        // Candidate and dependency block have different timestamps.
        // Should short-circuit the check and not recurse further.
        // Expected result: no cycle detected.

        let chain_id = 1;

        let candidate =
            BlockInfo { number: 10, hash: b256(10), parent_hash: b256(9), timestamp: 100 };
        let dep = BlockInfo { number: 9, hash: b256(9), parent_hash: b256(8), timestamp: 50 };

        let provider = MockProvider::default();
        let validator = MockValidator::default();

        let checker =
            CrossSafetyChecker::new(chain_id, &validator, &provider, SafetyLevel::CrossSafe);

        let result =
            checker.check_cyclic_dependency(&candidate, &dep, chain_id, &mut HashSet::new());
        assert!(result.is_ok());
    }

    #[test]
    fn stops_traversal_if_block_is_already_cross_safe() {
        // Scenario:
        // Dependency block is already cross-safe (head is ahead of it).
        // Should skip further traversal.
        // Expected result: no cycle detected.

        let chain_id = 1;

        let candidate =
            BlockInfo { number: 10, hash: b256(10), parent_hash: b256(9), timestamp: 100 };
        let dep = BlockInfo { number: 9, hash: b256(9), parent_hash: b256(8), timestamp: 100 };

        let mut provider = MockProvider::default();
        let validator = MockValidator::default();

        provider.expect_get_safety_head_ref().returning(|_, _| {
            Ok(BlockInfo {
                number: 10,
                hash: b256(10),
                parent_hash: b256(9),
                timestamp: 100, // head ahead
            })
        });

        let checker =
            CrossSafetyChecker::new(chain_id, &validator, &provider, SafetyLevel::CrossSafe);

        let result =
            checker.check_cyclic_dependency(&candidate, &dep, chain_id, &mut HashSet::new());
        assert!(result.is_ok());
    }
}
