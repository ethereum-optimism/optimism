//! Relation fixtures and the WP3-owned negative cases.
//!
//! The fixtures are a tiny synthetic private chain over a synthetic L1. Private attributes are
//! assembled here by hand from the L1 headers and receipts (`L1BlockInfoTx` + `decode_deposit`),
//! independently of the relation's in-guest `StatefulAttributesBuilder`; any divergence shows up
//! as a different executed block. Expected public values come from admission's own path,
//! `validate_projection_range` + `public_values` over the fixture's public data, never from
//! `execute`.

use super::*;
use alloy_consensus::{Eip658Value, Receipt, ReceiptEnvelope, Signed, TxEip1559};
use alloy_eips::{
    eip2935::{HISTORY_STORAGE_ADDRESS, HISTORY_STORAGE_CODE},
    eip4788::{BEACON_ROOTS_ADDRESS, BEACON_ROOTS_CODE},
};
use alloy_primitives::{Address, Log, LogData, Signature, address, hex};
use alloy_rpc_types_engine::PayloadAttributes;
use alloy_sol_types::{SolEvent, SolValue};
use alloy_trie::{EMPTY_ROOT_HASH, TrieAccount};
use kona_genesis::{
    ChainDependency, HardForkConfig, Predeploys, PrivateProjectionConfig, SystemConfig,
};
use kona_interop::ExecutingMessage;
use kona_mpt::{Nibbles, NoopTrieProvider, TrieNode, TrieProvider};
use kona_protocol::{DEPOSIT_EVENT_ABI_HASH, SingleBatch, decode_deposit};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::sync::Mutex;

pub(super) const CHAIN: u64 = 901;
/// Fixture `program_vkey` (a canonical BN254 scalar; the devstack `native-mock` value).
pub(super) const FIXTURE_VKEY: B256 = B256::with_last_byte(1);
const MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const INBOX: Address = address!("4200000000000000000000000000000000000022");
const REGISTRY: Address = address!("420000000000000000000000000000000000002e");
const DEPOSIT_CONTRACT: Address = address!("00000000000000000000000000000000000000dc");
const SYSTEM_CONFIG: Address = address!("00000000000000000000000000000000000005c0");
const BATCHER: Address = address!("00000000000000000000000000000000000000ba");
/// Index of the one valid user-deposit log in every fixture L1 block (§D.2.4 log counting).
const DEPOSIT_LOG_INDEX: usize = 2;

/// A fixture: relation input, witness, and admission-computed expected public values.
pub(super) type Fixture = (RelationInput, Witness, PublicValues);

pub(super) fn signed(to: Address, data: Bytes, nonce: u64) -> Bytes {
    let access_list = if to == INBOX {
        projection::validateMessageCall::abi_decode_validate(&data)
            .ok()
            .map(|call| {
                vec![alloy_eips::eip2930::AccessListItem {
                    address: INBOX,
                    storage_keys: projection::import_keys(&call).unwrap(),
                }]
                .into()
            })
            .unwrap_or_default()
    } else {
        Default::default()
    };
    TxEnvelope::Eip1559(Signed::new_unhashed(
        TxEip1559 {
            chain_id: CHAIN,
            nonce,
            gas_limit: 500_000,
            to: to.into(),
            input: data,
            access_list,
            ..Default::default()
        },
        Signature::test_signature(),
    ))
    .encoded_2718()
    .into()
}

fn save_trie(node: &TrieNode, out: &mut BTreeMap<B256, Bytes>) {
    let raw = alloy_rlp::encode(node);
    out.insert(keccak256(&raw), raw.into());
    match node {
        TrieNode::Branch { stack } => {
            for child in stack {
                save_trie(child, out);
            }
        }
        TrieNode::Extension { node, .. } => save_trie(node, out),
        _ => {}
    }
}

/// The L1 origin of private block `n`: two L2 blocks per epoch, starting after genesis epoch 100.
pub(super) const fn epoch_of(n: u64) -> u64 {
    100 + n.div_ceil(2)
}

pub(super) fn deposit_log(from: Address, to: Address, mint: u128, value: u64, gas: u64) -> Log {
    let mut opaque = U256::from(mint).to_be_bytes::<32>().to_vec();
    opaque.extend_from_slice(&U256::from(value).to_be_bytes::<32>());
    opaque.extend_from_slice(&gas.to_be_bytes());
    opaque.push(0);
    Log {
        address: DEPOSIT_CONTRACT,
        data: LogData::new_unchecked(
            vec![DEPOSIT_EVENT_ABI_HASH, from.into_word(), to.into_word(), B256::ZERO],
            (Bytes::from(opaque),).abi_encode_params().into(),
        ),
    }
}

pub(super) fn receipt(cumulative_gas_used: u64, logs: Vec<Log>) -> Bytes {
    ReceiptEnvelope::Eip1559(
        Receipt { status: Eip658Value::Eip658(true), cumulative_gas_used, logs }.with_bloom(),
    )
    .encoded_2718()
    .into()
}

/// The fixture L1 block `number`'s receipts: noise, a deposit-topic log from the wrong address,
/// and one genuine user deposit (log index 2).
pub(super) fn l1_receipts(number: u64) -> Vec<Bytes> {
    if number == 100 {
        return vec![];
    }
    vec![
        receipt(
            21_000,
            vec![
                Log {
                    address: Address::with_last_byte(0x77),
                    data: LogData::new_unchecked(vec![B256::repeat_byte(0x33)], Bytes::new()),
                },
                Log {
                    address: Address::with_last_byte(0x78),
                    ..deposit_log(BATCHER, BATCHER, 9, 9, 9)
                },
            ],
        ),
        receipt(
            63_000,
            vec![deposit_log(
                Address::with_last_byte(0x50),
                Address::with_last_byte(0x51),
                100,
                100,
                100_000,
            )],
        ),
    ]
}

