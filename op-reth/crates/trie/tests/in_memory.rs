//! Common test suite for [`OpProofsStore`] implementations.

use alloy_eips::{eip1898::BlockWithParent, NumHash};
use alloy_primitives::{map::HashMap, B256, U256};
use reth_optimism_trie::{
    db::MdbxProofsStorage, BlockStateDiff, InMemoryProofsStorage, OpProofsHashedCursorRO,
    OpProofsStorageError, OpProofsStore, OpProofsTrieCursorRO,
};
use reth_primitives_traits::Account;
use reth_trie::{
    updates::TrieUpdates, BranchNodeCompact, HashedPostState, HashedStorage, Nibbles, TrieMask,
};
use serial_test::serial;
use std::sync::Arc;
use tempfile::TempDir;
use test_case::test_case;

/// Helper to create a simple test branch node
fn create_test_branch() -> BranchNodeCompact {
    let mut state_mask = TrieMask::default();
    state_mask.set_bit(0);
    state_mask.set_bit(1);

    BranchNodeCompact {
        state_mask,
        tree_mask: TrieMask::default(),
        hash_mask: TrieMask::default(),
        hashes: Arc::new(vec![]),
        root_hash: None,
    }
}

/// Helper to create a variant test branch node for comparison tests
fn create_test_branch_variant() -> BranchNodeCompact {
    let mut state_mask = TrieMask::default();
    state_mask.set_bit(5);
    state_mask.set_bit(6);

    BranchNodeCompact {
        state_mask,
        tree_mask: TrieMask::default(),
        hash_mask: TrieMask::default(),
        hashes: Arc::new(vec![]),
        root_hash: None,
    }
}

/// Helper to create nibbles from a vector of u8 values
fn nibbles_from(vec: Vec<u8>) -> Nibbles {
    Nibbles::from_nibbles_unchecked(vec)
}

/// Helper to create a test account
fn create_test_account() -> Account {
    Account {
        nonce: 42,
        balance: U256::from(1000000),
        bytecode_hash: Some(B256::repeat_byte(0xBB)),
    }
}

/// Helper to create a test account with custom values
fn create_test_account_with_values(nonce: u64, balance: u64, code_hash_byte: u8) -> Account {
    Account {
        nonce,
        balance: U256::from(balance),
        bytecode_hash: Some(B256::repeat_byte(code_hash_byte)),
    }
}

fn create_mdbx_proofs_storage() -> MdbxProofsStorage {
    let path = TempDir::new().unwrap();
    MdbxProofsStorage::new(path.path()).unwrap()
}

/// Test basic storage and retrieval of earliest block number
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_earliest_block_operations<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    // Initially should be None
    let earliest = storage.get_earliest_block_number().await?;
    assert!(earliest.is_none());

    // Set earliest block
    let block_hash = B256::repeat_byte(0x42);
    storage.set_earliest_block_number(100, block_hash).await?;

    // Should retrieve the same values
    let earliest = storage.get_earliest_block_number().await?;
    assert_eq!(earliest, Some((100, block_hash)));

    Ok(())
}

/// Test storing and retrieving trie updates
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_trie_updates_operations<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let block_ref = BlockWithParent::new(B256::ZERO, NumHash::new(50, B256::repeat_byte(0x96)));
    let trie_updates = TrieUpdates::default();
    let post_state = HashedPostState::default();
    let block_state_diff =
        BlockStateDiff { trie_updates: trie_updates.clone(), post_state: post_state.clone() };

    // Store trie updates
    storage.store_trie_updates(block_ref, block_state_diff).await?;

    // Retrieve and verify
    let retrieved_diff = storage.fetch_trie_updates(block_ref.block.number).await?;
    assert_eq!(retrieved_diff.trie_updates, trie_updates);
    assert_eq!(retrieved_diff.post_state, post_state);

    Ok(())
}

// =============================================================================
// 1. Basic Cursor Operations
// =============================================================================

/// Test cursor operations on empty trie
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_cursor_empty_trie<S: OpProofsStore>(storage: S) -> Result<(), OpProofsStorageError> {
    let mut cursor = storage.account_trie_cursor(100)?;

    // All operations should return None on empty trie
    assert!(cursor.seek_exact(Nibbles::default())?.is_none());
    assert!(cursor.seek(Nibbles::default())?.is_none());
    assert!(cursor.next()?.is_none());
    assert!(cursor.current()?.is_none());

    Ok(())
}

/// Test cursor operations with single entry
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_cursor_single_entry<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2, 3]);
    let branch = create_test_branch();

    // Store single entry
    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;

    // Test seek_exact
    let result = cursor.seek_exact(path)?.unwrap();
    assert_eq!(result.0, path);

    // Test current position
    assert_eq!(cursor.current()?.unwrap(), path);

    // Test next from end should return None
    assert!(cursor.next()?.is_none());

    Ok(())
}

/// Test cursor operations with multiple entries
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_cursor_multiple_entries<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let paths = vec![
        nibbles_from(vec![1]),
        nibbles_from(vec![1, 2]),
        nibbles_from(vec![2]),
        nibbles_from(vec![2, 3]),
    ];
    let branch = create_test_branch();

    // Store multiple entries
    for path in &paths {
        storage.store_account_branches(vec![(*path, Some(branch.clone()))]).await?;
    }

    let mut cursor = storage.account_trie_cursor(100)?;

    // Test that we can iterate through all entries
    let mut found_paths = Vec::new();
    while let Some((path, _)) = cursor.next()? {
        found_paths.push(path);
    }

    assert_eq!(found_paths.len(), 4);
    // Paths should be in lexicographic order
    for i in 0..paths.len() {
        assert_eq!(found_paths[i], paths[i]);
    }

    Ok(())
}

// =============================================================================
// 2. Seek Operations
// =============================================================================

/// Test `seek_exact` with existing path
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_exact_existing_path<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2, 3]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    let result = cursor.seek_exact(path)?.unwrap();
    assert_eq!(result.0, path);

    Ok(())
}

/// Test `seek_exact` with non-existing path
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_exact_non_existing_path<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2, 3]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    let non_existing = nibbles_from(vec![4, 5, 6]);
    assert!(cursor.seek_exact(non_existing)?.is_none());

    Ok(())
}

/// Test `seek_exact` with empty path
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_exact_empty_path<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    let result = cursor.seek_exact(Nibbles::default())?.unwrap();
    assert_eq!(result.0, Nibbles::default());

    Ok(())
}

/// Test seek to existing path
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_to_existing_path<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2, 3]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    let result = cursor.seek(path)?.unwrap();
    assert_eq!(result.0, path);

    Ok(())
}

/// Test seek between existing nodes
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_between_existing_nodes<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path1 = nibbles_from(vec![1]);
    let path2 = nibbles_from(vec![3]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path1, Some(branch.clone()))]).await?;
    storage.store_account_branches(vec![(path2, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    // Seek to path between 1 and 3, should return path 3
    let seek_path = nibbles_from(vec![2]);
    let result = cursor.seek(seek_path)?.unwrap();
    assert_eq!(result.0, path2);

    Ok(())
}

/// Test seek after all nodes
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_after_all_nodes<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    // Seek to path after all nodes
    let seek_path = nibbles_from(vec![9]);
    assert!(cursor.seek(seek_path)?.is_none());

    Ok(())
}

/// Test seek before all nodes
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_seek_before_all_nodes<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![5]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    // Seek to path before all nodes, should return first node
    let seek_path = nibbles_from(vec![1]);
    let result = cursor.seek(seek_path)?.unwrap();
    assert_eq!(result.0, path);

    Ok(())
}

// =============================================================================
// 3. Navigation Tests
// =============================================================================

/// Test next without prior seek
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_next_without_prior_seek<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    // next() without prior seek should start from beginning
    let result = cursor.next()?.unwrap();
    assert_eq!(result.0, path);

    Ok(())
}

