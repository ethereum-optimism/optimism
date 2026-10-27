//! Client for the Succinct Prover Network RPC.
//!
//! Reuses the proto types and generated gRPC client from `sp1-sdk`. The SDK exposes balance and
//! nonce only for its own signer and has no transfer or withdraw helpers, so transfer and withdraw
//! bodies are built here and signed the way the SDK signs proof requests: EIP-191 over the
//! protobuf-encoded body, `MessageFormat::Binary`.

use std::{
    str::FromStr,
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use alloy_primitives::{Address, B256, U256};
use anyhow::{Context, Result, anyhow, ensure};
use sp1_sdk::network::proto::{
    network::prover_network_client::ProverNetworkClient,
    types::{
        BalanceLog, BalanceOperation, BidHistory, FulfillmentStatus, GetBalanceRequest,
        GetFilteredBalanceLogsRequest, GetFilteredBidHistoryRequest,
        GetFilteredProofRequestsRequest, GetNonceRequest, GetProofRequestDetailsRequest,
        GetProofRequestParamsRequest, GetTransactionDetailsRequest, GetTransferParamsRequest,
        GetWithdrawParamsRequest, MessageFormat, ProofMode, ProofRequest, TransactionDetails,
        TransactionVariant, TransferRequest, TransferRequestBody, WithdrawRequest,
        WithdrawRequestBody,
    },
};
use tonic::{
    Code, Status,
    transport::{Channel, ClientTlsConfig, Endpoint},
};

/// Default SPN mainnet RPC (`MAINNET_RPC_URL` in sp1-sdk).
pub(crate) const MAINNET_SPN_RPC: &str = "https://rpc.mainnet.succinct.xyz";

/// Most rows SPN returns per page.
pub(crate) const MAX_PAGE: u32 = 100;
const POLL_INTERVAL: Duration = Duration::from_secs(10);
/// How far before submission the `TRANSFER_IN` scan starts, to absorb clock skew with SPN.
const TRANSFER_SCAN_SLACK_SECS: u64 = 300;

/// Auctioneer and fee the network charges for a transfer or withdraw.
#[derive(Debug)]
pub(crate) struct TxParams {
    pub(crate) auctioneer: Address,
    pub(crate) fee: U256,
}

#[derive(Debug)]
pub(crate) struct Spn {
    client: ProverNetworkClient<Channel>,
}

impl Spn {
    pub(crate) async fn connect(url: &str) -> Result<Self> {
        // tonic's TLS needs a process-wide rustls provider; sp1-sdk installs ring the same way.
        let _ = rustls::crypto::ring::default_provider().install_default();
        let mut endpoint = Endpoint::new(url.to_string())
            .with_context(|| format!("invalid SPN RPC URL {url}"))?
            .timeout(Duration::from_secs(60))
            .connect_timeout(Duration::from_secs(15));
        if url.starts_with("https://") {
            endpoint = endpoint.tls_config(ClientTlsConfig::new().with_enabled_roots())?;
        }
        let channel =
            endpoint.connect().await.with_context(|| format!("connecting to SPN RPC {url}"))?;
        Ok(Self { client: ProverNetworkClient::new(channel) })
    }

    /// Spendable SPN balance of any address, in wei of PROVE.
    pub(crate) async fn balance(&self, address: Address) -> Result<U256> {
        let res = self
            .client
            .clone()
            .get_balance(GetBalanceRequest { address: address.to_vec() })
            .await
            .context("SPN GetBalance")?
            .into_inner();
        parse_amount(&res.amount).context("SPN GetBalance")
    }

    pub(crate) async fn transfer_params(&self) -> Result<TxParams> {
        let res = self
            .client
            .clone()
            .get_transfer_params(GetTransferParamsRequest {})
            .await
            .context("SPN GetTransferParams")?
            .into_inner();
        tx_params(&res.auctioneer, &res.fee)
    }

    pub(crate) async fn withdraw_params(&self) -> Result<TxParams> {
        let res = self
            .client
            .clone()
            .get_withdraw_params(GetWithdrawParamsRequest {})
            .await
            .context("SPN GetWithdrawParams")?
            .into_inner();
        tx_params(&res.auctioneer, &res.fee)
    }

    /// The network-global EIP-712 domain separator, as sp1-sdk obtains it for proof requests.
    pub(crate) async fn domain(&self) -> Result<Vec<u8>> {
        let domain = self
            .client
            .clone()
            .get_proof_request_params(GetProofRequestParamsRequest {
                mode: ProofMode::Compressed.into(),
            })
            .await
            .context("SPN GetProofRequestParams")?
            .into_inner()
            .domain;
        ensure!(domain.len() == 32, "SPN returned a {}-byte domain, expected 32", domain.len());
        Ok(domain)
    }

    /// Builds a transfer from `from` to `to` at `from`'s current nonce. The recipient does not
    /// sign.
    pub(crate) async fn transfer_body(
        &self,
        from: Address,
        to: Address,
        amount: U256,
        params: &TxParams,
        domain: &[u8],
    ) -> Result<TransferRequestBody> {
        Ok(TransferRequestBody {
            nonce: self.nonce(from).await?,
            to: to.to_vec(),
            amount: amount.to_string(),
            domain: domain.to_vec(),
            variant: TransactionVariant::TransferVariant.into(),
            auctioneer: params.auctioneer.to_vec(),
            fee: params.fee.to_string(),
        })
    }

    /// Builds a withdraw of `amount` from `account` to the same address on L1.
    pub(crate) async fn withdraw_body(
        &self,
        account: Address,
        amount: U256,
        params: &TxParams,
        domain: &[u8],
    ) -> Result<WithdrawRequestBody> {
        Ok(WithdrawRequestBody {
            nonce: self.nonce(account).await?,
            account: account.to_vec(),
            amount: amount.to_string(),
            domain: domain.to_vec(),
            variant: TransactionVariant::WithdrawVariant.into(),
            auctioneer: params.auctioneer.to_vec(),
            fee: params.fee.to_string(),
        })
    }

    /// Submits a signed transfer and returns the tx hash SPN assigns to it. That hash is not
    /// computable from the body: it differs from `keccak256(sender || body)`.
    pub(crate) async fn submit_transfer(
        &self,
        sender: Address,
        body: TransferRequestBody,
        signature: Vec<u8>,
    ) -> Result<B256> {
        let nonce = body.nonce;
        let res = self
            .client
            .clone()
            .transfer(TransferRequest {
                format: MessageFormat::Binary.into(),
                signature,
                body: Some(body),
            })
            .await
            .map_err(|status| submit_error("Transfer", &status, sender, nonce))?
            .into_inner();
        accepted_hash(&res.tx_hash, "Transfer", sender, nonce)
    }

    /// Submits a signed withdraw and returns the tx hash SPN assigns to it.
    pub(crate) async fn submit_withdraw(
        &self,
        sender: Address,
        body: WithdrawRequestBody,
        signature: Vec<u8>,
    ) -> Result<B256> {
        let nonce = body.nonce;
        let res = self
            .client
            .clone()
            .withdraw(WithdrawRequest {
                format: MessageFormat::Binary.into(),
                signature,
                body: Some(body),
            })
            .await
            .map_err(|status| submit_error("Withdraw", &status, sender, nonce))?
            .into_inner();
        accepted_hash(&res.tx_hash, "Withdraw", sender, nonce)
    }

    /// One page of `address`'s balance changes, newest first.
    pub(crate) async fn balance_logs(
        &self,
        address: Address,
        operation: Option<BalanceOperation>,
        minimum_timestamp: Option<u64>,
        limit: u32,
        page: u32,
    ) -> Result<Vec<BalanceLog>> {
        Ok(self
            .client
            .clone()
            .get_filtered_balance_logs(GetFilteredBalanceLogsRequest {
                address: Some(address.to_vec()),
                operation: operation.map(Into::into),
                minimum_timestamp,
                maximum_timestamp: None,
                limit: Some(limit),
                page: Some(page),
            })
            .await
            .context("SPN GetFilteredBalanceLogs")?
            .into_inner()
            .logs)
    }

    /// One page of proof requests made by `requester`, newest first.
    pub(crate) async fn proof_requests(
        &self,
        requester: Address,
        status: Option<FulfillmentStatus>,
        limit: u32,
        page: u32,
    ) -> Result<Vec<ProofRequest>> {
        Ok(self
            .client
            .clone()
            .get_filtered_proof_requests(GetFilteredProofRequestsRequest {
                requester: Some(requester.to_vec()),
                fulfillment_status: status.map(Into::into),
                limit: Some(limit),
                page: Some(page),
                ..Default::default()
            })
            .await
            .context("SPN GetFilteredProofRequests")?
            .into_inner()
            .requests)
    }

    pub(crate) async fn proof_request(&self, id: B256) -> Result<Option<ProofRequest>> {
        Ok(self
            .client
            .clone()
            .get_proof_request_details(GetProofRequestDetailsRequest { request_id: id.to_vec() })
            .await
            .context("SPN GetProofRequestDetails")?
            .into_inner()
            .request)
    }

    /// The latest `limit` bids on request `id`.
    pub(crate) async fn bids(&self, id: B256, limit: u32) -> Result<Vec<BidHistory>> {
        Ok(self
            .client
            .clone()
            .get_filtered_bid_history(GetFilteredBidHistoryRequest {
                request_id: id.to_vec(),
                limit: Some(limit),
                page: None,
                winner_first: None,
            })
            .await
            .context("SPN GetFilteredBidHistory")?
            .into_inner()
            .bids)
    }

    pub(crate) async fn transaction(&self, hash: B256) -> Result<Option<TransactionDetails>> {
        Ok(self
            .client
            .clone()
            .get_transaction_details(GetTransactionDetailsRequest { tx_hash: hash.to_vec() })
            .await
            .context("SPN GetTransactionDetails")?
            .into_inner()
            .transaction)
    }

    /// Polls until `to` has a `TRANSFER_IN` log for `hash`; `false` on timeout.
    ///
    /// The recipient's balance cannot confirm a transfer: a requester pays for proofs while it
    /// waits, so its balance can stay below `before + amount` after the credit lands.
    pub(crate) async fn wait_for_transfer_in(
        &self,
        to: Address,
        hash: B256,
        submitted_at: u64,
        timeout: Duration,
    ) -> Result<bool> {
        let since = submitted_at.saturating_sub(TRANSFER_SCAN_SLACK_SECS);
        let deadline = tokio::time::Instant::now() + timeout;
        loop {
            for page in 1.. {
                let logs = self
                    .balance_logs(
                        to,
                        Some(BalanceOperation::TransferIn),
                        Some(since),
                        MAX_PAGE,
                        page,
                    )
                    .await?;
                if logs.iter().any(|log| is_transfer_in(log, hash)) {
                    return Ok(true);
                }
                if logs.len() < MAX_PAGE as usize {
                    break;
                }
            }
            if tokio::time::Instant::now() >= deadline {
                return Ok(false);
            }
            tokio::time::sleep(POLL_INTERVAL).await;
        }
    }

    /// Polls until `address`'s SPN balance reaches `target`; `None` on timeout.
    pub(crate) async fn wait_for_balance(
        &self,
        address: Address,
        target: U256,
        timeout: Duration,
    ) -> Result<Option<U256>> {
        let deadline = tokio::time::Instant::now() + timeout;
        loop {
            let balance = self.balance(address).await?;
            if balance >= target {
                return Ok(Some(balance));
            }
            if tokio::time::Instant::now() >= deadline {
                return Ok(None);
            }
            tokio::time::sleep(POLL_INTERVAL).await;
        }
    }

    async fn nonce(&self, address: Address) -> Result<u64> {
        Ok(self
            .client
            .clone()
            .get_nonce(GetNonceRequest { address: address.to_vec() })
            .await
            .context("SPN GetNonce")?
            .into_inner()
            .nonce)
    }
}

pub(crate) fn unix_now() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).map(|d| d.as_secs()).unwrap_or_default()
}

