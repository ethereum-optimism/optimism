use alloy_consensus::constants::EMPTY_WITHDRAWALS;
use alloy_eips::{BlockNumberOrTag, Encodable2718, eip7685::Requests};
use alloy_primitives::{B256, U256, keccak256, private::alloy_rlp::Encodable};
use alloy_provider::{Identity, Provider, ProviderBuilder, RootProvider};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3, PayloadStatusEnum,
};
use futures::{StreamExt, TryStreamExt};
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpExecutionPayloadV4;
use std::path::{Path, PathBuf};
use testcontainers::bollard::{
    Docker,
    container::{
        AttachContainerOptions, Config, CreateContainerOptions, RemoveContainerOptions,
        StartContainerOptions, StopContainerOptions,
    },
    exec::{CreateExecOptions, StartExecResults},
    image::CreateImageOptions,
    secret::{ContainerCreateResponse, HostConfig},
};
use tokio::signal;
use tracing::{debug, warn};

use crate::tests::{EngineApi, Ipc};

const AUTH_CONTAINER_IPC_PATH: &str = "/home/op-reth-shared/auth.ipc";
const RPC_CONTAINER_IPC_PATH: &str = "/home/op-reth-shared/rpc.ipc";

/// This type represents an Optimism execution client node that is running inside a
/// docker container. This node is used to validate the correctness of the blocks built
/// by op-rbuilder.
///
/// When this node is attached to a `ChainDriver`, it will automatically catch up with the
/// provided chain and will transparently ingest all newly built blocks by the driver.
///
/// If the built payload fails to validate, then the driver block production function will
/// return an error during `ChainDriver::build_new_block`.
pub struct ExternalNode {
    engine_api: EngineApi<Ipc>,
    provider: RootProvider<Optimism>,
    docker: Docker,
    tempdir: PathBuf,
    container_id: String,
}

impl ExternalNode {
    /// Creates a new instance of `ExternalNode` that runs the `op-reth` client in a Docker container
    /// using the specified version tag.
    pub async fn reth_version(version_tag: &str) -> eyre::Result<Self> {
        let docker = Docker::connect_with_local_defaults()?;

        let tempdir = std::env::var("TESTS_TEMP_DIR")
            .map(PathBuf::from)
            .unwrap_or_else(|_| std::env::temp_dir());

        let tempdir = tempdir.join(format!("reth-shared-{}", nanoid::nanoid!()));
        let auth_ipc = tempdir.join("auth.ipc").to_string_lossy().to_string();
        let rpc_ipc = tempdir.join("rpc.ipc").to_string_lossy().to_string();

        std::fs::create_dir_all(&tempdir)
            .map_err(|_| eyre::eyre!("Failed to create temporary directory"))?;

        std::fs::write(
            tempdir.join("genesis.json"),
            include_str!("./artifacts/genesis.json.tmpl"),
        )
        .map_err(|_| eyre::eyre!("Failed to write genesis file"))?;

        // Create Docker container with reth EL client
        let container = create_container(&tempdir, &docker, version_tag).await?;

        docker
            .start_container(&container.id, None::<StartContainerOptions<String>>)
            .await?;

        // Wait for the container to be ready and IPCs to be created
        await_ipc_readiness(&docker, &container.id).await?;

        // IPC files created by the container have restrictive permissions,
        // so we need to relax them to allow the host to access them.
        relax_permissions(&docker, &container.id, AUTH_CONTAINER_IPC_PATH).await?;
        relax_permissions(&docker, &container.id, RPC_CONTAINER_IPC_PATH).await?;

        // Connect to the IPCs
        let engine_api = EngineApi::with_ipc(&auth_ipc);
        let provider = ProviderBuilder::<Identity, Identity, Optimism>::default()
            .connect_ipc(rpc_ipc.into())
            .await?;

        // spin up a task that will clean up the container on ctrl-c
        tokio::spawn({
            let docker = docker.clone();
            let container_id = container.id.clone();
            let tempdir = tempdir.clone();

            async move {
                if signal::ctrl_c().await.is_ok() {
                    cleanup(tempdir.clone(), docker.clone(), container_id.clone()).await;
                }
            }
        });

        Ok(Self {
            engine_api,
            provider,
            docker,
            tempdir,
            container_id: container.id,
        })
    }

