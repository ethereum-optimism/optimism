//! Frames

use crate::{
    FrameQueue, NextFrameProvider, OriginProvider, PipelineError, PipelineErrorKind,
    test_utils::TestFrameQueueProvider,
};
use alloc::{sync::Arc, vec, vec::Vec};
use alloy_primitives::Bytes;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, DERIVATION_VERSION_0, Frame};

/// A [`FrameQueue`] builder.
#[derive(Debug, Default)]
pub struct FrameQueueBuilder {
    origin: Option<BlockInfo>,
    config: Option<RollupConfig>,
    mock: Option<TestFrameQueueProvider>,
    expected_frames: Vec<Frame>,
    expected_err: Option<PipelineErrorKind>,
}

fn encode_frames(frames: &[Frame]) -> Bytes {
    let mut bytes = Vec::new();
    bytes.extend_from_slice(&[DERIVATION_VERSION_0]);
    for frame in frames.iter() {
        bytes.extend_from_slice(&frame.encode());
    }
    Bytes::from(bytes)
}

impl FrameQueueBuilder {
    /// Create a new [`FrameQueueBuilder`] instance.
    pub const fn new() -> Self {
        Self { origin: None, config: None, mock: None, expected_frames: vec![], expected_err: None }
    }

    /// Sets the rollup config.
    pub fn with_rollup_config(mut self, config: &RollupConfig) -> Self {
        self.config = Some(config.clone());
        self
    }

    /// Set the origin block.
    pub const fn with_origin(mut self, origin: BlockInfo) -> Self {
        self.origin = Some(origin);
        self
    }

    /// With expected frames.
    pub fn with_expected_frames(mut self, frames: &[Frame]) -> Self {
        self.expected_frames = frames.to_vec();
        self
    }

    /// Sets the expected error type.
    pub fn with_expected_err(mut self, err: PipelineErrorKind) -> Self {
        self.expected_err = Some(err);
        self
    }

    /// With raw frames.
    pub fn with_raw_frames(mut self, raw: Bytes) -> Self {
        let mock = self.mock.unwrap_or_else(|| TestFrameQueueProvider::new(vec![Ok(raw)]));
        self.mock = Some(mock);
        self
    }

    /// Adds frames to the mock provider.
    pub fn with_frames(mut self, frames: &[Frame]) -> Self {
        let encoded = encode_frames(frames);
        let mock = self.mock.unwrap_or_else(|| TestFrameQueueProvider::new(vec![Ok(encoded)]));
        self.mock = Some(mock);
        self
    }

    /// Build the [`FrameQueue`].
    pub fn build(self) -> FrameQueueAsserter {
        let mut mock = self.mock.unwrap_or_else(|| TestFrameQueueProvider::new(vec![]));
        if let Some(origin) = self.origin {
            mock.set_origin(origin);
        }
        let config = self.config.unwrap_or_default();
        let config = Arc::new(config);
        let err = self.expected_err.unwrap_or_else(|| PipelineError::Eof.temp());
        FrameQueueAsserter::new(FrameQueue::new(mock, config), self.expected_frames, err)
    }
}

/// The [`FrameQueueAsserter`] validates frame queue outputs.
#[derive(Debug)]
pub struct FrameQueueAsserter {
    inner: FrameQueue<TestFrameQueueProvider>,
    expected_frames: Vec<Frame>,
    expected_err: PipelineErrorKind,
}

impl FrameQueueAsserter {
    /// Create a new [`FrameQueueAsserter`] instance.
    pub const fn new(
        inner: FrameQueue<TestFrameQueueProvider>,
        expected_frames: Vec<Frame>,
        expected_err: PipelineErrorKind,
    ) -> Self {
        Self { inner, expected_frames, expected_err }
    }

    /// Asserts that holocene is active.
    pub fn holocene_active(&self, active: bool) {
        let holocene = self.inner.is_holocene_active(self.inner.origin().unwrap_or_default());
        if !active {
            assert!(!holocene);
        } else {
            assert!(holocene);
        }
    }

    /// Asserts that the frame queue returns with a missing origin error.
    pub async fn missing_origin(mut self) {
        let err = self.inner.next_frame().await.unwrap_err();
        assert_eq!(err, PipelineError::MissingOrigin.crit());
    }

    /// Asserts that the frame queue produces the expected frames.
    pub async fn next_frames(mut self) {
        for eframe in self.expected_frames.into_iter() {
            let frame = self.inner.next_frame().await.expect("unexpected frame");
            assert_eq!(frame, eframe);
        }
        let err = self.inner.next_frame().await.unwrap_err();
        assert_eq!(err, self.expected_err);
    }
}
