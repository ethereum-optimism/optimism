//! Runs the `log-commitment` guest ELF in the SP1 executor. Tests that need the ELF skip when
//! `KONA_SP1_ELF_DIR` is unset; build it with `just build-elfs-native` in `rust/kona/sp1`.

use std::{path::PathBuf, sync::Arc};

use alloy_consensus::{
    EMPTY_ROOT_HASH, Receipt, ReceiptWithBloom, proofs::ordered_trie_root_with_encoder,
};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, Log, LogData, keccak256};
use kona_preimage::PreimageKey;
use kona_sp1_client_utils::{
    log_commitment::log_commitment_public_values,
    witness::{BlobData, DefaultWitnessData, WitnessData, preimage_store::PreimageStore},
};
use kona_sp1_elfs::{ELF_DIR_ENV, load_elf_from};
use kona_sp1_super_range_executor::build_log_commitment_stdin;
use op_alloy_consensus::{OpDepositReceipt, OpDepositReceiptWithBloom, OpReceiptEnvelope};
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

/// The native program output over the same stdin items the guest reads.
fn native_public_values(blocks: &[(B256, Vec<OpReceiptEnvelope>)], stdin: &SP1Stdin) -> Vec<u8> {
    let receipts_roots: Vec<B256> =
        blocks.iter().map(|(receipts_root, _)| *receipts_root).collect();
    log_commitment_public_values(&receipts_roots, &stdin.buffer[1])
}

/// `count` logs with distinct addresses, `topics` topics, and `data` data bytes each.
fn logs(first_seed: u64, count: u64, topics: usize, data: usize) -> Vec<Log> {
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

fn with_bloom(logs: Vec<Log>) -> ReceiptWithBloom<Receipt> {
    ReceiptWithBloom {
        receipt: Receipt { status: true.into(), cumulative_gas_used: 21_000, logs },
        logs_bloom: Default::default(),
    }
}

fn deposit(logs: Vec<Log>) -> OpReceiptEnvelope {
    let receipt = with_bloom(logs);
    OpReceiptEnvelope::Deposit(OpDepositReceiptWithBloom {
        receipt: OpDepositReceipt {
            inner: receipt.receipt,
            deposit_nonce: Some(1),
            deposit_receipt_version: Some(1),
        },
        logs_bloom: receipt.logs_bloom,
    })
}

fn block(receipts: Vec<OpReceiptEnvelope>) -> (B256, Vec<OpReceiptEnvelope>) {
    let receipts_root =
        ordered_trie_root_with_encoder(&receipts, |receipt, out| receipt.encode_2718(out));
    (receipts_root, receipts)
}

/// Pins SP1 public values to the native output for a mixed batch.
#[tokio::test(flavor = "multi_thread")]
async fn sp1_public_values_match_native() {
    let Some(dir) = elf_dir() else { return };
    let mixed = block(vec![
        deposit(vec![]),
        OpReceiptEnvelope::Legacy(with_bloom(logs(0, 2, 1, 40))),
        OpReceiptEnvelope::Eip2930(with_bloom(vec![])),
        OpReceiptEnvelope::Eip1559(with_bloom(logs(2, 3, 4, 0))),
        OpReceiptEnvelope::Eip7702(with_bloom(logs(5, 1, 0, 100))),
    ]);
    let blocks = vec![
        mixed.clone(),
        block(vec![deposit(vec![])]),
        (EMPTY_ROOT_HASH, vec![]),
        block(
            (0..130)
                .map(|i| OpReceiptEnvelope::Eip1559(with_bloom(logs(100 + i, 1, 2, 8))))
                .collect(),
        ),
        mixed,
    ];
    let stdin = build_log_commitment_stdin(&blocks).unwrap();
    let native = native_public_values(&blocks, &stdin);

    let (public_values, report) = execute(&dir, stdin).await.unwrap();
    assert_eq!(report.exit_code, 0);
    assert_eq!(public_values.as_slice(), native.as_slice());
}

/// Pins a halt in the SP1 executor, not `L = 0`, when the witness omits a trie node.
#[tokio::test(flavor = "multi_thread")]
async fn sp1_missing_trie_node_halts() {
    let Some(dir) = elf_dir() else { return };
    let blocks = vec![block(
        (0..20).map(|i| OpReceiptEnvelope::Eip1559(with_bloom(logs(i, 1, 1, 8)))).collect(),
    )];
    let (receipts_root, receipts) = &blocks[0];
    let mut trie = kona_mpt::ordered_trie_with_encoder(receipts, |receipt, out| {
        receipt.encode_2718(out);
    });
    assert_eq!(trie.root(), *receipts_root);
    let mut store = PreimageStore::default();
    let mut omitted = false;
    for (path, node) in trie.take_proof_nodes().into_inner() {
        if !omitted && !path.is_empty() {
            omitted = true;
            continue;
        }
        store.save_preimage(PreimageKey::new_keccak256(*keccak256(&node)), node.to_vec()).unwrap();
    }
    assert!(omitted);
    let witness = DefaultWitnessData::from_parts(store, BlobData::default());
    let mut stdin = build_log_commitment_stdin(&blocks).unwrap();
    stdin.buffer[1] = rkyv::to_bytes::<rkyv::rancor::Error>(&witness).unwrap().to_vec();

    let (public_values, report) = execute(&dir, stdin).await.unwrap();
    // SP1 reports a guest panic as a non-zero exit code, not as an execution error.
    assert_ne!(report.exit_code, 0, "a missing trie node must halt the guest");
    assert!(public_values.as_slice().is_empty());
}

/// Measures one block of `log_count` logs with `data` bytes each, in one receipt.
async fn measure(log_count: u64, data: usize) {
    let dir = elf_dir().expect("set KONA_SP1_ELF_DIR to the built ELF directory");
    let blocks =
        vec![block(vec![OpReceiptEnvelope::Eip1559(with_bloom(logs(0, log_count, 0, data)))])];
    let stdin = build_log_commitment_stdin(&blocks).unwrap();
    let witness_bytes = stdin.buffer[1].len();
    let native = native_public_values(&blocks, &stdin);

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
