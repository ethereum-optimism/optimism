//! `ots_getTransactionError` on an SDM block.

use super::fixture::{REVERT_OUTPUT, REVERTING_INDEX, SettledBlock};
use alloy_primitives::Bytes;
use jsonrpsee::{core::client::ClientT, rpc_params};

/// Otterscan replays a reverted transaction on an SDM block and returns its revert data.
#[tokio::test]
async fn otterscan_returns_revert_output_on_sdm_block() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    let revert: Option<Bytes> = fixture
        .client
        .request("ots_getTransactionError", rpc_params![fixture.transaction_hash(REVERTING_INDEX)])
        .await?;

    assert_eq!(
        revert,
        Some(Bytes::copy_from_slice(&REVERT_OUTPUT.to_be_bytes::<32>())),
        "ots_getTransactionError revert data",
    );
    Ok(())
}
