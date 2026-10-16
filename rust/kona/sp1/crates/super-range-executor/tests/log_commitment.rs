//! Runs the `log-commitment` guest ELF in the SP1 executor. Tests that need the ELF skip when
//! `KONA_SP1_ELF_DIR` is unset; build it with `just build-elfs-native` in `rust/kona/sp1`.

use std::{path::PathBuf, sync::Arc};

use alloy_primitives::{Address, B256, Bytes, Log, LogData};
use kona_sp1_client_utils::log_commitment::{
    BlockReceipts, log_commitment_public_values, log_commitment_witness,
    test_utils::{eip1559, missing_trie_node_witness, mixed_blocks, receipts_root},
};
use kona_sp1_elfs::{ELF_DIR_ENV, load_elf_from};
use kona_sp1_super_range_executor::{
    MAX_ENTRIES_PER_BATCH, build_log_commitment_stdin, build_log_commitment_stdin_from_receipts,
    estimate_log_block_pgu, estimate_log_commitment_pgu,
};
use sp1_sdk::{Elf, ExecutionReport, Prover, ProverClient, SP1PublicValues, SP1Stdin};

const ELF_NAME: &str = "log-commitment-elf";

fn elf_dir() -> Option<PathBuf> {
    std::env::var(ELF_DIR_ENV).ok().map(PathBuf::from)
}

async fn execute(
    dir: &std::path::Path,
    stdin: SP1Stdin,
) -> anyhow::Result<(SP1PublicValues, ExecutionReport)> {
    let elf = Arc::<[u8]>::from(load_elf_from(dir, ELF_NAME)?);
    let client = ProverClient::builder().cpu().build().await;
    Ok(client.execute(Elf::Dynamic(elf), stdin).calculate_gas(true).await?)
}

/// The native program output for `blocks`, and the size of its witness.
fn native_public_values(blocks: &[BlockReceipts]) -> (Vec<u8>, usize) {
    let receipts_roots: Vec<B256> = blocks.iter().map(|block| block.receipts_root).collect();
    let witness = log_commitment_witness(blocks).unwrap();
    let witness = rkyv::to_bytes::<rkyv::rancor::Error>(&witness).unwrap();
    (log_commitment_public_values(&receipts_roots, &witness), witness.len())
}

/// Pins SP1 public values to the native output for a mixed batch with a duplicate root.
#[tokio::test(flavor = "multi_thread")]
async fn sp1_public_values_match_native() {
    let Some(dir) = elf_dir() else { return };
    let mut blocks = mixed_blocks();
    blocks.push(blocks[0].clone());
    let (native, _) = native_public_values(&blocks);

    let stdin = build_log_commitment_stdin_from_receipts(&blocks).unwrap();
    let (public_values, report) = execute(&dir, stdin).await.unwrap();
    assert_eq!(report.exit_code, 0);
    assert_eq!(public_values.as_slice(), native.as_slice());
}

/// Pins a halt in the SP1 executor, not `L = 0`, when the witness omits a trie node.
#[tokio::test(flavor = "multi_thread")]
async fn sp1_missing_trie_node_halts() {
    let Some(dir) = elf_dir() else { return };
    let (receipts_root, witness) = missing_trie_node_witness();
    let stdin = build_log_commitment_stdin(&[receipts_root], witness).unwrap();

    let (public_values, report) = execute(&dir, stdin).await.unwrap();
    // SP1 reports a guest panic as a non-zero exit code, not as an execution error.
    assert_ne!(report.exit_code, 0, "a missing trie node must halt the guest");
    assert!(public_values.as_slice().is_empty());
}

