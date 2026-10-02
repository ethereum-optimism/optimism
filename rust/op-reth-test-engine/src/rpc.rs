//! JSON-RPC surface for the test engine, served over a Unix socket (reth-ipc, go-ethereum
//! `rpc.DialIPC`-compatible) by the companion binary.
//!
//! Three namespaces serve what the `op-e2e/actions` harness drives: `engine_*` (the versioned
//! newPayload/forkchoiceUpdated/getPayload trio), `eth_*` (chain and state reads,
//! `eth_call`/`eth_estimateGas`, and `eth_sendRawTransaction` into a parking buffer), and
//! `optest_*` — the sequencing hooks that let a test choose a block's transactions (`includeTx`,
//! `includeNextTx`, `remainingBlockGas`, `forcedEmpty`, `setForceEmpty`) and the sync-backfill
//! hooks that stand in for devp2p sync (`syncTarget`, `blockPayloadByNumber`, `importBlock`).
//!
//! The engine's methods take `&mut self`, so the module context is an `Arc<Mutex<TestEngine>>` and
//! requests are served one at a time; a poisoned lock is recovered rather than propagated so one
//! failed request can't wedge the process.

use std::sync::{Arc, Mutex};

use alloy_consensus::transaction::{Recovered, SignerRecoverable, TransactionInfo};
use alloy_eips::eip7685::Requests;
use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_rpc_types_engine::{
    CancunPayloadFields, ForkchoiceState, PayloadId, PraguePayloadFields,
};
use alloy_serde::JsonStorageKey;
use jsonrpsee::{RpcModule, types::ErrorObjectOwned};

use op_alloy_rpc_types::Transaction as OpRpcTransaction;
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadSidecar,
    OpPayloadAttributes,
};
use reth_optimism_primitives::OpBlock;
use reth_payload_primitives::EngineApiMessageVersion;
use reth_rpc_engine_api::EngineApiError;
use reth_rpc_eth_types::error::RevertError;
use reth_storage_api::{StateProofProvider, StateProvider, StateProviderBox};
use serde_json::{Value, json};

use crate::{IncludeNextOutcome, IncludeTxOutcome, TestEngine};

/// Shared, mutably-accessed engine behind the RPC module.
pub type SharedEngine = Arc<Mutex<TestEngine>>;

/// JSON-RPC error code for engine and execution failures without a dedicated engine-API code. The
/// engine API spec assigns no meaning to it, but it lies in the engine-error range op-node's
/// `ErrorCode.IsEngineError` recognizes.
const ERR_CODE: i32 = -38000;

fn rpc_err(msg: impl std::fmt::Display) -> ErrorObjectOwned {
    ErrorObjectOwned::owned(ERR_CODE, msg.to_string(), None::<()>)
}

/// go-ethereum's JSON-RPC error code for reverted `eth_call`/`eth_estimateGas` executions.
const REVERT_ERR_CODE: i32 = 3;

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

