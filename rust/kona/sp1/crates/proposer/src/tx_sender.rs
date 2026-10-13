//! L1 transaction sending, modelled on Go `op-service/txmgr` (`sendTx`, `publishTx`,
//! `SendState`) for one key sending one transaction at a time.
//!
//! What a send does:
//! - Takes the `latest` nonce, so it replaces any transaction of ours still pending at that nonce.
//! - Estimates gas once and prices the transaction at the node's suggested tip plus twice the
//!   latest base fee, clamped by [`FeeCaps`].
//! - Bumps fees every 24s, and once when the pool rejects a publish as underpriced, using txmgr's
//!   replacement rule (at least +10% on tip and fee cap). When a bump is blocked by [`FeeCaps`] or
//!   fails, republishes the current version.
//! - Treats "already known", and a publish the node never answered, as published: a lost response
//!   can hide an accepted transaction. Polls every version it sent, concurrently and including ones
//!   whose publish failed, until one reaches [`NUM_CONFIRMATIONS`]. Receipt query errors are logged
//!   and retried.
//! - Aborts on "nonce too low" with txmgr's rules: at once if nothing was published, after 3
//!   otherwise, never while a version is mined.
//! - Bounds every RPC and signer call by 10s and the whole send by the caller's timeout.
//!
//! Not supported from txmgr, to keep the sender limited to what the proposer needs:
//! - Fee-limit multiplier and threshold; bumps are bounded by the tick rate, the send timeout and
//!   [`FeeCaps`].
//! - Minimum tip and base fee floors, and the max tip and base fee errors ([`FeeCaps`] clamps
//!   instead).
//! - Gas re-estimation on bumps, and caller-supplied gas limits.
//! - Rebroadcast of an unchanged transaction between bumps.
//! - Retries of "nonce too high", the mempool deadline, "address already reserved" handling and
//!   custom already-published errors.
//! - Blob transactions.
//! - Concurrent sends (`SendAsync`, `Queue`, cached nonces); callers serialize through
//!   `SignerLock`.
//! - Waiting for a mined transaction on shutdown, runtime reconfiguration and metrics.

use std::{collections::HashSet, future::IntoFuture, time::Duration};

use alloy_consensus::TxEnvelope;
use alloy_eips::{BlockNumberOrTag, Encodable2718};
use alloy_primitives::B256;
use alloy_provider::Provider;
use alloy_rpc_types_eth::{TransactionReceipt, TransactionRequest};
use anyhow::{Context, Result, anyhow, ensure};
use futures::{StreamExt, stream::FuturesUnordered};
use tokio::time::{Instant, MissedTickBehavior, interval_at, timeout};

use crate::signer::Signer;

/// Number of L1 confirmations required before a transaction is considered included.
pub const NUM_CONFIRMATIONS: u64 = 3;

/// Bound on each L1 RPC and signer call (Go txmgr `NetworkTimeout`).
const NETWORK_TIMEOUT: Duration = Duration::from_secs(10);

/// Interval between fee bumps of an unmined transaction (Go txmgr challenger
/// `ResubmissionTimeout`).
const RESUBMISSION_INTERVAL: Duration = Duration::from_secs(24);

/// Nonce-too-low publish errors tolerated after a successful publish before the send aborts
/// (Go txmgr `SafeAbortNonceTooLowCount`).
const SAFE_ABORT_NONCE_TOO_LOW_COUNT: u64 = 3;

/// Optional operator caps on EIP-1559 fee fields, applied to the initial fees and to every
/// bump. Unset caps leave the estimate.
#[derive(Clone, Copy, Debug, Default)]
pub struct FeeCaps {
    /// Cap on max fee per gas, in wei.
    pub max_fee_per_gas: Option<u128>,
    /// Cap on max priority fee per gas, in wei.
    pub max_priority_fee_per_gas: Option<u128>,
}

impl FeeCaps {
    /// Clamps both fees to the caps, then restores `tip <= fee_cap`.
    fn clamp(self, fees: Fees) -> Fees {
        let fee_cap = self.max_fee_per_gas.map_or(fees.fee_cap, |cap| fees.fee_cap.min(cap));
        let tip = self.max_priority_fee_per_gas.map_or(fees.tip, |cap| fees.tip.min(cap));
        Fees { tip: tip.min(fee_cap), fee_cap }
    }
}

/// EIP-1559 fees of a signed transaction, in wei.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct Fees {
    tip: u128,
    fee_cap: u128,
}