/// Test next after seek
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_next_after_seek<S: OpProofsStore>(storage: S) -> Result<(), OpProofsStorageError> {
    let path1 = nibbles_from(vec![1]);
    let path2 = nibbles_from(vec![2]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path1, Some(branch.clone()))]).await?;
    storage.store_account_branches(vec![(path2, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    cursor.seek(path1)?;

    // next() should return second node
    let result = cursor.next()?.unwrap();
    assert_eq!(result.0, path2);

    Ok(())
}

/// Test next at end of trie
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_next_at_end_of_trie<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;
    cursor.seek(path)?;

    // next() at end should return None
    assert!(cursor.next()?.is_none());

    Ok(())
}

/// Test multiple consecutive next calls
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_multiple_consecutive_next<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let paths = vec![nibbles_from(vec![1]), nibbles_from(vec![2]), nibbles_from(vec![3])];
    let branch = create_test_branch();

    for path in &paths {
        storage.store_account_branches(vec![(*path, Some(branch.clone()))]).await?;
    }

    let mut cursor = storage.account_trie_cursor(100)?;

    // Iterate through all with consecutive next() calls
    for expected_path in &paths {
        let result = cursor.next()?.unwrap();
        assert_eq!(result.0, *expected_path);
    }

    // Final next() should return None
    assert!(cursor.next()?.is_none());

    Ok(())
}

/// Test current after operations
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_current_after_operations<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path1 = nibbles_from(vec![1]);
    let path2 = nibbles_from(vec![2]);
    let branch = create_test_branch();

    storage.store_account_branches(vec![(path1, Some(branch.clone()))]).await?;
    storage.store_account_branches(vec![(path2, Some(branch.clone()))]).await?;

    let mut cursor = storage.account_trie_cursor(100)?;

    // Current should be None initially
    assert!(cursor.current()?.is_none());

    // After seek, current should track position
    cursor.seek(path1)?;
    assert_eq!(cursor.current()?.unwrap(), path1);

    // After next, current should update
    cursor.next()?;
    assert_eq!(cursor.current()?.unwrap(), path2);

    Ok(())
}

/// Test current with no prior operations
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_current_no_prior_operations<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let mut cursor = storage.account_trie_cursor(100)?;

    // Current should be None when no operations performed
    assert!(cursor.current()?.is_none());

    Ok(())
}

// =============================================================================
// 4. Block Number Filtering
// =============================================================================

/// Test same path with different blocks
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_same_path_different_blocks<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2]);
    let branch1 = create_test_branch();
    let branch2 = create_test_branch_variant();

    // Store same path at different blocks
    storage.store_account_branches(vec![(path, Some(branch1.clone()))]).await?;
    storage.store_account_branches(vec![(path, Some(branch2.clone()))]).await?;

    // Cursor with max_block_number=75 should see only block 50 data
    let mut cursor75 = storage.account_trie_cursor(75)?;
    let result75 = cursor75.seek_exact(path)?.unwrap();
    assert_eq!(result75.0, path);

    // Cursor with max_block_number=150 should see block 100 data (latest)
    let mut cursor150 = storage.account_trie_cursor(150)?;
    let result150 = cursor150.seek_exact(path)?.unwrap();
    assert_eq!(result150.0, path);

    Ok(())
}

/// Test deleted branch nodes
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_deleted_branch_nodes<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2]);
    let branch = create_test_branch();
    let block_ref = BlockWithParent::new(B256::ZERO, NumHash::new(100, B256::repeat_byte(0x96)));

    // Store branch node, then delete it (store None)
    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    // Cursor before deletion should see the node
    let mut cursor75 = storage.account_trie_cursor(75)?;
    assert!(cursor75.seek_exact(path)?.is_some());

    let mut block_state_diff = BlockStateDiff::default();
    block_state_diff.trie_updates.removed_nodes.insert(path);
    storage.store_trie_updates(block_ref, block_state_diff).await?;

    // Cursor after deletion should not see the node
    let mut cursor150 = storage.account_trie_cursor(150)?;
    assert!(cursor150.seek_exact(path)?.is_none());

    Ok(())
}

// =============================================================================
// 5. Hashed Address Filtering
// =============================================================================

/// Test account-specific cursor
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_account_specific_cursor<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2]);
    let addr1 = B256::repeat_byte(0x01);
    let addr2 = B256::repeat_byte(0x02);
    let branch = create_test_branch();

    // Store same path for different accounts (using storage branches)
    storage.store_storage_branches(addr1, vec![(path, Some(branch.clone()))]).await?;
    storage.store_storage_branches(addr2, vec![(path, Some(branch.clone()))]).await?;

    // Cursor for addr1 should only see addr1 data
    let mut cursor1 = storage.storage_trie_cursor(addr1, 100)?;
    let result1 = cursor1.seek_exact(path)?.unwrap();
    assert_eq!(result1.0, path);

    // Cursor for addr2 should only see addr2 data
    let mut cursor2 = storage.storage_trie_cursor(addr2, 100)?;
    let result2 = cursor2.seek_exact(path)?.unwrap();
    assert_eq!(result2.0, path);

    // Cursor for addr1 should not see addr2 data when iterating
    let mut cursor1_iter = storage.storage_trie_cursor(addr1, 100)?;
    let mut found_count = 0;
    while cursor1_iter.next()?.is_some() {
        found_count += 1;
    }
    assert_eq!(found_count, 1); // Should only see one entry (for addr1)

    Ok(())
}

/// Test state trie cursor
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_state_trie_cursor<S: OpProofsStore>(storage: S) -> Result<(), OpProofsStorageError> {
    let path = nibbles_from(vec![1, 2]);
    let addr = B256::repeat_byte(0x01);
    let branch = create_test_branch();

    // Store data for account trie and state trie
    storage.store_storage_branches(addr, vec![(path, Some(branch.clone()))]).await?;
    storage.store_account_branches(vec![(path, Some(branch.clone()))]).await?;

    // State trie cursor (None address) should only see state trie data
    let mut state_cursor = storage.account_trie_cursor(100)?;
    let result = state_cursor.seek_exact(path)?.unwrap();
    assert_eq!(result.0, path);

    // Verify state cursor doesn't see account data when iterating
    let mut state_cursor_iter = storage.account_trie_cursor(100)?;
    let mut found_count = 0;
    while state_cursor_iter.next()?.is_some() {
        found_count += 1;
    }

    assert_eq!(found_count, 1); // Should only see state trie entry

    Ok(())
}

/// Test mixed account and state data
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_mixed_account_state_data<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let path1 = nibbles_from(vec![1]);
    let path2 = nibbles_from(vec![2]);
    let addr = B256::repeat_byte(0x01);
    let branch = create_test_branch();

    // Store mixed account and state trie data
    storage.store_storage_branches(addr, vec![(path1, Some(branch.clone()))]).await?;
    storage.store_account_branches(vec![(path2, Some(branch.clone()))]).await?;

    // Account cursor should only see account data
    let mut account_cursor = storage.storage_trie_cursor(addr, 100)?;
    let mut account_paths = Vec::new();
    while let Some((path, _)) = account_cursor.next()? {
        account_paths.push(path);
    }
    assert_eq!(account_paths.len(), 1);
    assert_eq!(account_paths[0], path1);

    // State cursor should only see state data
    let mut state_cursor = storage.account_trie_cursor(100)?;
    let mut state_paths = Vec::new();
    while let Some((path, _)) = state_cursor.next()? {
        state_paths.push(path);
    }
    assert_eq!(state_paths.len(), 1);
    assert_eq!(state_paths[0], path2);

    Ok(())
}

// =============================================================================
// 6. Path Ordering Tests
// =============================================================================

