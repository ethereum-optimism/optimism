//! JSON-RPC surface for the test engine, served over a Unix socket (reth-ipc, go-ethereum
//! `rpc.DialIPC`-compatible) by the companion binary.
//!
//! Three namespaces serve what the `op-e2e/actions` harness drives: `engine_*` (the versioned
//! newPayload/forkchoiceUpdated/getPayload trio), `eth_*` (op-reth's own `eth_` API over the
//! engine's chain, except that `eth_sendRawTransaction` parks transactions in a buffer and the
//! `pending` nonce counts them), and `optest_*` — the sequencing hooks that let a test choose a
//! block's transactions (`includeTx`, `includeNextTx`, `remainingBlockGas`, `forcedEmpty`,
//! `setForceEmpty`) and a state dump (`dumpAccounts`). `debug_*` serves what kona-host reads from
//! an L2 node, as production op-reth serves it: reth's `debug_` API (`getRawHeader`,
//! `getRawReceipts`, `dbGet`) and op-reth's `debug_executePayload`.
//!
//! The engine's methods take `&mut self`, so the module context is an `Arc<Mutex<TestEngine>>`; a
//! poisoned lock is recovered rather than propagated so one failed request can't wedge the
//! process. The `eth_` and `debug_` reads query the chain's provider directly, without the engine
//! lock.

use std::sync::{Arc, Mutex};

use alloy_eips::{BlockId, eip7685::Requests};
use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_rpc_types_engine::{
    CancunPayloadFields, ForkchoiceState, PayloadId, PraguePayloadFields,
};
use jsonrpsee::{RpcModule, types::ErrorObjectOwned};
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadSidecar,
    OpPayloadAttributes,
};
use reth_network_api::noop::NoopNetwork;
use reth_optimism_evm::{OpEvmConfig, OpRethReceiptBuilder, tx::OpTxEnvConverter};
use reth_optimism_node::engine::validate_op_version_specific_fields;
use reth_optimism_payload_builder::{OpExecData, OpPayloadBuilder, OpPayloadBuilderAttributes};
use reth_optimism_primitives::OpTransactionSigned;
use reth_optimism_rpc::{
    OpEthApi,
    eth::{receipt::OpReceiptConverter, transaction::OpTxInfoMapper},
    witness::{DebugExecutionWitnessApiServer, OpDebugWitnessApi},
};
use reth_optimism_txpool::OpPooledTransaction;
use reth_payload_primitives::{EngineApiMessageVersion, PayloadOrAttributes};
use reth_rpc::{DebugApi, EthApiBuilder};
use reth_rpc_api::DebugApiServer;
use reth_rpc_engine_api::EngineApiError;
use reth_rpc_eth_api::{EthApiServer, RpcConverter, node::RpcNodeCoreAdapter};
use reth_tasks::Runtime;
use reth_transaction_pool::noop::NoopTransactionPool;
use serde_json::{Value, json};

use crate::{EphemeralChain, IncludeNextOutcome, IncludeTxOutcome, TestEngine, chain::Provider};

/// Shared, mutably-accessed engine behind the RPC module.
pub type SharedEngine = Arc<Mutex<TestEngine>>;

/// JSON-RPC error code for engine and execution failures without a dedicated engine-API code. The
/// engine API spec assigns no meaning to it, but it lies in the engine-error range op-node's
/// `ErrorCode.IsEngineError` recognizes.
const ERR_CODE: i32 = -38000;

fn rpc_err(msg: impl std::fmt::Display) -> ErrorObjectOwned {
    ErrorObjectOwned::owned(ERR_CODE, msg.to_string(), None::<()>)
}

/// go-ethereum's engine-API error code for an unknown payload id (`engine.UnknownPayload`).
/// op-node's build/seal path keys on exactly this code and remaps it to
/// `apis.BuildErrCodeUnknownPayload`, so a generic engine error here would break the sequencer-API
/// contract.
const UNKNOWN_PAYLOAD_ERR_CODE: i32 = -38001;

/// go-ethereum's engine-API error code for invalid payload attributes
/// (`eth.InvalidPayloadAttributes`). op-node's `startPayload` keys on exactly this code to classify
/// a forkchoice-update-with-attributes failure as a payload error (`BlockInsertPayloadErr`) and,
/// under Holocene for a derived block, request a deposits-only replacement. A generic engine error
/// here is instead read as a prestate problem and drives op-node into an endless reset loop.
const INVALID_PAYLOAD_ATTRIBUTES_ERR_CODE: i32 = -38003;