/// The node's fee suggestion: `eth_maxPriorityFeePerGas` and the latest block's base fee.
#[derive(Clone, Copy, Debug)]
struct SuggestedFees {
    tip: u128,
    base_fee: u128,
}

impl SuggestedFees {
    const fn fees(self) -> Fees {
        Fees { tip: self.tip, fee_cap: fee_cap_for(self.tip, self.base_fee) }
    }
}

/// Go txmgr `calcGasFeeCap`.
const fn fee_cap_for(tip: u128, base_fee: u128) -> u128 {
    tip.saturating_add(base_fee.saturating_mul(2))
}

/// `ceil(x * 1.1)`, the minimum replacement price geth accepts (Go txmgr `calcThresholdValue`).
const fn bump_threshold(x: u128) -> u128 {
    x.saturating_mul(110).saturating_add(99) / 100
}

/// Go txmgr `updateFees`: the lowest fees that satisfy the replacement rule for `old` while
/// paying at least the current suggestion.
const fn update_fees(old: Fees, suggested: SuggestedFees) -> Fees {
    let new = suggested.fees();
    let min_tip = bump_threshold(old.tip);
    let min_fee_cap = bump_threshold(old.fee_cap);
    match (new.tip >= min_tip, new.fee_cap >= min_fee_cap) {
        (true, true) => new,
        (true, false) => Fees { tip: new.tip, fee_cap: min_fee_cap },
        (false, true) => Fees { tip: min_tip, fee_cap: fee_cap_for(min_tip, suggested.base_fee) },
        (false, false) => Fees { tip: min_tip, fee_cap: min_fee_cap },
    }
}

/// Publication state of one logical transaction across its fee-bumped versions
/// (Go txmgr `SendState`).
#[derive(Debug, Default)]
struct SendState {
    published: bool,
    nonce_too_low: u64,
    mined: HashSet<B256>,
}

impl SendState {
    fn tx_mined(&mut self, hash: B256) {
        self.mined.insert(hash);
    }

    /// A reorg that leaves nothing mined restarts the nonce-too-low count, so the send waits
    /// out the full count again before aborting.
    fn tx_not_mined(&mut self, hash: B256) {
        if self.mined.remove(&hash) && self.mined.is_empty() {
            self.nonce_too_low = 0;
        }
    }

    fn is_waiting_for_confirmation(&self) -> bool {
        !self.mined.is_empty()
    }

    /// Errors when no version of the transaction can land anymore.
    fn critical_error(&self) -> Result<()> {
        if self.is_waiting_for_confirmation() {
            return Ok(());
        }
        ensure!(
            self.nonce_too_low < SAFE_ABORT_NONCE_TOO_LOW_COUNT,
            "nonce too low on {} publishes; another transaction used the nonce",
            self.nonce_too_low
        );
        ensure!(
            self.published || self.nonce_too_low == 0,
            "nonce too low before any publish; another transaction used the nonce"
        );
        Ok(())
    }
}

/// Signs and sends `request`, bumping fees until a version has [`NUM_CONFIRMATIONS`]
/// confirmations or `send_timeout` elapses. A reverted receipt is returned as `Ok`.
pub(crate) async fn send_transaction<P: Provider>(
    provider: &P,
    signer: &Signer,
    request: TransactionRequest,
    send_timeout: Duration,
    caps: FeeCaps,
) -> Result<TransactionReceipt> {
    timeout(send_timeout, send(provider, signer, request, caps))
        .await
        .map_err(|_| anyhow!("transaction not confirmed within {}s", send_timeout.as_secs()))?
}