/// Test lexicographic ordering
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_lexicographic_ordering<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let paths = vec![
        nibbles_from(vec![3, 1]),
        nibbles_from(vec![1, 2]),
        nibbles_from(vec![2]),
        nibbles_from(vec![1]),
    ];
    let branch = create_test_branch();

    // Store paths in random order
    for path in &paths {
        storage.store_account_branches(vec![(*path, Some(branch.clone()))]).await?;
    }

    let mut cursor = storage.account_trie_cursor(100)?;
    let mut found_paths = Vec::new();
    while let Some((path, _)) = cursor.next()? {
        found_paths.push(path);
    }

    // Should be returned in lexicographic order: [1], [1,2], [2], [3,1]
    let expected_order = vec![
        nibbles_from(vec![1]),
        nibbles_from(vec![1, 2]),
        nibbles_from(vec![2]),
        nibbles_from(vec![3, 1]),
    ];

    assert_eq!(found_paths, expected_order);

    Ok(())
}

/// Test path prefix scenarios
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_path_prefix_scenarios<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let paths = vec![
        nibbles_from(vec![1]),       // Prefix of next
        nibbles_from(vec![1, 2]),    // Extends first
        nibbles_from(vec![1, 2, 3]), // Extends second
    ];
    let branch = create_test_branch();

    for path in &paths {
        storage.store_account_branches(vec![(*path, Some(branch.clone()))]).await?;
    }

    let mut cursor = storage.account_trie_cursor(100)?;

    // Seek to prefix should find exact match
    let result = cursor.seek_exact(paths[0])?.unwrap();
    assert_eq!(result.0, paths[0]);

    // Next should go to next path, not skip prefixed paths
    let result = cursor.next()?.unwrap();
    assert_eq!(result.0, paths[1]);

    let result = cursor.next()?.unwrap();
    assert_eq!(result.0, paths[2]);

    Ok(())
}

/// Test complex nibble combinations
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_complex_nibble_combinations<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    // Test various nibble patterns including edge values
    let paths = vec![
        nibbles_from(vec![0]),
        nibbles_from(vec![0, 15]),
        nibbles_from(vec![15]),
        nibbles_from(vec![15, 0]),
        nibbles_from(vec![7, 8, 9]),
    ];
    let branch = create_test_branch();

    for path in &paths {
        storage.store_account_branches(vec![(*path, Some(branch.clone()))]).await?;
    }

    let mut cursor = storage.account_trie_cursor(100)?;
    let mut found_paths = Vec::new();
    while let Some((path, _)) = cursor.next()? {
        found_paths.push(path);
    }

    // All paths should be found and in correct order
    assert_eq!(found_paths.len(), 5);

    // Verify specific ordering for edge cases
    assert_eq!(found_paths[0], nibbles_from(vec![0]));
    assert_eq!(found_paths[1], nibbles_from(vec![0, 15]));
    assert_eq!(found_paths[4], nibbles_from(vec![15, 0]));

    Ok(())
}

// =============================================================================
// 7. Leaf Node Tests (Hashed Accounts and Storage)
// =============================================================================

/// Test store and retrieve single account
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_store_and_retrieve_single_account<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let account_key = B256::repeat_byte(0x01);
    let account = create_test_account();

    // Store account
    storage.store_hashed_accounts(vec![(account_key, Some(account))]).await?;

    // Retrieve via cursor
    let mut cursor = storage.account_hashed_cursor(100)?;
    let result = cursor.seek(account_key)?.unwrap();

    assert_eq!(result.0, account_key);
    assert_eq!(result.1.nonce, account.nonce);
    assert_eq!(result.1.balance, account.balance);
    assert_eq!(result.1.bytecode_hash, account.bytecode_hash);

    Ok(())
}

/// Test account cursor navigation
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_account_cursor_navigation<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let accounts = [
        (B256::repeat_byte(0x01), create_test_account()),
        (B256::repeat_byte(0x03), create_test_account()),
        (B256::repeat_byte(0x05), create_test_account()),
    ];

    // Store accounts
    let accounts_to_store: Vec<_> = accounts.iter().map(|(k, v)| (*k, Some(*v))).collect();
    storage.store_hashed_accounts(accounts_to_store).await?;

    let mut cursor = storage.account_hashed_cursor(100)?;

    // Test seeking to exact key
    let result = cursor.seek(accounts[1].0)?.unwrap();
    assert_eq!(result.0, accounts[1].0);

    // Test seeking to key that doesn't exist (should return next greater)
    let seek_key = B256::repeat_byte(0x02);
    let result = cursor.seek(seek_key)?.unwrap();
    assert_eq!(result.0, accounts[1].0); // Should find 0x03

    // Test next() navigation
    let result = cursor.next()?.unwrap();
    assert_eq!(result.0, accounts[2].0); // Should find 0x05

    // Test next() at end
    assert!(cursor.next()?.is_none());

    Ok(())
}

/// Test account block versioning
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_account_block_versioning<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let account_key = B256::repeat_byte(0x01);
    let account_v1 = create_test_account_with_values(1, 100, 0xBB);
    let account_v2 = create_test_account_with_values(2, 200, 0xDD);

    // Store account at different blocks
    storage.store_hashed_accounts(vec![(account_key, Some(account_v1))]).await?;

    // Cursor with max_block_number=75 should see v1
    let mut cursor75 = storage.account_hashed_cursor(75)?;
    let result75 = cursor75.seek(account_key)?.unwrap();
    assert_eq!(result75.1.nonce, account_v1.nonce);
    assert_eq!(result75.1.balance, account_v1.balance);

    storage.store_hashed_accounts(vec![(account_key, Some(account_v2))]).await?;

    // After update, Cursor with max_block_number=150 should see v2
    let mut cursor150 = storage.account_hashed_cursor(150)?;
    let result150 = cursor150.seek(account_key)?.unwrap();
    assert_eq!(result150.1.nonce, account_v2.nonce);
    assert_eq!(result150.1.balance, account_v2.balance);

    Ok(())
}

/// Test store and retrieve storage
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
async fn test_store_and_retrieve_storage<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let hashed_address = B256::repeat_byte(0x01);
    let storage_slots = vec![
        (B256::repeat_byte(0x10), U256::from(100)),
        (B256::repeat_byte(0x20), U256::from(200)),
        (B256::repeat_byte(0x30), U256::from(300)),
    ];

    // Store storage slots
    storage.store_hashed_storages(hashed_address, storage_slots.clone()).await?;

    // Retrieve via cursor
    let mut cursor = storage.storage_hashed_cursor(hashed_address, 100)?;

    // Test seeking to each slot
    for (key, expected_value) in &storage_slots {
        let result = cursor.seek(*key)?.unwrap();
        assert_eq!(result.0, *key);
        assert_eq!(result.1, *expected_value);
    }

    Ok(())
}

/// Test storage cursor navigation
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_storage_cursor_navigation<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let hashed_address = B256::repeat_byte(0x01);
    let storage_slots = vec![
        (B256::repeat_byte(0x10), U256::from(100)),
        (B256::repeat_byte(0x30), U256::from(300)),
        (B256::repeat_byte(0x50), U256::from(500)),
    ];

    storage.store_hashed_storages(hashed_address, storage_slots.clone()).await?;

    let mut cursor = storage.storage_hashed_cursor(hashed_address, 100)?;

    // Start from beginning with next()
    let mut found_slots = Vec::new();
    while let Some((key, value)) = cursor.next()? {
        found_slots.push((key, value));
    }

    assert_eq!(found_slots.len(), 3);
    assert_eq!(found_slots[0], storage_slots[0]);
    assert_eq!(found_slots[1], storage_slots[1]);
    assert_eq!(found_slots[2], storage_slots[2]);

    Ok(())
}

/// Test storage account isolation
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_storage_account_isolation<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let address1 = B256::repeat_byte(0x01);
    let address2 = B256::repeat_byte(0x02);
    let storage_key = B256::repeat_byte(0x10);

    // Store same storage key for different accounts
    storage.store_hashed_storages(address1, vec![(storage_key, U256::from(100))]).await?;
    storage.store_hashed_storages(address2, vec![(storage_key, U256::from(200))]).await?;

    // Verify each account sees only its own storage
    let mut cursor1 = storage.storage_hashed_cursor(address1, 100)?;
    let result1 = cursor1.seek(storage_key)?.unwrap();
    assert_eq!(result1.1, U256::from(100));

    let mut cursor2 = storage.storage_hashed_cursor(address2, 100)?;
    let result2 = cursor2.seek(storage_key)?.unwrap();
    assert_eq!(result2.1, U256::from(200));

    // Verify cursor1 doesn't see address2's storage
    let mut cursor1_iter = storage.storage_hashed_cursor(address1, 100)?;
    let mut count = 0;
    while cursor1_iter.next()?.is_some() {
        count += 1;
    }
    assert_eq!(count, 1); // Should only see one entry

    Ok(())
}

