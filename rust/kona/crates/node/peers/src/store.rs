//! Bootnode Store

use discv5::Enr;
use serde::Deserialize;
use std::{
    collections::VecDeque,
    io::{self, Write},
    path::{Path, PathBuf},
};
use tempfile::NamedTempFile;

/// The maximum number of peers that can be stored in the bootstore.
const MAX_PEERS: usize = 2048;

/// The mode the bootstore file is created with before the umask applies: the one
/// [`std::fs::File::create`] uses, so that it is readable like any other file the node writes.
#[cfg(unix)]
const FILE_MODE: u32 = 0o666;

/// The records a bootstore holds, oldest first: the contents of its file, and nothing else.
///
/// When the number of records exceeds `MAX_PEERS`, the oldest are removed to make room for new
/// ones.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
struct Records(VecDeque<Enr>);

impl Records {
    /// Parses the contents of a bootstore file, a JSON list of records, and returns the records
    /// with the entries it skipped.
    ///
    /// An entry that isn't a valid record is skipped, so that one bad entry doesn't lose the
    /// others. Empty contents hold no records: an earlier version truncated the file before it
    /// wrote it, and a crash in between left it empty.
    fn parse_json_slice(json: &[u8]) -> Result<(Self, Vec<SkippedEntry>), serde_json::Error> {
        let mut records = Self::default();
        let mut skipped = Vec::new();
        if json.iter().all(u8::is_ascii_whitespace) {
            return Ok((records, skipped));
        }
        let entries: Vec<serde_json::Value> = serde_json::from_slice(json)?;
        for entry in entries {
            match Enr::deserialize(&entry) {
                Ok(enr) => {
                    records.add(enr);
                }
                Err(error) => skipped.push(SkippedEntry { entry, error }),
            }
        }
        Ok((records, skipped))
    }

    /// Returns the contents of a bootstore file that holds these records.
    fn to_json_vec(&self) -> Result<Vec<u8>, serde_json::Error> {
        serde_json::to_vec(&self.0)
    }

    /// Adds `enr`, unless it is there already, and removes the oldest record if there are then
    /// too many. Returns whether the records changed.
    fn add(&mut self, enr: Enr) -> bool {
        if self.0.contains(&enr) {
            return false;
        }

        debug!(target: "bootstore", "Adding enr to the boot store: {}", enr);
        self.0.push_back(enr);

        if self.0.len() > MAX_PEERS {
            debug!(target: "bootstore", "Boot store exceeded maximum peers, removing oldest peer");
            self.0.pop_front();
        }
        true
    }
}

/// An entry of a bootstore file that isn't a node record, which [`BootStore::open`] skips. The
/// store's next save leaves it out.
#[derive(Debug)]
pub struct SkippedEntry {
    /// The entry, as it is in the file.
    pub entry: serde_json::Value,
    /// Why it isn't a node record.
    pub error: serde_json::Error,
}