/// A header of the fixture L1 chain, committing to `receipts`.
pub(super) fn l1_header(number: u64, parent_hash: B256, receipts: &[Bytes]) -> Header {
    Header {
        parent_hash,
        number,
        timestamp: 996 + 4 * (number - 100),
        mix_hash: keccak256(number.to_be_bytes()),
        base_fee_per_gas: Some(7),
        parent_beacon_block_root: Some(B256::repeat_byte(0xbe)),
        excess_blob_gas: Some(0),
        blob_gas_used: Some(0),
        receipts_root: ordered_list_preimages(receipts).0,
        withdrawals_root: Some(EMPTY_ROOT_HASH),
        gas_limit: 30_000_000,
        ..Default::default()
    }
}

/// Fixture L1 chain `100..=last`.
pub(super) fn l1_chain(last: u64) -> BTreeMap<u64, Header> {
    let mut out = BTreeMap::new();
    let mut parent = B256::repeat_byte(0x10);
    for number in 100..=last {
        let header = l1_header(number, parent, &l1_receipts(number));
        parent = header.hash_slow();
        out.insert(number, header);
    }
    out
}

/// Add an L1 block's header and receipts-trie nodes to the witness.
pub(super) fn add_l1_block(header: &Header, receipts: &[Bytes], out: &mut BTreeMap<B256, Bytes>) {
    out.insert(header.hash_slow(), alloy_rlp::encode(header).into());
    let (root, nodes) = ordered_list_preimages(receipts);
    assert_eq!(root, header.receipts_root);
    for node in nodes {
        out.insert(keccak256(&node), node);
    }
}

/// A preimage store the fixture's own executor shares with the fixture, so the storage-trie nodes
/// that later blocks re-read (a real chain's `debug_executionWitness` reports them) can be added
/// between blocks.
#[derive(Debug, Clone)]
struct SharedStore(Arc<Mutex<BTreeMap<B256, Bytes>>>);

impl TrieProvider for SharedStore {
    type Error = anyhow::Error;
    fn trie_node_by_hash(&self, hash: B256) -> Result<TrieNode> {
        Store(&self.0.lock().unwrap()).trie_node_by_hash(hash)
    }
}

impl kona_executor::TrieDBProvider for SharedStore {
    fn bytecode_by_hash(&self, hash: B256) -> Result<Bytes> {
        Store(&self.0.lock().unwrap()).bytecode_by_hash(hash)
    }
    fn header_by_hash(&self, hash: B256) -> Result<Header> {
        Store(&self.0.lock().unwrap()).header(hash)
    }
}

/// The storage the pre-execution system calls write: EIP-4788 (timestamp and beacon root in a
/// 8191-slot ring) and EIP-2935 (parent hash at `(number - 1) % 8191`).
#[derive(Debug, Default)]
struct SystemStorage {
    beacon_roots: BTreeMap<U256, U256>,
    history: BTreeMap<U256, U256>,
}

impl SystemStorage {
    fn record(&mut self, number: u64, timestamp: u64, beacon_root: B256, parent: B256) {
        let ts = U256::from(timestamp % 8191);
        self.beacon_roots.insert(ts, U256::from(timestamp));
        self.beacon_roots.insert(ts + U256::from(8191), U256::from_be_bytes(beacon_root.0));
        self.history.insert(U256::from((number - 1) % 8191), U256::from_be_bytes(parent.0));
    }

    fn tries(&self) -> [TrieNode; 2] {
        [&self.beacon_roots, &self.history].map(|slots| {
            let mut trie = TrieNode::Empty;
            for (slot, value) in slots {
                trie.insert(
                    &Nibbles::unpack(keccak256(slot.to_be_bytes::<32>())),
                    alloy_rlp::encode(value).into(),
                    &NoopTrieProvider,
                )
                .unwrap();
            }
            trie
        })
    }

    fn save(&self, out: &mut BTreeMap<B256, Bytes>) {
        for trie in self.tries() {
            save_trie(&trie, out);
        }
    }
}

fn scalar(base: u32, blob: u32) -> U256 {
    let mut word = [0u8; 32];
    word[0] = 1;
    word[24..28].copy_from_slice(&blob.to_be_bytes());
    word[28..32].copy_from_slice(&base.to_be_bytes());
    U256::from_be_bytes(word)
}

/// The projection's fee override of a private system config (`stateful.rs`).
fn projected(sys: &SystemConfig) -> SystemConfig {
    SystemConfig {
        scalar: U256::from(1) << 248,
        overhead: U256::ZERO,
        operator_fee_scalar: Some(0),
        operator_fee_constant: Some(0),
        min_base_fee: Some(0),
        gas_limit: i64::MAX as u64,
        ..*sys
    }
}

pub(super) fn fixture(replacements: usize) -> Fixture {
    fixture_with_sdm(replacements, false)
}

/// A real EVM range with an exported and an imported message, over `replacements` canonical
/// recovery blocks. Every block at an epoch change carries a forced user deposit from L1.
pub(super) fn fixture_with_sdm(replacements: usize, sdm: bool) -> Fixture {
    fixture_at(0, replacements, sdm)
}