/// Test storage block versioning
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_storage_block_versioning<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let hashed_address = B256::repeat_byte(0x01);
    let storage_key = B256::repeat_byte(0x10);

    // Store storage at different blocks
    storage.store_hashed_storages(hashed_address, vec![(storage_key, U256::from(100))]).await?;

    // Cursor with max_block_number=75 should see old value
    let mut cursor75 = storage.storage_hashed_cursor(hashed_address, 75)?;
    let result75 = cursor75.seek(storage_key)?.unwrap();
    assert_eq!(result75.1, U256::from(100));

    storage.store_hashed_storages(hashed_address, vec![(storage_key, U256::from(200))]).await?;
    // Cursor with max_block_number=150 should see new value
    let mut cursor150 = storage.storage_hashed_cursor(hashed_address, 150)?;
    let result150 = cursor150.seek(storage_key)?.unwrap();
    assert_eq!(result150.1, U256::from(200));

    Ok(())
}

/// Test storage zero value deletion
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_storage_zero_value_deletion<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let hashed_address = B256::repeat_byte(0x01);
    let storage_key = B256::repeat_byte(0x10);

    // Store non-zero value
    storage.store_hashed_storages(hashed_address, vec![(storage_key, U256::from(100))]).await?;

    // Cursor before deletion should see the value
    let mut cursor75 = storage.storage_hashed_cursor(hashed_address, 75)?;
    let result75 = cursor75.seek(storage_key)?.unwrap();
    assert_eq!(result75.1, U256::from(100));

    // "Delete" by storing zero value at block 100
    let mut block_state_diff = BlockStateDiff::default();
    let mut hashed_storage = HashedStorage::default();
    hashed_storage.storage.insert(storage_key, U256::ZERO);
    block_state_diff.post_state.storages.insert(hashed_address, hashed_storage);

    let block_ref: BlockWithParent =
        BlockWithParent::new(B256::ZERO, NumHash::new(100, B256::repeat_byte(0x96)));
    storage.store_trie_updates(block_ref, block_state_diff).await?;

    // Cursor after deletion should NOT see the entry (zero values are skipped)
    let mut cursor150 = storage.storage_hashed_cursor(hashed_address, 150)?;
    let result150 = cursor150.seek(storage_key)?;
    assert!(result150.is_none(), "Zero values should be skipped/deleted");

    Ok(())
}

/// Test that zero values are skipped during iteration
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_storage_cursor_skips_zero_values<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let hashed_address = B256::repeat_byte(0x01);

    // Create a mix of non-zero and zero value storage slots
    let storage_slots = vec![
        (B256::repeat_byte(0x10), U256::from(100)), // Non-zero
        (B256::repeat_byte(0x20), U256::ZERO),      // Zero value - should be skipped
        (B256::repeat_byte(0x30), U256::from(300)), // Non-zero
        (B256::repeat_byte(0x40), U256::ZERO),      // Zero value - should be skipped
        (B256::repeat_byte(0x50), U256::from(500)), // Non-zero
    ];

    // Store all slots
    storage.store_hashed_storages(hashed_address, storage_slots.clone()).await?;

    // Create cursor and iterate through all entries
    let mut cursor = storage.storage_hashed_cursor(hashed_address, 100)?;
    let mut found_slots = Vec::new();
    while let Some((key, value)) = cursor.next()? {
        found_slots.push((key, value));
    }

    // Should only find 3 non-zero values
    assert_eq!(found_slots.len(), 3, "Zero values should be skipped during iteration");

    // Verify the non-zero values are the ones we stored
    assert_eq!(found_slots[0], (B256::repeat_byte(0x10), U256::from(100)));
    assert_eq!(found_slots[1], (B256::repeat_byte(0x30), U256::from(300)));
    assert_eq!(found_slots[2], (B256::repeat_byte(0x50), U256::from(500)));

    // Verify seeking to a zero-value slot returns None or skips to next non-zero
    let mut seek_cursor = storage.storage_hashed_cursor(hashed_address, 100)?;
    let seek_result = seek_cursor.seek(B256::repeat_byte(0x20))?;

    // Should either return None or skip to the next non-zero value (0x30)
    if let Some((key, value)) = seek_result {
        assert_eq!(key, B256::repeat_byte(0x30), "Should skip zero value and find next non-zero");
        assert_eq!(value, U256::from(300));
    }

    Ok(())
}

/// Test empty cursors
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_empty_cursors<S: OpProofsStore>(storage: S) -> Result<(), OpProofsStorageError> {
    // Test empty account cursor
    let mut account_cursor = storage.account_hashed_cursor(100)?;
    assert!(account_cursor.seek(B256::repeat_byte(0x01))?.is_none());
    assert!(account_cursor.next()?.is_none());

    // Test empty storage cursor
    let mut storage_cursor = storage.storage_hashed_cursor(B256::repeat_byte(0x01), 100)?;
    assert!(storage_cursor.seek(B256::repeat_byte(0x10))?.is_none());
    assert!(storage_cursor.next()?.is_none());

    Ok(())
}

/// Test cursor boundary conditions
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_cursor_boundary_conditions<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    let account_key = B256::repeat_byte(0x80); // Middle value
    let account = create_test_account();

    storage.store_hashed_accounts(vec![(account_key, Some(account))]).await?;

    let mut cursor = storage.account_hashed_cursor(100)?;

    // Seek to minimum key should find our account
    let result = cursor.seek(B256::ZERO)?.unwrap();
    assert_eq!(result.0, account_key);

    // Seek to maximum key should find nothing
    assert!(cursor.seek(B256::repeat_byte(0xFF))?.is_none());

    // Seek to key just before our account should find our account
    let just_before = B256::repeat_byte(0x7F);
    let result = cursor.seek(just_before)?.unwrap();
    assert_eq!(result.0, account_key);

    Ok(())
}

/// Test large batch operations
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_large_batch_operations<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    // Create large batch of accounts
    let mut accounts = Vec::new();
    for i in 0..100 {
        let key = B256::from([i as u8; 32]);
        let account = create_test_account_with_values(i, i * 1000, (i + 1) as u8);
        accounts.push((key, Some(account)));
    }

    // Store in batch
    storage.store_hashed_accounts(accounts.clone()).await?;

    // Verify all accounts can be retrieved
    let mut cursor = storage.account_hashed_cursor(100)?;
    let mut found_count = 0;
    while cursor.next()?.is_some() {
        found_count += 1;
    }
    assert_eq!(found_count, 100);

    // Test specific account retrieval
    let test_key = B256::from([42u8; 32]);
    let result = cursor.seek(test_key)?.unwrap();
    assert_eq!(result.0, test_key);
    assert_eq!(result.1.nonce, 42);

    Ok(())
}

