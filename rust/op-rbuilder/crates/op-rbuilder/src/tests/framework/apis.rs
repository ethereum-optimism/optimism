use super::DEFAULT_JWT_TOKEN;
use alloy_eips::{BlockNumberOrTag, eip7685::Requests};
use alloy_primitives::B256;

use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus};
use core::{future::Future, marker::PhantomData};
use jsonrpsee::{
    core::{RpcResult, client::SubscriptionClientT},
    proc_macros::rpc,
};
use op_alloy_rpc_types_engine::OpExecutionPayloadV4;
use reth::rpc::types::engine::ForkchoiceState;
use reth_node_api::{EngineTypes, PayloadTypes};
use reth_optimism_node::OpEngineTypes;
use reth_optimism_rpc::engine::OpEngineApiClient;
use reth_payload_builder::PayloadId;
use reth_rpc_layer::{AuthClientLayer, JwtSecret};
use serde_json::Value;
use tracing::debug;

#[derive(Clone, Debug)]
pub enum Address {
    Ipc(String),
    Http(url::Url),
}

pub trait Protocol {
    fn client(
        jwt: JwtSecret,
        address: Address,
    ) -> impl Future<Output = impl SubscriptionClientT + Send + Sync + Unpin + 'static>;
}

pub struct Http;
impl Protocol for Http {
    async fn client(
        jwt: JwtSecret,
        address: Address,
    ) -> impl SubscriptionClientT + Send + Sync + Unpin + 'static {
        let Address::Http(url) = address else {
            unreachable!();
        };

        let secret_layer = AuthClientLayer::new(jwt);
        let middleware = tower::ServiceBuilder::default().layer(secret_layer);
        jsonrpsee::http_client::HttpClientBuilder::default()
            .set_http_middleware(middleware)
            .build(url)
            .expect("Failed to create http client")
    }
}

pub struct Ipc;
impl Protocol for Ipc {
    async fn client(
        _: JwtSecret, // ipc does not use JWT
        address: Address,
    ) -> impl SubscriptionClientT + Send + Sync + Unpin + 'static {
        let Address::Ipc(path) = address else {
            unreachable!();
        };
        reth_ipc::client::IpcClientBuilder::default()
            .build(&path)
            .await
            .expect("Failed to create ipc client")
    }
}

/// Helper for engine api operations
pub struct EngineApi<P: Protocol = Ipc> {
    address: Address,
    jwt_secret: JwtSecret,
    _tag: PhantomData<P>,
}

impl<P: Protocol> EngineApi<P> {
    async fn client(&self) -> impl SubscriptionClientT + Send + Sync + Unpin + 'static + use<P> {
        P::client(self.jwt_secret, self.address.clone()).await
    }
}

// http specific
impl EngineApi<Http> {
    pub fn with_http(url: &str) -> EngineApi<Http> {
        EngineApi::<Http> {
            address: Address::Http(url.parse().expect("Invalid URL")),
            jwt_secret: DEFAULT_JWT_TOKEN.parse().expect("Invalid JWT"),
            _tag: PhantomData,
        }
    }

    pub fn with_localhost_port(port: u16) -> EngineApi<Http> {
        EngineApi::<Http> {
            address: Address::Http(
                format!("http://localhost:{port}")
                    .parse()
                    .expect("Invalid URL"),
            ),
            jwt_secret: DEFAULT_JWT_TOKEN.parse().expect("Invalid JWT"),
            _tag: PhantomData,
        }
    }

    pub fn with_port(mut self, port: u16) -> Self {
        let Address::Http(url) = &mut self.address else {
            unreachable!();
        };

        url.set_port(Some(port)).expect("Invalid port");
        self
    }

    pub fn with_jwt_secret(mut self, jwt_secret: &str) -> Self {
        self.jwt_secret = jwt_secret.parse().expect("Invalid JWT");
        self
    }

    pub fn url(&self) -> &url::Url {
        let Address::Http(url) = &self.address else {
            unreachable!();
        };
        url
    }
}

// ipc specific
impl EngineApi<Ipc> {
    pub fn with_ipc(path: &str) -> EngineApi<Ipc> {
        EngineApi::<Ipc> {
            address: Address::Ipc(path.into()),
            jwt_secret: DEFAULT_JWT_TOKEN.parse().expect("Invalid JWT"),
            _tag: PhantomData,
        }
    }

    pub fn path(&self) -> &str {
        let Address::Ipc(path) = &self.address else {
            unreachable!();
        };
        path
    }
}

impl<P: Protocol> EngineApi<P> {
    pub async fn get_payload(
        &self,
        payload_id: PayloadId,
    ) -> eyre::Result<<OpEngineTypes as EngineTypes>::ExecutionPayloadEnvelopeV4> {
        debug!(
            "Fetching payload with id: {} at {}",
            payload_id,
            chrono::Utc::now()
        );
        Ok(
            OpEngineApiClient::<OpEngineTypes>::get_payload_v4(&self.client().await, payload_id)
                .await?,
        )
    }

    pub async fn new_payload(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Requests,
    ) -> eyre::Result<PayloadStatus> {
        debug!("Submitting new payload at {}...", chrono::Utc::now());
        Ok(OpEngineApiClient::<OpEngineTypes>::new_payload_v4(
            &self.client().await,
            payload,
            versioned_hashes,
            parent_beacon_block_root,
            execution_requests,
        )
        .await?)
    }

    pub async fn update_forkchoice(
        &self,
        current_head: B256,
        new_head: B256,
        payload_attributes: Option<<OpEngineTypes as PayloadTypes>::PayloadAttributes>,
    ) -> eyre::Result<ForkchoiceUpdated> {
        debug!("Updating forkchoice at {}...", chrono::Utc::now());
        Ok(OpEngineApiClient::<OpEngineTypes>::fork_choice_updated_v3(
            &self.client().await,
            ForkchoiceState {
                head_block_hash: new_head,
                safe_block_hash: current_head,
                finalized_block_hash: current_head,
            },
            payload_attributes,
        )
        .await?)
    }
}

#[rpc(server, client, namespace = "eth")]
pub trait BlockApi {
    #[method(name = "getBlockByNumber")]
    async fn get_block_by_number(
        &self,
        block_number: BlockNumberOrTag,
        include_txs: bool,
    ) -> RpcResult<Option<alloy_rpc_types_eth::Block>>;
}

pub fn generate_genesis(output: Option<String>) -> eyre::Result<()> {
    // Read the template file
    let template = include_str!("artifacts/genesis.json.tmpl");

    // Parse the JSON
    let mut genesis: Value = serde_json::from_str(template)?;

    // Update the timestamp field - example using current timestamp
    let timestamp = chrono::Utc::now().timestamp();
    if let Some(config) = genesis.as_object_mut() {
        // Assuming timestamp is at the root level - adjust path as needed
        config["timestamp"] = Value::String(format!("0x{timestamp:x}"));
    }

    // Write the result to the output file
    if let Some(output) = output {
        std::fs::write(&output, serde_json::to_string_pretty(&genesis)?)?;
        println!("Generated genesis file at: {output}");
    } else {
        println!("{}", serde_json::to_string_pretty(&genesis)?);
    }

    Ok(())
}
