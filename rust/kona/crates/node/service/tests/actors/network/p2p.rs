use crate::actors::network::mocks::builder::TestNetworkBuilder;
use alloy_rpc_types_engine::ExecutionPayloadV1;
use kona_rpc::NetworkAdminQuery;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::time::Duration;
use tokio::{task::yield_now, time::timeout};

#[tokio::test(flavor = "multi_thread")]
async fn test_p2p_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new();
    let network_1 = builder.build(vec![]).await;
    let enr_1 = network_1.peer_enr().await?;

    let network_2 = builder.build(vec![enr_1]).await;

    network_2.is_connected_to_with_retries(&network_1).await?;

    network_1.is_connected_to_with_retries(&network_2).await?;

    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn test_large_network_conn() -> anyhow::Result<()> {
    const NETWORKS: usize = 10;

    let mut builder = TestNetworkBuilder::new();

    let (mut networks, mut bootnodes) = (vec![], vec![]);

    for _ in 0..NETWORKS {
        let network = builder.build(bootnodes.clone()).await;
        let enr = network.peer_enr().await?;
        networks.push(network);
        bootnodes.push(enr);
    }

    for network in &networks {
        for other_network in &networks {
            if network.peer_id().await? == other_network.peer_id().await? {
                continue;
            }

            network.is_connected_to_with_retries(other_network).await?;
        }
    }

    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn pending_admin_admission_keeps_p2p_responsive_and_preserves_payloads() -> anyhow::Result<()>
{
    let mut network = TestNetworkBuilder::new().build(vec![]).await;
    let capacity = network.blocks_rx.max_capacity();
    let payloads: Vec<_> = (0..=capacity)
        .map(|number| {
            let mut block = OpBlock::default();
            block.header.number = number as u64;
            OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block))
        })
        .collect();

    timeout(Duration::from_secs(10), async {
        for payload in &payloads {
            network
                .admin_rpc_tx
                .send(NetworkAdminQuery::PostUnsafePayload { payload: payload.clone() })
                .await?;
        }
        // All target slots are occupied and every admin request has left its
        // mailbox: the actor must retain the final payload while admission waits.
        while network.blocks_rx.len() != capacity ||
            network.admin_rpc_tx.capacity() != network.admin_rpc_tx.max_capacity()
        {
            yield_now().await;
        }
        Ok::<(), anyhow::Error>(())
    })
    .await
    .expect("target must fill with one pending admin payload")?;

    timeout(Duration::from_secs(10), network.peer_info())
        .await
        .expect("P2P queries must respond while engine admission is full")?;

    // Releasing capacity must admit the retained payload without losing,
    // duplicating or reordering any earlier request.
    timeout(Duration::from_secs(10), async {
        for expected in payloads {
            let admitted = network.blocks_rx.recv().await.expect("network must remain live");
            assert_eq!(admitted, expected);
        }
    })
    .await
    .expect("every admin payload must be admitted after capacity is released");
    assert!(matches!(
        network.blocks_rx.try_recv(),
        Err(tokio::sync::mpsc::error::TryRecvError::Empty)
    ));
    Ok(())
}