/// Test wiped storage in [`HashedPostState`]
///
/// When `store_trie_updates` receives a [`HashedPostState`] with wiped=true for a storage entry,
/// it should iterate all existing values for that address and create deletion entries for them.
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_store_trie_updates_with_wiped_storage<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    use reth_trie::HashedStorage;

    let hashed_address = B256::repeat_byte(0x01);
    let block_ref: BlockWithParent =
        BlockWithParent::new(B256::ZERO, NumHash::new(100, B256::repeat_byte(0x96)));

    // First, store some storage values at block 50
    let storage_slots = vec![
        (B256::repeat_byte(0x10), U256::from(100)),
        (B256::repeat_byte(0x20), U256::from(200)),
        (B256::repeat_byte(0x30), U256::from(300)),
        (B256::repeat_byte(0x40), U256::from(400)),
    ];

    storage.store_hashed_storages(hashed_address, storage_slots.clone()).await?;

    // Verify all values are present at block 75
    let mut cursor75 = storage.storage_hashed_cursor(hashed_address, 75)?;
    let mut found_slots = Vec::new();
    while let Some((key, value)) = cursor75.next()? {
        found_slots.push((key, value));
    }
    assert_eq!(found_slots.len(), 4, "All storage slots should be present before wipe");
    assert_eq!(found_slots[0], (B256::repeat_byte(0x10), U256::from(100)));
    assert_eq!(found_slots[1], (B256::repeat_byte(0x20), U256::from(200)));
    assert_eq!(found_slots[2], (B256::repeat_byte(0x30), U256::from(300)));
    assert_eq!(found_slots[3], (B256::repeat_byte(0x40), U256::from(400)));

    // Now create a HashedPostState with wiped=true for this address at block 100
    let mut post_state = HashedPostState::default();
    let wiped_storage = HashedStorage::new(true); // wiped=true, empty storage map
    post_state.storages.insert(hashed_address, wiped_storage);

    let block_state_diff = BlockStateDiff { trie_updates: TrieUpdates::default(), post_state };

    // Store the wiped state
    storage.store_trie_updates(block_ref, block_state_diff).await?;

    // After wiping, cursor at block 150 should see NO storage values
    let mut cursor150 = storage.storage_hashed_cursor(hashed_address, 150)?;
    let mut found_slots_after_wipe = Vec::new();
    while let Some((key, value)) = cursor150.next()? {
        found_slots_after_wipe.push((key, value));
    }

    assert_eq!(
        found_slots_after_wipe.len(),
        0,
        "All storage slots should be deleted after wipe. Found: {:?}",
        found_slots_after_wipe
    );

    // Verify individual seeks also return None
    for (slot, _) in &storage_slots {
        let mut seek_cursor = storage.storage_hashed_cursor(hashed_address, 150)?;
        let result = seek_cursor.seek(*slot)?;
        assert!(
            result.is_none() || result.unwrap().0 != *slot,
            "Storage slot {:?} should be deleted after wipe",
            slot
        );
    }

    // Verify cursor at block 75 (before wipe) still sees all values
    let mut cursor75_after = storage.storage_hashed_cursor(hashed_address, 75)?;
    let mut found_slots_before_wipe = Vec::new();
    while let Some((key, value)) = cursor75_after.next()? {
        found_slots_before_wipe.push((key, value));
    }
    assert_eq!(
        found_slots_before_wipe.len(),
        4,
        "All storage slots should still be present when querying before wipe block"
    );

    Ok(())
}

/// Test that `store_trie_updates` properly stores branch nodes, leaf nodes, and removals
///
/// This test verifies that all data stored via `store_trie_updates` can be read back
/// through the cursor APIs.
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_store_trie_updates_comprehensive<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    use reth_trie::{updates::StorageTrieUpdates, HashedStorage};

    let block_ref = BlockWithParent::new(B256::ZERO, NumHash::new(100, B256::repeat_byte(0x96)));

    // Create comprehensive trie updates with branches, leaves, and removals
    let mut trie_updates = TrieUpdates::default();

    // Add account branch nodes
    let account_path1 = nibbles_from(vec![1, 2, 3]);
    let account_path2 = nibbles_from(vec![4, 5, 6]);
    let account_branch1 = create_test_branch();
    let account_branch2 = create_test_branch_variant();

    trie_updates.account_nodes.insert(account_path1, account_branch1.clone());
    trie_updates.account_nodes.insert(account_path2, account_branch2.clone());

    // Add removed account nodes
    let removed_account_path = nibbles_from(vec![7, 8, 9]);
    trie_updates.removed_nodes.insert(removed_account_path);

    // Add storage branch nodes for an address
    let hashed_address = B256::repeat_byte(0x42);
    let storage_path1 = nibbles_from(vec![1, 1]);
    let storage_path2 = nibbles_from(vec![2, 2]);
    let storage_branch = create_test_branch();

    let mut storage_trie = StorageTrieUpdates::default();
    storage_trie.storage_nodes.insert(storage_path1, storage_branch.clone());
    storage_trie.storage_nodes.insert(storage_path2, storage_branch.clone());

    // Add removed storage node
    let removed_storage_path = nibbles_from(vec![3, 3]);
    storage_trie.removed_nodes.insert(removed_storage_path);

    trie_updates.insert_storage_updates(hashed_address, storage_trie);

    // Create post state with accounts and storage
    let mut post_state = HashedPostState::default();

    // Add accounts
    let account1_addr = B256::repeat_byte(0x10);
    let account2_addr = B256::repeat_byte(0x20);
    let account1 = create_test_account_with_values(1, 1000, 0xAA);
    let account2 = create_test_account_with_values(2, 2000, 0xBB);

    post_state.accounts.insert(account1_addr, Some(account1));
    post_state.accounts.insert(account2_addr, Some(account2));

    // Add deleted account
    let deleted_account_addr = B256::repeat_byte(0x30);
    post_state.accounts.insert(deleted_account_addr, None);

    // Add storage for an address
    let storage_addr = B256::repeat_byte(0x50);
    let mut hashed_storage = HashedStorage::new(false);
    hashed_storage.storage.insert(B256::repeat_byte(0x01), U256::from(111));
    hashed_storage.storage.insert(B256::repeat_byte(0x02), U256::from(222));
    hashed_storage.storage.insert(B256::repeat_byte(0x03), U256::ZERO); // Deleted storage
    post_state.storages.insert(storage_addr, hashed_storage);

    let block_state_diff = BlockStateDiff { trie_updates, post_state };

    // Store the updates
    storage.store_trie_updates(block_ref, block_state_diff).await?;

    // ========== Verify Account Branch Nodes ==========
    let mut account_trie_cursor = storage.account_trie_cursor(block_ref.block.number + 10)?;

    // Should find the added branches
    let result1 = account_trie_cursor.seek_exact(account_path1)?;
    assert!(result1.is_some(), "Account branch node 1 should be found");
    assert_eq!(result1.unwrap().0, account_path1);

    let result2 = account_trie_cursor.seek_exact(account_path2)?;
    assert!(result2.is_some(), "Account branch node 2 should be found");
    assert_eq!(result2.unwrap().0, account_path2);

    // Removed node should not be found
    let removed_result = account_trie_cursor.seek_exact(removed_account_path)?;
    assert!(removed_result.is_none(), "Removed account node should not be found");

    // ========== Verify Storage Branch Nodes ==========
    let mut storage_trie_cursor =
        storage.storage_trie_cursor(hashed_address, block_ref.block.number + 10)?;

    let storage_result1 = storage_trie_cursor.seek_exact(storage_path1)?;
    assert!(storage_result1.is_some(), "Storage branch node 1 should be found");

    let storage_result2 = storage_trie_cursor.seek_exact(storage_path2)?;
    assert!(storage_result2.is_some(), "Storage branch node 2 should be found");

    // Removed storage node should not be found
    let removed_storage_result = storage_trie_cursor.seek_exact(removed_storage_path)?;
    assert!(removed_storage_result.is_none(), "Removed storage node should not be found");

    // ========== Verify Account Leaves ==========
    let mut account_cursor = storage.account_hashed_cursor(block_ref.block.number + 10)?;

    let acc1_result = account_cursor.seek(account1_addr)?;
    assert!(acc1_result.is_some(), "Account 1 should be found");
    assert_eq!(acc1_result.unwrap().0, account1_addr);
    assert_eq!(acc1_result.unwrap().1.nonce, 1);
    assert_eq!(acc1_result.unwrap().1.balance, U256::from(1000));

    let acc2_result = account_cursor.seek(account2_addr)?;
    assert!(acc2_result.is_some(), "Account 2 should be found");
    assert_eq!(acc2_result.unwrap().1.nonce, 2);

    // Deleted account should not be found
    let deleted_acc_result = account_cursor.seek(deleted_account_addr)?;
    assert!(
        deleted_acc_result.is_none() || deleted_acc_result.unwrap().0 != deleted_account_addr,
        "Deleted account should not be found"
    );

    // ========== Verify Storage Leaves ==========
    let mut storage_cursor =
        storage.storage_hashed_cursor(storage_addr, block_ref.block.number + 10)?;

    let slot1_result = storage_cursor.seek(B256::repeat_byte(0x01))?;
    assert!(slot1_result.is_some(), "Storage slot 1 should be found");
    assert_eq!(slot1_result.unwrap().1, U256::from(111));

    let slot2_result = storage_cursor.seek(B256::repeat_byte(0x02))?;
    assert!(slot2_result.is_some(), "Storage slot 2 should be found");
    assert_eq!(slot2_result.unwrap().1, U256::from(222));

    // Zero-valued storage should not be found (deleted)
    let slot3_result = storage_cursor.seek(B256::repeat_byte(0x03))?;
    assert!(
        slot3_result.is_none() || slot3_result.unwrap().0 != B256::repeat_byte(0x03),
        "Zero-valued storage slot should not be found"
    );

    // ========== Verify fetch_trie_updates can retrieve the data ==========
    let fetched_diff = storage.fetch_trie_updates(block_ref.block.number).await?;

    // Check that trie updates are stored
    assert_eq!(
        fetched_diff.trie_updates.account_nodes_ref().len(),
        2,
        "Should have 2 account nodes"
    );
    assert_eq!(
        fetched_diff.trie_updates.storage_tries_ref().len(),
        1,
        "Should have 1 storage trie"
    );

    // Check that post state is stored
    assert_eq!(
        fetched_diff.post_state.accounts.len(),
        3,
        "Should have 3 accounts (including deleted)"
    );
    assert_eq!(fetched_diff.post_state.storages.len(), 1, "Should have 1 storage entry");

    Ok(())
}