async fn send<P: Provider>(
    provider: &P,
    signer: &Signer,
    mut request: TransactionRequest,
    caps: FeeCaps,
) -> Result<TransactionReceipt> {
    // The proposer never deploys contracts; a request without a `to` address is a bug
    // upstream of the sender.
    ensure!(request.to.is_some(), "transaction request has no `to` address");
    let from = signer.address();
    request.from = Some(from);
    request.chain_id = Some(rpc("eth_chainId", provider.get_chain_id()).await?);
    request.nonce =
        Some(rpc("eth_getTransactionCount", provider.get_transaction_count(from).latest()).await?);
    request.gas = Some(rpc("eth_estimateGas", provider.estimate_gas(request.clone())).await?);
    let fees = caps.clamp(suggest_fees(provider).await?.fees());
    let tx = sign(signer, &request, fees).await?;

    let mut pending = PendingSend {
        provider,
        signer,
        caps,
        request,
        fees,
        tx,
        hashes: Vec::new(),
        state: SendState::default(),
    };
    let start = Instant::now();
    pending.publish(false).await;

    let mut resubmit = interval_at(start + RESUBMISSION_INTERVAL, RESUBMISSION_INTERVAL);
    resubmit.set_missed_tick_behavior(MissedTickBehavior::Delay);
    let poll_interval = provider.client().poll_interval();
    let mut poll = interval_at(start + poll_interval, poll_interval);
    poll.set_missed_tick_behavior(MissedTickBehavior::Delay);
    loop {
        pending.state.critical_error()?;
        tokio::select! {
            _ = resubmit.tick() => {
                if !pending.state.is_waiting_for_confirmation() {
                    pending.publish(true).await;
                }
            }
            _ = poll.tick() => {
                if let Some(receipt) = pending.query_receipts().await {
                    return Ok(receipt);
                }
            }
        }
    }
}

/// Why a publish did not return a hash.
enum PublishError {
    /// The node answered with an error, so it did not accept this version.
    Rejected(String),
    /// No answer from the node (timeout or transport failure); it may have accepted this version.
    Unknown(String),
}

/// One logical transaction: the unsigned request, its latest signed version and every hash
/// sent so far.
struct PendingSend<'a, P> {
    provider: &'a P,
    signer: &'a Signer,
    caps: FeeCaps,
    request: TransactionRequest,
    fees: Fees,
    tx: TxEnvelope,
    hashes: Vec<B256>,
    state: SendState,
}

impl<P: Provider> PendingSend<'_, P> {
    /// Publishes the current version, bumping its fees first when `bump` is set (Go txmgr
    /// `publishTx`). An underpriced rejection of an unbumped version bumps once and retries;
    /// any other failure waits for the next resubmission tick.
    async fn publish(&mut self, mut bump: bool) {
        loop {
            if bump && let Err(error) = self.bump_fees().await {
                tracing::warn!(
                    error = format!("{error:#}"),
                    "Unable to bump fees, republishing the current transaction"
                );
            }
            match self.send_raw().await {
                Err(PublishError::Rejected(error))
                    if !bump && error.contains("transaction underpriced") =>
                {
                    tracing::warn!(error, "Transaction underpriced, bumping fees");
                    bump = true;
                }
                result => return self.record_publish(result),
            }
        }
    }

    async fn send_raw(&self) -> Result<(), PublishError> {
        let raw = self.tx.encoded_2718();
        match timeout(NETWORK_TIMEOUT, self.provider.send_raw_transaction(&raw)).await {
            Ok(Ok(_)) => Ok(()),
            Ok(Err(error)) if error.as_error_resp().is_some() => {
                Err(PublishError::Rejected(error.to_string()))
            }
            Ok(Err(error)) => Err(PublishError::Unknown(error.to_string())),
            Err(_) => Err(PublishError::Unknown(format!(
                "eth_sendRawTransaction timed out after {NETWORK_TIMEOUT:?}"
            ))),
        }
    }

    /// Records a publish attempt. Every sent version is polled, even when its publish failed.
    /// A publish with no answer from the node counts as published for the nonce-too-low abort
    /// rules: the node may have accepted it, and a replacement then sees "nonce too low" once it
    /// is mined.
    fn record_publish(&mut self, result: Result<(), PublishError>) {
        let hash = *self.tx.tx_hash();
        let nonce = self.request.nonce;
        if !self.hashes.contains(&hash) {
            self.hashes.push(hash);
        }
        match result {
            Ok(()) => tracing::info!(
                %hash,
                nonce,
                tip = self.fees.tip,
                fee_cap = self.fees.fee_cap,
                "Published transaction"
            ),
            Err(PublishError::Rejected(error)) if error.contains("already known") => {
                tracing::debug!(%hash, nonce, "Transaction already in the pool");
            }
            Err(PublishError::Unknown(error)) => {
                tracing::warn!(%hash, nonce, error, "Publish outcome unknown, polling the transaction");
            }
            Err(PublishError::Rejected(error)) => {
                if error.contains("nonce too low") {
                    self.state.nonce_too_low += 1;
                }
                tracing::warn!(%hash, nonce, error, "Unable to publish transaction");
                return;
            }
        }
        self.state.published = true;
    }

    /// Re-signs at fees that replace the current version (Go txmgr `increaseGasPrice`).
    async fn bump_fees(&mut self) -> Result<()> {
        let fees = self.caps.clamp(update_fees(self.fees, suggest_fees(self.provider).await?));
        ensure!(
            fees.tip >= bump_threshold(self.fees.tip) &&
                fees.fee_cap >= bump_threshold(self.fees.fee_cap),
            "fee caps {:?} block the replacement of tip {} and fee cap {}",
            self.caps,
            self.fees.tip,
            self.fees.fee_cap
        );
        self.tx = sign(self.signer, &self.request, fees).await?;
        self.fees = fees;
        Ok(())
    }

    /// Queries every sent version concurrently and returns as soon as one mined version has
    /// [`NUM_CONFIRMATIONS`] confirmations, without waiting for slower queries (Go txmgr
    /// polls each version in its own goroutine).
    async fn query_receipts(&mut self) -> Option<TransactionReceipt> {
        let provider = self.provider;
        let mut queries: FuturesUnordered<_> = self
            .hashes
            .iter()
            .map(|&hash| async move {
                (
                    hash,
                    rpc("eth_getTransactionReceipt", provider.get_transaction_receipt(hash)).await,
                )
            })
            .collect();
        while let Some((hash, result)) = queries.next().await {
            let receipt = match result {
                Ok(receipt) => receipt,
                Err(error) => {
                    tracing::warn!(%hash, error = format!("{error:#}"), "Receipt query failed");
                    continue;
                }
            };
            let Some(block) = receipt.as_ref().and_then(|receipt| receipt.block_number) else {
                self.state.tx_not_mined(hash);
                continue;
            };
            self.state.tx_mined(hash);
            match rpc("eth_blockNumber", provider.get_block_number()).await {
                Ok(tip) if block + NUM_CONFIRMATIONS <= tip + 1 => return receipt,
                Ok(_) => {}
                Err(error) => {
                    tracing::warn!(error = format!("{error:#}"), "L1 head query failed");
                }
            }
        }
        None
    }
}

