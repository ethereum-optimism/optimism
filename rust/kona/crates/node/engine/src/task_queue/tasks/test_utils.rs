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

pub(super) struct ExpectedCall {
    method: &'static str,
    response: Value,
}

impl ExpectedCall {
    pub(super) fn success(method: &'static str, result: impl Serialize) -> Self {
        Self { method, response: json!({"jsonrpc": "2.0", "result": result}) }
    }

    pub(super) fn error(method: &'static str, code: i64) -> Self {
        Self {
            method,
            response: json!({"jsonrpc": "2.0", "error": {"code": code, "message": "injected engine error"}}),
        }
    }
}

pub(super) struct ObservedCall {
    pub(super) method: String,
    pub(super) params: Value,
}

#[derive(Default)]
pub(super) struct Script {
    expected: Mutex<VecDeque<ExpectedCall>>,
    observed: Mutex<Vec<ObservedCall>>,
}

impl Script {
    pub(super) fn new(expected: impl IntoIterator<Item = ExpectedCall>) -> Arc<Self> {
        Arc::new(Self {
            expected: Mutex::new(expected.into_iter().collect()),
            ..Default::default()
        })
    }

    pub(super) fn client(
        self: &Arc<Self>,
        cfg: Arc<RollupConfig>,
    ) -> OpEngineClient<RootProvider, RootProvider<Optimism>> {
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
        let l1 = tower::service_fn(|_: RequestPacket| -> TransportFut<'static> {
            Box::pin(async { Err(TransportErrorKind::custom_str("unexpected L1 call")) })
        });
        OpEngineClient::new(
            RootProvider::new(RpcClient::new(l1, false)),
            RootProvider::<Optimism>::new(RpcClient::new(transport, false)),
            cfg,
        )
    }

    pub(super) fn calls(&self) -> Vec<ObservedCall> {
        std::mem::take(&mut *self.observed.lock().unwrap())
    }

    pub(super) fn assert_finished(&self) {
        assert!(self.expected.lock().unwrap().is_empty(), "expected engine calls were not made");
    }
}