/// go-ethereum's engine-API error code for an inconsistent forkchoice state
/// (`engine.InvalidForkChoiceState`). op-node resets its view of the engine on it, where a generic
/// engine error would have it retry the same forkchoice update forever.
const INVALID_FORKCHOICE_STATE_ERR_CODE: i32 = -38002;

/// Map a `forkchoice_updated` error. Attributes failing the engine API's checks get the code
/// op-reth's engine API gives them (its own conversion): `-32602` for a missing or out-of-range
/// field, `-38003` for a field the fork version requires or forbids, `-38005` for an unsupported
/// fork. Attributes that cannot open a block on the head become `-38003`, a safe/finalized block
/// that is unknown or off the head's chain `-38002`; anything else is a generic engine error.
fn fcu_err(err: crate::Error) -> ErrorObjectOwned {
    let code = match err {
        crate::Error::MalformedPayloadAttributes(err) => {
            return EngineApiError::EngineObjectValidationError(err).into();
        }
        crate::Error::InvalidPayloadAttributes(_) => INVALID_PAYLOAD_ATTRIBUTES_ERR_CODE,
        crate::Error::UnknownForkchoiceBlock { .. } |
        crate::Error::NonCanonicalForkchoiceBlock { .. } => INVALID_FORKCHOICE_STATE_ERR_CODE,
        err => return rpc_err(err),
    };
    ErrorObjectOwned::owned(code, err.to_string(), None::<()>)
}

/// Map a `get_payload` error: an unknown payload id becomes go-ethereum's `-38001`; anything else
/// is a generic engine error.
fn get_payload_err(err: crate::Error) -> ErrorObjectOwned {
    match &err {
        crate::Error::UnknownPayloadId(_) => {
            ErrorObjectOwned::owned(UNKNOWN_PAYLOAD_ERR_CODE, err.to_string(), None::<()>)
        }
        _ => rpc_err(err),
    }
}

/// Lock the engine, recovering the guard if a previous handler panicked while holding it.
fn lock(engine: &SharedEngine) -> std::sync::MutexGuard<'_, TestEngine> {
    engine.lock().unwrap_or_else(|poisoned| poisoned.into_inner())
}

/// Reconstruct an [`OpExecutionData`] from `engine_newPayload*` arguments.
///
/// The sidecar carries the fork-specific fields: pre-Ecotone payloads have none, Ecotone/Fjord/
/// Granite/Holocene are v3 (Cancun beacon-root + blob versioned hashes), and Isthmus onward are v4
/// (additionally the Prague execution requests).
fn execution_data(
    payload: OpExecutionPayload,
    versioned_hashes: Vec<B256>,
    parent_beacon_block_root: Option<B256>,
    execution_requests: Vec<Bytes>,
) -> OpExecutionData {
    let sidecar =
        parent_beacon_block_root.map_or_else(OpExecutionPayloadSidecar::default, |root| {
            let cancun = CancunPayloadFields::new(root, versioned_hashes);
            if matches!(payload, OpExecutionPayload::V4(_)) {
                OpExecutionPayloadSidecar::v4(
                    cancun,
                    PraguePayloadFields::new(Requests::from_requests(execution_requests)),
                )
            } else {
                OpExecutionPayloadSidecar::v3(cancun)
            }
        });
    OpExecutionData::new(payload, sidecar)
}

/// Import `data`, carried by an `engine_newPayload` call of message version `version`.
///
/// Like op-reth's engine API, this first checks the payload against the fields `version` and the
/// forks active at its timestamp require, failing with the code op-reth gives a violation: `-38005`
/// for a timestamp in a fork `version` does not serve, `-32602` for a missing or unexpected field.
fn new_payload(
    engine: &SharedEngine,
    version: EngineApiMessageVersion,
    data: OpExecutionData,
) -> Result<Value, ErrorObjectOwned> {
    let data = OpExecData(data);
    let mut engine = lock(engine);
    validate_op_version_specific_fields(
        engine.chain.chain_spec().as_ref(),
        version,
        PayloadOrAttributes::from_execution_payload(&data),
    )
    .map_err(|err| ErrorObjectOwned::from(EngineApiError::EngineObjectValidationError(err)))?;
    let status = engine.new_payload(data.0).map_err(rpc_err)?;
    serde_json::to_value(status).map_err(rpc_err)
}

/// The components op-reth's `eth_` API runs on: the chain's provider, and no pool or network.
type EthNodeCore = RpcNodeCoreAdapter<
    Provider,
    NoopTransactionPool<OpPooledTransaction>,
    NoopNetwork,
    OpEvmConfig,
>;

/// op-reth's conversion of blocks, transactions and receipts to their OP RPC form.
type EthConverter = RpcConverter<
    Optimism,
    OpEvmConfig,
    OpReceiptConverter<Provider>,
    (),
    OpTxInfoMapper<Provider>,
    (),
    (),
    OpTxEnvConverter,
