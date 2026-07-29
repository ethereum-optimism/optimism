//! Verifies that two op-reth nodes negotiate the expected eth wire protocol version.
//!
//! The test is intentionally written to fail when upstream reth bumps the negotiated
//! version. That failure is the signal: it forces a conscious review of the new
//! version's behavior before we ship it, rather than silently inheriting a protocol
//! change on the next reth bump.

use alloy_rpc_types_admin::EthPeerInfo;
use reth_optimism_node::utils::setup;
use reth_rpc_api::servers::AdminApiServer;
use std::time::Duration;
use tokio::time::{sleep, timeout};

const EXPECTED_ETH_VERSION: u64 = 69;

#[tokio::test]
async fn peers_negotiate_eth_69() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let (nodes, _wallet) = setup(2).await?;
    let admin_a = nodes[0].inner.add_ons_handle.admin_api();

    let negotiated = timeout(Duration::from_secs(30), async {
        loop {
            let peers = admin_a.peers().await.expect("admin_peers rpc call");
            for p in peers {
                if let Some(EthPeerInfo::Info(info)) = p.protocols.eth {
                    return info.version;
                }
            }
            sleep(Duration::from_millis(200)).await;
        }
    })
    .await?;

    assert_eq!(
        negotiated, EXPECTED_ETH_VERSION,
        "expected eth/{EXPECTED_ETH_VERSION}, got eth/{negotiated}"
    );
    Ok(())
}
