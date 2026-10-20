//! L2 block reads for log-commitment batches, through the endpoints and transports that kona-host
//! uses.
//!
//! A read by hash is checked against that hash, and receipts against their receipts root, so a
//! node error is an error, never a different block.

use alloy_consensus::{Header, Sealed, proofs::ordered_trie_root_with_encoder};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{B256, Bytes, U64, keccak256};
use alloy_provider::{Provider, RootProvider};
use alloy_rlp::Decodable;
use alloy_rpc_client::BuiltInConnectionString;
use anyhow::{Context, Result, ensure};
use op_alloy_consensus::OpReceiptEnvelope;

/// An L2 execution node, read through `debug_getRawHeader` and `debug_getRawReceipts`.
#[derive(Debug, Clone)]
pub struct L2BlockClient {
    provider: RootProvider,
}

impl L2BlockClient {
    /// Connects to the node at `url` over HTTP, `WebSocket`, or IPC.
    pub async fn connect(url: &str) -> Result<Self> {
        let connection = url
            .parse::<BuiltInConnectionString>()
            .with_context(|| format!("invalid L2 node URL {url}"))?;
        let provider = match connection {
            BuiltInConnectionString::Http(url) => RootProvider::new_http(url),
            connection => RootProvider::connect_with(connection)
                .await
                .with_context(|| format!("failed to connect to L2 node {url}"))?,
        };
        Ok(Self { provider })
    }

    /// The node's chain ID.
    pub async fn chain_id(&self) -> Result<u64> {
        self.provider.get_chain_id().await.context("eth_chainId failed")
    }

    /// The header of block `hash`.
    pub async fn header_by_hash(&self, hash: B256) -> Result<Header> {
        let raw: Bytes = self
            .provider
            .client()
            .request("debug_getRawHeader", (hash,))
            .await
            .with_context(|| format!("debug_getRawHeader({hash}) failed"))?;
        decode_header(&raw, hash)
    }

    /// The header at `number`. Only its number is checked, so the caller trusts the node's
    /// canonical chain at that height.
    pub async fn header_by_number(&self, number: u64) -> Result<Sealed<Header>> {
        let raw: Bytes = self
            .provider
            .client()
            .request("debug_getRawHeader", (U64::from(number),))
            .await
            .with_context(|| format!("debug_getRawHeader({number}) failed"))?;
        let hash = keccak256(&raw);
        let header = decode_header(&raw, hash)?;
        ensure!(header.number == number, "node returned block {} for {number}", header.number);
        Ok(Sealed::new_unchecked(header, hash))
    }

    /// The receipts of block `hash`, which must hash to `receipts_root`.
    pub async fn receipts(
        &self,
        hash: B256,
        receipts_root: B256,
    ) -> Result<Vec<OpReceiptEnvelope>> {
        let raw: Vec<Bytes> = self
            .provider
            .client()
            .request("debug_getRawReceipts", (hash,))
            .await
            .with_context(|| format!("debug_getRawReceipts({hash}) failed"))?;
        decode_receipts(&raw, receipts_root).with_context(|| format!("receipts of block {hash}"))
    }
}

/// Decodes a raw header that must hash to `hash`.
fn decode_header(raw: &[u8], hash: B256) -> Result<Header> {
    let actual = keccak256(raw);
    ensure!(actual == hash, "header hashes to {actual}, expected {hash}");
    let mut buf = raw;
    let header =
        Header::decode(&mut buf).with_context(|| format!("header {hash} does not decode"))?;
    ensure!(buf.is_empty(), "header {hash} has {} trailing bytes", buf.len());
    Ok(header)
}

/// Decodes raw EIP-2718 receipts that must hash to `receipts_root`.
fn decode_receipts(raw: &[Bytes], receipts_root: B256) -> Result<Vec<OpReceiptEnvelope>> {
    let root = ordered_trie_root_with_encoder(raw, |receipt, out| out.extend_from_slice(receipt));
    ensure!(
        root == receipts_root,
        "receipts hash to {root}, expected receipts root {receipts_root}"
    );
    raw.iter()
        .enumerate()
        .map(|(index, receipt)| {
            OpReceiptEnvelope::decode_2718(&mut receipt.as_ref())
                .with_context(|| format!("receipt {index} does not decode"))
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use alloy_consensus::{Receipt, ReceiptWithBloom};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, LogData};
    use alloy_rlp::Encodable;

    use super::*;

    fn raw_receipts() -> Vec<Bytes> {
        (0..3u8)
            .map(|seed| {
                let log = alloy_primitives::Log {
                    address: Address::repeat_byte(seed),
                    data: LogData::new_unchecked(vec![B256::repeat_byte(seed)], Bytes::new()),
                };
                OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
                    receipt: Receipt {
                        status: true.into(),
                        cumulative_gas_used: 21_000 * u64::from(seed + 1),
                        logs: vec![log],
                    },
                    logs_bloom: Default::default(),
                })
                .encoded_2718()
                .into()
            })
            .collect()
    }

    /// Pins errors, not data, for a header or receipts that do not match their hash.
    #[test]
    fn l2_checks_reject_wrong_header_hash_and_receipts_root() {
        let header = Header { number: 7, timestamp: 14, ..Default::default() };
        let mut raw = Vec::new();
        header.encode(&mut raw);
        assert_eq!(decode_header(&raw, keccak256(&raw)).unwrap(), header);
        let err = decode_header(&raw, B256::repeat_byte(1)).unwrap_err();
        assert!(err.to_string().contains("hashes to"), "{err}");
        let mut trailing = raw.clone();
        trailing.push(0);
        let err = decode_header(&trailing, keccak256(&trailing)).unwrap_err();
        assert!(err.to_string().contains("trailing"), "{err}");

        let receipts = raw_receipts();
        let root = ordered_trie_root_with_encoder(&receipts, |r, out| out.extend_from_slice(r));
        assert_eq!(decode_receipts(&receipts, root).unwrap().len(), 3);
        let err = decode_receipts(&receipts[..2], root).unwrap_err();
        assert!(err.to_string().contains("expected receipts root"), "{err}");
        let mut garbled = receipts;
        garbled[1] = Bytes::from_static(&[0x02, 0xc0]);
        let root = ordered_trie_root_with_encoder(&garbled, |r, out| out.extend_from_slice(r));
        let err = decode_receipts(&garbled, root).unwrap_err();
        assert!(err.to_string().contains("receipt 1 does not decode"), "{err}");
    }
}
