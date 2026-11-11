//! Scripted RPC expectations over Alloy's mock transport.

use crate::{EngineClient, test_utils::TestRpcBlock};
use alloy_json_rpc::{ErrorPayload, RequestPacket, ResponsePacket};
use alloy_network::Network;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use alloy_rpc_types_eth::{Block as EthBlock, Transaction as EthTransaction};
use alloy_transport::{
    TransportError, TransportErrorKind, TransportFut,
    mock::{Asserter, MockResponse, MockTransport},
};
use kona_genesis::RollupConfig;
use serde::Serialize;
use serde_json::Value;
use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
    task::{Context, Poll},
    time::Duration,
};
use tower::Service;

/// Queues replies and checks the method and optional parameters of every outgoing RPC request.
/// Unexpected requests panic rather than silently satisfying a different operation's reply.
#[derive(Clone, Debug, Default)]
pub struct RpcMock {
    requests: Arc<Mutex<VecDeque<ExpectedRequest>>>,
}

#[derive(Debug)]
struct ExpectedRequest {
    method: &'static str,
    response: Option<MockResponse>,
    params: Option<Value>,
    delay: Duration,
}

impl RpcMock {
    /// Builds a real Alloy provider backed by this mock transport.
    pub fn provider<N: Network>(&self) -> RootProvider<N> {
        RootProvider::new(RpcClient::new(self.clone(), true))
    }

    /// Expects one call to `method` and returns the serialized response.
    pub fn expect(&self, method: &'static str, response: impl Serialize) {
        self.expect_with_delay(method, response, Duration::ZERO);
    }

    /// Expects a call with exact JSON parameters.
    pub fn expect_params(&self, method: &'static str, params: Value, response: impl Serialize) {
        self.push(
            method,
            Some(params),
            MockResponse::Success(serde_json::value::to_raw_value(&response).unwrap()),
            Duration::ZERO,
        );
    }

    /// Expects a full-block `eth_getBlockByNumber` call for `number` and answers it with `block`,
    /// or with null for `None`.
    pub fn expect_block_by_number(&self, number: u64, block: Option<TestRpcBlock>) {
        self.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!([format!("{number:#x}"), true]),
            block,
        );
    }

    /// Expects a header-only `eth_getBlockByNumber` call for `number` and answers it with `block`,
    /// or with null for `None`.
    pub fn expect_block_header_by_number(&self, number: u64, block: Option<TestRpcBlock>) {
        self.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!([format!("{number:#x}"), false]),
            block,
        );
    }

    /// Expects an `eth_syncing` call answered with no sync in progress, then an `eth_blockNumber`
    /// call answered with `latest_number`.
    pub fn expect_el_not_syncing_at(&self, latest_number: u64) {
        self.expect("eth_syncing", false);
        self.expect("eth_blockNumber", alloy_primitives::U64::from(latest_number));
    }

    /// Expects an `eth_syncing` call answered with a sync in progress whose current block is
    /// `current_number`.
    pub fn expect_el_syncing_at(&self, current_number: u64) {
        self.expect(
            "eth_syncing",
            serde_json::json!({
                "currentBlock": format!("{current_number:#x}"),
                "highestBlock": "0xffff",
                "startingBlock": "0x0",
            }),
        );
    }

    /// Expects an `engine_forkchoiceUpdatedV3` call for `forkchoice` without payload attributes and
    /// answers it with `status`.
    pub fn expect_forkchoice_updated(
        &self,
        forkchoice: ForkchoiceState,
        status: PayloadStatusEnum,
    ) {
        self.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, null]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(status)),
        );
    }

    /// Expects a call and delays its response, for recovery tests using Tokio's paused clock.
    pub fn expect_with_delay(
        &self,
        method: &'static str,
        response: impl Serialize,
        delay: Duration,
    ) {
        self.push(
            method,
            None,
            MockResponse::Success(serde_json::value::to_raw_value(&response).unwrap()),
            delay,
        );
    }

    /// Expects a call that fails with a JSON-RPC error.
    pub fn expect_error(&self, method: &'static str) {
        self.push(
            method,
            None,
            MockResponse::Failure(ErrorPayload::internal_error_message(
                "injected RPC failure".into(),
            )),
            Duration::ZERO,
        );
    }

    /// Expects a call that fails with a specific JSON-RPC error code.
    pub fn expect_error_code(&self, method: &'static str, code: i64) {
        self.push(
            method,
            None,
            MockResponse::Failure(ErrorPayload {
                code,
                message: "injected RPC failure".into(),
                data: None,
            }),
            Duration::ZERO,
        );
    }

    /// Expects a call that fails before receiving a JSON-RPC response.
    pub fn expect_transport_error(&self, method: &'static str) {
        self.requests.lock().unwrap().push_back(ExpectedRequest {
            method,
            params: None,
            response: None,
            delay: Duration::ZERO,
        });
    }

    fn push(
        &self,
        method: &'static str,
        params: Option<Value>,
        response: MockResponse,
        delay: Duration,
    ) {
        self.requests.lock().unwrap().push_back(ExpectedRequest {
            method,
            params,
            response: Some(response),
            delay,
        });
    }

    /// Checks that the operation consumed every expected request.
    pub fn assert_finished(&self) {
        assert!(self.requests.lock().unwrap().is_empty(), "unconsumed RPC expectations");
    }
}

impl Service<RequestPacket> for RpcMock {
    type Response = ResponsePacket;
    type Error = TransportError;
    type Future = TransportFut<'static>;

    fn poll_ready(&mut self, _cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        Poll::Ready(Ok(()))
    }

    fn call(&mut self, request: RequestPacket) -> Self::Future {
        let RequestPacket::Single(ref rpc) = request else {
            panic!("engine tests expect individual RPC requests");
        };
        let expected = self
            .requests
            .lock()
            .unwrap()
            .pop_front()
            .unwrap_or_else(|| panic!("unexpected RPC request: {}", rpc.method()));
        assert_eq!(rpc.method(), expected.method, "wrong RPC method");
        if let Some(params) = expected.params {
            let actual: Value =
                serde_json::from_str(rpc.params().expect("request has parameters").get()).unwrap();
            assert_eq!(actual, params, "wrong RPC parameters for {}", expected.method);
        }
        let replies = Asserter::new();
        let Some(reply) = expected.response else {
            return Box::pin(async {
                Err(TransportErrorKind::custom_str("injected transport failure"))
            });
        };
        replies.push(reply);
        let response = MockTransport::new(replies).call(request);
        Box::pin(async move {
            if !expected.delay.is_zero() {
                tokio::time::sleep(expected.delay).await;
            }
            response.await
        })
    }
}

/// Creates the production client with separate scripted L1 and L2 transports.
pub fn test_engine_client(cfg: Arc<RollupConfig>) -> (EngineClient, RpcMock, RpcMock) {
    let l1 = RpcMock::default();
    let l2 = RpcMock::default();
    let client = EngineClient::new(l1.provider(), l2.provider(), cfg);
    (client, l1, l2)
}

/// Expects the sync start a reset performs: the execution layer answering `block` for its
/// finalized, safe and latest blocks, the L1 lookup of its origin, and `forkchoice` being sent.
pub fn expect_sync_start(
    l1: &RpcMock,
    l2: &RpcMock,
    block: &TestRpcBlock,
    forkchoice: ForkchoiceState,
) {
    for label in ["finalized", "safe", "latest"] {
        l2.expect_params("eth_getBlockByNumber", serde_json::json!([label, true]), block);
    }
    l1.expect("eth_getBlockByHash", EthBlock::<EthTransaction>::default());
    l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Valid);
}
