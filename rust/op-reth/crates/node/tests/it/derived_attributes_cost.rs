//! Measures `engine_forkchoiceUpdated` and `engine_getPayload` for full derived blocks, with and
//! without a `0x7D`.
//!
//! This is a measurement, not a regression test. Run it in release mode and read the table:
//!
//! ```sh
//! cargo test --release -p reth-optimism-node --test it derived_attributes_cost \
//!     -- --ignored --nocapture
//! ```
//!
//! `SDM_BENCH_GAS_LIMIT` overrides the block gas limit (default 30M); `SDM_BENCH_RUNS` the number
//! of measured runs per variant (default 5).

use alloy_consensus::{Sealable, SignableTransaction, Signed, TxEip1559};
use alloy_eips::eip2718::Encodable2718;
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_network::TxSignerSync;
use alloy_primitives::{Address, B64, B256, Bytes, Signature, TxKind, U256, hex};
use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated};
use jsonrpsee::{
    core::{ClientError, client::ClientT},
    rpc_params,
};
use op_alloy_consensus::{OpTxEnvelope, SDMGasEntry, build_post_exec_tx, encode_jovian_extra_data};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV4;
use reth_chainspec::{BaseFeeParams, EthChainSpec};
use reth_e2e_test_utils::wallet::Wallet;
use reth_node_builder::{Node, NodeBuilder, NodeConfig};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_optimism_payload_builder::OpPayloadAttrs;
use reth_tasks::{RuntimeBuilder, RuntimeConfig, TokioConfig};
use std::{
    sync::Arc,
    time::{Duration, Instant},
};

const INVALID_PAYLOAD_ATTRIBUTES: i32 = -38003;
const SENDERS: usize = 20;
/// Below the EIP-7825 per-transaction cap.
const CALL_GAS: u64 = 10_000_000;
const MAX_FEE: u128 = 1_000_000_000_000;

/// Writes fresh slots from `calldata[0..32]` upward until less than 30k gas is left.
const SSTORE_LOOP: Address = Address::with_last_byte(0xA1);
const SSTORE_LOOP_CODE: &[u8] = &hex!("6000355b6175305a10601657600181556001016003565b00");
/// Re-hashes `mem[0..32]` until less than 5k gas is left.
const KECCAK_LOOP: Address = Address::with_last_byte(0xA2);
const KECCAK_LOOP_CODE: &[u8] = &hex!("5b6113885a1060145760206000206000526000565b00");
/// Copies a 128-byte ecrecover input from calldata, then calls the precompile until less than
/// 5k gas is left.
const ECRECOVER_LOOP: Address = Address::with_last_byte(0xA3);
const ECRECOVER_LOOP_CODE: &[u8] =
    &hex!("608060006000375b6113885a10602057602060806080600060015afa506007565b00");

#[derive(Clone, Copy, PartialEq, Eq)]
enum PostExec {
    /// No 0x7D: a payload job builds the block after FCU returns.
    Absent,
    /// A valid 0x7D: FCU builds the block while it creates the payload job.
    Valid,
    /// An invalid 0x7D: FCU fails when the build reaches the bad refund. It targets the last
    /// transaction, so the whole block executes first, as a malicious batcher would arrange.
    Invalid,
}

#[derive(Default)]
struct Samples {
    fcu: Vec<Duration>,
    get_payload: Vec<Duration>,
}

struct Workload {
    name: &'static str,
    transactions: Vec<Bytes>,
    /// A call `eth_call` must run without an exceptional halt, so a broken loop can't pass as a
    /// full block.
    sanity_call: Option<serde_json::Value>,
}

fn sign(signer: &impl TxSignerSync<Signature>, mut tx: TxEip1559) -> Signed<TxEip1559> {
    let signature = signer.sign_transaction_sync(&mut tx).expect("signing never fails");
    tx.into_signed(signature)
}

fn encoded(tx: Signed<TxEip1559>) -> Bytes {
    OpTxEnvelope::Eip1559(tx).encoded_2718().into()
}

async fn timed_fcu(
    client: &impl ClientT,
    head: ForkchoiceState,
    attrs: &OpPayloadAttrs,
) -> (Duration, Result<ForkchoiceUpdated, ClientError>) {
    let start = Instant::now();
    let result =
        client.request("engine_forkchoiceUpdatedV3", rpc_params![head, attrs.0.clone()]).await;
    (start.elapsed(), result)
}

fn post_exec(entries: Vec<SDMGasEntry>) -> Bytes {
    OpTxEnvelope::PostExec(build_post_exec_tx(1, entries).seal_slow()).encoded_2718().into()
}