    /// Creates a new instance of `ExternalNode` that runs the `op-reth` client in a Docker container
    /// using the latest version.
    pub async fn reth() -> eyre::Result<Self> {
        Self::reth_version("latest").await
    }
}

impl ExternalNode {
    /// Access to the RPC API of the validation node.
    pub fn provider(&self) -> &RootProvider<Optimism> {
        &self.provider
    }

    /// Access to the Engine API of the validation node.
    pub fn engine_api(&self) -> &EngineApi<Ipc> {
        &self.engine_api
    }
}

impl ExternalNode {
    /// Catches up this node with another node.
    ///
    /// This method will fail if this node is ahead of the provided chain or they do not
    /// share the same genesis block.
    pub async fn catch_up_with(&self, chain: &RootProvider<Optimism>) -> eyre::Result<()> {
        // check if we need to catch up
        let (latest_hash, latest_number) = chain.latest_block_hash_and_number().await?;
        let (our_latest_hash, our_latest_number) =
            self.provider.latest_block_hash_and_number().await?;

        // check if we can sync in the first place
        match (our_latest_number, latest_number) {
            (we, them) if we == them && our_latest_hash == latest_hash => {
                // we are already caught up and in sync with the provided chain
                return Ok(());
            }
            (we, them) if we == them && our_latest_hash != latest_hash => {
                // divergent histories, can't sync
                return Err(eyre::eyre!(
                    "External node is at the same height but has a different latest block hash: \
                    {we} == {them}, {our_latest_hash} != {latest_hash}",
                ));
            }
            (we, them) if we > them => {
                return Err(eyre::eyre!(
                    "External node is ahead of the provided chain: {we} > {them}",
                ));
            }
            (we, them) if we < them => {
                debug!("external node is behind the local chain: {we} < {them}, catching up...");

                // make sure that we share common history with the provided chain
                let hash_at_height = chain.hash_at_height(we).await?;
                if hash_at_height != our_latest_hash {
                    return Err(eyre::eyre!(
                        "External node does not share the same genesis block or history with \
                        the provided chain: {} != {} at height {}",
                        hash_at_height,
                        our_latest_hash,
                        we
                    ));
                }
            }
            _ => {}
        };

        // we are behind, let's catch up
        let mut our_current_height = our_latest_number + 1;

        while our_current_height <= latest_number {
            let payload = chain
                .execution_payload_for_block(our_current_height)
                .await?;

            let (latest_hash, _) = self.provider().latest_block_hash_and_number().await?;

            let status = self
                .engine_api()
                .new_payload(payload, vec![], B256::ZERO, Requests::default())
                .await?;

            if status.status != PayloadStatusEnum::Valid {
                return Err(eyre::eyre!(
                    "Failed to import block at height {our_current_height} into external validation node: {:?}",
                    status.status
                ));
            }

            let new_chain_hash = status.latest_valid_hash.unwrap_or_default();
            self.engine_api()
                .update_forkchoice(latest_hash, new_chain_hash, None)
                .await?;

            our_current_height += 1;
        }

        // sync complete, double check that we are in sync
        let (final_hash, final_number) = self.provider().latest_block_hash_and_number().await?;

        if final_hash != latest_hash || final_number != latest_number {
            return Err(eyre::eyre!(
                "Failed to sync external validation node: {:?} != {:?}, {:?} != {:?}",
                final_hash,
                latest_hash,
                final_number,
                latest_number
            ));
        }

        Ok(())
    }

    /// Posts a block to the external validation node for validation and sets it as the latest block
    /// in the fork choice.
    pub async fn post_block(&self, payload: &OpExecutionPayloadV4) -> eyre::Result<()> {
        let result = self
            .engine_api
            .new_payload(payload.clone(), vec![], B256::ZERO, Requests::default())
            .await?;

        let new_block_hash = payload.payload_inner.payload_inner.payload_inner.block_hash;
        debug!(
            "external validation node payload status for block {new_block_hash}: {:?}",
            result.status
        );

        if result.status != PayloadStatusEnum::Valid {
            return Err(eyre::eyre!(
                "Failed to validate block {new_block_hash} with external validation node."
            ));
        }

        let (latest_hash, _) = self.provider.latest_block_hash_and_number().await?;

        self.engine_api
            .update_forkchoice(latest_hash, new_block_hash, None)
            .await?;

        Ok(())
    }
}