/// Public hash of the canonical projection block carrying a non-genesis anchor's record.
pub(super) const PUBLIC_ANCHOR: B256 = B256::repeat_byte(0xa7);

/// As [`fixture_with_sdm`], anchored at private block `anchor` (0 = the private genesis). A
/// non-genesis anchor is a synthetic private block over the genesis state, carrying the L1-info
/// deposit of its epoch, so the relation's anchor path (transactions root, `L2BlockInfo`, system
/// config reconstruction) is exercised.
pub(super) fn fixture_at(anchor: u64, replacements: usize, sdm: bool) -> Fixture {
    let event = projection::SentMessage {
        destination: U256::from(100),
        target: Address::with_last_byte(8),
        messageNonce: U256::from(7),
        sender: Address::with_last_byte(9),
        message: Bytes::from_static(b"private hello"),
    };
    let log = event.encode_log_data();
    // Copy calldata, then emit LOG4 with the exact messenger event's topics.
    let mut code = vec![0x36, 0x5f, 0x5f, 0x37];
    for topic in log.topics().iter().rev() {
        code.push(0x7f);
        code.extend_from_slice(topic.as_slice());
    }
    code.extend_from_slice(&[0x36, 0x5f, 0xa4, 0x00]);
    let private_tx = signed(MESSENGER, log.data.clone(), 0);
    use alloy_consensus::transaction::SignerRecoverable;
    let sender = TxEnvelope::decode_2718_exact(&private_tx).unwrap().recover_signer().unwrap();
    let import = ExecutingMessage {
        payloadHash: B256::repeat_byte(0x42),
        identifier: kona_interop::MessageIdentifier {
            origin: MESSENGER,
            blockNumber: U256::from(50),
            logIndex: U256::from(2),
            timestamp: U256::from(900),
            chainId: U256::from(100),
        },
    }
    .encode_log_data();
    let mut inbox_code = vec![0x36, 0x5f, 0x5f, 0x37];
    for topic in import.topics().iter().rev() {
        inbox_code.push(0x7f);
        inbox_code.extend_from_slice(topic.as_slice());
    }
    inbox_code.extend_from_slice(&[0x36, 0x5f, 0xa2, 0x00]);
    let import_tx = signed(INBOX, import.data.clone(), 0);
    let importer = TxEnvelope::decode_2718_exact(&import_tx).unwrap().recover_signer().unwrap();
    let mut trie = TrieNode::Empty;
    for (addr, account) in [
        (
            importer,
            TrieAccount { balance: U256::from(1_000_000_000_000_000_000u64), ..Default::default() },
        ),
        (INBOX, TrieAccount { code_hash: keccak256(&inbox_code), ..Default::default() }),
        (
            sender,
            TrieAccount { balance: U256::from(1_000_000_000_000_000_000u64), ..Default::default() },
        ),
        (
            address!("4200000000000000000000000000000000000016"),
            TrieAccount { nonce: 1, ..Default::default() },
        ),
        (MESSENGER, TrieAccount { code_hash: keccak256(&code), ..Default::default() }),
        // The pre-execution system calls of every block, deposit-only recovery blocks included.
        (
            BEACON_ROOTS_ADDRESS,
            TrieAccount { code_hash: keccak256(&BEACON_ROOTS_CODE), ..Default::default() },
        ),
        (
            HISTORY_STORAGE_ADDRESS,
            TrieAccount { code_hash: keccak256(&HISTORY_STORAGE_CODE), ..Default::default() },
        ),
    ] {
        trie.insert(
            &Nibbles::unpack(keccak256(addr)),
            alloy_rlp::encode(account).into(),
            &NoopTrieProvider,
        )
        .unwrap();
    }
    let header = Header {
        state_root: trie.blind(),
        number: 0,
        timestamp: 1000,
        gas_limit: 30_000_000,
        base_fee_per_gas: Some(0),
        withdrawals_root: Some(EMPTY_ROOT_HASH),
        extra_data: if sdm {
            hex!("01000000fa000000060000000000000000").into()
        } else {
            hex!("00000000fa00000006").into()
        },
        ..Default::default()
    };
    let genesis_root = output(&header).unwrap();
    let count = replacements as u64 + 2;
    let l1 = l1_chain(epoch_of(anchor + count));
    let sys = SystemConfig {
        batcher_address: BATCHER,
        overhead: U256::ZERO,
        scalar: scalar(17, 19),
        gas_limit: 30_000_000,
        base_fee_scalar: Some(17),
        blob_base_fee_scalar: Some(19),
        eip1559_denominator: Some(250),
        eip1559_elasticity: Some(6),
        operator_fee_scalar: Some(23),
        operator_fee_constant: Some(29),
        min_base_fee: sdm.then_some(0),
        da_footprint_gas_scalar: sdm.then_some(400),
    };
    let mut private_cfg = RollupConfig {
        block_time: 2,
        l1_chain_id: 900,
        l2_chain_id: CHAIN.into(),
        deposit_contract_address: DEPOSIT_CONTRACT,
        l1_system_config_address: SYSTEM_CONFIG,
        hardforks: HardForkConfig {
            regolith_time: Some(0),
            canyon_time: Some(0),
            delta_time: Some(0),
            ecotone_time: Some(0),
            fjord_time: Some(0),
            granite_time: Some(0),
            holocene_time: Some(0),
            isthmus_time: Some(0),
            jovian_time: sdm.then_some(0),
            karst_time: sdm.then_some(0),
            lagoon_time: sdm.then_some(0),
            ..Default::default()
        },
        ..Default::default()
    };
    private_cfg.genesis.l2_time = 1000;
    private_cfg.genesis.l2 = BlockNumHash { number: 0, hash: header.hash_slow() };
    private_cfg.genesis.l1 = BlockNumHash { number: 100, hash: l1[&100].hash_slow() };
    private_cfg.genesis.system_config = Some(sys);
    let l1_cfg = L1ChainConfig { chain_id: 900, ..Default::default() };
    let dep_set = DependencySet {
        dependencies: [(CHAIN, ChainDependency {}), (902, ChainDependency {})].into(),
        override_message_expiry_window: None,
    };
    let private_config: Bytes = serde_json::to_vec(&private_cfg).unwrap().into();
    let l1_config: Bytes = serde_json::to_vec(&l1_cfg).unwrap().into();
    let mut cfg = private_cfg.clone();
    cfg.genesis.l2.hash = B256::repeat_byte(0x88);
    cfg.private_projection = Some(PrivateProjectionConfig {
        verifier: projection::SP1_PRIVATE_PROJECTION_V1.into(),
        allow_events: false,
        genesis_output_root: genesis_root,
        program_vkey: FIXTURE_VKEY,
        private_config_hash: projection::private_config_hash(&private_config, &l1_config),
        dependency_set_hash: projection::dependency_set_hash([U256::from(CHAIN), U256::from(902)]),
        mock_proofs: false,
    });
    let mut preimages = BTreeMap::new();
    save_trie(&trie, &mut preimages);
    preimages.insert(keccak256(&code), code.into());
    preimages.insert(keccak256(&inbox_code), inbox_code.into());
    preimages.insert(keccak256(&BEACON_ROOTS_CODE), BEACON_ROOTS_CODE.clone());
    preimages.insert(keccak256(&HISTORY_STORAGE_CODE), HISTORY_STORAGE_CODE.clone());
    preimages.insert(EMPTY_ROOT_HASH, Bytes::from_static(&[0x80]));
    for number in 101..=epoch_of(anchor + count) {
        add_l1_block(&l1[&number], &l1_receipts(number), &mut preimages);
    }
    let seq_of = |n: u64| u64::from(n > 0 && epoch_of(n) == epoch_of(n - 1));
    let (header, anchor_transactions, public_anchor) = if anchor == 0 {
        (header, vec![], cfg.genesis.l2)
    } else {
        let time = 1000 + anchor * 2;
        let (_, info) = L1BlockInfoTx::try_new_with_deposit_tx(
            &private_cfg,
            &l1_cfg,
            &sys,
            seq_of(anchor),
            &l1[&epoch_of(anchor)],
            time,
        )
        .unwrap();
        let txs: Vec<Bytes> = vec![info.encoded_2718().into()];
        let anchor_header = Header {
            parent_hash: B256::repeat_byte(0xaa),
            number: anchor,
            timestamp: time,
            transactions_root: tx_root(&txs),
            parent_beacon_block_root: Some(B256::ZERO),
            ..header
        };
        (anchor_header, txs, BlockNumHash { number: anchor, hash: PUBLIC_ANCHOR })
    };
    let root = output(&header).unwrap();
    let mut witness = Witness {
        anchor_header: header.clone(),
        anchor_transactions,
        private_data: Bytes::new(),
        preimages,
    };
    let l1_head = l1[&epoch_of(anchor + count)].hash_slow();
    let mut input = RelationInput {
        private_config,
        l1_config,
        projection_config: serde_json::to_vec(&cfg).unwrap().into(),
        dependency_set: serde_json::to_vec(&dep_set).unwrap().into(),
        parent_hash: public_anchor.hash,
        anchor: public_anchor,
        anchor_output: root,
        recovery: vec![],
        blocks: vec![],
        l1_head,
    };
    let shared = SharedStore(Arc::new(Mutex::new(std::mem::take(&mut witness.preimages))));
    let mut exec = StatelessL2Builder::new(
        &private_cfg,
        PostExecEvmFactoryAdapter::new(ZkvmOpEvmFactory),
        OpAlloyReceiptBuilder::default(),
        shared.clone(),
        NoopTrieHinter,
        header.seal_slow(),
    );
    let mut system = SystemStorage::default();
    let mut private_parent = witness.anchor_header.hash_slow();
    let mut parent_output = root;
    let mut terminal = None;
    let mut seq = seq_of(anchor);
    let mut private_span =
        SpanBatch { genesis_timestamp: 1000, chain_id: CHAIN, ..Default::default() };
    for i in 0..replacements + 2 {
        let n = anchor + i as u64 + 1;
        let time = 1000 + n * 2;
        let epoch = epoch_of(n);
        let l1_header = &l1[&epoch];
        let epoch_hash = l1_header.hash_slow();
        let new_epoch = epoch != epoch_of(n - 1);
        seq = if new_epoch { 0 } else { seq + 1 };
        let (_, info) = L1BlockInfoTx::try_new_with_deposit_tx(
            &private_cfg,
            &l1_cfg,
            &sys,
            seq,
            l1_header,
            time,
        )
        .unwrap();
        let mut deposits: Vec<Bytes> = vec![info.encoded_2718().into()];
        if new_epoch {
            let receipt = ReceiptEnvelope::decode_2718_exact(&l1_receipts(epoch)[1]).unwrap();
            let log = &receipt.logs()[0];
            deposits.push(decode_deposit(epoch_hash, DEPOSIT_LOG_INDEX, log).unwrap());
        }
        if sdm && i < replacements {
            deposits.push(
                OpTxEnvelope::from(op_alloy_consensus::build_post_exec_tx(n, vec![]))
                    .encoded_2718()
                    .into(),
            );
        }
        let attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: time,
                prev_randao: l1_header.mix_hash,
                suggested_fee_recipient: Predeploys::SEQUENCER_FEE_VAULT,
                withdrawals: Some(vec![]),
                parent_beacon_block_root: l1_header.parent_beacon_block_root,
                ..Default::default()
            },
            transactions: Some(deposits.clone()),
            gas_limit: Some(30_000_000),
            no_tx_pool: Some(true),
            eip_1559_params: Some(hex!("000000fa00000006").into()),
            min_base_fee: sdm.then_some(0),
        };
        let mut executing = attrs.clone();
        let mut txs =
            if i == replacements { vec![private_tx.clone(), import_tx.clone()] } else { vec![] };
        if sdm && i >= replacements {
            txs.push(
                OpTxEnvelope::from(op_alloy_consensus::build_post_exec_tx(n, vec![]))
                    .encoded_2718()
                    .into(),
            );
        }
        executing.transactions.as_mut().unwrap().extend(txs.clone());
        if i >= replacements {
            private_span
                .append_singular_batch(
                    SingleBatch {
                        parent_hash: private_parent,
                        epoch_num: epoch,
                        epoch_hash,
                        timestamp: time,
                        transactions: txs,
                    },
                    n,
                )
                .unwrap();
        }
        let result = exec.build_block(executing).unwrap();
        system.record(n, time, l1_header.parent_beacon_block_root.unwrap(), private_parent);
        system.save(&mut shared.0.lock().unwrap());
        let computed = exec.compute_output_root().unwrap();
        if i < replacements {
            // What projection derivation builds: the same L1 inputs under the fee override.
            let (_, public_info) = L1BlockInfoTx::try_new_with_deposit_tx(
                &cfg,
                &l1_cfg,
                &projected(&sys),
                seq,
                l1_header,
                time,
            )
            .unwrap();
            let mut public_txs = deposits.clone();
            public_txs[0] = public_info.encoded_2718().into();
            assert_ne!(public_txs[0], deposits[0], "private fee settings differ from projection");
            let mut projection_header = result.header.clone().unseal();
            projection_header.gas_limit = i64::MAX as u64;
            projection_header.transactions_root = tx_root(&public_txs);
            projection_header.parent_hash = input.parent_hash;
            input.parent_hash = projection_header.hash_slow();
            input.recovery.push(OpBlock {
                header: projection_header,
                body: alloy_consensus::BlockBody {
                    transactions: public_txs
                        .iter()
                        .map(|raw| OpTxEnvelope::decode_2718_exact(raw).unwrap())
                        .collect(),
                    ..Default::default()
                },
            });
            parent_output = computed;
        } else {
            let mut txs = vec![signed(
                REGISTRY,
                projection::recordOutputCall { outputRoot: computed }.abi_encode().into(),
                1,
            )];
            let receipts = &result.execution_result.receipts;
            let rendered =
                projection::rendered_logs(receipts.iter().flat_map(|r| r.logs())).unwrap();
            for rl in &rendered {
                let (to, data) = projection::replay_calldata(rl).unwrap();
                txs.push(signed(to, data, txs.len() as u64 + 1));
            }
            input.blocks.push(ProjectionBlock { timestamp: time, epoch, transactions: txs });
        }
        private_parent = result.header.hash();
        terminal = Some(result.header);
    }
    let mut raw = Vec::new();
    Batch::Span(private_span).encode(&mut raw).unwrap();
    let compressed =
        miniz_oxide::deflate::compress_to_vec_zlib(&alloy_rlp::encode(Bytes::from(raw)), 6);
    let mut data = vec![0];
    data.extend(Frame::new([0; 16], 0, compressed, true).encode());
    witness.private_data = data.into();
    witness.preimages = std::mem::take(&mut *shared.0.lock().unwrap());
    let terminal = terminal.unwrap();
    let recovery_hash = recovery_hash_of(&input.recovery);
    let claim = projection::RangeClaim {
        version: 2,
        firstBlock: anchor + replacements as u64 + 1,
        lastBlock: anchor + replacements as u64 + 2,
        privateTerminalBlockHash: terminal.hash(),
        privateTerminalParentHash: terminal.parent_hash,
        anchorBlock: anchor,
        anchorOutputRoot: root,
        parentOutputRoot: parent_output,
        recoveryHash: recovery_hash,
        l1Head: l1_head,
        rollupConfigHash: projection::config_hash(&cfg).unwrap(),
        depSetHash: cfg.private_projection.as_ref().unwrap().dependency_set_hash,
        privateDataHash: keccak256(&witness.private_data),
        proof: Bytes::new(),
    };
    input.blocks[0]
        .transactions
        .insert(0, signed(REGISTRY, projection::postClaimCall { claim }.abi_encode().into(), 0));
    let expected = expected_public_values(&input, &cfg);
    (input, witness, expected)
}

