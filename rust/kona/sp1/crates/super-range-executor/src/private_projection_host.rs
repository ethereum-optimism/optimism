//! Operator-side witness collection. None of this I/O runs inside derivation or the guest.
use alloy_eips::{BlockNumHash, Encodable2718};
use alloy_primitives::{B256, Bytes, hex, keccak256};
use alloy_rpc_types_engine::PayloadAttributes;
use anyhow::{Result, ensure};
use jsonrpsee::ws_client::{WsClient, WsClientBuilder};
use jsonrpsee_core::{client::ClientT, traits::ToRpcParams};
use jsonrpsee_http_client::{HttpClient, HttpClientBuilder};
use kona_genesis::RollupConfig;
use kona_sp1_client_utils::private_projection::{ProjectionBlock, PublicInputs, Witness, execute};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use serde::{Deserialize, de::DeserializeOwned};
use std::{collections::BTreeMap, io::Read, time::Duration};

#[derive(Deserialize)]
struct Request {
    private_rpc: String,
    projection_rpc: String,
    private_config: Bytes,
    projection_config: Bytes,
    dependency_set: Bytes,
    parent_hash: B256,
    anchor: BlockNumHash,
    anchor_output: B256,
    blocks: Vec<ProjectionBlock>,
    private_data: Bytes,
    expected_digest: B256,
}

#[derive(Deserialize)]
struct ExecutionWitness {
    state: Vec<Bytes>,
    codes: Vec<Bytes>,
    #[serde(default)]
    headers: Vec<Bytes>,
}

enum Rpc {
    Http(HttpClient),
    Ws(WsClient),
}
impl Rpc {
    async fn connect(url: &str) -> Result<Self> {
        if url.starts_with("ws://") || url.starts_with("wss://") {
            Ok(Self::Ws(
                WsClientBuilder::default()
                    .request_timeout(Duration::from_secs(60))
                    .max_response_size(100 * 1024 * 1024)
                    .build(url)
                    .await?,
            ))
        } else {
            Ok(Self::Http(
                HttpClientBuilder::default()
                    .request_timeout(Duration::from_secs(60))
                    .max_response_size(100 * 1024 * 1024)
                    .build(url)?,
            ))
        }
    }
    async fn request<R: DeserializeOwned, P: ToRpcParams + Send>(
        &self,
        method: &str,
        params: P,
    ) -> Result<R> {
        Ok(match self {
            Self::Http(client) => client.request(method, params).await?,
            Self::Ws(client) => client.request(method, params).await?,
        })
    }
}

async fn block(client: &Rpc, number: u64) -> Result<OpBlock> {
    let value: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> = client
        .request("eth_getBlockByNumber", jsonrpsee_core::rpc_params![format!("{number:#x}"), true])
        .await?;
    Ok(value.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner()))
}

fn attributes(block: &OpBlock, recovery: bool, cfg: &RollupConfig) -> Result<OpPayloadAttributes> {
    let h = &block.header;
    let eip_1559_params = if cfg.is_holocene_active(h.timestamp) {
        ensure!(h.extra_data.len() >= 9, "missing Holocene header parameters");
        Some(h.extra_data[1..9].try_into()?)
    } else {
        None
    };
    let min_base_fee = if cfg.is_jovian_active(h.timestamp) {
        ensure!(h.extra_data.len() == 17, "missing Jovian header parameters");
        Some(u64::from_be_bytes(h.extra_data[9..17].try_into()?))
    } else {
        None
    };
    Ok(OpPayloadAttributes {
        payload_attributes: PayloadAttributes {
            timestamp: h.timestamp,
            prev_randao: h.mix_hash,
            suggested_fee_recipient: h.beneficiary,
            withdrawals: block.body.withdrawals.as_ref().map(|w| w.to_vec()),
            parent_beacon_block_root: h.parent_beacon_block_root,
            ..Default::default()
        },
        transactions: Some(
            block
                .body
                .transactions
                .iter()
                .filter(|tx| recovery || matches!(tx, OpTxEnvelope::Deposit(_)))
                .map(|tx| tx.encoded_2718().into())
                .collect(),
        ),
        no_tx_pool: Some(true),
        gas_limit: Some(h.gas_limit),
        eip_1559_params,
        min_base_fee,
    })
}

