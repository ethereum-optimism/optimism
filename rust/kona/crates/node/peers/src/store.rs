//! Bootnode Store

use discv5::Enr;
use std::{collections::VecDeque, fs::File, io::Read, path::PathBuf};

/// The maximum number of peers that can be stored in the bootstore.
const MAX_PEERS: usize = 2048;

/// The records a bootstore holds, oldest first: the contents of its file, and nothing else.
///
/// When the number of records exceeds `MAX_PEERS`, the oldest are removed to make room for new
/// ones.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
struct Records(VecDeque<Enr>);

impl Records {
    /// Parses the contents of a bootstore file, a JSON list of records, and returns the records
    /// with the number of entries it skipped.
    ///
    /// An entry that isn't a valid record is skipped, so that one bad entry doesn't lose the
    /// others. Empty contents hold no records: an earlier version truncated the file before it
    /// wrote it, and a crash in between left it empty.
    fn parse_json_slice(json: &[u8]) -> Result<(Self, usize), serde_json::Error> {
        let mut records = Self::default();
        if json.iter().all(u8::is_ascii_whitespace) {
            return Ok((records, 0));
        }
        let entries: Vec<serde_json::Value> = serde_json::from_slice(json)?;
        let mut skipped = 0;
        for entry in entries {
            match serde_json::from_value::<Enr>(entry) {
                Ok(enr) => records.add(enr),
                Err(_) => skipped += 1,
            }
        }
        Ok((records, skipped))
    }

    /// Returns the contents of a bootstore file that holds these records.
    fn to_json_vec(&self) -> Result<Vec<u8>, serde_json::Error> {
        serde_json::to_vec(&self.0)
    }

    /// Adds `enr`, unless it is there already, and removes the oldest record if there are then
    /// too many.
    fn add(&mut self, enr: Enr) {
        if self.0.contains(&enr) {
            return;
        }

        debug!(target: "bootstore", "Adding enr to the boot store: {}", enr);
        self.0.push_back(enr);

        if self.0.len() > MAX_PEERS {
            debug!(target: "bootstore", "Boot store exceeded maximum peers, removing oldest peer");
            self.0.pop_front();
        }
    }
}

/// An error opening a [`BootStore`].
#[derive(Debug, thiserror::Error)]
pub enum BootStoreError {
    /// The default bootstore path is in the home directory, and there is none.
    #[error("no home directory for the default bootstore path")]
    NoHomeDirectory,
    /// The bootstore file, or its directory, can't be created, read or written.
    #[error("cannot open the bootstore {}", path.display())]
    Open {
        /// The path of the bootstore file.
        path: PathBuf,
        /// The reason.
        #[source]
        source: std::io::Error,
    },
    /// The file isn't a JSON list. It is left as it is: it may have been edited by hand, or be
    /// another file than the bootstore.
    #[error("the bootstore {} isn't a JSON list of node records: fix it or delete it", path.display())]
    Parse {
        /// The path of the bootstore file.
        path: PathBuf,
        /// The reason.
        #[source]
        source: serde_json::Error,
    },
}

/// The [`Enr`]s of peers, saved in a JSON file so that the node can find them again after a
/// restart, or kept only in memory.
#[derive(Debug)]
pub struct BootStore {
    /// The file the records are saved in, or `None` to keep them only in memory.
    path: Option<PathBuf>,
    /// The records.
    records: Records,
}

/// Where the bootstore file is.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BootStoreFile {
    /// Default path for the bootstore, ie `~/.kona/<chain_id>/bootstore.json`.
    Default {
        /// The l2 chain ID.
        chain_id: u64,
    },
    /// A custom path to the bootstore file.
    Custom(PathBuf),
}

impl BootStoreFile {
    /// Returns the path of the bootstore file.
    pub fn path(&self) -> Result<PathBuf, BootStoreError> {
        match self {
            Self::Default { chain_id } => {
                let mut path = dirs::home_dir().ok_or(BootStoreError::NoHomeDirectory)?;
                path.push(".kona");
                path.push(chain_id.to_string());
                path.push("bootstore.json");
                Ok(path)
            }
            Self::Custom(path) => Ok(path.clone()),
        }
    }
}

impl BootStore {
    /// Returns an empty store that keeps its records only in memory.
    pub fn in_memory() -> Self {
        Self { path: None, records: Records::default() }
    }

