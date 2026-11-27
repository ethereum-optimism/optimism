//! Derivation parity with op-node.
//!
//! `testdata/parity_vectors.json` holds inputs to derivation steps that op-node and kona both
//! implement, together with op-node's outputs. op-node's `TestParityVectors`
//! (`op-node/rollup/derive/parity_vectors_test.go`) writes the file and fails when op-node's
//! outputs change. This test runs kona on the same inputs. Each case must either match op-node or
//! be listed in [`KNOWN_DIFFERENCES`], and for a listed case kona's output must still be the one
//! pinned in `testdata/parity_kona_differences.json`. A change to either client therefore fails
//! here until the list, and with it the parity report, is updated. After changing a listed case,
//! rewrite the pinned outputs:
//!
//! ```text
//! UPDATE_PARITY_KONA_DIFFERENCES=1 cargo test -p kona-derive --features serde,test-utils --test parity
//! ```
//!
//! With `DERIVATION_PARITY_REPORT_DIR` set to an absolute path, the test also writes one JSON line
//! per case to `<dir>/vectors.jsonl`, the input to the parity report
//! (`go run ./op-service/testutils/parityreport/cmd <dir>`).

use std::{
    collections::{BTreeMap, HashMap, HashSet},
    fs::OpenOptions,
    io::Write,
    path::PathBuf,
    sync::Arc,
};

use alloy_consensus::{BlockBody, Eip658Value, Header, Receipt};
use alloy_eips::eip2718::{Decodable2718, Encodable2718};
use alloy_primitives::{Address, B256, Bytes, Log, hex};
use alloy_rlp::Decodable;
use kona_derive::{
    BatchStreamProvider, ChannelReader, PipelineError, PipelineErrorKind,
    test_utils::TestChannelReaderProvider,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_protocol::{
    Batch, BlockInfo, L1BlockInfoJovian, L1BlockInfoTx, decode_deposit, to_system_config,
};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use serde::{Deserialize, de::IgnoredAny};
use serde_json::{Value, json};

const VECTORS: &str = include_str!("../testdata/parity_vectors.json");

/// kona's outputs on the cases in [`KNOWN_DIFFERENCES`], keyed by group and case.
const KONA_DIFFERENCES_PATH: &str =
    concat!(env!("CARGO_MANIFEST_DIR"), "/testdata/parity_kona_differences.json");

/// The derivation steps the vectors cover.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
enum Group {
    ChannelReader,
    L1InfoDeposit,
    L1InfoDecoding,
    DepositLog,
    SystemConfigLog,
    SystemConfigFromBlock,
}

impl Group {
    const fn name(self) -> &'static str {
        match self {
            Self::ChannelReader => "channel reader",
            Self::L1InfoDeposit => "L1-info deposit",
            Self::L1InfoDecoding => "L1-info decoding",
            Self::DepositLog => "deposit log",
            Self::SystemConfigLog => "system config log",
            Self::SystemConfigFromBlock => "system config from L2 block",
        }
    }
}

/// A case on which kona's output differs from op-node's. Cases that show the same difference share
/// its description, which the parity report groups by.
struct KnownDifference {
    group: Group,
    case: &'static str,
    difference: &'static str,
}

const fn known(group: Group, case: &'static str, difference: &'static str) -> KnownDifference {
    KnownDifference { group, case, difference }
}