/// Map an engine error to a JSON-RPC error; a revert becomes geth's shape — code 3, a message
/// carrying the ABI-decoded reason when there is one, and the raw output as error data (which
/// go-ethereum clients read via `rpc.DataError`).
fn call_err(err: crate::Error) -> ErrorObjectOwned {
    match err {
        crate::Error::Revert(output) => {
            let data = format!("0x{}", alloy_primitives::hex::encode(&output));
            ErrorObjectOwned::owned(
                REVERT_ERR_CODE,
                RevertError::new(output).to_string(),
                Some(data),
            )
        }
        other => rpc_err(other),
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

/// Serialize an OP block as an `eth_getBlock*` result: the RPC header (which carries the block
/// hash) plus its transactions. With `full` false the `transactions` array is the transaction
/// hashes; with `full` true it is the full OP RPC transaction objects.
///
/// op-node's `sources.EthClient` reconstructs the `ExecutionPayload` and the `L2BlockRef` (from the
/// L1-info deposit) out of the full-transaction form, so the full path must emit exactly the JSON
/// go-ethereum's transaction decoder round-trips back to the same RLP the block was sealed with —
/// including the deposit-transaction fields. That is precisely what
/// [`OpRpcTransaction::from_transaction`] produces (it is what reth serves in production).
fn block_json(engine: &TestEngine, block: &OpBlock, full: bool) -> Result<Value, ErrorObjectOwned> {
    let header = alloy_rpc_types_eth::Header::new(block.header.clone());
    let block_hash = header.hash;
    let mut value = serde_json::to_value(&header).map_err(rpc_err)?;
    let txs = if full {
        full_transactions(engine, block, block_hash)?
    } else {
        let hashes: Vec<B256> = block.body.transactions.iter().map(|tx| tx.tx_hash()).collect();
        serde_json::to_value(hashes).map_err(rpc_err)?
    };
    if let Value::Object(map) = &mut value {
        map.insert("transactions".into(), txs);
        map.insert("uncles".into(), json!([]));
        // Post-Canyon OP blocks carry an (always empty) withdrawals list alongside the
        // withdrawals-root header field; op-node rejects a block that has the root but no list.
        if let Some(withdrawals) = &block.body.withdrawals {
            map.insert("withdrawals".into(), serde_json::to_value(withdrawals).map_err(rpc_err)?);
        }
    }
    Ok(value)
}

/// Serialize a block's transactions as full OP RPC transaction objects, in block order.
///
/// The signer is recovered per transaction (deposits recover to their `from` field), and a
/// deposit's nonce and receipt version are read from its receipt.
fn full_transactions(
    engine: &TestEngine,
    block: &OpBlock,
    block_hash: B256,
) -> Result<Value, ErrorObjectOwned> {
    let base_fee = block.header.base_fee_per_gas;
    let mut out = Vec::with_capacity(block.body.transactions.len());
    for (index, tx) in block.body.transactions.iter().enumerate() {
        let signer = tx.recover_signer().map_err(|e| rpc_err(format!("recover signer: {e}")))?;
        let recovered = Recovered::new_unchecked(tx.clone(), signer);
        let tx_info = engine
            .chain
            .rpc_transaction_info(
                tx,
                TransactionInfo {
                    hash: None,
                    index: Some(index as u64),
                    block_hash: Some(block_hash),
                    block_number: Some(block.header.number),
                    base_fee,
                    block_timestamp: Some(block.header.timestamp),
                },
            )
            .map_err(rpc_err)?;
        let rpc_tx = OpRpcTransaction::from_transaction(recovered, tx_info);
        out.push(serde_json::to_value(rpc_tx).map_err(rpc_err)?);
    }
    Ok(Value::Array(out))
}

/// Resolve a block-number-or-tag string (`latest`/`safe`/`finalized`/`earliest`/`pending` or a
/// `0x`-hex number) to a block hash, or `None` if the block is unknown.
fn resolve_block_hash(engine: &TestEngine, tag: &str) -> Result<Option<B256>, ErrorObjectOwned> {
    let hash = match tag {
        "latest" | "pending" => Some(engine.chain.latest_header().hash()),
        "safe" => engine.chain.safe_header().map(|h| h.hash()),
        "finalized" => engine.chain.finalized_header().map(|h| h.hash()),
        "earliest" => Some(engine.chain.genesis_hash()),
        // A 32-byte hex string is a block hash, not a number: op-node passes `blockHash.String()`
        // as the block tag to `eth_getProof`, and go-ethereum treats any 66-char `0x` string that
        // way. Resolve it to itself only if the block is known.
        hash if hash.len() == 66 && hash.starts_with("0x") => {
            let hash: B256 =
                hash.parse().map_err(|e| rpc_err(format!("invalid block hash {hash:?}: {e}")))?;
            engine.chain.sealed_header(hash).map_err(rpc_err)?.map(|_| hash)
        }
        num => {
            let n = u64::from_str_radix(num.trim_start_matches("0x"), 16)
                .map_err(|e| rpc_err(format!("invalid block number {num:?}: {e}")))?;
            engine.block_by_number(n).map_err(rpc_err)?.map(|block| block.header.hash_slow())
        }
    };
    Ok(hash)
}

/// Resolve a block-number-or-tag string to a block, or `None` if unknown.
fn resolve_block(engine: &TestEngine, tag: &str) -> Result<Option<OpBlock>, ErrorObjectOwned> {
    resolve_block_hash(engine, tag)?
        .map_or_else(|| Ok(None), |hash| engine.block_by_hash(hash).map_err(rpc_err))
}

/// The state provider at a block-number-or-tag, or a "block unknown" error. This is a real
/// historical overlay for blocks below the tip: reth composes the in-memory blocks' trie updates on
/// top of the persisted genesis state, so account/storage reads and `eth_getProof` are answered at
/// exactly that block's state — the property op-node's `OutputV0AtBlock` relies on when it verifies
/// a message-passer proof against a past block's state root.
fn state_at_tag(engine: &TestEngine, tag: &str) -> Result<StateProviderBox, ErrorObjectOwned> {
    let hash = resolve_block_hash(engine, tag)?
        .ok_or_else(|| rpc_err(format!("block {tag:?} is unknown")))?;
    engine
        .chain
        .state_at(hash)
        .map_err(rpc_err)?
        .ok_or_else(|| rpc_err(format!("no state for block {tag:?}")))
}

/// Encode a `u64` as a `0x`-prefixed hex quantity — the JSON form `eth_*` numeric results use.
fn quantity(n: u64) -> Value {
    Value::String(format!("0x{n:x}"))
}

/// Encode a [`U256`] as a `0x`-prefixed hex quantity (balances, storage values as quantities).
fn u256_quantity(v: U256) -> Value {
    Value::String(format!("0x{v:x}"))
}

/// Resolve the optional block tag argument shared by the account-read methods, defaulting to
/// `latest`.
fn tag_or_latest(tag: Option<String>) -> String {
    tag.unwrap_or_else(|| "latest".to_string())
}

/// Build the JSON-RPC module serving `engine_*`, `eth_*`, and `optest_*` over `engine`.
pub fn build_module(engine: SharedEngine) -> RpcModule<SharedEngine> {
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

    // The head a forkchoice update reported SYNCING for and has not since resolved, or the zero
    // hash. A real EL would snap-sync towards it over devp2p; the Go harness reads this and
    // backfills the missing blocks from the peer engine (`optest_blockPayloadByNumber` ->
    // `optest_importBlock`), standing in for the p2p sync this ephemeral engine cannot do.
    m.register_method("optest_syncTarget", |_params, ctx, _| {
        let target = lock(ctx).sync_target().unwrap_or(B256::ZERO);
        serde_json::to_value(target).map_err(rpc_err)
    })
    .expect("register method");

    // The canonical block at `number`, re-expressed as the full execution data an
    // `engine_newPayload` would carry (payload plus fork-specific sidecar), or null if unknown.
    // The sync backfill reads this from the peer engine and feeds it to `optest_importBlock`.
    m.register_method("optest_blockPayloadByNumber", |params, ctx, _| {
        let number: u64 = params.one().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match engine.block_by_number(number).map_err(rpc_err)? {
            Some(block) => {
                let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
                serde_json::to_value(OpExecutionData::new(payload, sidecar)).map_err(rpc_err)?
            }
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    // Import a full execution payload obtained from a peer engine during sync backfill: the
    // ordinary `new_payload` validate-execute path plus a head advance, as a real EL's block-sync
    // insertion does. It exists as a distinct method so the harness can transfer the whole
    // `OpExecutionData` in one value rather than reconstructing the versioned `engine_newPayload`
    // arguments.
    m.register_method("optest_importBlock", |params, ctx, _| {
        let data: OpExecutionData = params.one().map_err(rpc_err)?;
        let status = lock(ctx).import_block(data).map_err(rpc_err)?;
        serde_json::to_value(status).map_err(rpc_err)
    })
    .expect("register method");

    // --- eth_ ---

    m.register_method("eth_chainId", |_params, ctx, _| {
        Ok::<_, ErrorObjectOwned>(quantity(lock(ctx).chain.chain_id()))
    })
    .expect("register method");

    m.register_method("eth_blockNumber", |_params, ctx, _| {
        Ok::<_, ErrorObjectOwned>(quantity(lock(ctx).chain.latest_header().number))
    })
    .expect("register method");

    m.register_method("eth_getBlockByNumber", |params, ctx, _| {
        let (tag, full): (String, Option<bool>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match resolve_block(&engine, &tag)? {
            Some(block) => block_json(&engine, &block, full.unwrap_or(false))?,
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m.register_method("eth_getBlockByHash", |params, ctx, _| {
        let (hash, full): (B256, Option<bool>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match engine.block_by_hash(hash).map_err(rpc_err)? {
            Some(block) => block_json(&engine, &block, full.unwrap_or(false))?,
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    // Account state reads, all at a historical block tag. `eth_getProof` is the load-bearing one:
    // op-node runs it against the L2ToL1MessagePasser predeploy and verifies the returned account +
    // storage proof against the block's state root to derive the withdrawals (output) root.
    m.register_method("eth_getProof", |params, ctx, _| {
        let (address, keys, tag): (Address, Vec<JsonStorageKey>, Option<String>) =
            params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let state = state_at_tag(&engine, &tag_or_latest(tag))?;
        let slots: Vec<B256> = keys.iter().map(JsonStorageKey::as_b256).collect();
        // A default (empty) `TrieInput` is correct here: the historical overlay provider already
        // folds the in-memory blocks' trie changes into its own input before computing the proof.
        let proof = state.proof(Default::default(), address, &slots).map_err(rpc_err)?;
        serde_json::to_value(proof.into_eip1186_response(keys)).map_err(rpc_err)
    })
    .expect("register method");

    m.register_method("eth_getBalance", |params, ctx, _| {
        let (address, tag): (Address, Option<String>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let balance = state_at_tag(&engine, &tag_or_latest(tag))?
            .account_balance(&address)
            .map_err(rpc_err)?
            .unwrap_or_default();
        Ok::<Value, ErrorObjectOwned>(u256_quantity(balance))
    })
    .expect("register method");

    // A raw transaction is parked in the engine's pending buffer (no auto-inclusion); the Go tests'
    // `ActL2IncludeTx(from)` later includes it via `optest_includeNextTx`.
    m.register_method("eth_sendRawTransaction", |params, ctx, _| {
        let raw: Bytes = params.one().map_err(rpc_err)?;
        let hash = lock(ctx).send_raw_transaction(raw.as_ref()).map_err(rpc_err)?;
        serde_json::to_value(hash).map_err(rpc_err)
    })
    .expect("register method");

    m.register_method("eth_getTransactionCount", |params, ctx, _| {
        let (address, tag): (Address, Option<String>) = params.parse().map_err(rpc_err)?;
        let tag = tag_or_latest(tag);
        let engine = lock(ctx);
        // "pending" folds in the parked buffer so the caller's next-nonce read accounts for txs it
        // has already submitted but not yet had included; every other tag reads committed state.
        let nonce = if tag == "pending" {
            engine.pending_nonce(address).map_err(rpc_err)?
        } else {
            state_at_tag(&engine, &tag)?
                .account_nonce(&address)
                .map_err(rpc_err)?
                .unwrap_or_default()
        };
        Ok::<Value, ErrorObjectOwned>(quantity(nonce))
    })
    .expect("register method");

    m.register_method("eth_getCode", |params, ctx, _| {
        let (address, tag): (Address, Option<String>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let code = state_at_tag(&engine, &tag_or_latest(tag))?
            .account_code(&address)
            .map_err(rpc_err)?
            .map(|bytecode| bytecode.original_bytes())
            .unwrap_or_default();
        serde_json::to_value(code).map_err(rpc_err)
    })
    .expect("register method");

    // OP-enriched receipts. op-node's `FetchReceipts` (RPCKindStandard) calls
    // `eth_getBlockReceipts` with the block hash, falling back to batched
    // `eth_getTransactionReceipt`; both must return receipts whose consensus re-encoding
    // reproduces the header receipts-root (op-node's validateReceipts), which is exactly what
    // reth's OpReceiptConverter path preserves.
    m.register_method("eth_getBlockReceipts", |params, ctx, _| {
        let tag: String = params.one().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match resolve_block_hash(&engine, &tag)? {
            Some(hash) => match engine.rpc_receipts_by_block_hash(hash).map_err(rpc_err)? {
                Some(receipts) => serde_json::to_value(receipts).map_err(rpc_err)?,
                None => Value::Null,
            },
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m.register_method("eth_getTransactionReceipt", |params, ctx, _| {
        let tx_hash: B256 = params.one().map_err(rpc_err)?;
        let value = match lock(ctx).rpc_receipt_by_tx_hash(tx_hash).map_err(rpc_err)? {
            Some(receipt) => serde_json::to_value(receipt).map_err(rpc_err)?,
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m.register_method("eth_getTransactionByHash", |params, ctx, _| {
        let tx_hash: B256 = params.one().map_err(rpc_err)?;
        let engine = lock(ctx);
        let Some((tx, meta)) = engine.chain.transaction_by_hash(tx_hash).map_err(rpc_err)? else {
            return Ok(Value::Null);
        };
        let signer = tx.recover_signer().map_err(|e| rpc_err(format!("recover signer: {e}")))?;
        let tx_info = engine
            .chain
            .rpc_transaction_info(
                &tx,
                TransactionInfo {
                    hash: Some(tx_hash),
                    index: Some(meta.index),
                    block_hash: Some(meta.block_hash),
                    block_number: Some(meta.block_number),
                    base_fee: meta.base_fee,
                    block_timestamp: Some(meta.timestamp),
                },
            )
            .map_err(rpc_err)?;
        let rpc_tx =
            OpRpcTransaction::from_transaction(Recovered::new_unchecked(tx, signer), tx_info);
        serde_json::to_value(rpc_tx).map_err(rpc_err)
    })
    .expect("register method");

    // Read-only EVM execution at a block's state. The call-object argument is go-ethereum's
    // `toCallArg` form; the trailing block tag is optional (defaulting to latest, matching geth).
    m.register_method("eth_call", |params, ctx, _| {
        let mut seq = params.sequence();
        let request: alloy_rpc_types_eth::TransactionRequest = seq.next().map_err(rpc_err)?;
        let tag: Option<String> = seq.optional_next().map_err(rpc_err)?;
        let engine = lock(ctx);
        let hash = resolve_block_hash(&engine, &tag_or_latest(tag))?
            .ok_or_else(|| rpc_err("block is unknown"))?;
        let output = engine.eth_call(hash, request).map_err(call_err)?;
        serde_json::to_value(output).map_err(rpc_err)
    })
    .expect("register method");

    m.register_method("eth_estimateGas", |params, ctx, _| {
        let mut seq = params.sequence();
        let request: alloy_rpc_types_eth::TransactionRequest = seq.next().map_err(rpc_err)?;
        let tag: Option<String> = seq.optional_next().map_err(rpc_err)?;
        let engine = lock(ctx);
        let hash = resolve_block_hash(&engine, &tag_or_latest(tag))?
            .ok_or_else(|| rpc_err("block is unknown"))?;
        let gas = engine.estimate_gas(hash, request).map_err(call_err)?;
        Ok::<Value, ErrorObjectOwned>(quantity(gas))
    })
    .expect("register method");

    m.register_method("eth_getStorageAt", |params, ctx, _| {
        let (address, slot, tag): (Address, B256, Option<String>) =
            params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = state_at_tag(&engine, &tag_or_latest(tag))?
            .storage(address, slot)
            .map_err(rpc_err)?
            .unwrap_or_default();
        // eth_getStorageAt returns a full 32-byte word, not a trimmed quantity.
        serde_json::to_value(B256::from(value.to_be_bytes::<32>())).map_err(rpc_err)
    })
    .expect("register method");

    m
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::testsupport::{
        deposit_tx, depositor, encode, fcu, payload_attrs, test_engine, user_sender, user_tx,
    };
    use jsonrpsee::core::server::MethodsError;

    /// An engine whose head is block 1, holding a single deposit.
    fn engine_with_deposit() -> (SharedEngine, B256) {
        let mut engine = test_engine(user_sender());
        let genesis = engine.chain.genesis_hash();
        let attrs = payload_attrs(2, vec![encode(&deposit_tx(depositor()))], false);
        let id = engine
            .forkchoice_updated_as_op_node(fcu(genesis), Some(attrs))
            .unwrap()
            .payload_id
            .unwrap();
        let data = engine.get_payload(id).unwrap();
        let block_hash = data.payload.block_hash();
        assert!(engine.new_payload(data).unwrap().is_valid());
        assert!(engine.forkchoice_updated_as_op_node(fcu(block_hash), None).unwrap().is_valid());
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

    #[test]
    fn revert_with_an_overflowing_reason_length_does_not_panic() {
        // Error(string) selector, offset 0x20, then a length so large that adding the 64-byte
        // header overflows.
        let mut output = vec![0x08, 0xc3, 0x79, 0xa0];
        output.extend_from_slice(&U256::from(32).to_be_bytes::<32>());
        output.extend_from_slice(&U256::from(usize::MAX).to_be_bytes::<32>());
        let err = call_err(crate::Error::Revert(output.into()));
        assert_eq!(err.code(), REVERT_ERR_CODE);
        assert!(err.message().starts_with("execution reverted"), "{}", err.message());
    }

    #[test]
    fn revert_reason_is_decoded() {
        // Error(string) selector, offset 0x20, length 4, then "nope" right-padded to a word.
        let mut output = vec![0x08, 0xc3, 0x79, 0xa0];
        output.extend_from_slice(&U256::from(32).to_be_bytes::<32>());
        output.extend_from_slice(&U256::from(4).to_be_bytes::<32>());
        output.extend_from_slice(&B256::right_padding_from(b"nope").0);
        let err = call_err(crate::Error::Revert(output.into()));
        assert_eq!(err.message(), "execution reverted: nope");
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