fn env_or<T: std::str::FromStr>(key: &str, default: T) -> T {
    std::env::var(key).ok().and_then(|v| v.parse().ok()).unwrap_or(default)
}

fn median(samples: &[Duration]) -> Duration {
    let mut sorted = samples.to_vec();
    sorted.sort();
    sorted[sorted.len() / 2]
}

fn ms(d: Duration) -> String {
    format!("{:.1}", d.as_secs_f64() * 1000.0)
}

#[tokio::test(flavor = "multi_thread")]
#[ignore = "measurement; run in release with --ignored --nocapture"]
async fn derived_block_latency_on_full_blocks() -> eyre::Result<()> {
    let gas_limit: u64 = env_or("SDM_BENCH_GAS_LIMIT", 30_000_000);
    let runs: usize = env_or("SDM_BENCH_RUNS", 5);

    let mut genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    genesis.extra_data = encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism(), 0)?;
    genesis.base_fee_per_gas = Some(1_000_000_000);
    for (address, code) in [
        (SSTORE_LOOP, SSTORE_LOOP_CODE),
        (KECCAK_LOOP, KECCAK_LOOP_CODE),
        (ECRECOVER_LOOP, ECRECOVER_LOOP_CODE),
    ] {
        genesis
            .alloc
            .insert(address, GenesisAccount::default().with_code(Some(Bytes::from_static(code))));
    }
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build(),
    );
    let chain_id = chain_spec.chain_id();
    let genesis_header = chain_spec.genesis_header().clone();

    let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    // Production thread pools, not the 2-thread test runtime.
    let runtime = RuntimeBuilder::new(
        RuntimeConfig::default()
            .with_tokio(TokioConfig::existing_handle(tokio::runtime::Handle::current())),
    )
    .build()?;
    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(runtime)
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;
    let client = node.auth_server_handle().http_client();
    let head = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    let signers = Wallet::new(SENDERS).with_chain_id(chain_id).wallet_gen();
    let eip1559 = |i: usize, to: Address, gas_limit: u64, input: Bytes| {
        let tx = TxEip1559 {
            chain_id,
            nonce: (i / SENDERS) as u64,
            gas_limit,
            max_fee_per_gas: MAX_FEE,
            to: TxKind::Call(to),
            value: U256::from(1),
            input,
            ..Default::default()
        };
        sign(&signers[i % SENDERS], tx)
    };
    let calls = |to: Address, input: &dyn Fn(usize) -> Bytes| -> Vec<Bytes> {
        (0..(gas_limit / CALL_GAS) as usize)
            .map(|i| encoded(eip1559(i, to, CALL_GAS, input(i))))
            .collect()
    };
    let sanity = |to: Address, input: Bytes| {
        serde_json::json!({
            "from": signers[0].address(),
            "to": to,
            "gas": format!("{CALL_GAS:#x}"),
            "input": input,
        })
    };

    // A valid ecrecover input: a signature hash and its signature.
    let ecrecover_input = {
        let signed = eip1559(0, Address::ZERO, 21_000, Bytes::new());
        let signature = signed.signature();
        let mut input = signed.signature_hash().to_vec();
        input.extend(B256::from(U256::from(27 + u8::from(signature.v()))));
        input.extend(B256::from(signature.r()));
        input.extend(B256::from(signature.s()));
        Bytes::from(input)
    };
    let slot_base = |i: usize| Bytes::from(B256::from(U256::from(i + 1) << 128).to_vec());

    let workloads = [
        Workload {
            name: "ETH transfers to new accounts",
            transactions: (0..(gas_limit / 21_000) as usize)
                .map(|i| {
                    let to = Address::from_word(B256::from(U256::from(0x1_0000 + i)));
                    encoded(eip1559(i, to, 21_000, Bytes::new()))
                })
                .collect(),
            sanity_call: None,
        },
        Workload {
            name: "SSTORE to fresh slots",
            transactions: calls(SSTORE_LOOP, &slot_base),
            sanity_call: Some(sanity(SSTORE_LOOP, slot_base(0))),
        },
        Workload {
            name: "KECCAK256 loop",
            transactions: calls(KECCAK_LOOP, &|_| Bytes::new()),
            sanity_call: Some(sanity(KECCAK_LOOP, Bytes::new())),
        },
        Workload {
            name: "ecrecover precompile loop",
            transactions: calls(ECRECOVER_LOOP, &|_| ecrecover_input.clone()),
            sanity_call: Some(sanity(ECRECOVER_LOOP, ecrecover_input.clone())),
        },
    ];

    let derived = |transactions: Vec<Bytes>, run: usize| {
        let mut attrs = optimism_payload_attributes(genesis_header.timestamp + 2);
        attrs.0.no_tx_pool = Some(true);
        attrs.0.eip_1559_params = Some(B64::ZERO);
        attrs.0.min_base_fee = Some(0);
        attrs.0.gas_limit = Some(gas_limit);
        // A fresh payload id per run, so no run resolves an earlier run's job.
        attrs.0.payload_attributes.prev_randao = B256::from(U256::from(run));
        attrs.0.transactions = Some(transactions);
        attrs
    };

    println!(
        "\nblock gas limit {gas_limit}, median of {runs} runs, times in ms\n\
         FCU = engine_forkchoiceUpdatedV3 round trip; getPayload = engine_getPayloadV5 round trip\n"
    );
    println!(
        "| workload | txs | gas used | FCU, no 0x7D | FCU, valid 0x7D | FCU, invalid 0x7D | \
         getPayload, no 0x7D | getPayload, valid 0x7D | FCU+getPayload, no 0x7D | \
         FCU+getPayload, valid 0x7D |"
    );
    println!("|---|---|---|---|---|---|---|---|---|---|");

    for workload in workloads {
        if let Some(call) = &workload.sanity_call {
            let output: Result<Bytes, ClientError> =
                client.request("eth_call", rpc_params![call, "latest"]).await;
            assert!(output.is_ok(), "{}: loop halts exceptionally: {output:?}", workload.name);
        }

        let variants = [PostExec::Absent, PostExec::Valid, PostExec::Invalid];
        let mut samples: [Samples; 3] = Default::default();
        let mut gas_used = 0;
        // Run 0 warms up and is discarded.
        for run in 0..=runs {
            for (variant, samples) in variants.iter().zip(samples.iter_mut()) {
                let mut transactions = workload.transactions.clone();
                match variant {
                    PostExec::Absent => {}
                    PostExec::Valid => {
                        transactions.push(post_exec(vec![SDMGasEntry { index: 0, gas_refund: 1 }]))
                    }
                    PostExec::Invalid => transactions.push(post_exec(vec![SDMGasEntry {
                        index: workload.transactions.len() as u64 - 1,
                        gas_refund: gas_limit,
                    }])),
                }
                let attrs = derived(transactions, run * variants.len() + *variant as usize);

                let (fcu_time, result) = timed_fcu(&client, head, &attrs).await;
                if *variant == PostExec::Invalid {
                    match result {
                        Err(ClientError::Call(err)) => assert_eq!(
                            err.code(),
                            INVALID_PAYLOAD_ATTRIBUTES,
                            "{}: unexpected error {err:?}",
                            workload.name
                        ),
                        other => panic!("{}: expected -38003, got {other:?}", workload.name),
                    }
                    if run > 0 {
                        samples.fcu.push(fcu_time);
                    }
                    continue;
                }

                let updated = result?;
                assert!(updated.payload_status.is_valid(), "{}: {updated:?}", workload.name);
                let id = updated.payload_id.expect("derived attributes start a payload job");
                let start = Instant::now();
                let payload: OpExecutionPayloadEnvelopeV4 =
                    client.request("engine_getPayloadV5", rpc_params![id]).await?;
                let get_payload_time = start.elapsed();
                let block = &payload.execution_payload.payload_inner.payload_inner.payload_inner;
                assert_eq!(
                    block.transactions.len(),
                    attrs.0.transactions.as_ref().map_or(0, Vec::len),
                    "{}: the payload job dropped transactions",
                    workload.name
                );
                gas_used = block.gas_used;
                if run > 0 {
                    samples.fcu.push(fcu_time);
                    samples.get_payload.push(get_payload_time);
                }
            }
        }
        assert!(
            gas_used >= gas_limit * 9 / 10,
            "{}: block is not full ({gas_used} of {gas_limit})",
            workload.name
        );

        let [none, valid, invalid] = &samples;
        let total = |s: &Samples| -> Vec<Duration> {
            s.fcu.iter().zip(&s.get_payload).map(|(a, b)| *a + *b).collect()
        };
        println!(
            "| {} | {} | {gas_used} | {} | {} | {} | {} | {} | {} | {} |",
            workload.name,
            workload.transactions.len(),
            ms(median(&none.fcu)),
            ms(median(&valid.fcu)),
            ms(median(&invalid.fcu)),
            ms(median(&none.get_payload)),
            ms(median(&valid.get_payload)),
            ms(median(&total(none))),
            ms(median(&total(valid))),
        );
    }
    println!();

    Ok(())
}
