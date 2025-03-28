use crate::{
    supervisor::{parse_access_list_items_to_inbox_entries, ExecutingDescriptor, SupervisorClient},
    InvalidCrossTx,
};
use alloy_eips::eip2930::AccessList;
use alloy_primitives::TxHash;
use tracing::trace;

/// Extracts commitment from access list entries, pointing to 0x420..022 and validates them
/// against supervisor.
///
/// If commitment present pre-interop tx rejected.
///
/// Returns:
/// None - if tx is not cross chain,
/// Some(Ok(()) - if tx is valid cross chain,
/// Some(Err(e)) - if tx is not valid or interop is not active
pub async fn is_valid_cross_tx(
    access_list: Option<&AccessList>,
    hash: &TxHash,
    timestamp: u64,
    timeout: Option<u64>,
    is_interop_active: bool,
    client: Option<&SupervisorClient>,
) -> Option<Result<(), InvalidCrossTx>> {
    // We don't need to check for deposit transaction in here, because they won't come from
    // txpool
    let access_list = access_list?;
    let inbox_entries =
        parse_access_list_items_to_inbox_entries(access_list.iter()).copied().collect::<Vec<_>>();
    if inbox_entries.is_empty() {
        return None;
    }

    // Interop check
    if !is_interop_active {
        // No cross chain tx allowed before interop
        return Some(Err(InvalidCrossTx::CrossChainTxPreInterop))
    }
    let client = client.expect("supervisor client should be always set after interop is active");

    if let Err(err) = client
        .check_access_list(inbox_entries.as_slice(), ExecutingDescriptor::new(timestamp, timeout))
        .await
    {
        trace!(target: "txpool", hash=%hash, err=%err, "Cross chain transaction invalid");
        return Some(Err(InvalidCrossTx::ValidationError(err)));
    }
    Some(Ok(()))
}
