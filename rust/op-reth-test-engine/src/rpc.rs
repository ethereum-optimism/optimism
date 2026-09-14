//! JSON-RPC surface for the test engine, served over a Unix socket (reth-ipc, go-ethereum
//! `rpc.DialIPC`-compatible — the #20415 transport) by the companion binary.
//!
//! Three namespaces mirror what `op-e2e/actions` drives against the in-process op-geth engine:
//! `engine_*` (the versioned newPayload/forkchoiceUpdated/getPayload trio), read-only `eth_*`
//! queries, and `optest_*` — the sequencing hooks (`includeTx`, `remainingBlockGas`,
//! `forcedEmpty`, `setForceEmpty`) that replace the direct `L2EngineAPI` method calls.
//!
//! The engine's methods take `&mut self`, so the module context is an `Arc<Mutex<TestEngine>>`.
//! Action tests are single-threaded, so the coarse lock is not a throughput concern; a poisoned
//! lock is recovered rather than propagated so one failed request can't wedge the process.

use std::sync::{Arc, Mutex};

use alloy_eips::{eip2718::Encodable2718, eip7685::Requests};
use alloy_primitives::{B256, Bytes, keccak256};
use alloy_rpc_types_engine::{
    CancunPayloadFields, ForkchoiceState, PayloadId, PraguePayloadFields,
};
use jsonrpsee::{RpcModule, types::ErrorObjectOwned};
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadSidecar,
    OpPayloadAttributes,
};
use reth_optimism_primitives::OpBlock;
use serde_json::{Value, json};

use crate::{IncludeTxOutcome, TestEngine};

/// Shared, mutably-accessed engine behind the RPC module.
pub type SharedEngine = Arc<Mutex<TestEngine>>;

/// JSON-RPC error code for engine/execution errors (in go-ethereum's server error range).
const ERR_CODE: i32 = -38000;

fn rpc_err(msg: impl std::fmt::Display) -> ErrorObjectOwned {
    ErrorObjectOwned::owned(ERR_CODE, msg.to_string(), None::<()>)
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
/// hash) plus the block's transaction hashes. Faithful enough for the chain-shape assertions the
/// action tests make; full transaction objects are not needed by any caller yet.
fn block_json(block: &OpBlock) -> Result<Value, ErrorObjectOwned> {
    let header = alloy_rpc_types_eth::Header::new(block.header.clone());
    let mut value = serde_json::to_value(&header).map_err(rpc_err)?;
    let tx_hashes: Vec<B256> =
        block.body.transactions.iter().map(|tx| keccak256(tx.encoded_2718())).collect();
    if let Value::Object(map) = &mut value {
        map.insert("transactions".into(), serde_json::to_value(tx_hashes).map_err(rpc_err)?);
        map.insert("uncles".into(), json!([]));
    }
    Ok(value)
}

/// Resolve a block-number-or-tag string (`latest`/`safe`/`finalized`/`earliest`/`pending` or a
/// `0x`-hex number) to a block, or `None` if unknown.
fn resolve_block(engine: &TestEngine, tag: &str) -> Result<Option<OpBlock>, ErrorObjectOwned> {
    let hash = match tag {
        "latest" | "pending" => Some(engine.chain.latest_header().hash()),
        "safe" => engine.chain.safe_header().map(|h| h.hash()),
        "finalized" => engine.chain.finalized_header().map(|h| h.hash()),
        "earliest" => return engine.block_by_number(0).map_err(rpc_err),
        num => {
            let n = u64::from_str_radix(num.trim_start_matches("0x"), 16)
                .map_err(|e| rpc_err(format!("invalid block number {num:?}: {e}")))?;
            return engine.block_by_number(n).map_err(rpc_err);
        }
    };
    hash.map_or_else(|| Ok(None), |hash| engine.block_by_hash(hash).map_err(rpc_err))
}

/// Encode a `u64` as a `0x`-prefixed hex quantity — the JSON form `eth_*` numeric results use.
fn quantity(n: u64) -> Value {
    Value::String(format!("0x{n:x}"))
}

/// Build the JSON-RPC module serving `engine_*`, `eth_*`, and `optest_*` over `engine`.
pub fn build_module(engine: SharedEngine) -> RpcModule<SharedEngine> {
    let mut m = RpcModule::new(engine);

    // --- engine_ ---

    let register_fcu = |m: &mut RpcModule<SharedEngine>, name: &'static str| {
        m.register_method(name, |params, ctx, _| {
            let (state, attrs): (ForkchoiceState, Option<OpPayloadAttributes>) =
                params.parse().map_err(rpc_err)?;
            let updated = lock(ctx).forkchoice_updated(state, attrs).map_err(rpc_err)?;
            serde_json::to_value(updated).map_err(rpc_err)
        })
        .expect("register method");
    };
    register_fcu(&mut m, "engine_forkchoiceUpdatedV1");
    register_fcu(&mut m, "engine_forkchoiceUpdatedV2");
    register_fcu(&mut m, "engine_forkchoiceUpdatedV3");

    let register_get_payload = |m: &mut RpcModule<SharedEngine>, name: &'static str| {
        m.register_method(name, |params, ctx, _| {
            let id: PayloadId = params.one().map_err(rpc_err)?;
            let data = lock(ctx).get_payload(id).map_err(rpc_err)?;
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

    m.register_method("eth_chainId", |_params, ctx, _| {
        Ok::<_, ErrorObjectOwned>(quantity(lock(ctx).chain.chain_id()))
    })
    .expect("register method");

    m.register_method("eth_blockNumber", |_params, ctx, _| {
        Ok::<_, ErrorObjectOwned>(quantity(lock(ctx).chain.latest_header().number))
    })
    .expect("register method");

    m.register_method("eth_getBlockByNumber", |params, ctx, _| {
        // Second param (full-transactions) is accepted for compatibility but ignored: results
        // always carry transaction hashes.
        let (tag, _full): (String, Option<bool>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match resolve_block(&engine, &tag)? {
            Some(block) => block_json(&block)?,
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m.register_method("eth_getBlockByHash", |params, ctx, _| {
        let (hash, _full): (B256, Option<bool>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match engine.block_by_hash(hash).map_err(rpc_err)? {
            Some(block) => block_json(&block)?,
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m
}