const KNOWN_DIFFERENCES: &[KnownDifference] = &[
    known(Group::ChannelReader, "span batch before Delta", PRE_DELTA_SPAN_BATCH),
    known(
        Group::ChannelReader,
        "brotli channel with a format error after a large second batch",
        BROTLI_FORMAT_ERROR,
    ),
    known(
        Group::ChannelReader,
        "span batch with a deposit transaction, then a valid batch",
        SPAN_BATCH_DERIVATION_FAILURE,
    ),
    known(
        Group::ChannelReader,
        "span batch with a deposit transaction, then a valid batch, before Holocene",
        SPAN_BATCH_DERIVATION_FAILURE,
    ),
    known(
        Group::ChannelReader,
        "span batch with max fee per gas above 128 bits",
        WIDE_SPAN_BATCH_FEE,
    ),
    known(
        Group::ChannelReader,
        "span batch with max priority fee per gas above 128 bits",
        WIDE_SPAN_BATCH_FEE,
    ),
    known(
        Group::ChannelReader,
        "span batch with legacy gas price above 128 bits",
        WIDE_SPAN_BATCH_FEE,
    ),
    known(
        Group::ChannelReader,
        "span batch with access-list gas price above 128 bits",
        WIDE_SPAN_BATCH_FEE,
    ),
    known(
        Group::ChannelReader,
        "span batch with set-code max fee per gas above 128 bits",
        WIDE_SPAN_BATCH_FEE,
    ),
    known(Group::L1InfoDeposit, "unknown ecotone scalar version", INVALID_ECOTONE_SCALAR),
    known(Group::L1InfoDeposit, "ecotone scalar with non-zero padding", INVALID_ECOTONE_SCALAR),
    known(Group::L1InfoDeposit, "custom L1 blob schedule, L1 at cancun", BUILT_IN_BLOB_PARAMS),
    known(Group::L1InfoDeposit, "custom L1 blob schedule, L1 at prague", BUILT_IN_BLOB_PARAMS),
    known(Group::L1InfoDeposit, "custom L1 blob schedule, L1 at osaka", BUILT_IN_BLOB_PARAMS),
    known(Group::L1InfoDeposit, "pectra blob schedule ending after L1 osaka", PECTRA_BLOB_SCHEDULE),
    known(Group::L1InfoDecoding, "bedrock calldata in an ecotone block", L1_INFO_OTHER_FORK),
    known(Group::L1InfoDecoding, "ecotone calldata in a bedrock block", L1_INFO_OTHER_FORK),
    known(Group::L1InfoDecoding, "ecotone calldata in an isthmus block", L1_INFO_OTHER_FORK),
    known(Group::L1InfoDecoding, "isthmus calldata in a jovian block", L1_INFO_OTHER_FORK),
    known(Group::L1InfoDecoding, "jovian calldata in an ecotone block", L1_INFO_OTHER_FORK),
    known(Group::L1InfoDecoding, "bedrock number with non-zero padding", L1_INFO_PADDING),
    known(Group::L1InfoDecoding, "bedrock batcher with non-zero padding", L1_INFO_PADDING),
    known(Group::L1InfoDecoding, "ecotone batcher with non-zero padding", L1_INFO_PADDING),
    known(Group::L1InfoDecoding, "bedrock base fee above 64 bits", L1_INFO_WIDE_FEE),
    known(Group::L1InfoDecoding, "ecotone base fee above 64 bits", L1_INFO_WIDE_FEE),
    known(Group::L1InfoDecoding, "ecotone blob base fee above 128 bits", L1_INFO_WIDE_FEE),
    known(Group::DepositLog, "non-zero padding after the opaque data", DEPOSIT_DATA_PADDING),
    known(Group::DepositLog, "extra word after the opaque data", DEPOSIT_EXTRA_WORD),
    known(Group::DepositLog, "from topic with non-zero padding", DEPOSIT_TOPIC_PADDING),
    known(Group::DepositLog, "to topic with non-zero padding", DEPOSIT_TOPIC_PADDING),
    known(Group::SystemConfigLog, "eip1559 params with a zero denominator", EIP1559_ZERO_VALUE),
    known(Group::SystemConfigLog, "eip1559 params with a zero elasticity", EIP1559_ZERO_VALUE),
    known(Group::SystemConfigLog, "eip1559 params with non-zero padding", EIP1559_PADDING),
    known(Group::SystemConfigLog, "four topics", CONFIG_UPDATE_TOPICS),
    known(Group::SystemConfigLog, "update type with non-zero high bytes", CONFIG_UPDATE_TYPE),
    known(
        Group::SystemConfigFromBlock,
        "pre-holocene block with extra data",
        PRE_HOLOCENE_EXTRA_DATA,
    ),
    known(
        Group::SystemConfigFromBlock,
        "ecotone block with a bedrock L1-info deposit",
        L1_INFO_OTHER_FORK,
    ),
];