/// The recovery commitment admission's `ContextCollector` builds (word 9).
pub(super) fn recovery_hash_of(recovery: &[OpBlock]) -> B256 {
    recovery.iter().rev().fold(B256::ZERO, |h, b| projection::recovery_step(h, b, &encode_all(b)))
}

/// Admission's statement over the fixture's public data (the verifier's independent path).
pub(super) fn expected_public_values(input: &RelationInput, cfg: &RollupConfig) -> PublicValues {
    let span = SpanBatch {
        batches: input
            .blocks
            .iter()
            .map(|b| SpanBatchElement {
                timestamp: b.timestamp,
                epoch_num: b.epoch,
                transactions: b.transactions.clone(),
            })
            .collect(),
        ..Default::default()
    };
    let statement = projection::validate_projection_range(
        cfg,
        projection::ProjectionContext {
            parent_hash: input.parent_hash,
            l1_head: input.l1_head,
            continuation: projection::Continuation {
                anchor: input.anchor,
                output_root: input.anchor_output,
                recovery_hash: recovery_hash_of(&input.recovery),
            },
        },
        &span,
        &projection::StubVerifier,
    )
    .unwrap();
    projection::public_values(&statement)
}

fn write_fixture(name: &str, fixture: &Fixture) {
    if let Ok(dir) = std::env::var("PRIVATE_PROJECTION_FIXTURE_DIR") {
        std::fs::create_dir_all(&dir).unwrap();
        let (input, witness, expected) = fixture;
        std::fs::write(
            format!("{dir}/{name}.json"),
            serde_json::to_vec_pretty(&(input, witness, Bytes::copy_from_slice(expected))).unwrap(),
        )
        .unwrap();
    }
}

