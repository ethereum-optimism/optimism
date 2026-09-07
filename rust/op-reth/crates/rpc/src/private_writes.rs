//! Opaque net state changes, extracted from the full block execution outcome.
use alloy_primitives::{Address, B256, KECCAK256_EMPTY, U256, keccak256};
use reth_revm::db::BundleState;
use serde::{Deserialize, Serialize};

/// A private state identifier and commitment to its canonical resulting value.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WriteRecord {
    /// Domain-separated field/slot identifier.
    pub tag: B256,
    /// Domain-separated commitment to the new value.
    pub value_commitment: B256,
    /// Block boundary at which the value was recorded.
    pub block_number: u64,
}

/// Full-block net changes tied to an exact private block hash.
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BlockWrites {
    /// Exact reexecuted block, never a number resolved independently of its parent.
    pub block_hash: B256,
    /// Sorted unique records; an empty vector means no net changes.
    pub writes: Vec<WriteRecord>,
}

fn tag(chain: u64, kind: u8, address: Address, slot: B256) -> B256 {
    let mut input = b"optimism.private-interop.write-key.v1".to_vec();
    input.extend_from_slice(&chain.to_be_bytes());
    input.push(kind);
    input.extend_from_slice(address.as_slice());
    input.extend_from_slice(slot.as_slice());
    keccak256(input)
}

fn record(chain: u64, kind: u8, address: Address, slot: B256, value: B256) -> WriteRecord {
    let tag = tag(chain, kind, address, slot);
    let mut input = b"optimism.private-interop.write-value.v1".to_vec();
    input.extend_from_slice(tag.as_slice());
    input.extend_from_slice(value.as_slice());
    WriteRecord { tag, value_commitment: keccak256(input), block_number: 0 }
}

/// Includes system operations, fees and deposits. Reverted execution contributes only
/// its surviving effects. Storage wipes invalidate ALL slots through kind 4.
pub fn collect_writes(chain: u64, block_number: u64, bundle: &BundleState) -> Vec<WriteRecord> {
    let mut out = Vec::new();
    for (address, account) in &bundle.state {
        let fields = |info: Option<&revm::state::AccountInfo>| {
            [
                B256::from(U256::from(u8::from(info.is_some()))),
                B256::from(info.map_or(U256::ZERO, |i| i.balance)),
                B256::from(U256::from(info.map_or(0, |i| i.nonce))),
                info.map_or(KECCAK256_EMPTY, |i| i.code_hash),
            ]
        };
        let before = fields(account.original_info.as_ref());
        let after = fields(account.info.as_ref());
        for (kind, (old, new)) in before.into_iter().zip(after).enumerate() {
            if old != new {
                out.push(record(chain, kind as u8, *address, B256::ZERO, new));
            }
        }
        if account.was_destroyed() {
            out.push(record(chain, 4, *address, B256::ZERO, B256::ZERO));
        }
        if account.info.is_some() {
            for (slot, value) in &account.storage {
                if value.is_changed() || (account.was_destroyed() && !value.present_value.is_zero())
                {
                    out.push(record(
                        chain,
                        5,
                        *address,
                        B256::from(*slot),
                        B256::from(value.present_value),
                    ));
                }
            }
        }
    }
    for r in &mut out {
        r.block_number = block_number;
    }
    out.sort_unstable_by_key(|r| r.tag);
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_revm::db::{AccountStatus, BundleAccount, states::StorageSlot};
    use revm::state::AccountInfo;

    fn bundle(account: BundleAccount) -> BundleState {
        let mut bundle = BundleState::default();
        bundle.state.insert(Address::repeat_byte(0x11), account);
        bundle
    }

    #[test]
    fn matches_shared_go_vector() {
        let vector: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../../op-private-interop/writes/testdata/storage-write.json"
        ))
        .unwrap();
        let expected: WriteRecord = serde_json::from_value(vector["record"].clone()).unwrap();
        let mut actual = record(
            902,
            5,
            Address::repeat_byte(0x11),
            B256::from(U256::from(5)),
            B256::from(U256::from(7)),
        );
        actual.block_number = 42;
        assert_eq!(actual, expected);
        let mut encoded = Vec::new();
        encoded.extend_from_slice(actual.tag.as_slice());
        encoded.extend_from_slice(actual.value_commitment.as_slice());
        encoded.extend_from_slice(&actual.block_number.to_be_bytes());
        let expected_bytes: alloy_primitives::Bytes =
            serde_json::from_value(vector["encoded"].clone()).unwrap();
        assert_eq!(encoded.as_slice(), expected_bytes.as_ref());
    }

    #[test]
    fn ignores_restored_storage_but_keeps_fees_and_nonce() {
        let original = AccountInfo { balance: U256::from(100), ..Default::default() };
        let updated = AccountInfo { balance: U256::from(90), nonce: 1, ..original.clone() };
        let mut account = BundleAccount::new(
            Some(original),
            Some(updated),
            Default::default(),
            AccountStatus::Changed,
        );
        account.storage.insert(U256::ZERO, StorageSlot::new_changed(U256::from(7), U256::from(7)));
        let records = collect_writes(902, 42, &bundle(account));
        assert_eq!(records.len(), 2);
        assert!(records.iter().all(|r| r.block_number == 42));
        assert!(
            records.iter().any(|r| r.tag == tag(902, 1, Address::repeat_byte(0x11), B256::ZERO))
        );
        assert!(
            records.iter().any(|r| r.tag == tag(902, 2, Address::repeat_byte(0x11), B256::ZERO))
        );
    }

    #[test]
    fn deletion_marks_untouched_storage_and_creation_marks_existence() {
        let account = BundleAccount::new(
            Some(AccountInfo::default()),
            None,
            Default::default(),
            AccountStatus::Destroyed,
        );
        let records = collect_writes(902, 42, &bundle(account));
        assert_eq!(records.len(), 2);
        assert!(
            records.iter().any(|r| r.tag == tag(902, 4, Address::repeat_byte(0x11), B256::ZERO))
        );
        let created = BundleAccount::new(
            None,
            Some(AccountInfo::default()),
            Default::default(),
            AccountStatus::InMemoryChange,
        );
        let records = collect_writes(902, 42, &bundle(created));
        assert_eq!(records.len(), 1);
        assert_eq!(records[0].tag, tag(902, 0, Address::repeat_byte(0x11), B256::ZERO));
    }

    #[test]
    fn recreation_publishes_surviving_slots_even_if_value_is_restored() {
        let mut account = BundleAccount::new(
            Some(AccountInfo::default()),
            Some(AccountInfo::default()),
            Default::default(),
            AccountStatus::DestroyedChanged,
        );
        account
            .storage
            .insert(U256::from(5), StorageSlot::new_changed(U256::from(7), U256::from(7)));
        let records = collect_writes(902, 42, &bundle(account));
        assert_eq!(records.len(), 2);
        assert!(
            records.iter().any(
                |r| r.tag == tag(902, 5, Address::repeat_byte(0x11), B256::from(U256::from(5)))
            )
        );
        assert!(records.windows(2).all(|w| w[0].tag < w[1].tag));
    }
}
