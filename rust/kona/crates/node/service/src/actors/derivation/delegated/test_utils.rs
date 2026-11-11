//! A scripted derivation delegate and sync status builder shared by the delegated actor's tests.

use super::{DerivationDelegateClientError, DerivationDelegateProvider};
use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};
use tokio::sync::watch;

pub(super) type SyncStatusReply = Result<SyncStatus, DerivationDelegateClientError>;

/// What a poll with no reply scripted does.
#[derive(Clone, Copy, Debug)]
enum UnscriptedPoll {
    /// Fails like an unreachable delegate, which the actor skips.
    Fails,
    /// Panics, so a test that scripts every poll catches an extra one.
    Panics,
}

/// Answers each poll with the next scripted reply. It counts every poll, and separately the polls
/// it answered with a sync status.
#[derive(Clone, Debug)]
pub(super) struct ScriptedDelegate {
    answered_polls: watch::Sender<u32>,
    polls: watch::Sender<u32>,
    replies: Arc<Mutex<VecDeque<SyncStatusReply>>>,
    unscripted_poll: UnscriptedPoll,
}

impl ScriptedDelegate {
    /// A delegate whose every poll must be scripted.
    pub(super) fn strict(replies: impl IntoIterator<Item = SyncStatusReply>) -> Self {
        Self::new(replies, UnscriptedPoll::Panics)
    }

    /// A delegate that fails the polls nothing is scripted for, so waits can span them.
    pub(super) fn lenient() -> Self {
        Self::new([], UnscriptedPoll::Fails)
    }

    fn new(
        replies: impl IntoIterator<Item = SyncStatusReply>,
        unscripted_poll: UnscriptedPoll,
    ) -> Self {
        Self {
            answered_polls: watch::Sender::new(0),
            polls: watch::Sender::new(0),
            replies: Arc::new(Mutex::new(replies.into_iter().collect())),
            unscripted_poll,
        }
    }

    /// Scripts `status` as the reply to the next unanswered poll.
    pub(super) fn reports(&self, status: SyncStatus) {
        self.replies.lock().unwrap().push_back(Ok(status));
    }

    /// Counts the polls answered with a sync status.
    pub(super) fn answered_polls(&self) -> watch::Receiver<u32> {
        self.answered_polls.subscribe()
    }

    /// Counts every poll, answered or not.
    pub(super) fn polls(&self) -> watch::Receiver<u32> {
        self.polls.subscribe()
    }

    pub(super) fn assert_finished(&self) {
        let unanswered = self.replies.lock().unwrap().len();
        assert_eq!(unanswered, 0, "{unanswered} scripted sync status replies were never polled");
    }
}

#[async_trait]
impl DerivationDelegateProvider for ScriptedDelegate {
    async fn fetch_sync_status(&self) -> SyncStatusReply {
        self.polls.send_modify(|polls| *polls += 1);
        let Some(reply) = self.replies.lock().unwrap().pop_front() else {
            return match self.unscripted_poll {
                UnscriptedPoll::Fails => Err(DerivationDelegateClientError::FetchFailed(
                    "no sync status scripted".into(),
                )),
                UnscriptedPoll::Panics => panic!("unexpected sync status poll"),
            };
        };
        if reply.is_ok() {
            self.answered_polls.send_modify(|polls| *polls += 1);
        }
        reply
    }
}

/// A sync status with these labels, its unsafe head at the local-safe head.
pub(super) fn sync_status(
    current_l1: BlockInfo,
    local_safe: L2BlockInfo,
    safe: L2BlockInfo,
    finalized: L2BlockInfo,
) -> SyncStatus {
    SyncStatus {
        current_l1,
        current_l1_finalized: BlockInfo::default(),
        finalized_l1: BlockInfo::default(),
        finalized_l2: finalized,
        head_l1: BlockInfo::default(),
        local_safe_l2: local_safe,
        safe_l1: BlockInfo::default(),
        safe_l2: safe,
        unsafe_l2: local_safe,
    }
}