/// Test that `replace_updates` properly applies hashed/trie storage updates to the DB
///
/// This test verifies the bug fix where `replace_updates` was only storing `trie_updates`
/// and `post_states` directly without populating the internal data structures
/// (`hashed_accounts`, `hashed_storages`, `account_branches`, `storage_branches`).
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_replace_updates_applies_all_updates<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    use reth_trie::{updates::StorageTrieUpdates, HashedStorage};

    let block_ref_50 = BlockWithParent::new(B256::ZERO, NumHash::new(50, B256::repeat_byte(0x96)));

    // ========== Setup: Store initial state at blocks 50, 100, 101 ==========
    let initial_account_addr = B256::repeat_byte(0x10);
    let initial_account = create_test_account_with_values(1, 1000, 0xAA);

    let initial_storage_addr = B256::repeat_byte(0x20);
    let initial_storage_slot = B256::repeat_byte(0x01);
    let initial_storage_value = U256::from(100);

    let initial_branch_path = nibbles_from(vec![1, 2, 3]);
    let initial_branch = create_test_branch();

    // Store initial data at block 50
    let mut initial_trie_updates_50 = TrieUpdates::default();
    initial_trie_updates_50.account_nodes.insert(initial_branch_path, initial_branch.clone());

    let mut initial_post_state_50 = HashedPostState::default();
    initial_post_state_50.accounts.insert(initial_account_addr, Some(initial_account));

    let initial_diff_50 =
        BlockStateDiff { trie_updates: initial_trie_updates_50, post_state: initial_post_state_50 };
    storage.store_trie_updates(block_ref_50, initial_diff_50).await?;

    // Store data at block 100 (common block)
    let mut initial_trie_updates_100 = TrieUpdates::default();
    let common_branch_path = nibbles_from(vec![4, 5, 6]);
    initial_trie_updates_100.account_nodes.insert(common_branch_path, initial_branch.clone());

    let mut initial_post_state_100 = HashedPostState::default();
    let mut initial_storage_100 = HashedStorage::new(false);
    initial_storage_100.storage.insert(initial_storage_slot, initial_storage_value);
    initial_post_state_100.storages.insert(initial_storage_addr, initial_storage_100);

    let initial_diff_100 = BlockStateDiff {
        trie_updates: initial_trie_updates_100,
        post_state: initial_post_state_100,
    };

    let block_ref_100 =
        BlockWithParent::new(block_ref_50.block.hash, NumHash::new(100, B256::repeat_byte(0x97)));

    storage.store_trie_updates(block_ref_100, initial_diff_100).await?;

    // Store data at block 101 (will be replaced)
    let mut initial_trie_updates_101 = TrieUpdates::default();
    let old_branch_path = nibbles_from(vec![7, 8, 9]);
    initial_trie_updates_101.account_nodes.insert(old_branch_path, initial_branch.clone());

    let mut initial_post_state_101 = HashedPostState::default();
    let old_account_addr = B256::repeat_byte(0x30);
    let old_account = create_test_account_with_values(99, 9999, 0xFF);
    initial_post_state_101.accounts.insert(old_account_addr, Some(old_account));

    let initial_diff_101 = BlockStateDiff {
        trie_updates: initial_trie_updates_101,
        post_state: initial_post_state_101,
    };
    let block_ref_101 =
        BlockWithParent::new(block_ref_100.block.hash, NumHash::new(101, B256::repeat_byte(0x98)));
    storage.store_trie_updates(block_ref_101, initial_diff_101).await?;

    let block_ref_102 =
        BlockWithParent::new(block_ref_101.block.hash, NumHash::new(102, B256::repeat_byte(0x99)));

    // ========== Verify initial state exists ==========
    // Verify block 50 data exists
    let mut cursor_initial = storage.account_trie_cursor(75)?;
    assert!(
        cursor_initial.seek_exact(initial_branch_path)?.is_some(),
        "Initial branch should exist before replace"
    );

    // Verify block 101 old data exists
    let mut cursor_old = storage.account_trie_cursor(150)?;
    assert!(
        cursor_old.seek_exact(old_branch_path)?.is_some(),
        "Old branch at block 101 should exist before replace"
    );

    let mut account_cursor_old = storage.account_hashed_cursor(150)?;
    assert!(
        account_cursor_old.seek(old_account_addr)?.is_some(),
        "Old account at block 101 should exist before replace"
    );

    // ========== Call replace_updates to replace blocks after 100 ==========
    let mut blocks_to_add: HashMap<BlockWithParent, BlockStateDiff> = HashMap::default();

    // New data for block 101
    let new_account_addr = B256::repeat_byte(0x40);
    let new_account = create_test_account_with_values(5, 5000, 0xCC);

    let new_storage_addr = B256::repeat_byte(0x50);
    let new_storage_slot = B256::repeat_byte(0x02);
    let new_storage_value = U256::from(999);

    let new_branch_path = nibbles_from(vec![10, 11, 12]);
    let new_branch = create_test_branch_variant();

    let storage_branch_path = nibbles_from(vec![5, 5]);
    let storage_hashed_addr = B256::repeat_byte(0x60);

    let mut new_trie_updates = TrieUpdates::default();
    new_trie_updates.account_nodes.insert(new_branch_path, new_branch.clone());

    // Add storage trie updates
    let mut storage_trie = StorageTrieUpdates::default();
    storage_trie.storage_nodes.insert(storage_branch_path, new_branch.clone());
    new_trie_updates.insert_storage_updates(storage_hashed_addr, storage_trie);

    let mut new_post_state = HashedPostState::default();
    new_post_state.accounts.insert(new_account_addr, Some(new_account));

    let mut new_storage = HashedStorage::new(false);
    new_storage.storage.insert(new_storage_slot, new_storage_value);
    new_post_state.storages.insert(new_storage_addr, new_storage);

    blocks_to_add.insert(
        block_ref_101,
        BlockStateDiff { trie_updates: new_trie_updates, post_state: new_post_state },
    );

    // New data for block 102
    let block_102_account_addr = B256::repeat_byte(0x70);
    let block_102_account = create_test_account_with_values(10, 10000, 0xDD);

    let mut trie_updates_102 = TrieUpdates::default();
    let block_102_branch_path = nibbles_from(vec![15, 14, 13]);
    trie_updates_102.account_nodes.insert(block_102_branch_path, new_branch.clone());

    let mut post_state_102 = HashedPostState::default();
    post_state_102.accounts.insert(block_102_account_addr, Some(block_102_account));

    blocks_to_add.insert(
        block_ref_102,
        BlockStateDiff { trie_updates: trie_updates_102, post_state: post_state_102 },
    );

    // Execute replace_updates
    storage.replace_updates(100, blocks_to_add).await?;

    // ========== Verify that data up to block 100 still exists ==========
    let mut cursor_50 = storage.account_trie_cursor(75)?;
    assert!(
        cursor_50.seek_exact(initial_branch_path)?.is_some(),
        "Block 50 branch should still exist after replace"
    );

    let mut cursor_100 = storage.account_trie_cursor(100)?;
    assert!(
        cursor_100.seek_exact(common_branch_path)?.is_some(),
        "Block 100 branch should still exist after replace"
    );

    let mut storage_cursor_100 = storage.storage_hashed_cursor(initial_storage_addr, 100)?;
    let result_100 = storage_cursor_100.seek(initial_storage_slot)?;
    assert!(result_100.is_some(), "Block 100 storage should still exist after replace");
    assert_eq!(
        result_100.unwrap().1,
        initial_storage_value,
        "Block 100 storage value should be unchanged"
    );

    // ========== Verify that old data after block 100 is gone ==========
    let mut cursor_old_gone = storage.account_trie_cursor(150)?;
    assert!(
        cursor_old_gone.seek_exact(old_branch_path)?.is_none(),
        "Old branch at block 101 should be removed after replace"
    );

    let mut account_cursor_old_gone = storage.account_hashed_cursor(150)?;
    let old_acc_result = account_cursor_old_gone.seek(old_account_addr)?;
    assert!(
        old_acc_result.is_none() || old_acc_result.unwrap().0 != old_account_addr,
        "Old account at block 101 should be removed after replace"
    );

    // ========== Verify new data is properly accessible via cursors ==========

    // Verify new account branch nodes
    let mut trie_cursor = storage.account_trie_cursor(150)?;
    let branch_result = trie_cursor.seek_exact(new_branch_path)?;
    assert!(branch_result.is_some(), "New account branch should be accessible via cursor");
    assert_eq!(branch_result.unwrap().0, new_branch_path);

    // Verify new storage branch nodes
    let mut storage_trie_cursor = storage.storage_trie_cursor(storage_hashed_addr, 150)?;
    let storage_branch_result = storage_trie_cursor.seek_exact(storage_branch_path)?;
    assert!(storage_branch_result.is_some(), "New storage branch should be accessible via cursor");
    assert_eq!(storage_branch_result.unwrap().0, storage_branch_path);

    // Verify new hashed accounts
    let mut account_cursor = storage.account_hashed_cursor(150)?;
    let account_result = account_cursor.seek(new_account_addr)?;
    assert!(account_result.is_some(), "New account should be accessible via cursor");
    assert_eq!(account_result.as_ref().unwrap().0, new_account_addr);
    assert_eq!(account_result.as_ref().unwrap().1.nonce, new_account.nonce);
    assert_eq!(account_result.as_ref().unwrap().1.balance, new_account.balance);
    assert_eq!(account_result.as_ref().unwrap().1.bytecode_hash, new_account.bytecode_hash);

    // Verify new hashed storages
    let mut storage_cursor = storage.storage_hashed_cursor(new_storage_addr, 150)?;
    let storage_result = storage_cursor.seek(new_storage_slot)?;
    assert!(storage_result.is_some(), "New storage should be accessible via cursor");
    assert_eq!(storage_result.as_ref().unwrap().0, new_storage_slot);
    assert_eq!(storage_result.as_ref().unwrap().1, new_storage_value);

    // Verify block 102 data
    let mut trie_cursor_102 = storage.account_trie_cursor(150)?;
    let branch_result_102 = trie_cursor_102.seek_exact(block_102_branch_path)?;
    assert!(branch_result_102.is_some(), "Block 102 branch should be accessible");
    assert_eq!(branch_result_102.unwrap().0, block_102_branch_path);

    let mut account_cursor_102 = storage.account_hashed_cursor(150)?;
    let account_result_102 = account_cursor_102.seek(block_102_account_addr)?;
    assert!(account_result_102.is_some(), "Block 102 account should be accessible");
    assert_eq!(account_result_102.as_ref().unwrap().0, block_102_account_addr);
    assert_eq!(account_result_102.as_ref().unwrap().1.nonce, block_102_account.nonce);

    // Verify fetch_trie_updates returns the new data
    let fetched_101 = storage.fetch_trie_updates(101).await?;
    assert_eq!(
        fetched_101.trie_updates.account_nodes_ref().len(),
        1,
        "Should have 1 account branch node at block 101"
    );
    assert!(
        fetched_101.trie_updates.account_nodes_ref().contains_key(&new_branch_path),
        "New branch path should be in trie_updates"
    );
    assert_eq!(fetched_101.post_state.accounts.len(), 1, "Should have 1 account at block 101");
    assert!(
        fetched_101.post_state.accounts.contains_key(&new_account_addr),
        "New account should be in post_state"
    );

    Ok(())
}