/// An error opening a [`BootStore`].
#[derive(Debug, thiserror::Error)]
pub enum BootStoreError {
    /// The default bootstore path is in the home directory, and there is none.
    #[error("no home directory for the default bootstore path")]
    NoHomeDirectory,
    /// The bootstore file can't be read.
    #[error("cannot read the bootstore {}", path.display())]
    Open {
        /// The path of the bootstore file.
        path: PathBuf,
        /// The reason.
        #[source]
        source: std::io::Error,
    },
    /// The bootstore file can't be saved.
    #[error("cannot save the bootstore {}", path.display())]
    Save {
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
    /// Whether the file differs from the records: they changed since they were last saved, the
    /// file is missing, or it holds entries that aren't records.
    dirty: bool,
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

/// Returns the directory of the bootstore file at `path`, where a save writes its temporary file.
fn dir_of(path: &Path) -> &Path {
    match path.parent() {
        Some(dir) if !dir.as_os_str().is_empty() => dir,
        _ => Path::new("."),
    }
}

/// Creates a temporary file next to the bootstore file at `path`, for a save to rename over it.
/// Its name starts with the bootstore's, so that one left by a crash is easy to tell, and it is
/// removed when dropped, unless persisted.
///
/// `tempfile` would create it readable by its owner only, which the rename keeps, so it gets
/// [`FILE_MODE`] instead.
fn temp_file(path: &Path) -> io::Result<NamedTempFile> {
    let mut prefix = path.file_name().unwrap_or_default().to_owned();
    prefix.push(".");
    let mut builder = tempfile::Builder::new();
    builder.prefix(&prefix).suffix(".tmp");
    #[cfg(unix)]
    builder.permissions(std::os::unix::fs::PermissionsExt::from_mode(FILE_MODE));
    builder.tempfile_in(dir_of(path))
}

impl BootStore {
    /// Returns an empty store that keeps its records only in memory.
    pub fn in_memory() -> Self {
        Self { path: None, records: Records::default(), dirty: false }
    }

    /// Opens the store saved at `path`, and returns it with the entries it skipped because they
    /// aren't valid records, for the caller to log. They count as a change, so the next
    /// [`Self::sync`] writes the file without them. A
    /// missing file holds no records.
    ///
    /// Only reads the file, so that other readers can open it while the node saves it. If the file
    /// is missing, the first [`Self::sync`] creates it, and its directory. The node saves the store
    /// once it has opened it, so that a bootstore it can't write stops it from starting.
    ///
    /// Fails if the file can't be read, or isn't a JSON list, which is left as it is.
    pub fn open(path: PathBuf) -> Result<(Self, Vec<SkippedEntry>), BootStoreError> {
        let (json, missing) = match std::fs::read(&path) {
            Ok(json) => (json, false),
            Err(err) if err.kind() == io::ErrorKind::NotFound => (Vec::new(), true),
            Err(source) => return Err(BootStoreError::Open { path, source }),
        };
        let (records, skipped) = Records::parse_json_slice(&json)
            .map_err(|source| BootStoreError::Parse { path: path.clone(), source })?;
        let dirty = missing || !skipped.is_empty();
        Ok((Self { path: Some(path), records, dirty }, skipped))
    }

    /// Adds an [`Enr`] to the store, removing the oldest if there are then too many. The file
    /// changes at the next [`Self::sync`].
    pub fn add_enr(&mut self, enr: Enr) {
        self.dirty |= self.records.add(enr);
    }

    /// Returns the records, oldest first.
    pub fn iter(&self) -> impl Iterator<Item = &Enr> {
        self.records.0.iter()
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

    /// Keeps only the records `keep` accepts, and returns how many it removed. The file changes at
    /// the next [`Self::sync`].
    pub fn retain(&mut self, keep: impl FnMut(&Enr) -> bool) -> usize {
        let len = self.records.0.len();
        self.records.0.retain(keep);
        let removed = len - self.records.0.len();
        self.dirty |= removed > 0;
        removed
    }

    /// Saves the records, if they changed since they were last saved or the file is missing, and
    /// returns whether they did, whether or not the store has a file to save them to.
    ///
    /// A store with a file writes them to a temporary file next to it, creating the directory if
    /// it is missing, and renames that over it, so that a reader sees either the old file or the
    /// new one, whole, and a crash during a save keeps the old one. A temporary file left by a
    /// failed write is removed, and the next save tries again.
    pub fn sync(&mut self) -> Result<bool, BootStoreError> {
        if !self.dirty {
            return Ok(false);
        }
        if let Some(path) = &self.path {
            Self::save(path, &self.records)
                .map_err(|source| BootStoreError::Save { path: path.clone(), source })?;
        }
        self.dirty = false;
        Ok(true)
    }

    /// Saves `records` at `path`, see [`Self::sync`].
    fn save(path: &Path, records: &Records) -> io::Result<()> {
        std::fs::create_dir_all(dir_of(path))?;
        let mut file = temp_file(path)?;
        file.write_all(&records.to_json_vec()?)?;
        file.as_file().sync_all()?;
        file.persist(path)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use enr::CombinedKey;
    use std::{fs::File, io::Read};

    /// A new record with an IPv4 address.
    fn record() -> Enr {
        Enr::builder()
            .ip4(std::net::Ipv4Addr::new(1, 1, 1, 1))
            .udp4(9222)
            .build(&CombinedKey::generate_secp256k1())
            .unwrap()
    }

    /// A store opened at a path whose directory doesn't exist is empty, and writes nothing until
    /// it saves, which creates the directory and the file.
    #[test]
    fn test_save_creates_the_directory() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("datadir").join("bootstore.json");

        let (mut store, skipped) = BootStore::open(path.clone()).unwrap();
        assert!(store.is_empty());
        assert!(skipped.is_empty());
        assert!(!path.parent().unwrap().exists(), "opening the store wrote to the disk");

        assert!(store.sync().unwrap(), "a save didn't create the missing file");
        assert!(path.is_file());
    }

    /// Opening and saving leave no temporary file behind, only the bootstore.
    #[test]
    fn test_open_and_save_leave_no_temporary_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");

        let (mut store, _) = BootStore::open(path).unwrap();
        store.add_enr(record());
        store.sync().unwrap();

        let files: Vec<_> = std::fs::read_dir(dir.path())
            .unwrap()
            .map(|entry| entry.unwrap().file_name())
            .collect();
        assert_eq!(files, ["bootstore.json"]);
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
        assert!(skipped.is_empty());
    }

    /// An entry that isn't a record is skipped, and returned with the reason, and the others are
    /// kept.
    #[test]
    fn test_open_skips_entries_that_are_not_records() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let kept = record();
        let json = serde_json::json!(["enr:-not-base64!", kept.to_base64(), 42]);
        std::fs::write(&path, json.to_string()).unwrap();

        let (mut store, skipped) = BootStore::open(path.clone()).unwrap();

        assert_eq!(store.iter().collect::<Vec<_>>(), [&kept]);
        let skipped: Vec<_> = skipped.into_iter().map(|skipped| skipped.entry).collect();
        assert_eq!(skipped, [serde_json::json!("enr:-not-base64!"), serde_json::json!(42)]);

        // The next save writes the file without them.
        assert!(store.sync().unwrap(), "skipped entries weren't a change to save");
        let saved: serde_json::Value =
            serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
        assert_eq!(saved, serde_json::json!([kept.to_base64()]));
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
        assert!(skipped.is_empty());
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

    /// A saved bootstore is readable by other users, such as a sidecar or an operator's tools
    /// running under another user, as a file the node writes directly would be: the records it
    /// holds are public. The umask decides that mode, so the test compares with such a file.
    #[cfg(unix)]
    #[test]
    fn test_saved_file_is_readable_by_others() {
        use std::os::unix::fs::PermissionsExt;

        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        store.add_enr(record());
        store.sync().unwrap();
        let written = dir.path().join("written");
        File::create(&written).unwrap();

        let mode = |path| std::fs::metadata(path).unwrap().permissions().mode() & 0o777;
        assert_eq!(
            mode(&path),
            mode(&written),
            "the saved bootstore has another mode than a file the node writes directly"
        );
    }

    /// A save writes the file only if the store changed since it was opened or last saved, so
    /// that a node doesn't rewrite its bootstore every store interval for nothing. A save that
    /// writes replaces the file, which then has a new inode.
    #[cfg(unix)]
    #[test]
    fn test_save_without_changes_leaves_the_file() {
        use std::os::unix::fs::MetadataExt;

        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        store.add_enr(record());
        store.sync().unwrap();
        let inode = || std::fs::metadata(&path).unwrap().ino();

        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        let opened = inode();
        store.sync().unwrap();
        assert_eq!(inode(), opened, "a save rewrote the file it had just opened");

        store.add_enr(record());
        store.sync().unwrap();
        let changed = inode();
        assert_ne!(changed, opened, "a save didn't write a change");

        store.sync().unwrap();
        assert_eq!(inode(), changed, "a save rewrote the file it had just saved");
    }

    /// A save says whether there were changes to save, whether or not the store has a file: so a
    /// caller can tell the same way for either.
    #[rstest::rstest]
    #[case::in_memory(false)]
    #[case::file(true)]
    fn test_save_says_whether_there_were_changes(#[case] file: bool) {
        let dir = tempfile::tempdir().unwrap();
        let mut store = if file {
            BootStore::open(dir.path().join("bootstore.json")).unwrap().0
        } else {
            BootStore::in_memory()
        };
        // A missing file is a change to save, so that it gets created.
        assert_eq!(store.sync().unwrap(), file);

        let added = record();
        store.add_enr(added.clone());
        assert!(store.sync().unwrap(), "an added record wasn't a change");
        assert!(!store.sync().unwrap(), "a save without changes said it had some");

        store.add_enr(added);
        assert!(!store.sync().unwrap(), "a record added again was a change");

        store.retain(|_| true);
        assert!(!store.sync().unwrap(), "keeping every record was a change");
        store.retain(|_| false);
        assert!(store.sync().unwrap(), "a removed record wasn't a change");
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

    /// A directory can't be read as the bootstore.
    #[test]
    fn test_open_directory_fails() {
        let dir = tempfile::tempdir().unwrap();

        let err = BootStore::open(dir.path().to_path_buf()).unwrap_err();

        assert!(matches!(err, BootStoreError::Open { .. }), "{err}");
    }
}