impl Drop for ExternalNode {
    fn drop(&mut self) {
        // Block on cleaning up the container
        let docker = self.docker.clone();
        let container_id = self.container_id.clone();
        let tempdir = self.tempdir.clone();
        tokio::spawn(async move {
            cleanup(tempdir, docker, container_id).await;
        });
    }
}

async fn create_container(
    tempdir: &Path,
    docker: &Docker,
    version_tag: &str,
) -> eyre::Result<ContainerCreateResponse> {
    let host_config = HostConfig {
        binds: Some(vec![format!(
            "{}:/home/op-reth-shared:rw",
            tempdir.display()
        )]),
        ..Default::default()
    };

    // first pull the image locally
    let mut pull_stream = docker.create_image(
        Some(CreateImageOptions {
            from_image: "ghcr.io/paradigmxyz/op-reth".to_string(),
            tag: version_tag.into(),
            ..Default::default()
        }),
        None,
        None,
    );

    while let Some(pull_result) = pull_stream.try_next().await? {
        debug!(
            "Pulling 'ghcr.io/paradigmxyz/op-reth:{version_tag}' locally: {:?}",
            pull_result
        );
    }

    // Don't expose any ports, as we will only use IPC for communication.
    let container_config = Config {
        image: Some(format!("ghcr.io/paradigmxyz/op-reth:{version_tag}")),
        entrypoint: Some(vec!["op-reth".to_string()]),
        cmd: Some(
            vec![
                "node",
                "--chain=/home/op-reth-shared/genesis.json",
                "--auth-ipc",
                &format!("--auth-ipc.path={AUTH_CONTAINER_IPC_PATH}"),
                &format!("--ipcpath={RPC_CONTAINER_IPC_PATH}"),
                "--disable-discovery",
                "--no-persist-peers",
                "--max-outbound-peers=0",
                "--max-inbound-peers=0",
                "--trusted-only",
            ]
            .into_iter()
            .map(String::from)
            .collect(),
        ),
        host_config: Some(host_config),
        ..Default::default()
    };

    Ok(docker
        .create_container(
            Some(CreateContainerOptions::<String>::default()),
            container_config,
        )
        .await?)
}

async fn relax_permissions(docker: &Docker, container: &str, path: &str) -> eyre::Result<()> {
    let exec = docker
        .create_exec(
            container,
            CreateExecOptions {
                cmd: Some(vec!["chmod", "777", path]),
                attach_stdout: Some(true),
                attach_stderr: Some(true),
                ..Default::default()
            },
        )
        .await?;

    let StartExecResults::Attached { mut output, .. } = docker.start_exec(&exec.id, None).await?
    else {
        return Err(eyre::eyre!("Failed to start exec for relaxing permissions"));
    };

    while let Some(Ok(output)) = output.next().await {
        use testcontainers::bollard::container::LogOutput::*;
        match output {
            StdErr { message } => {
                return Err(eyre::eyre!(
                    "Failed to relax permissions for {path}: {}",
                    String::from_utf8_lossy(&message)
                ));
            }
            _ => continue,
        };
    }

    Ok(())
}

async fn await_ipc_readiness(docker: &Docker, container: &str) -> eyre::Result<()> {
    let mut attach_stream = docker
        .attach_container(
            container,
            Some(AttachContainerOptions::<String> {
                stdout: Some(true),
                stderr: Some(true),
                stream: Some(true),
                logs: Some(true),
                ..Default::default()
            }),
        )
        .await?;

    let mut rpc_ipc_started = false;
    let mut auth_ipc_started = false;

    // wait for the node to start and signal that IPCs are ready
    while let Some(Ok(output)) = attach_stream.output.next().await {
        use testcontainers::bollard::container::LogOutput;
        match output {
            LogOutput::StdOut { message } | LogOutput::StdErr { message } => {
                let message = String::from_utf8_lossy(&message);
                if message.contains(AUTH_CONTAINER_IPC_PATH) {
                    auth_ipc_started = true;
                }

                if message.contains(RPC_CONTAINER_IPC_PATH) {
                    rpc_ipc_started = true;
                }

                if message.to_lowercase().contains("error") {
                    return Err(eyre::eyre!("Failed to start op-reth container: {message}."));
                }
            }
            LogOutput::StdIn { .. } | LogOutput::Console { .. } => {}
        }

        if auth_ipc_started && rpc_ipc_started {
            break;
        }
    }

    if !auth_ipc_started || !rpc_ipc_started {
        return Err(eyre::eyre!(
            "Failed to start op-reth container: IPCs not ready"
        ));
    }

    Ok(())
}