/// Test that pure deletions (nodes only in `removed_nodes`) are properly stored
///
/// This test verifies that when a node appears only in `removed_nodes` (not in updates),
/// it is properly stored as a deletion and subsequent queries return None for that path.
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_pure_deletions_stored_correctly<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    use reth_trie::updates::StorageTrieUpdates;

    // ========== Setup: Store initial branch nodes at block 50 ==========
    let account_path1 = nibbles_from(vec![1, 2, 3]);
    let account_path2 = nibbles_from(vec![4, 5, 6]);
    let storage_path1 = nibbles_from(vec![7, 8, 9]);
    let storage_path2 = nibbles_from(vec![10, 11, 12]);
    let storage_address = B256::repeat_byte(0x42);

    let initial_branch = create_test_branch();

    let mut initial_trie_updates = TrieUpdates::default();
    initial_trie_updates.account_nodes.insert(account_path1, initial_branch.clone());
    initial_trie_updates.account_nodes.insert(account_path2, initial_branch.clone());

    let mut storage_trie = StorageTrieUpdates::default();
    storage_trie.storage_nodes.insert(storage_path1, initial_branch.clone());
    storage_trie.storage_nodes.insert(storage_path2, initial_branch.clone());
    initial_trie_updates.insert_storage_updates(storage_address, storage_trie);

    let initial_diff = BlockStateDiff {
        trie_updates: initial_trie_updates,
        post_state: HashedPostState::default(),
    };

    let block_ref_50 = BlockWithParent::new(B256::ZERO, NumHash::new(50, B256::repeat_byte(0x96)));

    storage.store_trie_updates(block_ref_50, initial_diff).await?;

    // Verify initial state exists at block 75
    let mut cursor_75 = storage.account_trie_cursor(75)?;
    assert!(
        cursor_75.seek_exact(account_path1)?.is_some(),
        "Initial account branch 1 should exist at block 75"
    );
    assert!(
        cursor_75.seek_exact(account_path2)?.is_some(),
        "Initial account branch 2 should exist at block 75"
    );

    let mut storage_cursor_75 = storage.storage_trie_cursor(storage_address, 75)?;
    assert!(
        storage_cursor_75.seek_exact(storage_path1)?.is_some(),
        "Initial storage branch 1 should exist at block 75"
    );
    assert!(
        storage_cursor_75.seek_exact(storage_path2)?.is_some(),
        "Initial storage branch 2 should exist at block 75"
    );

    // ========== At block 100: Mark paths as deleted (ONLY in removed_nodes) ==========
    let mut deletion_trie_updates = TrieUpdates::default();

    // Add to removed_nodes ONLY (no updates)
    deletion_trie_updates.removed_nodes.insert(account_path1);

    // Do the same for storage branch
    let mut deletion_storage_trie = StorageTrieUpdates::default();
    deletion_storage_trie.removed_nodes.insert(storage_path1);
    deletion_trie_updates.insert_storage_updates(storage_address, deletion_storage_trie);

    let deletion_diff = BlockStateDiff {
        trie_updates: deletion_trie_updates,
        post_state: HashedPostState::default(),
    };

    let block_ref_100 =
        BlockWithParent::new(B256::repeat_byte(0x96), NumHash::new(100, B256::repeat_byte(0x97)));

    storage.store_trie_updates(block_ref_100, deletion_diff).await?;

    // ========== Verify that deleted nodes return None at block 150 ==========

    // Deleted account branch should not be found
    let mut cursor_150 = storage.account_trie_cursor(150)?;
    let account_result = cursor_150.seek_exact(account_path1)?;
    assert!(account_result.is_none(), "Deleted account branch should return None at block 150");

    // Non-deleted account branch should still exist
    let account_result2 = cursor_150.seek_exact(account_path2)?;
    assert!(
        account_result2.is_some(),
        "Non-deleted account branch should still exist at block 150"
    );

    // Deleted storage branch should not be found
    let mut storage_cursor_150 = storage.storage_trie_cursor(storage_address, 150)?;
    let storage_result = storage_cursor_150.seek_exact(storage_path1)?;
    assert!(storage_result.is_none(), "Deleted storage branch should return None at block 150");

    // Non-deleted storage branch should still exist
    let storage_result2 = storage_cursor_150.seek_exact(storage_path2)?;
    assert!(
        storage_result2.is_some(),
        "Non-deleted storage branch should still exist at block 150"
    );

    // ========== Verify that the nodes still exist at block 75 (before deletion) ==========
    let mut cursor_75_after = storage.account_trie_cursor(75)?;
    assert!(
        cursor_75_after.seek_exact(account_path1)?.is_some(),
        "Deleted node should still exist at block 75 (before deletion)"
    );

    let mut storage_cursor_75_after = storage.storage_trie_cursor(storage_address, 75)?;
    assert!(
        storage_cursor_75_after.seek_exact(storage_path1)?.is_some(),
        "Deleted storage node should still exist at block 75 (before deletion)"
    );

    // ========== Verify iteration skips deleted nodes ==========
    let mut cursor_iter = storage.account_trie_cursor(150)?;
    let mut found_paths = Vec::new();
    while let Some((path, _)) = cursor_iter.next()? {
        found_paths.push(path);
    }

    assert!(!found_paths.contains(&account_path1), "Iteration should skip deleted node");
    assert!(found_paths.contains(&account_path2), "Iteration should include non-deleted node");

    Ok(())
}

