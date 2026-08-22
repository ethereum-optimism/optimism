//! Byte sink and cursor used by the store record encodings.
//!
//! Every record here is fixed-width fields in a known order, big-endian for integers so that
//! keys sort chronologically. These two helpers keep the offset arithmetic in one place and turn
//! every short read into [`StoreError::DataCorruption`] naming the record.

use crate::error::StoreError;
use alloy_primitives::{B256, U256};

/// Appends fixed-width fields to a byte buffer.
#[derive(Debug, Default)]
pub(crate) struct Sink {
    buf: Vec<u8>,
}

impl Sink {
    /// Returns a sink with room for `capacity` bytes.
    pub(crate) fn with_capacity(capacity: usize) -> Self {
        Self { buf: Vec::with_capacity(capacity) }
    }

    pub(crate) fn put_u8(&mut self, value: u8) {
        self.buf.push(value);
    }

    pub(crate) fn put_u32(&mut self, value: u32) {
        self.buf.extend_from_slice(&value.to_be_bytes());
    }

    pub(crate) fn put_u64(&mut self, value: u64) {
        self.buf.extend_from_slice(&value.to_be_bytes());
    }

    pub(crate) fn put_u256(&mut self, value: U256) {
        self.buf.extend_from_slice(&value.to_be_bytes::<32>());
    }

    pub(crate) fn put_b256(&mut self, value: B256) {
        self.buf.extend_from_slice(value.as_slice());
    }

    /// Returns the accumulated bytes.
    pub(crate) fn into_vec(self) -> Vec<u8> {
        self.buf
    }
}

/// Reads fixed-width fields from a byte slice, in the order they were written.
#[derive(Debug)]
pub(crate) struct Cursor<'a> {
    buf: &'a [u8],
    offset: usize,
    record: &'static str,
}

impl<'a> Cursor<'a> {
    /// Returns a cursor over `buf`. `record` names the record in any corruption error.
    pub(crate) const fn new(buf: &'a [u8], record: &'static str) -> Self {
        Self { buf, offset: 0, record }
    }

    fn take(&mut self, len: usize) -> Result<&'a [u8], StoreError> {
        let end = self.offset.checked_add(len).ok_or(StoreError::DataCorruption(self.record))?;
        let slice =
            self.buf.get(self.offset..end).ok_or(StoreError::DataCorruption(self.record))?;
        self.offset = end;
        Ok(slice)
    }

    pub(crate) fn take_u8(&mut self) -> Result<u8, StoreError> {
        Ok(self.take(1)?[0])
    }

    pub(crate) fn take_u32(&mut self) -> Result<u32, StoreError> {
        let bytes: [u8; 4] =
            self.take(4)?.try_into().map_err(|_| StoreError::DataCorruption(self.record))?;
        Ok(u32::from_be_bytes(bytes))
    }

    pub(crate) fn take_u64(&mut self) -> Result<u64, StoreError> {
        let bytes: [u8; 8] =
            self.take(8)?.try_into().map_err(|_| StoreError::DataCorruption(self.record))?;
        Ok(u64::from_be_bytes(bytes))
    }

    pub(crate) fn take_u256(&mut self) -> Result<U256, StoreError> {
        Ok(U256::from_be_slice(self.take(32)?))
    }

    pub(crate) fn take_b256(&mut self) -> Result<B256, StoreError> {
        Ok(B256::from_slice(self.take(32)?))
    }

    /// Asserts the record was consumed exactly. Trailing bytes mean the declared field counts
    /// disagree with the stored length, which is corruption, not a longer record.
    pub(crate) const fn finish(self) -> Result<(), StoreError> {
        if self.offset == self.buf.len() {
            Ok(())
        } else {
            Err(StoreError::DataCorruption(self.record))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fields_round_trip_in_order() {
        let mut sink = Sink::with_capacity(0);
        sink.put_u8(1);
        sink.put_u32(2);
        sink.put_u64(3);
        sink.put_u256(U256::from(4));
        sink.put_b256(B256::repeat_byte(5));
        let encoded = sink.into_vec();
        assert_eq!(encoded.len(), 1 + 4 + 8 + 32 + 32);

        let mut cursor = Cursor::new(&encoded, "test");
        assert_eq!(cursor.take_u8().unwrap(), 1);
        assert_eq!(cursor.take_u32().unwrap(), 2);
        assert_eq!(cursor.take_u64().unwrap(), 3);
        assert_eq!(cursor.take_u256().unwrap(), U256::from(4));
        assert_eq!(cursor.take_b256().unwrap(), B256::repeat_byte(5));
        cursor.finish().unwrap();
    }

    #[test]
    fn a_short_read_names_the_record() {
        let mut cursor = Cursor::new(&[0u8; 3], "widget");
        assert!(matches!(cursor.take_u64(), Err(StoreError::DataCorruption("widget"))));
    }

    #[test]
    fn trailing_bytes_are_corruption() {
        let cursor = Cursor::new(&[0u8; 3], "widget");
        assert!(matches!(cursor.finish(), Err(StoreError::DataCorruption("widget"))));
    }
}