fn err(input: &RelationInput, witness: &Witness) -> String {
    format!("{:#}", execute(input, witness).expect_err("relation must reject"))
}

#[test]
fn executes_private_range_and_recovery() {
    for replacements in [0, 1, 3] {
        let fixture = fixture(replacements);
        let (input, witness, expected) = &fixture;
        let pv = execute(input, witness).unwrap();
        assert_eq!(pv, *expected, "native public values equal admission's");
        assert_ne!(public_value_word(&pv, 20), input.anchor_output);
        assert_eq!(pv, execute(input, witness).unwrap());
        write_fixture(&format!("range-{replacements}"), &fixture);
    }
}

/// Deposit-only recovery blocks run the EIP-4788 and EIP-2935 system calls like any other block,
/// so the relation's witness must carry those contracts: without the beacon-roots code the first
/// recovery block fails in its pre-execution system call.
#[test]
fn recovery_blocks_run_system_calls() {
    for (anchor, replacements) in [(0, 3), (5, 2)] {
        let (input, witness, expected) = fixture_at(anchor, replacements, false);
        assert_eq!(execute(&input, &witness).unwrap(), expected);
        for code in [&BEACON_ROOTS_CODE, &HISTORY_STORAGE_CODE] {
            let mut bad = witness.clone();
            bad.preimages.remove(&keccak256(code));
            let e = err(&input, &bad);
            assert!(e.contains(&format!("block {}", anchor + 1)), "{e}");
            assert!(e.contains("missing witness preimage"), "{e}");
        }
        // The beacon-roots storage the first recovery block wrote is re-read by the second one's
        // system call: exactly the preimage a host must ship for deposit-only blocks.
        let mut system = SystemStorage::default();
        let first = anchor + 1;
        system.record(
            first,
            1000 + first * 2,
            B256::repeat_byte(0xbe),
            witness.anchor_header.hash_slow(),
        );
        let [beacon_roots, _] = system.tries();
        let mut bad = witness.clone();
        assert!(bad.preimages.remove(&beacon_roots.blind()).is_some());
        let e = err(&input, &bad);
        assert!(e.contains(&format!("block {}", first + 1)), "{e}");
        assert!(e.contains("beacon root contract call"), "{e}");
    }
}