/// Test that updates take precedence over removals when both are present
///
/// This test verifies that when a path appears in both `removed_nodes` and `account_nodes`,
/// the update from `account_nodes` takes precedence. This is critical for correctness
/// when processing trie updates that both remove and update the same node.
#[test_case(InMemoryProofsStorage::new(); "InMemory")]
#[test_case(create_mdbx_proofs_storage(); "Mdbx")]
#[tokio::test]
#[serial]
async fn test_updates_take_precedence_over_removals<S: OpProofsStore>(
    storage: S,
) -> Result<(), OpProofsStorageError> {
    use reth_trie::updates::StorageTrieUpdates;

    // ========== Setup: Store initial branch nodes at block 50 ==========
    let account_path = nibbles_from(vec![1, 2, 3]);
    let storage_path = nibbles_from(vec![4, 5, 6]);
    let storage_address = B256::repeat_byte(0x42);

    let initial_branch = create_test_branch();

    let mut initial_trie_updates = TrieUpdates::default();
    initial_trie_updates.account_nodes.insert(account_path, initial_branch.clone());

    let mut storage_trie = StorageTrieUpdates::default();
    storage_trie.storage_nodes.insert(storage_path, initial_branch.clone());
    initial_trie_updates.insert_storage_updates(storage_address, storage_trie);

    let initial_diff = BlockStateDiff {
        trie_updates: initial_trie_updates,
        post_state: HashedPostState::default(),
    };

    let block_ref_50 = BlockWithParent::new(B256::ZERO, NumHash::new(50, B256::repeat_byte(0x96)));

    storage.store_trie_updates(block_ref_50, initial_diff).await?;

    // Verify initial state exists at block 75
    let mut cursor_75 = storage.account_trie_cursor(75)?;
    assert!(
        cursor_75.seek_exact(account_path)?.is_some(),
        "Initial account branch should exist at block 75"
    );

    let mut storage_cursor_75 = storage.storage_trie_cursor(storage_address, 75)?;
    assert!(
        storage_cursor_75.seek_exact(storage_path)?.is_some(),
        "Initial storage branch should exist at block 75"
    );

    // ========== At block 100: Add paths to BOTH removed_nodes AND account_nodes ==========
    // This simulates a scenario where a node is both removed and updated
    // The update should take precedence
    let updated_branch = create_test_branch_variant();

    let mut conflicting_trie_updates = TrieUpdates::default();

    // Add to removed_nodes
    conflicting_trie_updates.removed_nodes.insert(account_path);

    // Also add to account_nodes (this should take precedence)
    conflicting_trie_updates.account_nodes.insert(account_path, updated_branch.clone());

    // Do the same for storage branch
    let mut conflicting_storage_trie = StorageTrieUpdates::default();
    conflicting_storage_trie.removed_nodes.insert(storage_path);
    conflicting_storage_trie.storage_nodes.insert(storage_path, updated_branch.clone());
    conflicting_trie_updates.insert_storage_updates(storage_address, conflicting_storage_trie);

    let conflicting_diff = BlockStateDiff {
        trie_updates: conflicting_trie_updates,
        post_state: HashedPostState::default(),
    };

    let block_ref_100 =
        BlockWithParent::new(B256::repeat_byte(0x96), NumHash::new(100, B256::repeat_byte(0x97)));

    storage.store_trie_updates(block_ref_100, conflicting_diff).await?;

    // ========== Verify that updates took precedence at block 150 ==========

    // Account branch should exist (not deleted) with the updated value
    let mut cursor_150 = storage.account_trie_cursor(150)?;
    let account_result = cursor_150.seek_exact(account_path)?;
    assert!(
        account_result.is_some(),
        "Account branch should exist at block 150 (update should take precedence over removal)"
    );
    let (found_path, found_branch) = account_result.unwrap();
    assert_eq!(found_path, account_path);
    // Verify it's the updated branch, not the initial one
    assert_eq!(
        found_branch.state_mask, updated_branch.state_mask,
        "Account branch should be the updated version, not the initial one"
    );

    // Storage branch should exist (not deleted) with the updated value
    let mut storage_cursor_150 = storage.storage_trie_cursor(storage_address, 150)?;
    let storage_result = storage_cursor_150.seek_exact(storage_path)?;
    assert!(
        storage_result.is_some(),
        "Storage branch should exist at block 150 (update should take precedence over removal)"
    );
    let (found_storage_path, found_storage_branch) = storage_result.unwrap();
    assert_eq!(found_storage_path, storage_path);
    // Verify it's the updated branch
    assert_eq!(
        found_storage_branch.state_mask, updated_branch.state_mask,
        "Storage branch should be the updated version, not the initial one"
    );

    // ========== Verify that the old version still exists at block 75 ==========
    let mut cursor_75_after = storage.account_trie_cursor(75)?;
    let result_75 = cursor_75_after.seek_exact(account_path)?;
    assert!(result_75.is_some(), "Initial version should still exist at block 75");
    let (_, branch_75) = result_75.unwrap();
    assert_eq!(
        branch_75.state_mask, initial_branch.state_mask,
        "Block 75 should see the initial branch, not the updated one"
    );

    Ok(())
}
