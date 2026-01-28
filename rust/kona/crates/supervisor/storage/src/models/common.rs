//! Common model types used across various storage tables.

use derive_more::{Deref, DerefMut};
use reth_codecs::Compact;
use serde::{Deserialize, Serialize};

/// Wrapper for `Vec<u64>` to represent a list of numbers.
// todo: add support for Vec<64> in table
#[derive(
    Deref, DerefMut, Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize, Compact,
)]
pub struct U64List(pub Vec<u64>);

#[cfg(test)]
mod tests {
    use super::*;
    use reth_codecs::Compact;

    #[test]
    fn test_u64list_compact_empty() {
        let original_list = U64List(Vec::new());

        let mut buffer = Vec::new();
        let bytes_written = original_list.to_compact(&mut buffer);

        assert_eq!(
            bytes_written,
            buffer.len(),
            "Bytes written should match buffer length for empty list"
        );
        let (deserialized_list, remaining_buf) = U64List::from_compact(&buffer, bytes_written);

        assert_eq!(
            original_list, deserialized_list,
            "Original and deserialized empty lists should be equal"
        );
        assert!(
            remaining_buf.is_empty(),
            "Remaining buffer should be empty after deserialization of empty list"
        );
    }

    #[test]
    fn test_u64list_compact_with_data() {
        let original_list = U64List(vec![10, 20, 30, 40, 50]);

        let mut buffer = Vec::new();
        let bytes_written = original_list.to_compact(&mut buffer);

        assert_eq!(
            bytes_written,
            buffer.len(),
            "Bytes written should match buffer length for list with data"
        );
        let (deserialized_list, remaining_buf) = U64List::from_compact(&buffer, bytes_written);

        assert_eq!(
            original_list, deserialized_list,
            "Original and deserialized lists with data should be equal"
        );
        assert!(
            remaining_buf.is_empty(),
            "Remaining buffer should be empty after deserialization of list with data"
        );
    }

    #[test]
    fn test_u64list_deref() {
        let list = U64List(vec![1, 2, 3]);
        assert_eq!(list.len(), 3);
        assert_eq!(list[0], 1);
        assert!(!list.is_empty());
    }

    #[test]
    fn test_u64list_deref_mut() {
        let mut list = U64List(vec![1, 2, 3]);
        list.push(4);
        assert_eq!(list.0, vec![1, 2, 3, 4]);

        list.sort();
        assert_eq!(list.0, vec![1, 2, 3, 4]);

        list.clear();
        assert!(list.is_empty());
    }
}
