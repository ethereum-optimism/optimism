//! The interop authority epoch a node derives under.

use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::core::RpcResult;
use kona_rpc::InteropNodeApiServer;
use std::{
    fs::{self, File},
    io,
    path::PathBuf,
};
use tokio::sync::watch;

/// The interop authority epoch a node took on by deriving again from its finalized head, kept in
/// a file so that a restart knows whether the authority moved to another epoch meanwhile.
///
/// Without a file it is held in memory only, so every start derives again from the finalized head
/// once the authority answers.
#[derive(Debug)]
pub struct AppliedEpoch {
    /// The file holding the epoch, as a hex string.
    path: Option<PathBuf>,
    current: watch::Sender<Option<B256>>,
}

impl AppliedEpoch {
    /// The epoch recorded at `path`, or none when there is no file yet. A file that does not hold
    /// an epoch counts as none, with a warning: the node then derives again, which is always
    /// safe.
    pub fn load(path: Option<PathBuf>) -> io::Result<Self> {
        let epoch = match &path {
            None => None,
            Some(path) => match fs::read_to_string(path) {
                Ok(text) => match text.trim().parse::<B256>() {
                    Ok(epoch) => Some(epoch),
                    Err(err) => {
                        warn!(target: "authority", %err, path = %path.display(), "The applied epoch file holds no epoch; deriving again");
                        None
                    }
                },
                Err(err) if err.kind() == io::ErrorKind::NotFound => None,
                Err(err) => return Err(err),
            },
        };
        Ok(Self { path, current: watch::channel(epoch).0 })
    }

    /// The epoch the node derives under, if any.
    pub fn get(&self) -> Option<B256> {
        *self.current.borrow()
    }

    /// A view of the epoch the node derives under.
    pub fn watch(&self) -> watch::Receiver<Option<B256>> {
        self.current.subscribe()
    }

    /// Records that the node derives under `epoch`: in the file, written aside, synced and
    /// renamed over the last one, so a crash leaves the last epoch or this one. The epoch is held
    /// also if the file cannot be written.
    pub fn store(&self, epoch: B256) -> io::Result<()> {
        let written = self.path.as_ref().map_or(Ok(()), |path| {
            let aside = path.with_extension("tmp");
            fs::write(&aside, format!("{epoch}\n"))?;
            File::open(&aside)?.sync_all()?;
            fs::rename(&aside, path)?;
            match path.parent() {
                Some(dir) if !dir.as_os_str().is_empty() => File::open(dir)?.sync_all(),
                _ => Ok(()),
            }
        });
        self.current.send_replace(Some(epoch));
        written
    }
}

/// Serves the authority the epoch the node derives under, so it reads the node only once the
/// node took its epoch on.
#[derive(Debug)]
pub struct AppliedEpochRpc(pub watch::Receiver<Option<B256>>);

#[async_trait]
impl InteropNodeApiServer for AppliedEpochRpc {
    async fn applied_epoch(&self) -> RpcResult<Option<B256>> {
        Ok(*self.0.borrow())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A stored epoch is there after a restart; a missing or broken file is none.
    #[test]
    fn a_stored_epoch_survives_a_restart() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("authority-epoch");
        assert_eq!(AppliedEpoch::load(Some(path.clone())).unwrap().get(), None);

        let applied = AppliedEpoch::load(Some(path.clone())).unwrap();
        let view = applied.watch();
        applied.store(B256::repeat_byte(7)).unwrap();
        assert_eq!(*view.borrow(), Some(B256::repeat_byte(7)));
        applied.store(B256::repeat_byte(8)).unwrap();

        assert_eq!(
            AppliedEpoch::load(Some(path.clone())).unwrap().get(),
            Some(B256::repeat_byte(8))
        );
        assert!(!path.with_extension("tmp").exists());
        fs::write(&path, "not an epoch").unwrap();
        assert_eq!(AppliedEpoch::load(Some(path)).unwrap().get(), None);
    }

    #[test]
    fn without_a_file_the_epoch_is_held_in_memory() {
        let applied = AppliedEpoch::load(None).unwrap();
        applied.store(B256::repeat_byte(1)).unwrap();

        assert_eq!(applied.get(), Some(B256::repeat_byte(1)));
        assert_eq!(AppliedEpoch::load(None).unwrap().get(), None);
    }
}