    /// Opens the store saved at `path`, and returns it with the number of entries it skipped
    /// because they aren't valid records.
    ///
    /// The file, and its directory, are created if they are missing, so that a path that can't
    /// be written fails here rather than at every [`Self::sync`].
    ///
    /// Fails if the file can't be created or read, or isn't a JSON list, which is left as it is.
    pub fn open(path: PathBuf) -> Result<(Self, usize), BootStoreError> {
        let open_error = |source| BootStoreError::Open { path: path.clone(), source };
        if let Some(dir) = path.parent() {
            std::fs::create_dir_all(dir).map_err(open_error)?;
        }
        let mut file = File::options()
            .read(true)
            .write(true)
            .create(true)
            .truncate(false)
            .open(&path)
            .map_err(open_error)?;
        let mut json = Vec::new();
        file.read_to_end(&mut json).map_err(open_error)?;
        let (records, skipped) = Records::parse_json_slice(&json)
            .map_err(|source| BootStoreError::Parse { path: path.clone(), source })?;
        Ok((Self { path: Some(path), records }, skipped))
    }

    /// Adds an [`Enr`] to the store, removing the oldest if there are then too many. The file
    /// changes at the next [`Self::sync`].
    pub fn add_enr(&mut self, enr: Enr) {
        self.records.add(enr);
    }

    /// Returns the records, oldest first.
    pub fn iter(&self) -> impl Iterator<Item = &Enr> {
        self.records.0.iter()
    }

    /// Returns the number of peers in the bootstore that
    /// have the [`crate::OpStackEnr::OP_CL_KEY`] in the ENR.
    pub fn valid_peers(&self) -> Vec<&Enr> {
        self.iter()
            .filter(|enr| enr.get_raw_rlp(crate::OpStackEnr::OP_CL_KEY.as_bytes()).is_some())
            .collect()
    }

    /// Returns the number of peers that contain the
    /// [`crate::OpStackEnr::OP_CL_KEY`] in the ENR *and*
    /// have the correct chain id and version.
    pub fn valid_peers_with_chain_id(&self, chain_id: u64) -> Vec<&Enr> {
        self.iter().filter(|enr| crate::EnrValidation::validate(enr, chain_id).is_valid()).collect()
    }

    /// Returns the number of peers in the in-memory store.
    pub fn len(&self) -> usize {
        self.records.0.len()
    }

    /// Returns if the in-memory store is empty.
    pub fn is_empty(&self) -> bool {
        self.records.0.is_empty()
    }

    /// Merges the given list of [`Enr`]s with the current list of peers.
    pub fn merge(&mut self, peers: impl IntoIterator<Item = Enr>) {
        peers.into_iter().for_each(|peer| self.records.add(peer));
    }

    /// Keeps only the records `keep` accepts, and returns how many it removed. The file changes at
    /// the next [`Self::sync`].
    pub fn retain(&mut self, keep: impl FnMut(&Enr) -> bool) -> usize {
        let len = self.records.0.len();
        self.records.0.retain(keep);
        len - self.records.0.len()
    }

    /// Writes the records to the store's file, if it has one.
    pub fn sync(&mut self) -> Result<(), std::io::Error> {
        if let Some(path) = &self.path {
            std::fs::write(path, self.records.to_json_vec()?)?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use enr::CombinedKey;

    /// A new record with an IPv4 address.
    fn record() -> Enr {
        Enr::builder()
            .ip4(std::net::Ipv4Addr::new(1, 1, 1, 1))
            .udp4(9222)
            .build(&CombinedKey::generate_secp256k1())
            .unwrap()
    }

    /// A store opened at a path whose directory doesn't exist is empty, and creates the directory
    /// and the file, so that a path it couldn't write would fail here.
    #[test]
    fn test_open_creates_the_file_and_its_directory() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("datadir").join("bootstore.json");

        let (store, skipped) = BootStore::open(path.clone()).unwrap();

        assert!(store.is_empty());
        assert_eq!(skipped, 0);
        assert!(path.is_file());
    }

    /// The records a store saves are the ones it opens again.
    #[test]
    fn test_saved_records_open_again() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let records = [record(), record()];
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        for record in &records {
            store.add_enr(record.clone());
        }
        store.sync().unwrap();

        let (store, skipped) = BootStore::open(path).unwrap();

        assert_eq!(store.iter().cloned().collect::<Vec<_>>(), records);
        assert_eq!(skipped, 0);
    }