async fn cleanup(tempdir: PathBuf, docker: Docker, container_id: String) {
    // This is a no-op function that will be spawned to clean up the container on ctrl-c
    // or Drop.
    debug!(
        "Cleaning up external node resources at {} [{container_id}]...",
        tempdir.display()
    );

    if !tempdir.exists() {
        return; // If the tempdir does not exist, there's nothing to clean up.
    }

    // Block on cleaning up the container
    if let Err(e) = docker
        .stop_container(&container_id, None::<StopContainerOptions>)
        .await
    {
        warn!("Failed to stop container {}: {}", container_id, e);
    }

    if let Err(e) = docker
        .remove_container(
            &container_id,
            Some(RemoveContainerOptions {
                force: true,
                ..Default::default()
            }),
        )
        .await
    {
        warn!("Failed to remove container {}: {}", container_id, e);
    }

    // Clean up the temporary directory
    std::fs::remove_dir_all(&tempdir).expect("Failed to remove temporary directory");
}

trait OptimismProviderExt {
    async fn hash_at_height(&self, height: u64) -> eyre::Result<B256>;
    async fn latest_block_hash_and_number(&self) -> eyre::Result<(B256, u64)>;
    async fn execution_payload_for_block(&self, number: u64) -> eyre::Result<OpExecutionPayloadV4>;
}

impl OptimismProviderExt for RootProvider<Optimism> {
    async fn hash_at_height(&self, height: u64) -> eyre::Result<B256> {
        let block = self
            .get_block_by_number(BlockNumberOrTag::Number(height))
            .await?
            .ok_or_else(|| eyre::eyre!("No block found at height {}", height))?;
        Ok(block.header.hash)
    }

    async fn latest_block_hash_and_number(&self) -> eyre::Result<(B256, u64)> {
        let block = self
            .get_block_by_number(BlockNumberOrTag::Latest)
            .await?
            .ok_or_else(|| eyre::eyre!("No latest block found"))?;
        Ok((block.header.hash, block.header.number))
    }

    async fn execution_payload_for_block(&self, number: u64) -> eyre::Result<OpExecutionPayloadV4> {
        let block = self
            .get_block_by_number(BlockNumberOrTag::Number(number))
            .full()
            .await?
            .ok_or_else(|| eyre::eyre!("No block found at height {}", number))?;

        let withdrawals = block.withdrawals.clone().unwrap_or_default();

        // Calculate the withdrawals root properly
        let withdrawals_root = if withdrawals.is_empty() {
            EMPTY_WITHDRAWALS
        } else {
            // Calculate the Merkle Patricia Trie root of the withdrawals
            let mut buf = Vec::new();
            withdrawals.encode(&mut buf);
            keccak256(&buf)
        };

        let payload = OpExecutionPayloadV4 {
            payload_inner: ExecutionPayloadV3 {
                payload_inner: ExecutionPayloadV2 {
                    payload_inner: ExecutionPayloadV1 {
                        parent_hash: block.header.parent_hash,
                        fee_recipient: block.header.beneficiary,
                        state_root: block.header.state_root,
                        receipts_root: block.header.receipts_root,
                        logs_bloom: block.header.logs_bloom,
                        prev_randao: block.header.mix_hash,
                        block_number: block.header.number,
                        gas_limit: block.header.gas_limit,
                        gas_used: block.header.gas_used,
                        timestamp: block.header.timestamp,
                        extra_data: block.header.extra_data.clone(),
                        base_fee_per_gas: U256::from(
                            block.header.base_fee_per_gas.unwrap_or_default(),
                        ),
                        block_hash: block.header.hash,
                        transactions: block
                            .transactions
                            .into_transactions_vec()
                            .into_iter()
                            .map(|tx| tx.as_ref().encoded_2718().into())
                            .collect(),
                    },
                    withdrawals: block.withdrawals.unwrap_or_default().to_vec(),
                },
                blob_gas_used: block.header.inner.blob_gas_used.unwrap_or_default(),
                excess_blob_gas: block.header.inner.excess_blob_gas.unwrap_or_default(),
            },
            withdrawals_root,
        };

        Ok(payload)
    }
}
