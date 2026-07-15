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

use alloy_consensus::transaction::{Recovered, SignerRecoverable, TransactionInfo};
use alloy_eips::{eip2718::Encodable2718, eip7685::Requests};
use alloy_primitives::{Address, B256, Bytes, U256, keccak256};
use alloy_rpc_types_engine::{
    CancunPayloadFields, ForkchoiceState, PayloadId, PraguePayloadFields,
};
use alloy_serde::JsonStorageKey;
use jsonrpsee::{RpcModule, types::ErrorObjectOwned};
use op_alloy_consensus::transaction::{OpDepositInfo, OpTransactionInfo};
use op_alloy_rpc_types::Transaction as OpRpcTransaction;
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadSidecar,
    OpPayloadAttributes,
};
use reth_optimism_primitives::OpBlock;
use reth_storage_api::{StateProofProvider, StateProvider, StateProviderBox};
use serde_json::{Value, json};

use crate::{IncludeNextOutcome, IncludeTxOutcome, TestEngine};

/// Shared, mutably-accessed engine behind the RPC module.
pub type SharedEngine = Arc<Mutex<TestEngine>>;

/// JSON-RPC error code for engine/execution errors (in go-ethereum's server error range).
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

/// Map a `forkchoice_updated` error: invalid payload attributes become go-ethereum's `-38003`;
/// anything else is a generic engine error.
fn fcu_err(err: crate::Error) -> ErrorObjectOwned {
    match &err {
        crate::Error::InvalidPayloadAttributes(_) => ErrorObjectOwned::owned(
            INVALID_PAYLOAD_ATTRIBUTES_ERR_CODE,
            err.to_string(),
            None::<()>,
        ),
        _ => rpc_err(err),
    }
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
            ErrorObjectOwned::owned(REVERT_ERR_CODE, revert_msg(&output), Some(data))
        }
        other => rpc_err(other),
    }
}

/// Format a revert as geth does: `execution reverted` plus the decoded `Error(string)` reason if
/// the output carries one.
fn revert_msg(output: &[u8]) -> String {
    // Error(string) selector, then ABI-encoded (offset, length, bytes).
    const ERROR_STRING_SELECTOR: [u8; 4] = [0x08, 0xc3, 0x79, 0xa0];
    let reason = output.strip_prefix(&ERROR_STRING_SELECTOR[..]).and_then(|abi| {
        let len = usize::try_from(U256::from_be_slice(abi.get(32..64)?)).ok()?;
        abi.get(64..64 + len).map(|s| String::from_utf8_lossy(s).into_owned())
    });
    reason.map_or_else(
        || "execution reverted".to_string(),
        |reason| format!("execution reverted: {reason}"),
    )
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
fn block_json(block: &OpBlock, full: bool) -> Result<Value, ErrorObjectOwned> {
    let header = alloy_rpc_types_eth::Header::new(block.header.clone());
    let block_hash = header.hash;
    let mut value = serde_json::to_value(&header).map_err(rpc_err)?;
    let txs = if full {
        full_transactions(block, block_hash)?
    } else {
        let hashes: Vec<B256> =
            block.body.transactions.iter().map(|tx| keccak256(tx.encoded_2718())).collect();
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
/// The signer is recovered per transaction (deposits recover to their `from` field). The
/// deposit-nonce/receipt-version RPC fields are receipt-derived and not needed to reconstruct the
/// transaction, so they are left unset — op-node rebuilds the deposit from the consensus fields.
fn full_transactions(block: &OpBlock, block_hash: B256) -> Result<Value, ErrorObjectOwned> {
    let base_fee = block.header.base_fee_per_gas;
    let mut out = Vec::with_capacity(block.body.transactions.len());
    for (index, tx) in block.body.transactions.iter().enumerate() {
        let signer = tx.recover_signer().map_err(|e| rpc_err(format!("recover signer: {e}")))?;
        let recovered = Recovered::new_unchecked(tx.clone(), signer);
        let tx_info = OpTransactionInfo::new(
            TransactionInfo {
                hash: None,
                index: Some(index as u64),
                block_hash: Some(block_hash),
                block_number: Some(block.header.number),
                base_fee,
                block_timestamp: Some(block.header.timestamp),
            },
            OpDepositInfo::default(),
        );
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

    let register_fcu = |m: &mut RpcModule<SharedEngine>, name: &'static str| {
        m.register_method(name, |params, ctx, _| {
            let (state, attrs): (ForkchoiceState, Option<OpPayloadAttributes>) =
                params.parse().map_err(rpc_err)?;
            let updated = lock(ctx).forkchoice_updated(state, attrs).map_err(fcu_err)?;
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
    // computes the eligible nonce, executes the matching parked tx, and drains it from the buffer.
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
            Some(block) => block_json(&block, full.unwrap_or(false))?,
            None => Value::Null,
        };
        Ok::<Value, ErrorObjectOwned>(value)
    })
    .expect("register method");

    m.register_method("eth_getBlockByHash", |params, ctx, _| {
        let (hash, full): (B256, Option<bool>) = params.parse().map_err(rpc_err)?;
        let engine = lock(ctx);
        let value = match engine.block_by_hash(hash).map_err(rpc_err)? {
            Some(block) => block_json(&block, full.unwrap_or(false))?,
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
    // `ActL2IncludeTx(from)` later drains it via `optest_includeNextTx`.
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
        let tx_info = OpTransactionInfo::new(
            TransactionInfo {
                hash: Some(tx_hash),
                index: Some(meta.index),
                block_hash: Some(meta.block_hash),
                block_number: Some(meta.block_number),
                base_fee: meta.base_fee,
                block_timestamp: Some(meta.timestamp),
            },
            OpDepositInfo::default(),
        );
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