/// Every span after the first anchors at a non-genesis private block.
#[test]
fn executes_from_non_genesis_anchor() {
    for (anchor, replacements, sdm) in [(5, 0, false), (4, 0, false), (5, 2, false), (6, 3, true)] {
        let fixture = fixture_at(anchor, replacements, sdm);
        let (input, witness, expected) = &fixture;
        assert_eq!(execute(input, witness).unwrap(), *expected, "anchor {anchor}");
        assert_eq!(public_value_word(expected, 6), B256::from(U256::from(anchor)));
        let mut bad = witness.clone();
        bad.anchor_transactions[0] = Bytes::from_static(&[0x7e]);
        assert!(err(input, &bad).contains("private anchor transactions"));
        let mut bad = witness.clone();
        bad.anchor_transactions.clear();
        assert!(err(input, &bad).contains("private anchor transactions"));
        if (anchor, replacements) == (5, 2) {
            write_fixture("anchored-recovery", &fixture);
        }
    }
    // The genesis anchor carries no transactions.
    let (input, mut witness, _) = fixture(0);
    witness.anchor_transactions.push(Bytes::from_static(&[0x7e]));
    assert!(err(&input, &witness).contains("private genesis anchor transactions"));
}

#[test]
fn executes_lagoon_recovery_post_exec() {
    let fixture = fixture_with_sdm(3, true);
    let (input, witness, expected) = &fixture;
    assert_eq!(execute(input, witness).unwrap(), *expected);
    write_fixture("lagoon-recovery", &fixture);
}