// A span batch that decodes but fails span-batch derivation, for example on a deposit
// transaction. A batch that fails to decode at all ends the channel in both clients.
const SPAN_BATCH_DERIVATION_FAILURE: &str = "Span batch that fails span-batch derivation: kona drops the rest of its channel, op-node drops only that batch";
const WIDE_SPAN_BATCH_FEE: &str = "Span batch fee fields above 128 bits: kona keeps only the low 128 bits, op-node keeps the full value";
// op-node hands on the batches its brotli reader has already flushed, which takes enough decoded
// data between them and the error.
const BROTLI_FORMAT_ERROR: &str = "Brotli channel with a format error: kona discards the batches decoded before the error, op-node keeps them";
// Both clients drop the batch; only the stage that drops it differs.
const PRE_DELTA_SPAN_BATCH: &str = "Span batch before Delta: kona passes it on for batch validation to drop, op-node drops it in the channel reader";
const INVALID_ECOTONE_SCALAR: &str = "Invalid Ecotone fee scalar in the system config: kona builds the L1-info deposit, op-node halts derivation";
const BUILT_IN_BLOB_PARAMS: &str = "L1-info blob base fee: kona uses built-in Cancun, Prague and Osaka blob parameters, op-node reads them from the L1 chain config";
const PECTRA_BLOB_SCHEDULE: &str = "Pectra blob schedule override: kona drops it once L1 activates Osaka or a BPO fork, op-node applies it whatever L1 fork is active";
const L1_INFO_OTHER_FORK: &str =
    "L1-info calldata in another fork's layout: kona decodes it, op-node rejects it";
const L1_INFO_PADDING: &str =
    "L1-info calldata with non-zero padding: kona ignores the padding, op-node rejects it";
// Derivation reads neither fee from an L2 block's L1-info deposit.
const L1_INFO_WIDE_FEE: &str = "L1-info base fee above 64 bits or blob base fee above 128 bits: kona keeps only the low bits, op-node keeps the full value";
const DEPOSIT_DATA_PADDING: &str = "TransactionDeposited data with non-zero padding: kona halts derivation, op-node accepts the deposit";
const DEPOSIT_EXTRA_WORD: &str = "TransactionDeposited data with an extra word after the opaque data: kona accepts the deposit, op-node halts derivation";
const DEPOSIT_TOPIC_PADDING: &str = "TransactionDeposited from or to topic with non-zero padding: kona halts derivation, op-node accepts the deposit";
const EIP1559_ZERO_VALUE: &str = "EIP-1559 params update with a zero denominator or elasticity: kona applies it, op-node rejects it";
const EIP1559_PADDING: &str =
    "EIP-1559 params update with non-zero padding: kona rejects it, op-node applies it";
const CONFIG_UPDATE_TOPICS: &str =
    "ConfigUpdate log with more than three topics: kona applies it, op-node rejects it";
