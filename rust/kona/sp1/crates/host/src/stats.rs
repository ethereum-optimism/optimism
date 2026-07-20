//! Execution statistics for a range execution.

use std::fmt;

use crate::fetcher::BlockInfo;
use kona_sp1_client_utils::precompiles::cycle_tracker::keys;
use num_format::{Locale, ToFormattedString};
use serde::{Deserialize, Serialize};
use sp1_sdk::ExecutionReport;

/// Statistics for the range execution.
#[allow(missing_docs)]
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ExecutionStats {
    pub l1_head: u64,
    pub batch_start: u64,
    pub batch_end: u64,
    /// The wall clock time to generate the witness.
    pub witness_generation_time_sec: u64,
    /// The wall clock time to execute the range on the machine.
    pub total_execution_time_sec: u64,
    pub total_instruction_count: u64,
    pub oracle_verify_instruction_count: u64,
    pub derivation_instruction_count: u64,
    pub block_execution_instruction_count: u64,
    pub blob_verification_instruction_count: u64,
    pub total_sp1_gas: u64,
    pub nb_blocks: u64,
    pub nb_transactions: u64,
    pub eth_gas_used: u64,
    pub l1_fees: u128,
    pub total_tx_fees: u128,
    pub cycles_per_block: u64,
    pub cycles_per_transaction: u64,
    pub transactions_per_block: u64,
    pub gas_used_per_block: u64,
    pub gas_used_per_transaction: u64,
    pub bn_pair_cycles: u64,
    pub bn_add_cycles: u64,
    pub bn_mul_cycles: u64,
    pub kzg_eval_cycles: u64,
    pub ec_recover_cycles: u64,
    pub p256_verify_cycles: u64,
}

/// Write a statistic to the formatter.
fn write_stat(f: &mut fmt::Formatter<'_>, label: &str, value: u64) -> fmt::Result {
    writeln!(f, "| {:<30} | {:>25} |", label, value.to_formatted_string(&Locale::en))
}

impl fmt::Display for ExecutionStats {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "+--------------------------------+---------------------------+")?;
        writeln!(f, "| {:<30} | {:<25} |", "Metric", "Value")?;
        writeln!(f, "+--------------------------------+---------------------------+")?;
        write_stat(f, "Batch Start", self.batch_start)?;
        write_stat(f, "Batch End", self.batch_end)?;
        write_stat(f, "Witness Generation (seconds)", self.witness_generation_time_sec)?;
        write_stat(f, "Execution Duration (seconds)", self.total_execution_time_sec)?;
        write_stat(f, "Total Instruction Count", self.total_instruction_count)?;
        write_stat(f, "Oracle Verify Cycles", self.oracle_verify_instruction_count)?;
        write_stat(f, "Derivation Cycles", self.derivation_instruction_count)?;
        write_stat(f, "Block Execution Cycles", self.block_execution_instruction_count)?;
        write_stat(f, "Blob Verification Cycles", self.blob_verification_instruction_count)?;
        write_stat(f, "Total SP1 Gas", self.total_sp1_gas)?;
        write_stat(f, "Number of Blocks", self.nb_blocks)?;
        write_stat(f, "Number of Transactions", self.nb_transactions)?;
        write_stat(f, "Ethereum Gas Used", self.eth_gas_used)?;
        write_stat(f, "Cycles per Block", self.cycles_per_block)?;
        write_stat(f, "Cycles per Transaction", self.cycles_per_transaction)?;
        write_stat(f, "Transactions per Block", self.transactions_per_block)?;
        write_stat(f, "Gas Used per Block", self.gas_used_per_block)?;
        write_stat(f, "Gas Used per Transaction", self.gas_used_per_transaction)?;
        write_stat(f, "BN Pair Cycles", self.bn_pair_cycles)?;
        write_stat(f, "BN Add Cycles", self.bn_add_cycles)?;
        write_stat(f, "BN Mul Cycles", self.bn_mul_cycles)?;
        write_stat(f, "KZG Eval Cycles", self.kzg_eval_cycles)?;
        write_stat(f, "EC Recover Cycles", self.ec_recover_cycles)?;
        write_stat(f, "P256 Verify Cycles", self.p256_verify_cycles)?;
        writeln!(f, "+--------------------------------+---------------------------+")
    }
}