#[test]
fn rejects_tampered_execution_and_publication() {
    for replacements in [0, 3] {
        let (input, witness, _) = fixture(replacements);
        let mut bad = witness.clone();
        bad.anchor_header.state_root = B256::ZERO;
        assert!(err(&input, &bad).contains("anchor output"));
        let mut bad = witness.clone();
        bad.private_data = Bytes::from_static(b"tampered");
        assert!(err(&input, &bad).contains("private data commitment"));
        let mut bad = witness.clone();
        for value in bad.preimages.values_mut() {
            *value = Bytes::from_static(b"wrong");
        }
        assert!(execute(&input, &bad).is_err());
        let mut bad = input.clone();
        bad.blocks[1].transactions[0] = signed(
            REGISTRY,
            projection::recordOutputCall { outputRoot: B256::repeat_byte(3) }.abi_encode().into(),
            1,
        );
        assert!(err(&bad, &witness).contains("published checkpoint"));
        let mut bad = input.clone();
        bad.dependency_set =
            Bytes::from_static(br#"{"dependencies":{"901":{},"902":{},"903":{}}}"#);
        assert!(err(&bad, &witness).contains("dependency-set commitment"));
        let mut bad = input.clone();
        let mut projection: serde_json::Value =
            serde_json::from_slice(&bad.projection_config).unwrap();
        projection["block_time"] = 3.into();
        bad.projection_config = serde_json::to_vec(&projection).unwrap().into();
        assert!(err(&bad, &witness).contains("geometry"));
        let mut bad = input.clone();
        let mut projection: serde_json::Value =
            serde_json::from_slice(&bad.projection_config).unwrap();
        projection["private_projection"]["program_vkey"] =
            B256::with_last_byte(2).to_string().into();
        bad.projection_config = serde_json::to_vec(&projection).unwrap().into();
        assert!(err(&bad, &witness).contains("rollup config hash"));
        let mut bad = input.clone();
        bad.parent_hash = B256::repeat_byte(2);
        assert!(err(&bad, &witness).contains("canonical public parent"));
        if replacements > 0 {
            let mut bad = input.clone();
            bad.recovery[0].body.transactions.clear();
            assert!(err(&bad, &witness).contains("transaction root"));
            let mut bad = input.clone();
            bad.recovery.reverse();
            assert!(err(&bad, &witness).contains("recovery ancestry"));
        }
    }
}

/// N10b: the retired prover-supplied attribute path cannot come back through the transport.
#[test]
fn rejects_prover_supplied_attributes() {
    let (input, witness, _) = fixture(0);
    let mut value = serde_json::to_value((&input, &witness)).unwrap();
    value[0]["attributes"] = serde_json::json!([]);
    let err = execute_encoded(&serde_json::to_vec(&value).unwrap()).unwrap_err();
    assert!(err.to_string().contains("unknown field `attributes`"), "{err}");
}

/// A forged `TransactionDeposited` log can enter only by changing an L1 receipts root, which
/// changes the L1 header and breaks the hash chain from `l1Head`, or by swapping trie nodes,
/// which leaves the committed root unresolvable.
#[test]
fn rejects_forged_deposit() {
    for replacements in [0, 3] {
        let (input, witness, _) = fixture(replacements);
        let forged_log = deposit_log(
            Address::with_last_byte(0x66),
            Address::with_last_byte(0x66),
            1_000_000_000,
            0,
            100_000,
        );
        let mut forged = l1_receipts(101);
        forged.push(receipt(84_000, vec![forged_log]));
        let honest = l1_chain(101);
        let header = l1_header(101, honest[&100].hash_slow(), &forged);

        // Forged receipts behind a re-hashed header: nothing links it to l1Head.
        let mut bad = witness.clone();
        add_l1_block(&header, &forged, &mut bad.preimages);
        bad.preimages.remove(&honest[&101].hash_slow());
        let e = err(&input, &bad);
        assert!(e.contains("missing witness preimage"), "{e}");

        // Forged receipt nodes under the honest header: the honest root no longer resolves.
        let mut bad = witness.clone();
        let (_, honest_nodes) = ordered_list_preimages(&l1_receipts(101));
        for node in &honest_nodes {
            bad.preimages.remove(&keccak256(node));
        }
        for node in ordered_list_preimages(&forged).1 {
            bad.preimages.insert(keccak256(&node), node);
        }
        let e = err(&input, &bad);
        assert!(e.contains("receipts trie"), "{e}");

        // A preimage under the wrong key is never accepted.
        let mut bad = witness.clone();
        bad.preimages
            .insert(keccak256(&honest_nodes[0]), ordered_list_preimages(&forged).1[0].clone());
        assert!(err(&input, &bad).contains("incorrect witness preimage"));
    }
    // A forged deposit in a canonical recovery block no longer corresponds to L1.
    let (input, witness, _) = fixture(3);
    let mut bad = input;
    let OpTxEnvelope::Deposit(user) = bad.recovery[0].body.transactions[1].clone() else {
        panic!("fixture recovery block carries a user deposit");
    };
    let mut forged = user.into_inner();
    forged.mint += 1;
    bad.recovery[0].body.transactions[1] = OpTxEnvelope::from(forged);
    bad.recovery[0].header.transactions_root = tx_root(&encode_all(&bad.recovery[0]));
    let mut parent = bad.anchor.hash;
    for block in &mut bad.recovery {
        block.header.parent_hash = parent;
        parent = block.header.hash_slow();
    }
    bad.parent_hash = parent;
    let e = err(&bad, &witness);
    assert!(e.contains("claim continuation mismatch"), "{e}");
}

/// §D.1: a semantic change to the private config changes execution; a byte-only change is
/// still caught by admission, because word 3 is the hash of the exact bytes.
#[test]
fn binds_private_config() {
    let (input, witness, expected) = fixture(1);
    let mut bad = input.clone();
    let mut private: serde_json::Value = serde_json::from_slice(&bad.private_config).unwrap();
    private["genesis"]["system_config"]["operatorFeeScalar"] = 24.into();
    bad.private_config = serde_json::to_vec(&private).unwrap().into();
    // The operator fee enters the private L1-info deposit, so recovery executes differently.
    assert!(err(&bad, &witness).contains("private parent after recovery"));

    let mut bad = input.clone();
    bad.private_config = [bad.private_config.as_ref(), b" "].concat().into();
    let pv = execute(&bad, &witness).unwrap();
    for word in 0..21 {
        assert_eq!(
            public_value_word(&pv, word) == public_value_word(&expected, word),
            word != 3,
            "only privateConfigHash (word 3) differs, at word {word}"
        );
    }

    let mut bad = input;
    bad.l1_config = [bad.l1_config.as_ref(), b" "].concat().into();
    let pv = execute(&bad, &witness).unwrap();
    assert_ne!(public_value_word(&pv, 3), public_value_word(&expected, 3));
}

/// §D.3: the anchor output must be the private anchor header's `OutputV0`.
#[test]
fn rejects_wrong_anchor_output() {
    for replacements in [0, 3] {
        let (input, witness, _) = fixture(replacements);
        let mut bad = input.clone();
        bad.anchor_output = B256::repeat_byte(0x99);
        assert!(err(&bad, &witness).contains("private anchor output"));
        let mut bad = witness.clone();
        bad.anchor_header.timestamp += 2;
        assert!(err(&input, &bad).contains("private anchor timestamp"));
    }
}

/// §D.2.1: `input.l1_head` is the claim's, and the L1 witness hangs off it.
#[test]
fn binds_l1_head() {
    let (input, witness, _) = fixture(1);
    let mut bad = input.clone();
    bad.l1_head = B256::repeat_byte(0x55);
    assert!(err(&bad, &witness).contains("l1 head"));
    let mut bad = witness;
    bad.preimages.remove(&input.l1_head);
    assert!(err(&input, &bad).contains("l1Head"));
}

fn resign(block: &mut ProjectionBlock) {
    for (nonce, raw) in block.transactions.iter_mut().enumerate() {
        let tx = TxEnvelope::decode_2718_exact(raw).unwrap();
        *raw = signed(tx.to().unwrap(), tx.input().clone(), nonce as u64);
    }
}

/// §D.5: an omitted, extra or misplaced replay is rejected by `execute`.
#[test]
fn rejects_incomplete_or_reordered_messages() {
    for replacements in [0, 3] {
        let (input, witness, _) = fixture(replacements);
        // Block 0: claim, record, export replay, import replay.
        assert_eq!(input.blocks[0].transactions.len(), 4);

        let mut bad = input.clone();
        bad.blocks[0].transactions.remove(2);
        assert!(err(&bad, &witness).contains("projection messages differ"), "omitted export");
        let mut bad = input.clone();
        bad.blocks[0].transactions.remove(3);
        assert!(err(&bad, &witness).contains("projection messages differ"), "omitted import");

        let mut bad = input.clone();
        let extra = bad.blocks[0].transactions[3].clone();
        bad.blocks[0].transactions.push(extra);
        resign(&mut bad.blocks[0]);
        assert!(err(&bad, &witness).contains("projection messages differ"), "extra import");

        let mut bad = input.clone();
        bad.blocks[0].transactions.swap(2, 3);
        assert!(err(&bad, &witness).contains("projection messages differ"), "swapped");

        let mut bad = input.clone();
        let moved = bad.blocks[0].transactions.remove(3);
        bad.blocks[1].transactions.push(moved);
        resign(&mut bad.blocks[1]);
        assert!(err(&bad, &witness).contains("projection messages differ"), "moved");

        // Same fields, different message bytes.
        let mut bad = input.clone();
        let tx = TxEnvelope::decode_2718_exact(&bad.blocks[0].transactions[2]).unwrap();
        let mut call = projection::replaySentMessageCall::abi_decode_validate(tx.input()).unwrap();
        call.message = Bytes::from_static(b"private hellO");
        bad.blocks[0].transactions[2] = signed(MESSENGER, call.abi_encode().into(), 2);
        assert!(err(&bad, &witness).contains("projection messages differ"), "altered message");
    }
}

/// Native relation throughput on the `lagoon-recovery` shape with a long recovery interval
/// (spec §G.3a item 2). Run with `--ignored --nocapture`; builds are optimized in `test`.
#[test]
#[ignore = "benchmark"]
fn recovery_throughput() {
    let replacements = 1_000;
    let (input, witness, expected) = fixture_with_sdm(replacements, true);
    let started = std::time::Instant::now();
    assert_eq!(execute(&input, &witness).unwrap(), expected);
    let elapsed = started.elapsed();
    let blocks = replacements + 2;
    eprintln!(
        "native relation: {blocks} blocks ({replacements} recovery) in {elapsed:.2?}: {:.0} blocks/s",
        blocks as f64 / elapsed.as_secs_f64()
    );
}