>;

/// op-reth's `eth_` API over the engine's chain.
type EthApi = OpEthApi<EthNodeCore, EthConverter>;

/// op-reth's `eth_` API over `chain`'s provider, with no transaction pool or network behind it,
/// spawning its tasks on `runtime`.
///
/// The proof window is unbounded: op-node verifies withdrawal proofs against the state of past
/// blocks, which reth's default window of zero blocks would reject. `eth_chainId` reads the network
/// handle, so the noop network carries the chain's id.
fn op_eth_api(chain: &EphemeralChain, runtime: Runtime) -> EthApi {
    let provider = chain.provider().clone();
    let evm_config = OpEvmConfig::new(chain.chain_spec(), OpRethReceiptBuilder::default());
    let converter = RpcConverter::new(OpReceiptConverter::new(provider.clone()))
        .with_mapper(OpTxInfoMapper::new(provider.clone()))
        .with_tx_env_converter(OpTxEnvConverter);
    let inner = EthApiBuilder::new(
        provider,
        NoopTransactionPool::<OpPooledTransaction>::new(),
        NoopNetwork::default().with_chain_id(chain.chain_id()),
        evm_config,
    )
    .with_rpc_converter(converter)
    .eth_proof_window(u64::MAX)
    .task_spawner(runtime)
    .build_inner();
    OpEthApi::new(inner, None, U256::ZERO, None, false)
}

/// The `debug_` methods kona-host reads from an L2 node, spawning their work on `runtime`:
/// reth's `debug_` API over `eth_api`, and op-reth's `debug_executePayload`, which builds a
/// payload on `chain` with no transaction pool and returns its execution witness. The production
/// node wires both the same way.
fn debug_api(chain: &EphemeralChain, eth_api: EthApi, runtime: Runtime) -> jsonrpsee::Methods {
    let debug =
        DebugApi::new(eth_api, runtime.blocking_guard(), &runtime, futures::stream::empty());

    let provider = chain.provider().clone();
    let evm_config = OpEvmConfig::new(chain.chain_spec(), OpRethReceiptBuilder::default());
    let builder = OpPayloadBuilder::new(
        NoopTransactionPool::<OpPooledTransaction>::new(),
        provider.clone(),
        evm_config.clone(),
    );
    let witness =
        OpDebugWitnessApi::<_, _, _, OpPayloadBuilderAttributes<OpTransactionSigned>>::new(
            provider, runtime, builder, evm_config,
        );

    let mut methods = DebugApiServer::into_rpc(debug);
    methods
        .merge(DebugExecutionWitnessApiServer::into_rpc(witness))
        .expect("debug_executePayload is not part of reth's debug_ API");
    methods.into()
}