/// Measures one block of `log_count` logs with `data` bytes each, in one receipt.
async fn measure(log_count: u64, data: usize) {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let logs = (0..log_count)
        .map(|seed| Log {
            address: Address::left_padding_from(&seed.to_be_bytes()),
            data: LogData::new_unchecked(vec![], Bytes::from(vec![seed as u8; data])),
        })
        .collect();
    let receipts = vec![eip1559(logs)];
    let blocks = vec![BlockReceipts { receipts_root: receipts_root(&receipts), receipts }];
    let (native, witness_bytes) = native_public_values(&blocks);

    let stdin = build_log_commitment_stdin_from_receipts(&blocks).unwrap();
    let (public_values, report) = execute(&dir, stdin).await.unwrap();
    assert_eq!(report.exit_code, 0);
    assert_eq!(public_values.as_slice(), native.as_slice());
    println!(
        "RESULT logs={log_count} data={data} witness_bytes={witness_bytes} instructions={} \
         pgu={:?}",
        report.total_instruction_count(),
        report.gas(),
    );
}

/// A full 30M-gas block of 1 KB logs.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "manual SP1 cost measurement"]
async fn measure_data_heavy_block() {
    measure(3_480, 1_024).await;
}

/// A full 30M-gas block of empty logs.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "manual SP1 cost measurement"]
async fn measure_log_heavy_block() {
    measure(75_000, 0).await;
}

/// A block with one log.
#[tokio::test(flavor = "multi_thread")]
#[ignore = "manual SP1 cost measurement"]
async fn measure_single_log_block() {
    measure(1, 0).await;
}

/// `count` logs with distinct addresses, `topics` topics, and `data` data bytes each.
fn shaped_logs(first_seed: u64, count: u64, topics: usize, data: usize) -> Vec<Log> {
    (first_seed..first_seed + count)
        .map(|seed| Log {
            address: Address::left_padding_from(&seed.to_be_bytes()),
            data: LogData::new_unchecked(
                vec![B256::repeat_byte(0x11); topics],
                Bytes::from(vec![seed as u8; data]),
            ),
        })
        .collect()
}

/// A block with one receipt for each entry of `logs`.
fn block(logs: Vec<Vec<Log>>) -> BlockReceipts {
    let receipts: Vec<_> = logs.into_iter().map(eip1559).collect();
    BlockReceipts { receipts_root: receipts_root(&receipts), receipts }
}

/// Pins a batch estimate at or above the measured PGU for each log shape.
#[tokio::test(flavor = "multi_thread")]
async fn log_estimate_bounds_measured_pgu() {
    let Some(dir) = elf_dir() else { return };
    let mixed: Vec<_> = (0..8u64)
        .map(|i| block(vec![shaped_logs(i * 100, i * 10, (i % 5) as usize, (i * 64) as usize); 3]))
        .collect();
    let at_cap: Vec<_> =
        (0..MAX_ENTRIES_PER_BATCH as u64).map(|i| block(vec![shaped_logs(i, 1, 1, 32)])).collect();
    let shapes = [
        ("one log", vec![block(vec![shaped_logs(0, 1, 0, 0)])]),
        ("3,480 logs of 1 KB", vec![block(vec![shaped_logs(0, 3_480, 0, 1_024)])]),
        ("75,000 empty logs", vec![block(vec![shaped_logs(0, 75_000, 0, 0)])]),
        ("16,000 logs of 4 topics", vec![block(vec![shaped_logs(0, 16_000, 4, 0)])]),
        ("1,428 receipts without logs", vec![block(vec![vec![]; 1_428])]),
        ("one 100 KB log", vec![block(vec![shaped_logs(0, 1, 0, 100 * 1_024)])]),
        ("8 mixed blocks", mixed),
        ("entry cap", at_cap),
    ];
    for (shape, blocks) in shapes {
        let estimate = estimate_log_commitment_pgu(
            blocks.iter().map(|block| estimate_log_block_pgu(&block.receipts)),
        );
        let (native, _) = native_public_values(&blocks);
        let stdin = build_log_commitment_stdin_from_receipts(&blocks).unwrap();
        let (public_values, report) = execute(&dir, stdin).await.unwrap();
        assert_eq!(report.exit_code, 0, "{shape}");
        assert_eq!(public_values.as_slice(), native.as_slice(), "{shape}");
        let measured = report.gas().expect("gas is calculated");
        println!("RESULT shape={shape:?} measured={measured} estimate={estimate}");
        assert!(measured <= estimate, "{shape}: measured {measured} PGU over estimate {estimate}");
    }
}