pub(crate) fn parse_amount(amount: &str) -> Result<U256> {
    U256::from_str(amount).with_context(|| format!("invalid PROVE amount {amount:?}"))
}

fn is_transfer_in(log: &BalanceLog, hash: B256) -> bool {
    BalanceOperation::try_from(log.operation) == Ok(BalanceOperation::TransferIn) &&
        log.tx_hash == hash.as_slice()
}

/// SPN refused the request outright for these codes, so nothing was accepted. Any other failure
/// (timeout, unavailable, internal) may have happened after SPN accepted it.
const fn is_refusal(code: Code) -> bool {
    matches!(
        code,
        Code::InvalidArgument |
            Code::PermissionDenied |
            Code::FailedPrecondition |
            Code::Unauthenticated |
            Code::NotFound |
            Code::AlreadyExists |
            Code::OutOfRange
    )
}

fn submit_error(method: &str, status: &Status, sender: Address, nonce: u64) -> anyhow::Error {
    let cause = anyhow!("{:?}: {}", status.code(), status.message());
    if is_refusal(status.code()) {
        return cause.context(format!("SPN refused the {method} (sender {sender}, nonce {nonce})"));
    }
    cause.context(format!(
        "SPN {method} failed and SPN may still have accepted it (sender {sender}, nonce {nonce}); \
         check `spn-wallet history {sender}` before sending again"
    ))
}