/// Run the real native relation before creating the explicitly insecure envelope.
/// Private bytes stay in memory/stdin; stdout contains only public envelope bytes.
pub(super) async fn publish() -> Result<()> {
    let mut encoded = Vec::new();
    std::io::stdin().take(128 * 1024 * 1024 + 1).read_to_end(&mut encoded)?;
    ensure!(encoded.len() <= 128 * 1024 * 1024, "publication request too large");
    let r: Request = serde_json::from_slice(&encoded)?;
    let cfg: RollupConfig = serde_json::from_slice(&r.private_config)?;
    let projection: RollupConfig = serde_json::from_slice(&r.projection_config)?;
    ensure!(
        projection.private_projection.as_ref().is_some_and(|c| c.verifier == "execution-mock-v1"),
        "publication request requires execution-mock-v1"
    );
    ensure!(!r.blocks.is_empty(), "empty range");
    ensure!(
        cfg.block_time != 0 && r.blocks[0].timestamp >= cfg.genesis.l2_time,
        "invalid range geometry"
    );
    let first = cfg
        .genesis
        .l2
        .number
        .checked_add((r.blocks[0].timestamp - cfg.genesis.l2_time) / cfg.block_time)
        .ok_or_else(|| anyhow::anyhow!("range overflow"))?;
    let last = first
        .checked_add(r.blocks.len() as u64 - 1)
        .ok_or_else(|| anyhow::anyhow!("range overflow"))?;
    ensure!(r.anchor.number < first, "invalid anchor");
    let private = Rpc::connect(&r.private_rpc).await?;
    let public = Rpc::connect(&r.projection_rpc).await?;
    let anchor_header = block(&private, r.anchor.number).await?.header;
    let mut preimages = BTreeMap::new();
    let mut attrs = Vec::new();
    let mut recovery = Vec::new();
    let mut parent = anchor_header.hash_slow();
    let mut identities = Vec::new();
    for number in r.anchor.number + 1..=last {
        let b = block(&private, number).await?;
        ensure!(b.header.parent_hash == parent, "private chain changed during witness collection");
        parent = b.header.hash_slow();
        identities.push((number, parent));
        let w: ExecutionWitness = private
            .request("debug_executionWitness", jsonrpsee_core::rpc_params![format!("{number:#x}")])
            .await?;
        for raw in w.state.into_iter().chain(w.codes).chain(w.headers) {
            preimages.insert(keccak256(&raw), raw);
        }
        // The output-root computation reads the message-passer account even if
        // this block never touched it. Add the parent-state account proof.
        let proof: alloy_rpc_types_eth::EIP1186AccountProofResponse = private
            .request(
                "eth_getProof",
                jsonrpsee_core::rpc_params![
                    "0x4200000000000000000000000000000000000016",
                    Vec::<String>::new(),
                    format!("{:#x}", number - 1)
                ],
            )
            .await?;
        for raw in proof.account_proof {
            preimages.insert(keccak256(&raw), raw);
        }
        if number < first {
            let replacement = block(&public, number).await?;
            attrs.push(attributes(&b, true, &cfg)?);
            recovery.push(replacement);
        } else {
            attrs.push(attributes(&b, false, &cfg)?);
        }
    }
    // Number-based debug RPC must not silently mix witnesses from two branches.
    // Re-execution also binds the anchor, private-data hash and terminal identity.
    for (number, hash) in identities {
        ensure!(
            block(&private, number).await?.header.hash_slow() == hash,
            "private witness reorged"
        );
    }
    let inputs = PublicInputs {
        private_config: r.private_config,
        projection_config: r.projection_config,
        dependency_set: r.dependency_set,
        parent_hash: r.parent_hash,
        anchor: r.anchor,
        anchor_output: r.anchor_output,
        recovery,
        attributes: attrs,
        blocks: r.blocks,
    };
    let witness = Witness { anchor_header, private_data: r.private_data, preimages };
    let result = execute(&inputs, &witness)?;
    ensure!(
        result.admission_digest == r.expected_digest,
        "native execution returned a different statement"
    );
    let mut proof = b"optimism.private-execution.mock.v1\0".to_vec();
    proof.extend_from_slice(result.admission_digest.as_slice());
    println!("0x{}", hex::encode(proof));
    Ok(())
}