/// Build the JSON-RPC module serving `engine_*`, `eth_*`, `debug_*`, and `optest_*` over `engine`.
///
/// Call it inside a tokio runtime: the `eth_` and `debug_` APIs spawn their tasks through a reth
/// task runtime that reuses its handle, with the small thread pools of reth's test runtime.
pub fn build_module(engine: SharedEngine) -> RpcModule<SharedEngine> {
    let runtime = Runtime::test();
    let (eth_api, debug) = {
        let engine = lock(&engine);
        let eth_api = op_eth_api(&engine.chain, runtime.clone());
        let debug = debug_api(&engine.chain, eth_api.clone(), runtime);
        (eth_api, debug)
    };
    let mut m = RpcModule::new(engine);

    // --- engine_ ---

    let register_fcu = |m: &mut RpcModule<SharedEngine>,
                        name: &'static str,
                        version: EngineApiMessageVersion| {
        m.register_method(name, move |params, ctx, _| {
            let (state, attrs): (ForkchoiceState, Option<OpPayloadAttributes>) =
                params.parse().map_err(rpc_err)?;
            let updated = lock(ctx).forkchoice_updated(version, state, attrs).map_err(fcu_err)?;
            serde_json::to_value(updated).map_err(rpc_err)
        })
        .expect("register method");
    };
    register_fcu(&mut m, "engine_forkchoiceUpdatedV1", EngineApiMessageVersion::V1);
    register_fcu(&mut m, "engine_forkchoiceUpdatedV2", EngineApiMessageVersion::V2);
    register_fcu(&mut m, "engine_forkchoiceUpdatedV3", EngineApiMessageVersion::V3);

    let register_get_payload = |m: &mut RpcModule<SharedEngine>, name: &'static str| {
        m.register_method(name, |params, ctx, _| {
            let id: PayloadId = params.one().map_err(rpc_err)?;
            let data = lock(ctx).get_payload(id).map_err(get_payload_err)?;
            let envelope = OpExecutionPayloadEnvelope::try_from(data).map_err(rpc_err)?;
            serde_json::to_value(envelope).map_err(rpc_err)
        })
        .expect("register method");
    };
    register_get_payload(&mut m, "engine_getPayloadV2");
    register_get_payload(&mut m, "engine_getPayloadV3");
    register_get_payload(&mut m, "engine_getPayloadV4");
    register_get_payload(&mut m, "engine_getPayloadV5");

    m.register_method("engine_newPayloadV2", |params, ctx, _| {
        let payload: OpExecutionPayload = params.one().map_err(rpc_err)?;
        let data = execution_data(payload, Vec::new(), None, Vec::new());
        new_payload(ctx, EngineApiMessageVersion::V2, data)
    })
    .expect("register method");

    m.register_method("engine_newPayloadV3", |params, ctx, _| {
        let (payload, versioned_hashes, beacon_root): (
            OpExecutionPayload,
            Vec<B256>,
            Option<B256>,
        ) = params.parse().map_err(rpc_err)?;
        let data = execution_data(payload, versioned_hashes, beacon_root, Vec::new());
        new_payload(ctx, EngineApiMessageVersion::V3, data)
    })
    .expect("register method");

    m.register_method("engine_newPayloadV4", |params, ctx, _| {
        let (payload, versioned_hashes, beacon_root, execution_requests): (
            OpExecutionPayload,
            Vec<B256>,
            Option<B256>,
            Vec<Bytes>,
        ) = params.parse().map_err(rpc_err)?;
        let data = execution_data(payload, versioned_hashes, beacon_root, execution_requests);
        new_payload(ctx, EngineApiMessageVersion::V4, data)
    })
    .expect("register method");

    // --- optest_ ---

    m.register_method("optest_includeTx", |params, ctx, _| {
        let raw: Bytes = params.one().map_err(rpc_err)?;
        let value = match lock(ctx).include_tx(None, raw.as_ref()).map_err(rpc_err)? {
            IncludeTxOutcome::Included { tx_hash, gas_used } => {
                json!({ "txHash": tx_hash, "gasUsed": gas_used })
            }
            // Force-empty dropped the tx (op-geth returns nil, nil); the caller treats null as "not
            // included".
            IncludeTxOutcome::Skipped => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    // Include the next parked transaction from `from` (the reframed `ActL2IncludeTx`): the engine
    // computes the eligible nonce and executes the matching parked tx, which stays parked until
    // the committed state passes its nonce.
    m.register_method("optest_includeNextTx", |params, ctx, _| {
        let from: Address = params.one().map_err(rpc_err)?;
        let value = match lock(ctx).include_next_tx(from).map_err(rpc_err)? {
            IncludeNextOutcome::Included { tx_hash, gas_used } => {
                json!({ "txHash": tx_hash, "gasUsed": gas_used })
            }
            IncludeNextOutcome::Skipped => json!({ "skipped": true }),
            IncludeNextOutcome::NoTx => json!({ "noTx": true }),
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m.register_method("optest_remainingBlockGas", |_params, ctx, _| {
        Ok::<_, ErrorObjectOwned>(Value::from(lock(ctx).remaining_block_gas(None)))
    })
    .expect("register method");

    m.register_method("optest_forcedEmpty", |_params, ctx, _| {
        Ok::<_, ErrorObjectOwned>(Value::from(lock(ctx).forced_empty(None)))
    })
    .expect("register method");

    m.register_method("optest_setForceEmpty", |params, ctx, _| {
        let value: bool = params.one().map_err(rpc_err)?;
        lock(ctx).set_force_empty(None, value).map_err(rpc_err)?;
        Ok::<Value, ErrorObjectOwned>(Value::Bool(true))
    })
    .expect("register method");

    // `optest_dumpAccounts(blockHash, addresses)`: the state of the listed accounts at a block, as
    // a go-ethereum `types.GenesisAlloc` decodes it: an object keyed by address whose values are
    // `{"nonce", "balance", "code", "storage": {slot: value}}` (hex quantities and bytes, full
    // 32-byte slots and values). An account that does not exist is absent, `code` is absent for
    // an account without code and `storage` when it holds no non-zero slot. An unknown block is
    // an error. See `EphemeralChain::dump_accounts`.
    m.register_method("optest_dumpAccounts", |params, ctx, _| {
        let (hash, addresses): (B256, Vec<Address>) = params.parse().map_err(rpc_err)?;
        let dump = lock(ctx)
            .chain
            .dump_accounts(hash, &addresses)
            .map_err(rpc_err)?
            .ok_or_else(|| rpc_err(format!("unknown block {hash}")))?;
        serde_json::to_value(dump).map_err(rpc_err)
    })
    .expect("register method");

    // --- eth_ ---

    // A raw transaction is parked in the engine's pending buffer (no auto-inclusion); the Go tests'
    // `ActL2IncludeTx(from)` later includes it via `optest_includeNextTx`.
    m.register_method("eth_sendRawTransaction", |params, ctx, _| {
        let raw: Bytes = params.one().map_err(rpc_err)?;
        let hash = lock(ctx).send_raw_transaction(raw.as_ref()).map_err(rpc_err)?;
        serde_json::to_value(hash).map_err(rpc_err)
    })
    .expect("register method");

    // "pending" folds in the parked buffer so the caller's next-nonce read accounts for txs it has
    // already submitted but not yet had included; every other block reads committed state.
    let nonce_api = eth_api.clone();
    m.register_async_method("eth_getTransactionCount", move |params, ctx, _| {
        let eth_api = nonce_api.clone();
        async move {
            let (address, block): (Address, Option<BlockId>) = params.parse()?;
            if block.is_some_and(|block| block.is_pending()) {
                let nonce = lock(&ctx).pending_nonce(address).map_err(rpc_err)?;
                return Ok(U256::from(nonce));
            }
            EthApiServer::transaction_count(&eth_api, address, block).await
        }
    })
    .expect("register method");

    let mut eth = eth_api.into_rpc();
    for method in ["eth_sendRawTransaction", "eth_getTransactionCount"] {
        eth.remove_method(method);
    }
    m.merge(eth).expect("eth_ methods do not clash with the engine's own");
    m.merge(debug).expect("debug_ methods do not clash with the engine's own");

    m
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::testsupport::{
        FUNDED_BALANCE, deposit_tx, depositor, ecotone_test_engine, encode, fcu, next_env,
        payload_attrs, payload_of, sequence_forced, test_engine, test_engine_with_accounts,
        user_sender, user_tx,
    };
    use alloy_consensus::{Header, proofs::ordered_trie_root_encoded};
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::{TxKind, address, bytes, keccak256};
    use alloy_rlp::Decodable;
    use jsonrpsee::core::server::MethodsError;
    use op_alloy_consensus::TxDeposit;
    use reth_optimism_evm::PostExecMode;
    use std::collections::BTreeMap;

    /// An engine whose head is block 1, holding a single deposit.
    fn engine_with_deposit() -> (SharedEngine, B256) {
        let mut engine = test_engine(user_sender());
        let genesis = engine.chain.genesis_hash();
        let attrs = payload_attrs(2, vec![encode(&deposit_tx(depositor()))], false);
        let id =
            engine.forkchoice_updated_auto(fcu(genesis), Some(attrs)).unwrap().payload_id.unwrap();
        let data = engine.get_payload(id).unwrap();
        let block_hash = data.payload.block_hash();
        assert!(engine.new_payload(data).unwrap().is_valid());
        assert!(engine.forkchoice_updated_auto(fcu(block_hash), None).unwrap().is_valid());
        (Arc::new(Mutex::new(engine)), block_hash)
    }

    fn assert_deposit_fields(tx: &Value) {
        assert_eq!(tx["type"], "0x7e", "{tx}");
        assert_eq!(tx["nonce"], "0x0", "deposit nonce missing: {tx}");
        assert_eq!(tx["depositReceiptVersion"], "0x1", "receipt version missing: {tx}");
    }

    #[tokio::test]
    async fn deposit_transactions_carry_their_nonce_and_receipt_version() {
        let (engine, block_hash) = engine_with_deposit();
        let module = build_module(engine);
        let deposit_hash = deposit_tx(depositor()).tx_hash();

        let tx: Value = module.call("eth_getTransactionByHash", [deposit_hash]).await.unwrap();
        assert_deposit_fields(&tx);

        let block: Value = module.call("eth_getBlockByHash", (block_hash, true)).await.unwrap();
        assert_deposit_fields(&block["transactions"][0]);
    }

    #[tokio::test]
    async fn raw_header_and_receipts_round_trip() {
        let (engine, block_hash) = engine_with_deposit();
        let module = build_module(engine);

        let raw: Bytes = module.call("debug_getRawHeader", [block_hash]).await.expect("by hash");
        let header = Header::decode(&mut raw.as_ref()).expect("decode header");
        assert_eq!(header.hash_slow(), block_hash);
        let by_number: Bytes = module.call("debug_getRawHeader", ["0x1"]).await.expect("by number");
        assert_eq!(by_number, raw);

        let receipts: Vec<Bytes> =
            module.call("debug_getRawReceipts", [block_hash]).await.expect("receipts");
        assert_eq!(receipts.len(), 1);
        assert_eq!(ordered_trie_root_encoded(&receipts), header.receipts_root);
    }

    /// A predeploy address (`GasPriceOracle`'s) holding [`store_code`].
    const PREDEPLOY: Address = address!("0x420000000000000000000000000000000000000F");

    /// Sets slot 1 to `0x2a` and clears slot 0:
    /// `PUSH1 0x2a PUSH1 1 SSTORE PUSH1 0 PUSH1 0 SSTORE STOP`.
    fn store_code() -> Bytes {
        bytes!("0x602a600155600060005500")
    }

    fn word(value: u64) -> B256 {
        B256::from(U256::from(value))
    }

    /// An engine whose genesis puts [`store_code`] at [`PREDEPLOY`], with slots 0 and 2 set.
    fn engine_with_predeploy() -> TestEngine {
        let storage = BTreeMap::from([(word(0), word(7)), (word(2), word(9))]);
        test_engine_with_accounts(
            user_sender(),
            [(
                PREDEPLOY,
                GenesisAccount {
                    nonce: Some(1),
                    code: Some(store_code()),
                    storage: Some(storage),
                    ..Default::default()
                },
            )],
        )
    }

    /// A deposit to `to` with `input`; `source` tells deposits apart.
    fn deposit(source: u8, to: TxKind, input: Bytes) -> Bytes {
        let deposit = TxDeposit {
            source_hash: B256::repeat_byte(source),
            from: depositor(),
            to,
            gas_limit: 100_000,
            input,
            ..Default::default()
        };
        encode(&deposit.into())
    }

    /// A deposit calling [`PREDEPLOY`].
    fn call_predeploy(source: u8) -> Bytes {
        deposit(source, TxKind::Call(PREDEPLOY), Bytes::new())
    }

    /// The code [`DEPLOY_INITCODE`] deploys.
    const DEPLOYED_CODE: Bytes = bytes!("0xdeadbe");

    /// Returns [`DEPLOYED_CODE`]: `PUSH3 0xdeadbe PUSH1 0 MSTORE PUSH1 3 PUSH1 29 RETURN`.
    const DEPLOY_INITCODE: Bytes = bytes!("0x62deadbe6000526003601df3");

    #[tokio::test]
    async fn execute_payload_witnesses_the_parent_state_and_touched_code() {
        let mut engine = engine_with_predeploy();
        let genesis = engine.chain.genesis_hash();
        let forced = vec![call_predeploy(1), deposit(3, TxKind::Create, DEPLOY_INITCODE)];
        let parent = sequence_forced(&mut engine, genesis, 2, forced, &[]).payload.block_hash();
        let parent_root = engine.chain.sealed_header(parent).unwrap().expect("parent").state_root;
        let module = build_module(Arc::new(Mutex::new(engine)));

        let attrs = payload_attrs(4, vec![call_predeploy(2)], true);
        let witness: Value =
            module.call("debug_executePayload", (parent, attrs)).await.expect("witness");
        let state: Vec<Bytes> = serde_json::from_value(witness["state"].clone()).unwrap();
        assert!(state.iter().any(|node| keccak256(node) == parent_root), "parent state root");
        let codes: Vec<Bytes> = serde_json::from_value(witness["codes"].clone()).unwrap();
        assert!(codes.contains(&store_code()), "touched code");

        // kona-host fetches code by its hash, in the `0x63 || codeHash` key form.
        let key: Bytes = [&[0x63], keccak256(store_code()).as_slice()].concat().into();
        let code: Bytes = module.call("debug_dbGet", [key]).await.expect("code");
        assert_eq!(code, store_code());
        // Code deployed in a block that is not persisted is served as well.
        let key: Bytes = [&[0x63], keccak256(&DEPLOYED_CODE).as_slice()].concat().into();
        let code: Bytes = module.call("debug_dbGet", [key]).await.expect("deployed code");
        assert_eq!(code, DEPLOYED_CODE);
    }

    #[tokio::test]
    async fn dump_accounts_reflects_storage_changes() {
        let mut engine = engine_with_predeploy();
        let genesis = engine.chain.genesis_hash();
        let block = sequence_forced(&mut engine, genesis, 2, vec![call_predeploy(1)], &[])
            .payload
            .block_hash();
        // The dumped block does not touch the predeploy; its parent wrote the storage.
        let child = sequence_forced(&mut engine, block, 4, vec![], &[]).payload.block_hash();
        let module = build_module(Arc::new(Mutex::new(engine)));
        let missing = Address::with_last_byte(0x77);

        let dump: BTreeMap<Address, GenesisAccount> = module
            .call("optest_dumpAccounts", (child, [PREDEPLOY, user_sender(), missing]))
            .await
            .expect("dump");
        let predeploy = &dump[&PREDEPLOY];
        assert_eq!(
            predeploy.storage,
            Some(BTreeMap::from([(word(1), word(0x2a)), (word(2), word(9))]))
        );
        assert_eq!(predeploy.code, Some(store_code()));
        assert_eq!(predeploy.nonce, Some(1));
        let sender = &dump[&user_sender()];
        assert_eq!(sender.balance, FUNDED_BALANCE);
        assert_eq!((&sender.code, &sender.storage), (&None, &None));
        assert!(!dump.contains_key(&missing), "absent account");

        let at_genesis: BTreeMap<Address, GenesisAccount> = module
            .call("optest_dumpAccounts", (genesis, [PREDEPLOY]))
            .await
            .expect("dump at genesis");
        assert_eq!(
            at_genesis[&PREDEPLOY].storage,
            Some(BTreeMap::from([(word(0), word(7)), (word(2), word(9))]))
        );

        let unknown = B256::repeat_byte(0xab);
        assert!(
            module.call::<_, Value>("optest_dumpAccounts", (unknown, [PREDEPLOY])).await.is_err(),
            "unknown block"
        );
    }

    #[tokio::test]
    async fn chain_id_is_the_chain_specs() {
        let engine = test_engine(user_sender());
        let expected = engine.chain.chain_id();
        let module = build_module(Arc::new(Mutex::new(engine)));
        let chain_id: U256 = module.call("eth_chainId", [(); 0]).await.expect("chain id");
        assert_eq!(chain_id, U256::from(expected));
    }

    #[tokio::test]
    async fn pending_nonce_counts_parked_transactions() {
        let module = build_module(Arc::new(Mutex::new(test_engine(user_sender()))));
        let _: B256 =
            module.call("eth_sendRawTransaction", [encode(&user_tx(0))]).await.expect("park tx");

        let pending: U256 = module
            .call("eth_getTransactionCount", (user_sender(), "pending"))
            .await
            .expect("pending nonce");
        assert_eq!(pending, U256::from(1));
        let latest: U256 = module
            .call("eth_getTransactionCount", (user_sender(), "latest"))
            .await
            .expect("latest nonce");
        assert_eq!(latest, U256::ZERO);
    }

    #[tokio::test]
    async fn eth_call_reads_state_and_surfaces_reverts_as_geth_does() {
        // Returns the 32-byte word 0x2a: `PUSH1 0x2a PUSH1 0 MSTORE PUSH1 0x20 PUSH1 0 RETURN`.
        let returner = address!("0x00000000000000000000000000000000000000aa");
        // Always reverts with empty output: `PUSH1 0 PUSH1 0 REVERT`.
        let reverter = address!("0x00000000000000000000000000000000000000bb");
        let engine = test_engine_with_accounts(
            user_sender(),
            [
                (
                    returner,
                    GenesisAccount {
                        code: Some(bytes!("0x602a60005260206000f3")),
                        ..Default::default()
                    },
                ),
                (
                    reverter,
                    GenesisAccount { code: Some(bytes!("0x60006000fd")), ..Default::default() },
                ),
            ],
        );
        let module = build_module(Arc::new(Mutex::new(engine)));

        let out: Bytes = module
            .call("eth_call", [json!({ "from": user_sender(), "to": returner })])
            .await
            .expect("call");
        assert_eq!(out, Bytes::from(U256::from(42u64).to_be_bytes::<32>()));

        match module
            .call::<_, Bytes>("eth_call", [json!({ "from": user_sender(), "to": reverter })])
            .await
        {
            Err(MethodsError::JsonRpc(err)) => assert_eq!(err.code(), 3, "{err:?}"),
            other => panic!("expected a revert error, got {other:?}"),
        }
    }

    /// Open a build on genesis over `engine_forkchoiceUpdatedV3` with `attrs` and return the
    /// JSON-RPC error code the update fails with.
    async fn fcu_error_code(attrs: OpPayloadAttributes) -> i32 {
        fcu_error_code_via("engine_forkchoiceUpdatedV3", attrs).await
    }

    /// Open a build on genesis over the forkchoice update `method` with `attrs` and return the
    /// JSON-RPC error code the update fails with.
    async fn fcu_error_code_via(method: &str, attrs: OpPayloadAttributes) -> i32 {
        let engine = test_engine(user_sender());
        let genesis = engine.chain.genesis_hash();
        let module = build_module(Arc::new(Mutex::new(engine)));
        match module.call::<_, Value>(method, (fcu(genesis), attrs)).await {
            Err(MethodsError::JsonRpc(err)) => err.code(),
            other => panic!("expected a JSON-RPC error, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn attributes_through_a_method_version_the_fork_does_not_use_are_rejected() {
        // The fixture chain runs Ecotone from genesis, so its attributes belong to V3.
        let attrs = payload_attrs(2, vec![], false);
        let code = fcu_error_code_via("engine_forkchoiceUpdatedV2", attrs.clone()).await;
        assert_eq!(code, -38003, "parent beacon block root through V2");

        let mut attrs = attrs;
        attrs.payload_attributes.parent_beacon_block_root = None;
        let code = fcu_error_code_via("engine_forkchoiceUpdatedV2", attrs.clone()).await;
        assert_eq!(code, -38005, "Ecotone timestamp through V2");

        let code = fcu_error_code_via("engine_forkchoiceUpdatedV3", attrs).await;
        assert_eq!(code, -38003, "no parent beacon block root through V3");
    }

    #[tokio::test]
    async fn missing_op_attribute_fields_are_invalid_params() {
        let mut attrs = payload_attrs(2, vec![], false);
        attrs.eip_1559_params = None;
        assert_eq!(fcu_error_code(attrs).await, -32602);
    }

    #[tokio::test]
    async fn attributes_that_cannot_open_a_block_are_invalid_payload_attributes() {
        let mut attrs = payload_attrs(2, vec![], false);
        attrs.payload_attributes.withdrawals = None;
        assert_eq!(fcu_error_code(attrs).await, -38003, "withdrawals missing after Canyon");

        // Genesis has timestamp 0.
        assert_eq!(fcu_error_code(payload_attrs(0, vec![], false)).await, -38003, "timestamp");

        let late_deposit = deposit_tx(Address::with_last_byte(0xdf));
        let forced =
            vec![encode(&deposit_tx(depositor())), encode(&user_tx(0)), encode(&late_deposit)];
        assert_eq!(fcu_error_code(payload_attrs(2, forced, true)).await, -38003, "deposit order");
    }

    #[test]
    fn unsupported_fork_attributes_map_to_unsupported_fork() {
        let err = crate::Error::MalformedPayloadAttributes(
            reth_payload_primitives::EngineObjectValidationError::UnsupportedFork,
        );
        assert_eq!(fcu_err(err).code(), -38005);
    }

    #[test]
    fn invalid_forkchoice_pointers_map_to_invalid_forkchoice_state() {
        let hash = B256::repeat_byte(0x01);
        for err in [
            crate::Error::UnknownForkchoiceBlock { which: crate::ForkchoicePointer::Safe, hash },
            crate::Error::NonCanonicalForkchoiceBlock {
                which: crate::ForkchoicePointer::Finalized,
                hash,
            },
        ] {
            assert_eq!(fcu_err(err).code(), -38002);
        }
    }

    /// Call the `engine_newPayload` `method` with block 1 of an Ecotone chain, passing the
    /// parameters that method version takes.
    async fn new_payload_on_ecotone(method: &str) -> Result<Value, MethodsError> {
        let engine = ecotone_test_engine(user_sender());
        let genesis = engine.chain.latest_header().hash();
        let built = engine
            .chain
            .assemble_block(
                genesis,
                next_env(2, Bytes::new()),
                PostExecMode::Disabled,
                &[deposit_tx(depositor())],
            )
            .expect("assemble block");
        let data = payload_of(&built.block.clone_sealed_block().into_block());
        let beacon_root = data.sidecar.parent_beacon_block_root().expect("Ecotone beacon root");
        let module = build_module(Arc::new(Mutex::new(engine)));
        let hashes = Vec::<B256>::new();
        if method == "engine_newPayloadV3" {
            module.call(method, (data.payload, hashes, beacon_root)).await
        } else {
            module.call(method, (data.payload, hashes, beacon_root, Vec::<Bytes>::new())).await
        }
    }

    #[tokio::test]
    async fn new_payload_through_a_method_version_the_fork_does_not_use_is_rejected() {
        match new_payload_on_ecotone("engine_newPayloadV4").await {
            Err(MethodsError::JsonRpc(err)) => assert_eq!(err.code(), -38005, "{err:?}"),
            other => panic!("expected an unsupported-fork error, got {other:?}"),
        }

        let status = new_payload_on_ecotone("engine_newPayloadV3").await.expect("V3 at Ecotone");
        assert_eq!(status["status"], "VALID", "{status}");
    }
}