fn accepted_hash(returned: &[u8], method: &str, sender: Address, nonce: u64) -> Result<B256> {
    B256::try_from(returned).map_err(|_| {
        anyhow!(
            "SPN accepted the {method} (sender {sender}, nonce {nonce}) but returned a {}-byte tx \
             hash; check `spn-wallet history {sender}` before sending again",
            returned.len()
        )
    })
}

fn tx_params(auctioneer: &[u8], fee: &str) -> Result<TxParams> {
    Ok(TxParams {
        auctioneer: Address::try_from(auctioneer).context("SPN returned an invalid auctioneer")?,
        fee: parse_amount(fee).context("SPN returned an invalid fee")?,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    const HASH: B256 = B256::repeat_byte(0xab);

    fn log(operation: i32, hash: B256) -> BalanceLog {
        BalanceLog {
            address: vec![0; 20],
            operation,
            amount: "1".into(),
            tx_hash: hash.to_vec(),
            created_at: 0,
        }
    }

    #[test]
    fn matches_only_a_transfer_in_with_the_submitted_hash() {
        assert!(is_transfer_in(&log(BalanceOperation::TransferIn.into(), HASH), HASH));
        // The sender's TRANSFER_OUT and TRANSFER_FEE_OUT logs carry the same hash.
        assert!(!is_transfer_in(&log(BalanceOperation::TransferOut.into(), HASH), HASH));
        assert!(!is_transfer_in(&log(BalanceOperation::TransferFeeOut.into(), HASH), HASH));
        assert!(!is_transfer_in(&log(BalanceOperation::TransferIn.into(), B256::ZERO), HASH));
        assert!(!is_transfer_in(&log(i32::MAX, HASH), HASH));
    }

    #[test]
    fn treats_only_definite_refusals_as_not_accepted() {
        assert!(is_refusal(Code::PermissionDenied));
        assert!(is_refusal(Code::InvalidArgument));
        assert!(!is_refusal(Code::Unavailable));
        assert!(!is_refusal(Code::DeadlineExceeded));
        assert!(!is_refusal(Code::Internal));
        assert!(!is_refusal(Code::Unknown));
    }
}