async fn suggest_fees<P: Provider>(provider: &P) -> Result<SuggestedFees> {
    let tip = rpc("eth_maxPriorityFeePerGas", provider.get_max_priority_fee_per_gas()).await?;
    let block = rpc("eth_getBlockByNumber", provider.get_block_by_number(BlockNumberOrTag::Latest))
        .await?
        .context("latest L1 block not found")?;
    let base_fee = block.header.base_fee_per_gas.context("latest L1 block has no base fee")?;
    Ok(SuggestedFees { tip, base_fee: base_fee.into() })
}

async fn sign(signer: &Signer, request: &TransactionRequest, fees: Fees) -> Result<TxEnvelope> {
    let mut request = request.clone();
    request.max_priority_fee_per_gas = Some(fees.tip);
    request.max_fee_per_gas = Some(fees.fee_cap);
    timeout(NETWORK_TIMEOUT, signer.sign_transaction(request))
        .await
        .map_err(|_| anyhow!("signing timed out after {NETWORK_TIMEOUT:?}"))?
}

async fn rpc<F, T, E>(method: &str, call: F) -> Result<T>
where
    F: IntoFuture<Output = Result<T, E>>,
    E: std::error::Error + Send + Sync + 'static,
{
    timeout(NETWORK_TIMEOUT, call)
        .await
        .map_err(|_| anyhow!("{method} timed out after {NETWORK_TIMEOUT:?}"))?
        .with_context(|| format!("{method} failed"))
}

#[cfg(test)]
mod tests {
    use std::{
        collections::{HashMap, HashSet, VecDeque},
        sync::Arc,
        task::{Context as TaskContext, Poll},
    };