    /// An entry that isn't a record is skipped and counted, and the others are kept.
    #[test]
    fn test_open_skips_entries_that_are_not_records() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let kept = record();
        let json = serde_json::json!(["enr:-not-base64!", kept.to_base64(), 42]);
        std::fs::write(&path, json.to_string()).unwrap();

        let (store, skipped) = BootStore::open(path).unwrap();

        assert_eq!(store.iter().collect::<Vec<_>>(), [&kept]);
        assert_eq!(skipped, 2);
    }

    /// An empty file, which an earlier version left after a crash, holds no records.
    #[rstest::rstest]
    #[case::empty("")]
    #[case::whitespace(" \n")]
    fn test_open_empty_file(#[case] contents: &str) {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        std::fs::write(&path, contents).unwrap();

        let (store, skipped) = BootStore::open(path).unwrap();

        assert!(store.is_empty());
        assert_eq!(skipped, 0);
    }

    /// A file that isn't a JSON list fails to open, and is left as it is: it may be another file
    /// than the bootstore, given by mistake.
    #[rstest::rstest]
    #[case::not_json("not json")]
    #[case::object(r#"{"peers": []}"#)]
    #[case::binary("\u{0}\u{1}")]
    fn test_open_leaves_a_file_that_is_not_a_list(#[case] contents: &str) {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        std::fs::write(&path, contents).unwrap();

        let err = BootStore::open(path.clone()).unwrap_err();

        assert!(matches!(err, BootStoreError::Parse { .. }), "{err}");
        assert_eq!(std::fs::read_to_string(path).unwrap(), contents);
    }

    /// A reader that opened the file before a save reads the file as it was when it opened it,
    /// whole, rather than what the save left, which is empty or cut off while the save writes it.
    /// So a save replaces the file rather than writing over it, which also keeps the old file if
    /// the node crashes during a save.
    #[test]
    fn test_save_replaces_the_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        store.add_enr(record());
        store.sync().unwrap();
        let before = std::fs::read(&path).unwrap();
        let mut reader = File::open(&path).unwrap();

        store.add_enr(record());
        store.sync().unwrap();

        let mut read = Vec::new();
        reader.read_to_end(&mut read).unwrap();
        assert_eq!(read, before, "a save wrote over the file a reader had open");
    }

    /// A reader never sees the file empty or cut off while the store saves it, as can happen when
    /// the store truncates the file before it writes it: an operator's tools, or the node's own
    /// tests, may read it while the node runs. The gap is short, so this test may miss it, unlike
    /// [`test_save_replaces_the_file`].
    ///
    /// Each save adds a record, so that the store has something to write each time.
    #[test]
    fn test_reader_never_sees_a_partial_file() {
        use std::sync::{
            Arc,
            atomic::{AtomicBool, Ordering},
        };

        const RECORDS: usize = 500;
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        for _ in 0..RECORDS {
            store.add_enr(record());
        }
        store.sync().unwrap();

        let done = Arc::new(AtomicBool::new(false));
        let reader = std::thread::spawn({
            let done = Arc::clone(&done);
            move || {
                let mut partial = 0;
                while !done.load(Ordering::Relaxed) {
                    let json = std::fs::read(&path).unwrap();
                    match Records::parse_json_slice(&json) {
                        Ok((records, _)) if records.0.len() >= RECORDS => {}
                        _ => partial += 1,
                    }
                }
                partial
            }
        });
        for _ in 0..1000 {
            store.add_enr(record());
            store.sync().unwrap();
        }
        done.store(true, Ordering::Relaxed);

        let partial = reader.join().unwrap();
        assert_eq!(partial, 0, "a reader saw the bootstore empty or cut off {partial} times");
    }

    /// A directory can't be opened as the bootstore.
    #[test]
    fn test_open_directory_fails() {
        let dir = tempfile::tempdir().unwrap();

        let err = BootStore::open(dir.path().to_path_buf()).unwrap_err();

        assert!(matches!(err, BootStoreError::Open { .. }), "{err}");
    }
}
