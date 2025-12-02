//! Transaction Types

use crate::Frame;
use alloc::vec::Vec;
use alloy_primitives::Bytes;

/// BatchTransaction is a set of [`Frame`]s that can be [Into::into] [`Bytes`].
/// if the size exceeds the desired threshold.
#[derive(Debug, Clone)]
pub struct BatchTransaction {
    /// The frames in the batch.
    pub frames: Vec<Frame>,
    /// The size of the potential transaction.
    pub size: usize,
}

impl BatchTransaction {
    /// Returns the size of the transaction.
    pub const fn size(&self) -> usize {
        self.size
    }

    /// Returns if the transaction has reached the max frame count.
    pub const fn is_full(&self, max_frames: u16) -> bool {
        self.frames.len() as u16 >= max_frames
    }

    /// Returns the [`BatchTransaction`] as a [`Bytes`].
    pub fn to_bytes(&self) -> Bytes {
        self.frames
            .iter()
            .fold(Vec::new(), |mut acc, frame| {
                acc.append(&mut frame.encode());
                acc
            })
            .into()
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_batch_transaction() {
        let frame = Frame { id: [0xFF; 16], number: 0xEE, data: vec![0xDD; 50], is_last: true };
        let batch = BatchTransaction { frames: vec![frame.clone(); 5], size: 5 * frame.size() };
        let bytes: Bytes = batch.to_bytes();
        let bytes =
            [crate::DERIVATION_VERSION_0].iter().chain(bytes.iter()).copied().collect::<Vec<_>>();
        let frames = Frame::parse_frames(&bytes).unwrap();
        assert_eq!(frames, vec![frame; 5]);
    }
}