const CONFIG_UPDATE_TYPE: &str = "ConfigUpdate type with non-zero high bytes: kona reads the low bytes and applies it, op-node rejects it";
const PRE_HOLOCENE_EXTRA_DATA: &str = "Pre-Holocene L2 block with non-empty extraData: kona rebuilds its system config, op-node rejects the block";

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct Vectors {
    rollup_configs: HashMap<String, RollupConfig>,
    l1_chain_configs: HashMap<String, L1ChainConfig>,
    channel_reader: Vec<Case<ChannelInput>>,
    l1_info_deposit: Vec<Case<L1InfoDepositInput>>,
    l1_info_decode: Vec<Case<L1InfoDecodeInput>>,
    deposit_log: Vec<Case<DepositLogInput>>,
    system_config_log: Vec<Case<SystemConfigLogInput>>,
    system_config_from_block: Vec<Case<SystemConfigFromBlockInput>>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct Case<I> {
    name: String,
    input: I,
    op_node: Value,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ChannelInput {
    rollup_config: String,
    origin_time: u64,
    channel: Bytes,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct L1InfoDepositInput {
    rollup_config: String,
    l1_chain_config: String,
    system_config: SystemConfig,
    sequence_number: u64,
    l1_header: Bytes,
    l2_timestamp: u64,
}

/// kona decodes L1-info calldata by its selector, whatever the block's fork, so it reads neither
/// the rollup config nor the block's timestamp.
#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct L1InfoDecodeInput {
    #[serde(rename = "rollupConfig")]
    _rollup_config: IgnoredAny,
    #[serde(rename = "l2Timestamp")]
    _l2_timestamp: IgnoredAny,
    calldata: Bytes,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct LogInput {
    address: Address,
    topics: Vec<B256>,
    data: Bytes,
}

impl LogInput {
    fn to_log(&self) -> Log {
        Log::new_unchecked(self.address, self.topics.clone(), self.data.clone())
    }
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct DepositLogInput {
    log: LogInput,
    block_hash: B256,
    log_index: usize,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct SystemConfigLogInput {
    rollup_config: String,
    l1_time: u64,
    system_config: SystemConfig,
    log: LogInput,
}

/// An L2 block with only the fields the system config rebuild reads.
#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct SystemConfigFromBlockInput {
    rollup_config: String,
    number: u64,
    timestamp: u64,
    gas_limit: u64,
    extra_data: Bytes,
    l1_info_tx: Bytes,
}

/// The outcome of one case.
struct Outcome {
    group: Group,
    case: String,
    op_node: Value,
    kona: Value,
}

impl Outcome {
    fn new<I>(group: Group, case: &Case<I>, kona: Value) -> Self {
        Self { group, case: case.name.clone(), op_node: case.op_node.clone(), kona }
    }

    fn agrees(&self) -> bool {
        self.op_node == self.kona
    }

    fn key(&self) -> String {
        format!("{} / {}", self.group.name(), self.case)
    }

    fn known_difference(&self) -> Option<&'static str> {
        KNOWN_DIFFERENCES
            .iter()
            .find(|known| known.group == self.group && known.case == self.case)
            .map(|known| known.difference)
    }
}

impl Vectors {
    fn rollup_config(&self, name: &str) -> &RollupConfig {
        self.rollup_configs.get(name).unwrap_or_else(|| panic!("unknown rollup config {name}"))
    }

    fn l1_chain_config(&self, name: &str) -> &L1ChainConfig {
        self.l1_chain_configs.get(name).unwrap_or_else(|| panic!("unknown L1 chain config {name}"))
    }

    async fn outcomes(&self) -> Vec<Outcome> {
        let mut outcomes = Vec::new();
        for case in &self.channel_reader {
            let kona =
                channel_reader(self.rollup_config(&case.input.rollup_config), &case.input).await;
            outcomes.push(Outcome::new(Group::ChannelReader, case, kona));
        }
        push_outcomes(&mut outcomes, Group::L1InfoDeposit, &self.l1_info_deposit, |input| {
            self.l1_info_deposit(input)
        });
        push_outcomes(&mut outcomes, Group::L1InfoDecoding, &self.l1_info_decode, l1_info_decode);
        push_outcomes(&mut outcomes, Group::DepositLog, &self.deposit_log, deposit_log);
        push_outcomes(&mut outcomes, Group::SystemConfigLog, &self.system_config_log, |input| {
            system_config_log(self.rollup_config(&input.rollup_config), input)
        });
        push_outcomes(
            &mut outcomes,
            Group::SystemConfigFromBlock,
            &self.system_config_from_block,
            |input| system_config_from_block(self.rollup_config(&input.rollup_config), input),
        );
        outcomes
    }

    fn l1_info_deposit(&self, input: &L1InfoDepositInput) -> Value {
        let header = Header::decode(&mut input.l1_header.as_ref()).expect("valid L1 header");
        bytes_or_error(
            L1BlockInfoTx::try_new_with_deposit_tx(
                self.rollup_config(&input.rollup_config),
                self.l1_chain_config(&input.l1_chain_config),
                &input.system_config,
                input.sequence_number,
                &header,
                input.l2_timestamp,
            )
            .map(|(_, deposit)| deposit.encoded_2718()),
        )
    }
}

fn push_outcomes<I>(
    outcomes: &mut Vec<Outcome>,
    group: Group,
    cases: &[Case<I>],
    kona: impl Fn(&I) -> Value,
) {
    outcomes.extend(cases.iter().map(|case| Outcome::new(group, case, kona(&case.input))));
}

/// Reads every batch of one channel, as op-node's runner does.
async fn channel_reader(cfg: &RollupConfig, input: &ChannelInput) -> Value {
    let provider = TestChannelReaderProvider {
        data: vec![Ok(Some(input.channel.clone()))],
        block_info: Some(BlockInfo { timestamp: input.origin_time, ..Default::default() }),
        reset: false,
    };
    let mut reader = ChannelReader::new(provider, Arc::new(cfg.clone()));
    let mut batches = Vec::new();
    for _ in 0..1000 {
        match reader.next_batch().await {
            Ok(batch) => batches.push(batch_json(&batch)),
            Err(PipelineErrorKind::Temporary(PipelineError::Eof)) => {
                return json!({ "batches": batches });
            }
            Err(PipelineErrorKind::Temporary(_)) => {}
            Err(_) => return json!({ "batches": batches, "error": true }),
        }
    }
    panic!("the channel reader did not run out of data");
}

fn batch_json(batch: &Batch) -> Value {
    let txs = |txs: &[Bytes]| txs.iter().map(hex::encode_prefixed).collect::<Vec<_>>();
    match batch {
        Batch::Single(batch) => json!({ "singular": {
            "parentHash": hex::encode_prefixed(batch.parent_hash),
            "epochNum": batch.epoch_num,
            "epochHash": hex::encode_prefixed(batch.epoch_hash),
            "timestamp": batch.timestamp,
            "transactions": txs(&batch.transactions),
        }}),
        Batch::Span(batch) => json!({ "span": {
            "parentCheck": hex::encode_prefixed(batch.parent_check),
            "l1OriginCheck": hex::encode_prefixed(batch.l1_origin_check),
            "blocks": batch.batches.iter().map(|block| json!({
                "epochNum": block.epoch_num,
                "timestamp": block.timestamp,
                "transactions": txs(&block.transactions),
            })).collect::<Vec<_>>(),
        }}),
    }
}

/// Decodes the calldata and encodes the value again in the layout it was decoded from. op-node
/// succeeds only on the layout of the block's fork, which is the one kona picks from the selector,
/// so the two encodings are equal exactly when the decoded values are. The L1-info deposit group
/// pins kona's encoders against op-node's.
fn l1_info_decode(input: &L1InfoDecodeInput) -> Value {
    bytes_or_error(
        L1BlockInfoTx::decode_calldata(&input.calldata).map(|info| info.encode_calldata()),
    )
}

fn deposit_log(input: &DepositLogInput) -> Value {
    bytes_or_error(decode_deposit(input.block_hash, input.log_index, &input.log.to_log()))
}

/// Applies a successful receipt with the log to the system config.
fn system_config_log(cfg: &RollupConfig, input: &SystemConfigLogInput) -> Value {
    let mut config = input.system_config;
    let receipt = Receipt {
        status: Eip658Value::Eip658(true),
        cumulative_gas_used: 0,
        logs: vec![input.log.to_log()],
    };
    // A malformed update is skipped, as in op-node, so the config is the outcome to compare.
    let _ = config.update_with_receipts(
        &[receipt],
        cfg.l1_system_config_address,
        cfg.is_ecotone_active(input.l1_time),
    );
    system_config_json(&config)
}

/// Rebuilds the system config from an L2 block, as a pipeline reset does.
fn system_config_from_block(cfg: &RollupConfig, input: &SystemConfigFromBlockInput) -> Value {
    let l1_info_tx =
        OpTxEnvelope::decode_2718(&mut input.l1_info_tx.as_ref()).expect("valid L1-info deposit");
    let block = OpBlock {
        header: Header {
            number: input.number,
            timestamp: input.timestamp,
            gas_limit: input.gas_limit,
            extra_data: input.extra_data.clone(),
            ..Default::default()
        },
        body: BlockBody { transactions: vec![l1_info_tx], ommers: vec![], withdrawals: None },
    };
    to_system_config(&block, cfg).map_or_else(
        |_| error_output(),
        |config| json!({ "systemConfig": system_config_json(&config) }),
    )
}

/// A system config by the values derivation reads from it.
fn system_config_json(config: &SystemConfig) -> Value {
    json!({
        "batcherAddr": hex::encode_prefixed(config.batcher_address),
        "overhead": hex::encode_prefixed(B256::from(config.overhead)),
        "scalar": hex::encode_prefixed(B256::from(config.scalar)),
        "gasLimit": config.gas_limit,
        "eip1559Denominator": config.eip1559_denominator.unwrap_or_default(),
        "eip1559Elasticity": config.eip1559_elasticity.unwrap_or_default(),
        "operatorFeeScalar": config.operator_fee_scalar.unwrap_or_default(),
        "operatorFeeConstant": config.operator_fee_constant.unwrap_or_default(),
        "minBaseFee": config.min_base_fee.unwrap_or_default(),
        // Derivation reads a missing or zero scalar as the default.
        "daFootprintGasScalar": match config.da_footprint_gas_scalar {
            None | Some(0) => L1BlockInfoJovian::DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            Some(scalar) => scalar,
        },
    })
}

fn bytes_or_error<B: AsRef<[u8]>, E>(result: Result<B, E>) -> Value {
    result.map_or_else(|_| error_output(), |bytes| json!({ "bytes": hex::encode_prefixed(bytes) }))
}

fn error_output() -> Value {
    json!({ "error": true })
}

/// Appends one parity record per case to `<DERIVATION_PARITY_REPORT_DIR>/vectors.jsonl`.
fn write_report(outcomes: &[Outcome]) {
    let Some(dir) = std::env::var_os("DERIVATION_PARITY_REPORT_DIR") else { return };
    let dir = PathBuf::from(dir);
    std::fs::create_dir_all(&dir).expect("create the report directory");
    let mut file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(dir.join("vectors.jsonl"))
        .expect("open the report file");
    for outcome in outcomes {
        let mut record = json!({
            "layer": "vectors",
            "group": outcome.group.name(),
            "case": outcome.case,
            "agree": outcome.agrees(),
        });
        if !outcome.agrees() {
            record["divergence"] = json!(outcome.known_difference().unwrap_or("not yet described"));
        }
        writeln!(file, "{record}").expect("write a report record");
    }
}

#[tokio::test]
async fn derivation_parity_with_op_node() {
    let vectors: Vectors = serde_json::from_str(VECTORS).expect("valid parity vectors");
    let outcomes = vectors.outcomes().await;
    write_report(&outcomes);

    let update = std::env::var_os("UPDATE_PARITY_KONA_DIFFERENCES").is_some();
    let pinned: BTreeMap<String, Value> = if update {
        BTreeMap::new()
    } else {
        let data = std::fs::read_to_string(KONA_DIFFERENCES_PATH).expect("read kona's differences");
        serde_json::from_str(&data).expect("valid kona differences")
    };

    let mut failures = Vec::new();
    let mut kona_differences = BTreeMap::new();
    for outcome in &outcomes {
        let key = outcome.key();
        match (outcome.agrees(), outcome.known_difference()) {
            (true, None) => {}
            (true, Some(_)) => failures
                .push(format!("{key}: kona now matches op-node, remove it from KNOWN_DIFFERENCES")),
            (false, None) => failures.push(format!(
                "{key}: kona differs from op-node\n  op-node: {}\n  kona:    {}",
                outcome.op_node, outcome.kona
            )),
            (false, Some(_)) => {
                if !update && pinned.get(&key) != Some(&outcome.kona) {
                    failures.push(format!(
                        "{key}: kona's output changed. Check its description in KNOWN_DIFFERENCES \
                         and rewrite the pinned outputs with UPDATE_PARITY_KONA_DIFFERENCES=1\n  \
                         kona: {}",
                        outcome.kona
                    ));
                }
                kona_differences.insert(key, outcome.kona.clone());
            }
        }
    }
    if update {
        let data = serde_json::to_string_pretty(&kona_differences).expect("encode kona's outputs");
        std::fs::write(KONA_DIFFERENCES_PATH, data + "\n").expect("write kona's differences");
    }
    for key in pinned.keys().filter(|key| !kona_differences.contains_key(*key)) {
        failures
            .push(format!("{key}: pinned in {KONA_DIFFERENCES_PATH} but not a known difference"));
    }
    for known in KNOWN_DIFFERENCES {
        if !outcomes.iter().any(|o| o.group == known.group && o.case == known.case) {
            failures.push(format!(
                "{} / {}: listed in KNOWN_DIFFERENCES but not a case",
                known.group.name(),
                known.case
            ));
        }
    }
    let listed: HashSet<_> = KNOWN_DIFFERENCES.iter().map(|k| (k.group, k.case)).collect();
    if listed.len() != KNOWN_DIFFERENCES.len() {
        failures.push("KNOWN_DIFFERENCES lists a case twice".to_string());
    }

    let differing = outcomes.iter().filter(|o| !o.agrees()).count();
    println!(
        "{} cases: {} agree with op-node, {} differ",
        outcomes.len(),
        outcomes.len() - differing,
        differing
    );
    assert!(failures.is_empty(), "{}", failures.join("\n"));
}
