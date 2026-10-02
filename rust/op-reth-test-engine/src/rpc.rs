//! JSON-RPC surface for the test engine, served over a Unix socket (reth-ipc, go-ethereum
//! `rpc.DialIPC`-compatible) by the companion binary.
//!
//! Three namespaces serve what the `op-e2e/actions` harness drives: `engine_*` (the versioned
//! newPayload/forkchoiceUpdated/getPayload trio), `eth_*` (op-reth's own `eth_` API over the
//! engine's chain, except that `eth_sendRawTransaction` parks transactions in a buffer and the
//! `pending` nonce counts them), and `optest_*` — the sequencing hooks that let a test choose a
//! block's transactions (`includeTx`, `includeNextTx`, `remainingBlockGas`, `forcedEmpty`,
//! `setForceEmpty`).
//!
//! The engine's methods take `&mut self`, so the module context is an `Arc<Mutex<TestEngine>>`; a
//! poisoned lock is recovered rather than propagated so one failed request can't wedge the
//! process. The `eth_` reads query the chain's provider directly, without the engine lock.

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
use reth_optimism_rpc::{
    OpEthApi,
    eth::{receipt::OpReceiptConverter, transaction::OpTxInfoMapper},
};
use reth_optimism_txpool::OpPooledTransaction;
use reth_payload_primitives::EngineApiMessageVersion;
use reth_rpc::EthApiBuilder;
use reth_rpc_engine_api::EngineApiError;
use reth_rpc_eth_api::{EthApiServer, RpcConverter, node::RpcNodeCoreAdapter};
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

/// op-reth's `eth_` API over `chain`'s provider, with no transaction pool or network behind it.
///
/// The proof window is unbounded: op-node verifies withdrawal proofs against the state of past
/// blocks, which reth's default window of zero blocks would reject. `eth_chainId` reads the network
/// handle, so the noop network carries the chain's id.
fn op_eth_api(chain: &EphemeralChain) -> OpEthApi<EthNodeCore, EthConverter> {
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
    .build_inner();
    OpEthApi::new(inner, None, U256::ZERO, None, false)
}

/// Build the JSON-RPC module serving `engine_*`, `eth_*`, and `optest_*` over `engine`.
///
/// Must be called inside a tokio runtime: op-reth's `eth_` API spawns its block cache and its
/// fee-history task on the current one.
pub fn build_module(engine: SharedEngine) -> RpcModule<SharedEngine> {
    let eth_api = op_eth_api(&lock(&engine).chain);
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
        let status = lock(ctx).new_payload(data).map_err(rpc_err)?;
        serde_json::to_value(status).map_err(rpc_err)
    })
    .expect("register method");

    m.register_method("engine_newPayloadV3", |params, ctx, _| {
        let (payload, versioned_hashes, beacon_root): (
            OpExecutionPayload,
            Vec<B256>,
            Option<B256>,
        ) = params.parse().map_err(rpc_err)?;
        let data = execution_data(payload, versioned_hashes, beacon_root, Vec::new());
        let status = lock(ctx).new_payload(data).map_err(rpc_err)?;
        serde_json::to_value(status).map_err(rpc_err)
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
        let status = lock(ctx).new_payload(data).map_err(rpc_err)?;
        serde_json::to_value(status).map_err(rpc_err)
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

    m
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::testsupport::{
        deposit_tx, depositor, encode, fcu, payload_attrs, test_engine, test_engine_with_accounts,
        user_sender, user_tx,
    };
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::{address, bytes};
    use jsonrpsee::core::server::MethodsError;

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
}