    use alloy_consensus::{Eip658Value, Receipt, ReceiptEnvelope, ReceiptWithBloom, Transaction};
    use alloy_eips::Decodable2718;
    use alloy_json_rpc::{ErrorPayload, RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_primitives::{Address, Bloom, Bytes, TxKind};
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::Block;
    use alloy_signer_local::PrivateKeySigner;
    use alloy_transport::{TransportError, TransportFut};
    use parking_lot::Mutex;
    use serde_json::{Value, json};

    use super::*;

    const TIP: u128 = 1_000_000_000;
    const BASE_FEE: u64 = 10_000_000_000;
    const LATEST_NONCE: u64 = 7;
    const HEAD: u64 = 100;
    const SEND_TIMEOUT: Duration = Duration::from_secs(180);

    /// What the scripted node does with one `eth_sendRawTransaction`.
    #[derive(Clone, Copy)]
    enum Publish {
        Accept,
        Reject(&'static str),
        /// The request is lost: the node never sees the transaction.
        Hang,
        /// The node accepts the transaction but the response is lost.
        AcceptWithoutResponse,
    }

    /// Scripted L1 node. Publishes follow `publishes` in order (then accept); receipts exist
    /// for the transactions in `mined`, keyed to their block.
    struct Chain {
        publishes: VecDeque<Publish>,
        mine_on_accept: bool,
        receipt_errors: usize,
        hung_receipts: HashSet<B256>,
        head: u64,
        sent: Vec<TxEnvelope>,
        mined: HashMap<B256, u64>,
    }

    impl Chain {
        fn mine(&mut self, index: usize, block: u64) {
            let hash = *self.sent[index].tx_hash();
            self.mined.insert(hash, block);
        }

        /// `None` leaves the request unanswered.
        fn respond(&mut self, method: &str, params: &Value) -> Option<Result<Value, String>> {
            let value = match method {
                "eth_chainId" => json!("0x1"),
                "eth_getTransactionCount" => {
                    assert_eq!(params[1], "latest");
                    json!(format!("{LATEST_NONCE:#x}"))
                }
                "eth_estimateGas" => json!("0x186a0"),
                "eth_maxPriorityFeePerGas" => json!(format!("{TIP:#x}")),
                "eth_getBlockByNumber" => {
                    let mut block = Block::<alloy_rpc_types_eth::Transaction>::default();
                    block.header.inner.base_fee_per_gas = Some(BASE_FEE);
                    serde_json::to_value(block).unwrap()
                }
                "eth_blockNumber" => json!(format!("{:#x}", self.head)),
                "eth_sendRawTransaction" => {
                    let raw: Bytes = serde_json::from_value(params[0].clone()).unwrap();
                    let tx = TxEnvelope::decode_2718(&mut raw.as_ref()).unwrap();
                    let hash = *tx.tx_hash();
                    self.sent.push(tx);
                    let publish = self.publishes.pop_front().unwrap_or(Publish::Accept);
                    match publish {
                        Publish::Reject(error) => return Some(Err(error.to_string())),
                        Publish::Hang => return None,
                        Publish::Accept | Publish::AcceptWithoutResponse => {}
                    }
                    if self.mine_on_accept {
                        self.mined.insert(hash, self.head - 2);
                    }
                    if matches!(publish, Publish::AcceptWithoutResponse) {
                        return None;
                    }
                    json!(hash)
                }
                "eth_getTransactionReceipt" => {
                    if self.receipt_errors > 0 {
                        self.receipt_errors -= 1;
                        return Some(Err("receipt backend unavailable".to_string()));
                    }
                    let hash: B256 = serde_json::from_value(params[0].clone()).unwrap();
                    if self.hung_receipts.contains(&hash) {
                        return None;
                    }
                    self.mined.get(&hash).map_or(Value::Null, |&block| {
                        serde_json::to_value(receipt(hash, block)).unwrap()
                    })
                }
                other => panic!("unexpected RPC {other}"),
            };
            Some(Ok(value))
        }
    }

    fn receipt(hash: B256, block: u64) -> TransactionReceipt {
        TransactionReceipt {
            inner: ReceiptEnvelope::Eip1559(ReceiptWithBloom::new(
                Receipt {
                    status: Eip658Value::Eip658(true),
                    cumulative_gas_used: 0,
                    logs: Vec::new(),
                },
                Bloom::ZERO,
            )),
            transaction_hash: hash,
            transaction_index: Some(0),
            block_hash: Some(B256::ZERO),
            block_number: Some(block),
            gas_used: 0,
            effective_gas_price: 0,
            blob_gas_used: None,
            blob_gas_price: None,
            from: Address::ZERO,
            to: None,
            contract_address: None,
        }
    }

    #[derive(Clone)]
    struct MockL1(Arc<Mutex<Chain>>);

    impl tower::Service<RequestPacket> for MockL1 {
        type Response = ResponsePacket;
        type Error = TransportError;
        type Future = TransportFut<'static>;

        fn poll_ready(&mut self, _: &mut TaskContext<'_>) -> Poll<Result<(), Self::Error>> {
            Poll::Ready(Ok(()))
        }

        fn call(&mut self, request: RequestPacket) -> Self::Future {
            let chain = self.0.clone();
            Box::pin(async move {
                let RequestPacket::Single(request) = request else {
                    panic!("the sender issues no batch requests")
                };
                let params = request
                    .params()
                    .map_or(Value::Null, |params| serde_json::from_str(params.get()).unwrap());
                let Some(result) = chain.lock().respond(request.method(), &params) else {
                    return std::future::pending().await;
                };
                let payload = match result {
                    Ok(value) => {
                        ResponsePayload::Success(serde_json::value::to_raw_value(&value).unwrap())
                    }
                    Err(message) => ResponsePayload::Failure(ErrorPayload {
                        code: -32000,
                        message: message.into(),
                        data: None,
                    }),
                };
                Ok(ResponsePacket::Single(Response { id: request.id().clone(), payload }))
            })
        }
    }

    fn chain(configure: impl FnOnce(&mut Chain)) -> Arc<Mutex<Chain>> {
        let mut chain = Chain {
            publishes: VecDeque::new(),
            mine_on_accept: false,
            receipt_errors: 0,
            hung_receipts: HashSet::new(),
            head: HEAD,
            sent: Vec::new(),
            mined: HashMap::new(),
        };
        configure(&mut chain);
        Arc::new(Mutex::new(chain))
    }

    async fn send_with_caps(
        chain: Arc<Mutex<Chain>>,
        send_timeout: Duration,
        caps: FeeCaps,
    ) -> Result<TransactionReceipt> {
        let provider = RootProvider::new(RpcClient::new(MockL1(chain), true));
        let signer = Signer::LocalSigner(PrivateKeySigner::random());
        let request = TransactionRequest {
            to: Some(TxKind::Call(Address::repeat_byte(0x11))),
            ..Default::default()
        };
        send_transaction(&provider, &signer, request, send_timeout, caps).await
    }

    async fn send(chain: Arc<Mutex<Chain>>) -> Result<TransactionReceipt> {
        send_with_caps(chain, SEND_TIMEOUT, FeeCaps::default()).await
    }

    fn assert_replaces(old: &TxEnvelope, new: &TxEnvelope) {
        assert_eq!(new.nonce(), old.nonce());
        assert!(
            new.max_priority_fee_per_gas().unwrap() >=
                bump_threshold(old.max_priority_fee_per_gas().unwrap())
        );
        assert!(new.max_fee_per_gas() >= bump_threshold(old.max_fee_per_gas()));
    }

    /// After a restart our earlier transaction still holds the nonce; the pool rejects the
    /// new one as an underpriced replacement until its fees are bumped.
    #[tokio::test(start_paused = true)]
    async fn replaces_a_stuck_transaction_at_the_latest_nonce() {
        let chain = chain(|chain| {
            chain.publishes = [Publish::Reject("replacement transaction underpriced")].into();
            chain.mine_on_accept = true;
        });
        let start = Instant::now();
        let receipt = send(chain.clone()).await.unwrap();
        assert!(start.elapsed() < RESUBMISSION_INTERVAL, "replaced only at the resubmission tick");
        let chain = chain.lock();
        assert_eq!(chain.sent.len(), 2);
        assert_eq!(chain.sent[0].nonce(), LATEST_NONCE);
        assert_replaces(&chain.sent[0], &chain.sent[1]);
        assert_eq!(receipt.transaction_hash, *chain.sent[1].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn bumps_fees_each_resubmission_until_mined() {
        let chain = chain(|_| {});
        let handle = tokio::spawn(send(chain.clone()));
        tokio::time::sleep(RESUBMISSION_INTERVAL * 2 + Duration::from_secs(1)).await;
        {
            let mut chain = chain.lock();
            assert_eq!(chain.sent.len(), 3);
            chain.mine(2, HEAD - 2);
        }
        let receipt = handle.await.unwrap().unwrap();
        let chain = chain.lock();
        assert_replaces(&chain.sent[0], &chain.sent[1]);
        assert_replaces(&chain.sent[1], &chain.sent[2]);
        assert_eq!(receipt.transaction_hash, *chain.sent[2].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn hung_publish_times_out_and_is_retried() {
        let chain = chain(|chain| {
            chain.publishes = [Publish::Hang].into();
            chain.mine_on_accept = true;
        });
        let receipt = send(chain.clone()).await.unwrap();
        let chain = chain.lock();
        assert_eq!(chain.sent.len(), 2);
        assert_eq!(receipt.transaction_hash, *chain.sent[1].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn publish_with_a_lost_response_is_still_confirmed() {
        let chain = chain(|chain| {
            chain.publishes = [Publish::AcceptWithoutResponse].into();
            chain.mine_on_accept = true;
        });
        let receipt = send(chain.clone()).await.unwrap();
        let chain = chain.lock();
        assert_eq!(chain.sent.len(), 1);
        assert_eq!(receipt.transaction_hash, *chain.sent[0].tx_hash());
    }

    /// The original was accepted without a response and mines while its replacement is
    /// rejected as "nonce too low"; the send must not abort as if nothing was published.
    #[tokio::test(start_paused = true)]
    async fn lost_response_original_mined_during_replacement_is_confirmed() {
        let chain = chain(|chain| {
            chain.publishes =
                [Publish::AcceptWithoutResponse, Publish::Reject("nonce too low")].into();
        });
        let handle = tokio::spawn(send(chain.clone()));
        tokio::time::sleep(RESUBMISSION_INTERVAL + Duration::from_secs(1)).await;
        {
            let mut chain = chain.lock();
            assert_eq!(chain.sent.len(), 2);
            chain.mine(0, HEAD - 2);
        }
        let receipt = handle.await.unwrap().unwrap();
        assert_eq!(receipt.transaction_hash, *chain.lock().sent[0].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn hung_receipt_queries_do_not_delay_a_confirmed_replacement() {
        let chain = chain(|_| {});
        let handle = tokio::spawn(send(chain.clone()));
        tokio::time::sleep(RESUBMISSION_INTERVAL + Duration::from_secs(1)).await;
        {
            let mut chain = chain.lock();
            assert_eq!(chain.sent.len(), 2);
            let stale = *chain.sent[0].tx_hash();
            chain.hung_receipts.insert(stale);
            chain.mine(1, HEAD - 2);
        }
        let mined_at = Instant::now();
        let receipt = handle.await.unwrap().unwrap();
        assert!(mined_at.elapsed() < NETWORK_TIMEOUT, "waited on the stale receipt query");
        assert_eq!(receipt.transaction_hash, *chain.lock().sent[1].tx_hash());
    }

    /// "already known" counts as a publish, so a later nonce-too-low on the replacement does
    /// not abort while the original may still be mined.
    #[tokio::test(start_paused = true)]
    async fn already_known_counts_as_published() {
        let chain = chain(|chain| {
            chain.publishes =
                [Publish::Reject("already known"), Publish::Reject("nonce too low")].into();
        });
        let handle = tokio::spawn(send(chain.clone()));
        tokio::time::sleep(RESUBMISSION_INTERVAL + Duration::from_secs(1)).await;
        {
            let mut chain = chain.lock();
            assert_eq!(chain.sent.len(), 2);
            chain.mine(0, HEAD - 2);
        }
        let receipt = handle.await.unwrap().unwrap();
        assert_eq!(receipt.transaction_hash, *chain.lock().sent[0].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn receipt_errors_do_not_abort_the_send() {
        let chain = chain(|chain| {
            chain.mine_on_accept = true;
            chain.receipt_errors = 3;
        });
        let receipt = send(chain.clone()).await.unwrap();
        let chain = chain.lock();
        assert_eq!(chain.sent.len(), 1);
        assert_eq!(receipt.transaction_hash, *chain.sent[0].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn waits_for_required_confirmations() {
        let chain = chain(|_| {});
        let handle = tokio::spawn(send(chain.clone()));
        tokio::time::sleep(Duration::from_secs(1)).await;
        chain.lock().mine(0, HEAD);
        for head in HEAD..HEAD + NUM_CONFIRMATIONS - 1 {
            chain.lock().head = head;
            tokio::time::sleep(Duration::from_secs(5)).await;
            assert!(!handle.is_finished(), "confirmed at head {head}");
        }
        chain.lock().head = HEAD + NUM_CONFIRMATIONS - 1;
        let receipt = handle.await.unwrap().unwrap();
        assert_eq!(receipt.block_number, Some(HEAD));
    }

    #[tokio::test(start_paused = true)]
    async fn aborts_on_nonce_too_low_before_any_publish() {
        let chain = chain(|chain| chain.publishes = [Publish::Reject("nonce too low")].into());
        let error = send(chain.clone()).await.unwrap_err().to_string();
        assert!(error.contains("nonce too low before any publish"), "{error}");
        assert_eq!(chain.lock().sent.len(), 1);
    }

    /// The original version was mined while the replacement was in flight.
    #[tokio::test(start_paused = true)]
    async fn keeps_waiting_when_a_replacement_hits_nonce_too_low() {
        let chain = chain(|chain| {
            chain.publishes = [Publish::Accept, Publish::Reject("nonce too low")].into();
        });
        let handle = tokio::spawn(send(chain.clone()));
        tokio::time::sleep(RESUBMISSION_INTERVAL + Duration::from_secs(1)).await;
        {
            let mut chain = chain.lock();
            assert_eq!(chain.sent.len(), 2);
            chain.mine(0, HEAD - 2);
        }
        let receipt = handle.await.unwrap().unwrap();
        assert_eq!(receipt.transaction_hash, *chain.lock().sent[0].tx_hash());
    }

    #[tokio::test(start_paused = true)]
    async fn aborts_after_repeated_nonce_too_low_once_published() {
        let nonce_too_low = Publish::Reject("nonce too low");
        let chain = chain(|chain| {
            chain.publishes = [Publish::Accept, nonce_too_low, nonce_too_low, nonce_too_low].into();
        });
        let error = send(chain).await.unwrap_err().to_string();
        assert!(error.contains("nonce too low on 3 publishes"), "{error}");
    }

    #[tokio::test(start_paused = true)]
    async fn gives_up_at_the_send_timeout() {
        let chain = chain(|_| {});
        let error = send_with_caps(chain.clone(), Duration::from_secs(60), FeeCaps::default())
            .await
            .unwrap_err()
            .to_string();
        assert!(error.contains("not confirmed within 60s"), "{error}");
        assert_eq!(chain.lock().sent.len(), 3);
    }

    #[tokio::test(start_paused = true)]
    async fn fee_caps_bound_bumps() {
        let chain = chain(|_| {});
        let suggested = SuggestedFees { tip: TIP, base_fee: BASE_FEE.into() }.fees();
        let caps = FeeCaps {
            max_fee_per_gas: Some(suggested.fee_cap),
            max_priority_fee_per_gas: Some(suggested.tip),
        };
        send_with_caps(chain.clone(), Duration::from_secs(60), caps).await.unwrap_err();
        let chain = chain.lock();
        assert_eq!(chain.sent.len(), 3);
        assert!(chain.sent.iter().all(|tx| tx.tx_hash() == chain.sent[0].tx_hash()));
    }

    #[test]
    fn bump_threshold_rounds_up() {
        assert_eq!(bump_threshold(0), 0);
        assert_eq!(bump_threshold(1), 2);
        assert_eq!(bump_threshold(100), 110);
        assert_eq!(bump_threshold(101), 112);
    }

    #[test]
    fn update_fees_follows_go_branches() {
        let old = Fees { tip: 100, fee_cap: 1_000 };
        let cases = [
            // Suggestion clears both thresholds: use it.
            ((200, 1_000), Fees { tip: 200, fee_cap: 2_200 }),
            // Tip clears, fee cap does not: threshold fee cap.
            ((200, 100), Fees { tip: 200, fee_cap: 1_100 }),
            // Fee cap clears, tip does not: threshold tip, fee cap recomputed from it.
            ((50, 1_000), Fees { tip: 110, fee_cap: 2_110 }),
            // Neither clears: both thresholds.
            ((50, 100), Fees { tip: 110, fee_cap: 1_100 }),
        ];
        for ((tip, base_fee), expected) in cases {
            assert_eq!(update_fees(old, SuggestedFees { tip, base_fee }), expected);
        }
    }

    #[test]
    fn fee_caps_clamp() {
        let fees = Fees { tip: 50, fee_cap: 100 };
        assert_eq!(FeeCaps::default().clamp(fees), fees);
        let both = FeeCaps { max_fee_per_gas: Some(60), max_priority_fee_per_gas: Some(10) };
        assert_eq!(both.clamp(fees), Fees { tip: 10, fee_cap: 60 });
        let fee_cap_only = FeeCaps { max_fee_per_gas: Some(30), max_priority_fee_per_gas: None };
        assert_eq!(fee_cap_only.clamp(fees), Fees { tip: 30, fee_cap: 30 });
    }

    #[test]
    fn reorg_restarts_the_nonce_too_low_count() {
        let hash = B256::repeat_byte(1);
        let mut state = SendState { published: true, ..Default::default() };
        state.nonce_too_low = SAFE_ABORT_NONCE_TOO_LOW_COUNT;
        state.tx_mined(hash);
        assert!(state.critical_error().is_ok());
        state.tx_not_mined(hash);
        assert_eq!(state.nonce_too_low, 0);
        assert!(state.critical_error().is_ok());
    }
}