impl ExecutionStats {
    /// Create a new execution stats.
    pub fn new(
        l1_head: u64,
        block_data: &[BlockInfo],
        report: &ExecutionReport,
        witness_generation_time_sec: u64,
        total_execution_time_sec: u64,
    ) -> Self {
        if block_data.is_empty() {
            return Self::default();
        }

        // Sort the block data by block number.
        let mut block_data = block_data.to_vec();
        block_data.sort_by_key(|b| b.block_number);

        let get_cycles = |key: &str| *report.cycle_tracker.get(key).unwrap_or(&0);

        let nb_blocks = block_data.len() as u64;
        let nb_transactions = block_data.iter().map(|b| b.transaction_count).sum();
        let total_gas_used: u64 = block_data.iter().map(|b| b.gas_used).sum();
        let total_instructions = report.total_instruction_count();

        let safe_div = |a: u64, b: u64| a.checked_div(b).unwrap_or_default();

        Self {
            l1_head,
            // The "block data" does not include the first block (as it's not executed), so we need
            // to subtract 1 to give the user back the block corresponding to the
            // blockhash they're proving from.
            batch_start: block_data[0].block_number - 1,
            batch_end: block_data[block_data.len() - 1].block_number,
            total_instruction_count: total_instructions,
            total_sp1_gas: report.gas().unwrap_or(0),
            block_execution_instruction_count: get_cycles("block-execution"),
            oracle_verify_instruction_count: get_cycles("oracle-verify"),
            derivation_instruction_count: get_cycles("payload-derivation"),
            blob_verification_instruction_count: get_cycles("blob-verification"),
            bn_add_cycles: get_cycles(keys::BN_ADD),
            bn_mul_cycles: get_cycles(keys::BN_MUL),
            bn_pair_cycles: get_cycles(keys::BN_PAIR),
            kzg_eval_cycles: get_cycles(keys::KZG_EVAL),
            ec_recover_cycles: get_cycles(keys::EC_RECOVER),
            p256_verify_cycles: get_cycles(keys::P256_VERIFY),
            nb_transactions,
            eth_gas_used: block_data.iter().map(|b| b.gas_used).sum(),
            l1_fees: block_data.iter().map(|b| b.total_l1_fees).sum(),
            total_tx_fees: block_data.iter().map(|b| b.total_tx_fees).sum(),
            nb_blocks,
            cycles_per_block: safe_div(total_instructions, nb_blocks),
            cycles_per_transaction: safe_div(total_instructions, nb_transactions),
            transactions_per_block: safe_div(nb_transactions, nb_blocks),
            gas_used_per_block: safe_div(total_gas_used, nb_blocks),
            gas_used_per_transaction: safe_div(total_gas_used, nb_transactions),
            witness_generation_time_sec,
            total_execution_time_sec,
        }
    }
}

/// A [`ExecutionStats`] that can be displayed as Markdown.
#[derive(Debug)]
pub struct MarkdownExecutionStats(ExecutionStats);

impl MarkdownExecutionStats {
    /// Creates a [`MarkdownExecutionStats`].
    pub const fn new(inner: ExecutionStats) -> Self {
        Self(inner)
    }
}

impl fmt::Display for MarkdownExecutionStats {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "| {:<30} | {:<25} |", "Metric", "Value")?;
        writeln!(f, "|--------------------------------|---------------------------|")?;
        write_stat(f, "Batch Start", self.0.batch_start)?;
        write_stat(f, "Batch End", self.0.batch_end)?;
        write_stat(f, "Witness Generation (seconds)", self.0.witness_generation_time_sec)?;
        write_stat(f, "Execution Duration (seconds)", self.0.total_execution_time_sec)?;
        write_stat(f, "Total Instruction Count", self.0.total_instruction_count)?;
        write_stat(f, "Oracle Verify Cycles", self.0.oracle_verify_instruction_count)?;
        write_stat(f, "Derivation Cycles", self.0.derivation_instruction_count)?;
        write_stat(f, "Block Execution Cycles", self.0.block_execution_instruction_count)?;
        write_stat(f, "Blob Verification Cycles", self.0.blob_verification_instruction_count)?;
        write_stat(f, "Total SP1 Gas", self.0.total_sp1_gas)?;
        write_stat(f, "Number of Blocks", self.0.nb_blocks)?;
        write_stat(f, "Number of Transactions", self.0.nb_transactions)?;
        write_stat(f, "Ethereum Gas Used", self.0.eth_gas_used)?;
        write_stat(f, "Cycles per Block", self.0.cycles_per_block)?;
        write_stat(f, "Cycles per Transaction", self.0.cycles_per_transaction)?;
        write_stat(f, "Transactions per Block", self.0.transactions_per_block)?;
        write_stat(f, "Gas Used per Block", self.0.gas_used_per_block)?;
        write_stat(f, "Gas Used per Transaction", self.0.gas_used_per_transaction)?;
        write_stat(f, "BN Pair Cycles", self.0.bn_pair_cycles)?;
        write_stat(f, "BN Add Cycles", self.0.bn_add_cycles)?;
        write_stat(f, "BN Mul Cycles", self.0.bn_mul_cycles)?;
        write_stat(f, "KZG Eval Cycles", self.0.kzg_eval_cycles)?;
        write_stat(f, "EC Recover Cycles", self.0.ec_recover_cycles)?;
        write_stat(f, "P256 Verify Cycles", self.0.p256_verify_cycles)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn empty_report() -> ExecutionReport {
        ExecutionReport::default()
    }

    #[test]
    fn stats_default_when_block_data_empty() {
        let stats = ExecutionStats::new(100, &[], &empty_report(), 1, 2);
        assert_eq!(stats.nb_blocks, 0);
        assert_eq!(stats.batch_start, 0);
        assert_eq!(stats.batch_end, 0);
    }

    #[test]
    fn stats_handles_zero_transaction_ranges() {
        let block_data = [BlockInfo {
            block_number: 10,
            transaction_count: 0,
            gas_used: 0,
            total_l1_fees: 0,
            total_tx_fees: 0,
        }];

        let stats = ExecutionStats::new(100, &block_data, &empty_report(), 1, 2);
        assert_eq!(stats.nb_transactions, 0);
        assert_eq!(stats.cycles_per_transaction, 0);
        assert_eq!(stats.gas_used_per_transaction, 0);
    }
}
