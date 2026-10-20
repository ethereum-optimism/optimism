//! A strict RPC script for shared engine-operation tests.

use crate::OpEngineClient;
use alloy_json_rpc::{RequestPacket, ResponsePacket};
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport::{TransportErrorKind, TransportFut};
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
use serde::Serialize;
use serde_json::{Value, json};
use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};

/// One expected RPC request and response.
#[derive(Debug)]
pub struct ExpectedCall {
    method: &'static str,
    response: Value,
}

impl ExpectedCall {
    /// Expects a successful Engine API result.
    pub fn success(method: &'static str, result: impl Serialize) -> Self {
        Self { method, response: json!({"jsonrpc": "2.0", "result": result}) }
    }

    /// Expects an Engine API error with the supplied code.
    pub fn error(method: &'static str, code: i64) -> Self {
        Self {
            method,
            response: json!({"jsonrpc": "2.0", "error": {"code": code, "message": "injected engine error"}}),
        }
    }
}

/// An RPC request observed by the script.
#[derive(Debug)]
pub struct ObservedCall {
    /// JSON-RPC method.
    pub method: String,
    /// Serialized request parameters.
    pub params: Value,
}

/// Strict ordered RPC responses for operation and owner tests.
#[derive(Debug, Default)]
pub struct Script {
    expected: Mutex<VecDeque<ExpectedCall>>,
    observed: Mutex<Vec<ObservedCall>>,
}

impl Script {
    /// Creates an ordered script.
    pub fn new(expected: impl IntoIterator<Item = ExpectedCall>) -> Arc<Self> {
        Arc::new(Self {
            expected: Mutex::new(expected.into_iter().collect()),
            ..Default::default()
        })
    }

    /// Makes a provider for the selected network using this script.
    pub fn provider<N: alloy_network::Network>(self: &Arc<Self>) -> RootProvider<N> {
        let script = self.clone();
        let transport = tower::service_fn(move |request: RequestPacket| -> TransportFut<'static> {
            let script = script.clone();
            Box::pin(async move {
                let RequestPacket::Single(request) = request else { panic!("unexpected batch") };
                let expected =
                    script.expected.lock().unwrap().pop_front().expect("unexpected engine call");
                assert_eq!(request.method(), expected.method);
                script.observed.lock().unwrap().push(ObservedCall {
                    method: request.method().to_owned(),
                    params: serde_json::from_str(request.params().unwrap().get()).unwrap(),
                });
                let mut response = expected.response;
                response["id"] = serde_json::to_value(request.id()).unwrap();
                Ok(serde_json::from_value::<ResponsePacket>(response).unwrap())
            })
        });
        RootProvider::new(RpcClient::new(transport, false))
    }

    /// Makes an engine client whose L2 calls use this script; unexpected L1 calls fail.
    pub fn client(
        self: &Arc<Self>,
        cfg: Arc<RollupConfig>,
    ) -> OpEngineClient<RootProvider, RootProvider<Optimism>> {
        let l1 = tower::service_fn(|_: RequestPacket| -> TransportFut<'static> {
            Box::pin(async { Err(TransportErrorKind::custom_str("unexpected L1 call")) })
        });
        OpEngineClient::new(RootProvider::new(RpcClient::new(l1, false)), self.provider(), cfg)
    }

    /// Takes the observed requests.
    pub fn calls(&self) -> Vec<ObservedCall> {
        std::mem::take(&mut *self.observed.lock().unwrap())
    }

    /// Asserts that every scripted request occurred.
    pub fn assert_finished(&self) {
        assert!(self.expected.lock().unwrap().is_empty(), "expected engine calls were not made");
    }
}
