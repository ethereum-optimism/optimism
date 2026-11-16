//! Bootnode Store

use discv5::Enr;
use enr::NodeId;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    cmp::Ordering,
    collections::BTreeMap,
    io::{self, Write},
    net::{SocketAddr, SocketAddrV4, SocketAddrV6},
    path::{Path, PathBuf},
    time::SystemTime,
};
use tempfile::NamedTempFile;

/// The maximum number of peers that can be stored in the bootstore.
const MAX_PEERS: usize = 2048;

/// The format version of the bootstore files that [`Records::to_json_vec`] writes.
/// [`Contents::from_json_slice`] reads this one and every earlier one.
const FILE_VERSION: u64 = 1;

/// The mode the bootstore file is created with before the umask applies: the one
/// [`std::fs::File::create`] uses, so that it is readable like any other file the node writes.
#[cfg(unix)]
const FILE_MODE: u32 = 0o666;

/// The records a bootstore holds, one for each node, by node ID: the contents of its file, and
/// nothing else.
///
/// When a new node would make the records exceed `MAX_PEERS`, one is evicted to make room for it,
/// see [`Records::evict`].
#[derive(Debug, Default, Clone, PartialEq, Eq)]
struct Records(BTreeMap<[u8; 32], Entry>);

impl Records {
    /// Parses the contents of a bootstore file. An entry that isn't valid, or that [`Self::add`]
    /// doesn't keep, is skipped, so that one bad entry doesn't lose the others.
    ///
    /// Empty contents hold no records: an earlier version truncated the file before it wrote it,
    /// and a crash in between left it empty.
    fn parse_json_slice(json: &[u8]) -> Result<Parsed, ParseError> {
        let mut parsed = Parsed { records: Self::default(), skipped: Vec::new(), current: false };
        if json.iter().all(u8::is_ascii_whitespace) {
            return Ok(parsed);
        }
        let contents = Contents::from_json_slice(json)?;
        parsed.current = contents.version() == FILE_VERSION;
        type ParseEntry = fn(&Value) -> Result<Entry, serde_json::Error>;
        let (entries, parse_entry): (_, ParseEntry) = match contents {
            Contents::V0(enrs) => (enrs, |value| Enr::deserialize(value).map(Entry::candidate)),
            Contents::V1(entries) => (entries, |value| Entry::deserialize(value)),
        };
        for value in entries {
            let entry = match parse_entry(&value) {
                Ok(entry) => entry,
                Err(error) => {
                    parsed
                        .skipped
                        .push(SkippedEntry { entry: value, reason: SkipReason::Invalid(error) });
                    continue;
                }
            };
            let skipped = match parsed.records.add(entry) {
                Added::New { evicted: None } => continue,
                Added::New { evicted: Some(evicted) } => {
                    SkippedEntry::of(&evicted, SkipReason::OverLimit)
                }
                Added::Replaced { old } => SkippedEntry::of(&old, SkipReason::Stale),
                Added::Same { .. } => SkippedEntry { entry: value, reason: SkipReason::Duplicate },
                Added::Stale => SkippedEntry { entry: value, reason: SkipReason::Stale },
            };
            parsed.skipped.push(skipped);
        }
        Ok(parsed)
    }

    /// Returns the contents of a bootstore file that holds these records, in node ID order.
    fn to_json_vec(&self) -> Result<Vec<u8>, serde_json::Error> {
        let entries = self.0.values().map(serde_json::to_value).collect::<Result<_, _>>()?;
        Contents::V1(entries).to_json_vec()
    }

    /// Adds `entry`, keeping one record for each node, the one with the highest `seq`:
    ///
    /// - A record of a new node is added, evicting another if there are then too many.
    /// - A record with a higher `seq` replaces the stored one, and keeps its evidence for the
    ///   addresses it keeps, see [`Entry::replaced_by`].
    /// - The same record replaces the stored one too, which takes only its newer evidence.
    /// - A record with a lower `seq`, or another with the same `seq`, is ignored.
    fn add(&mut self, entry: Entry) -> Added {
        let key = entry.enr.node_id().raw();
        if let Some(stored) = self.0.get_mut(&key) {
            return match entry.enr.seq().cmp(&stored.enr.seq()) {
                Ordering::Greater => {
                    let replacement = stored.replaced_by(entry);
                    Added::Replaced { old: std::mem::replace(stored, replacement) }
                }
                Ordering::Equal if entry.enr == stored.enr => {
                    let replacement = stored.replaced_by(entry);
                    let refreshed = *stored != replacement;
                    *stored = replacement;
                    Added::Same { refreshed }
                }
                Ordering::Equal | Ordering::Less => Added::Stale,
            };
        }
        let evicted = if self.0.len() >= MAX_PEERS { self.evict() } else { None };
        self.0.insert(key, entry);
        Added::New { evicted }
    }

    /// Removes and returns the record to evict to make room for a new one: the one whose UDP
    /// address a session confirmed longest ago, those that no session confirmed first. Of those
    /// confirmed at the same time, the first in node ID order.
    fn evict(&mut self) -> Option<Entry> {
        let (&key, _) = self.0.iter().min_by_key(|(_, entry)| entry.udp_verified_at)?;
        self.0.remove(&key)
    }
}

/// A record a bootstore holds, with the evidence that the node it names is at its addresses. A
/// record without `tcp_verified_at` is a candidate: nothing showed that gossip can reach it.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
struct Entry {
    /// The record.
    enr: Enr,
    /// When a discovery session last confirmed the record's UDP address.
    #[serde(default, skip_serializing_if = "Option::is_none", with = "unix_secs")]
    udp_verified_at: Option<SystemTime>,
    /// When a gossip connection we dialed last reached the record's TCP address, and the peer ID
    /// of its key.
    #[serde(default, skip_serializing_if = "Option::is_none", with = "unix_secs")]
    tcp_verified_at: Option<SystemTime>,
}

impl Entry {
    /// Returns an entry for `enr` without evidence.
    const fn candidate(enr: Enr) -> Self {
        Self { enr, udp_verified_at: None, tcp_verified_at: None }
    }

    /// Returns `newer`, a record of the same node with a higher `seq`, or the same record, with the
    /// evidence of this entry for the addresses that `newer` keeps, where it is newer: a timestamp
    /// is kept only if the sockets it proves are unchanged. A record has one IP for each family,
    /// which UDP and TCP share, so a new IP drops both, and a new TCP port only `tcp_verified_at`.
    /// The same record keeps all its sockets, so it takes the newer of each timestamp.
    fn replaced_by(&self, mut newer: Self) -> Self {
        if udp_sockets(&self.enr) == udp_sockets(&newer.enr) {
            newer.udp_verified_at = newer.udp_verified_at.max(self.udp_verified_at);
        }
        if tcp_sockets(&self.enr) == tcp_sockets(&newer.enr) {
            newer.tcp_verified_at = newer.tcp_verified_at.max(self.tcp_verified_at);
        }
        newer
    }
}

/// What [`Records::add`] did with an entry.
#[derive(Debug)]
enum Added {
    /// The records held no record of its node. `evicted` made room for it.
    New {
        /// The entry evicted to make room, if there were too many.
        evicted: Option<Entry>,
    },
    /// It replaced an older record of its node.
    Replaced {
        /// The entry it replaced.
        old: Entry,
    },
    /// The records held the same record.
    Same {
        /// Whether the stored record took newer evidence from it.
        refreshed: bool,
    },
    /// The records held a newer record of its node, or another with the same `seq`, and kept it.
    Stale,
}

impl Added {
    /// Returns whether the records changed.
    const fn changed(&self) -> bool {
        match self {
            Self::New { .. } | Self::Replaced { .. } => true,
            Self::Same { refreshed } => *refreshed,
            Self::Stale => false,
        }
    }
}

/// The contents of a bootstore file, in each format version. The entries are left as JSON:
/// [`Records::parse_json_slice`] reads them one by one, so that one that isn't valid doesn't lose
/// the others.
#[derive(Debug)]
enum Contents {
    /// Format version 0, which has no version field: a list of records, without evidence.
    V0(Vec<Value>),
    /// Format version 1, `{"version": 1, "records": [...]}`, whose records are [`Entry`]s.
    V1(Vec<Value>),
}

impl Contents {
    /// Parses the contents of a bootstore file in any format version up to [`FILE_VERSION`].
    fn from_json_slice(json: &[u8]) -> Result<Self, ParseError> {
        /// The first format version after [`FILE_VERSION`], from a newer version of the node. As
        /// the last arm of the match below, it makes the match stop compiling when
        /// [`FILE_VERSION`] goes up without an arm for the new version.
        const FIRST_NEWER_VERSION: u64 = FILE_VERSION + 1;

        /// The field of a file with a version field that gives its format.
        #[derive(Deserialize)]
        struct Versioned {
            version: u64,
        }

        /// The fields of a format version that lists its entries in `records`.
        #[derive(Deserialize)]
        struct RecordValues {
            records: Vec<Value>,
        }

        let json: Value = serde_json::from_slice(json).map_err(ParseError::Json)?;
        if let Value::Array(records) = json {
            return Ok(Self::V0(records));
        }
        let Versioned { version } = Versioned::deserialize(&json).map_err(ParseError::Json)?;
        match version {
            0 => Err(ParseError::Json(serde::de::Error::custom(
                "format version 0 is a list of records, without a version",
            ))),
            1 => {
                let RecordValues { records } =
                    RecordValues::deserialize(json).map_err(ParseError::Json)?;
                Ok(Self::V1(records))
            }
            FIRST_NEWER_VERSION.. => Err(ParseError::Newer(version)),
        }
    }

    /// Returns the contents of a bootstore file in this format version.
    fn to_json_vec(&self) -> Result<Vec<u8>, serde_json::Error> {
        match self {
            Self::V0(records) => serde_json::to_vec(records),
            Self::V1(records) => {
                serde_json::to_vec(&serde_json::json!({"version": 1, "records": records}))
            }
        }
    }

    /// Returns the format version.
    const fn version(&self) -> u64 {
        match self {
            Self::V0(_) => 0,
            Self::V1(_) => 1,
        }
    }
}

/// What [`Records::parse_json_slice`] read from a bootstore file.
#[derive(Debug)]
struct Parsed {
    /// The records it kept.
    records: Records,
    /// The entries it skipped.
    skipped: Vec<SkippedEntry>,
    /// Whether the file is in the format [`FILE_VERSION`], which the store writes. One that isn't
    /// is written again in that format.
    current: bool,
}

/// Why the contents of a file can't be read as a bootstore.
#[derive(Debug)]
enum ParseError {
    /// It isn't in any format version up to [`FILE_VERSION`].
    Json(serde_json::Error),
    /// It is in a format version after [`FILE_VERSION`].
    Newer(u64),
}

/// An entry of a bootstore file that [`BootStore::open`] skipped. The store's next save leaves it
/// out.
#[derive(Debug)]
pub struct SkippedEntry {
    /// The entry, as it is in the file, or the record it holds.
    pub entry: Value,
    /// Why it was skipped.
    pub reason: SkipReason,
}

impl SkippedEntry {
    /// Returns the entry of the record of `entry`, which the store skipped for `reason`.
    fn of(entry: &Entry, reason: SkipReason) -> Self {
        Self { entry: Value::String(entry.enr.to_base64()), reason }
    }
}

/// Why [`BootStore::open`] skipped an entry of its file.
#[derive(Debug, thiserror::Error)]
pub enum SkipReason {
    /// The entry isn't a node record.
    #[error("not a node record: {0}")]
    Invalid(serde_json::Error),
    /// The entry is the same record as an earlier one.
    #[error("a duplicate of another entry")]
    Duplicate,
    /// Another entry holds a record of the same node with a higher `seq`, or one with the same
    /// `seq` that came first.
    #[error("superseded by another record of its node")]
    Stale,
    /// The file holds more than the store keeps, and this record was evicted to make room.
    #[error("over the limit of {} records", MAX_PEERS)]
    OverLimit,
}

/// An error opening or saving a [`BootStore`].
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
    /// The file isn't a bootstore: neither a JSON list of records, nor an object with a format
    /// version and entries. It is left as it is: it may have been edited by hand, or be another
    /// file than the bootstore.
    #[error("the bootstore {} isn't a bootstore file: fix it or delete it", path.display())]
    Parse {
        /// The path of the bootstore file.
        path: PathBuf,
        /// The reason.
        #[source]
        source: serde_json::Error,
    },
    /// The file is in a format version that a newer version of the node wrote. It is left as it
    /// is, for that version.
    #[error("the bootstore {} has format version {version}, which a newer version of the node wrote: upgrade, or delete the file", path.display())]
    NewerVersion {
        /// The path of the bootstore file.
        path: PathBuf,
        /// The version of the file's format.
        version: u64,
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
    /// file is missing or in an earlier format, or it holds entries the store skipped.
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

    /// Opens the store saved at `path`, and returns it with the entries it skipped, for the caller
    /// to log: those that aren't valid records, and those it doesn't keep, such as an older record
    /// of a node than another entry. They count as a change, so the next [`Self::sync`] writes the
    /// file without them. A missing file holds no records.
    ///
    /// A file in the format earlier versions wrote, a list of records, is read as records without
    /// evidence, and the next [`Self::sync`] writes it in the current format.
    ///
    /// Only reads the file, so that other readers can open it while the node saves it. If the file
    /// is missing, the first [`Self::sync`] creates it, and its directory. The node saves the store
    /// once it has opened it, so that a bootstore it can't create stops it from starting.
    ///
    /// Fails if the file can't be read, isn't a bootstore, or is in a format version a newer
    /// version of the node wrote. Such a file is left as it is.
    pub fn open(path: PathBuf) -> Result<(Self, Vec<SkippedEntry>), BootStoreError> {
        let json = match std::fs::read(&path) {
            Ok(json) => json,
            Err(err) if err.kind() == io::ErrorKind::NotFound => Vec::new(),
            Err(source) => return Err(BootStoreError::Open { path, source }),
        };
        let Parsed { records, skipped, current } =
            Records::parse_json_slice(&json).map_err(|err| match err {
                ParseError::Json(source) => BootStoreError::Parse { path: path.clone(), source },
                ParseError::Newer(version) => {
                    BootStoreError::NewerVersion { path: path.clone(), version }
                }
            })?;
        // A missing file isn't in the current format either.
        let dirty = !current || !skipped.is_empty();
        Ok((Self { path: Some(path), records, dirty }, skipped))
    }

    /// Adds `enr` without evidence, as a candidate. The store keeps one record for each node, the
    /// one with the highest `seq`, so `enr` replaces the stored record of its node only if its
    /// `seq` is higher. A new node evicts another if there are then too many. The file changes at
    /// the next [`Self::sync`].
    pub fn add_enr(&mut self, enr: Enr) {
        self.add(Entry::candidate(enr));
    }

    /// Notes that a discovery session with the node `node_id`, whose packets came from `socket`,
    /// confirmed the UDP address of its stored record at `at`, if the record's UDP address of the
    /// family of `socket` is `socket`. discv5 doesn't check that for a record without a UDP
    /// address of that family, nor for an unverifiable record. A session from another socket
    /// leaves the record's evidence as it is: it says nothing against the record's address.
    ///
    /// Does nothing for a node the store doesn't hold, nor for a time before the one it has. A
    /// newer time changes the file at the next [`Self::sync`].
    pub fn set_udp_verified(&mut self, node_id: NodeId, socket: SocketAddr, at: SystemTime) {
        let Some(entry) = self.records.0.get_mut(&node_id.raw()) else {
            return;
        };
        // A socket that listens on both families reports an IPv4 sender at an IPv4-mapped IPv6
        // address, which the record holds as its IPv4 one.
        let socket = SocketAddr::new(socket.ip().to_canonical(), socket.port());
        let recorded = match socket {
            SocketAddr::V4(_) => entry.enr.udp4_socket().map(SocketAddr::V4),
            SocketAddr::V6(_) => entry.enr.udp6_socket().map(SocketAddr::V6),
        };
        if recorded != Some(socket) {
            return;
        }
        if entry.udp_verified_at < Some(at) {
            entry.udp_verified_at = Some(at);
            self.dirty = true;
        }
    }

    /// Adds `entry`, see [`Records::add`].
    fn add(&mut self, entry: Entry) {
        let added = self.records.add(entry);
        if let Added::New { evicted: Some(evicted) } = &added {
            debug!(target: "bootstore", enr = %evicted.enr, "Bootstore full, evicted a record");
        }
        self.dirty |= added.changed();
    }

    /// Returns the records, in node ID order.
    pub fn iter(&self) -> impl Iterator<Item = &Enr> {
        self.records.0.values().map(|entry| &entry.enr)
    }

    /// Returns the records of peers of the chain `chain_id`: those with an
    /// [`crate::OpStackEnr::OP_CL_KEY`] of that chain and a version this node knows, in node ID
    /// order.
    pub fn valid_peers_with_chain_id(&self, chain_id: u64) -> impl Iterator<Item = &Enr> {
        self.iter().filter(move |enr| crate::EnrValidation::validate(enr, chain_id).is_valid())
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
    pub fn retain(&mut self, mut keep: impl FnMut(&Enr) -> bool) -> usize {
        let len = self.records.0.len();
        self.records.0.retain(|_, entry| keep(&entry.enr));
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
    /// failed write is removed, and the next save tries again. The directory isn't synced after
    /// the rename, so after a power loss the file may be the one from before the save.
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

/// Returns the UDP sockets of `enr`, which a session confirms.
fn udp_sockets(enr: &Enr) -> (Option<SocketAddrV4>, Option<SocketAddrV6>) {
    (enr.udp4_socket(), enr.udp6_socket())
}

/// Returns the TCP sockets of `enr`, which a gossip connection confirms.
fn tcp_sockets(enr: &Enr) -> (Option<SocketAddrV4>, Option<SocketAddrV6>) {
    (enr.tcp4_socket(), enr.tcp6_socket())
}

/// Serializes an optional time as whole seconds since the Unix epoch, the format of the bootstore
/// file's timestamps.
mod unix_secs {
    use serde::{Deserialize, Deserializer, Serializer};
    use std::time::{Duration, SystemTime, UNIX_EPOCH};

    /// Serializes `time` as the whole seconds since the epoch. A time before the epoch, from a
    /// clock set wrong, is saved as the epoch itself: the oldest time the format can hold.
    pub(super) fn serialize<S: Serializer>(
        time: &Option<SystemTime>,
        serializer: S,
    ) -> Result<S::Ok, S::Error> {
        match time {
            Some(time) => serializer
                .serialize_u64(time.duration_since(UNIX_EPOCH).map_or(0, |since| since.as_secs())),
            None => serializer.serialize_none(),
        }
    }

    /// Deserializes a time, failing for one that `SystemTime` can't hold.
    pub(super) fn deserialize<'de, D: Deserializer<'de>>(
        deserializer: D,
    ) -> Result<Option<SystemTime>, D::Error> {
        Option::<u64>::deserialize(deserializer)?
            .map(|secs| {
                UNIX_EPOCH
                    .checked_add(Duration::from_secs(secs))
                    .ok_or_else(|| serde::de::Error::custom("a time out of range"))
            })
            .transpose()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use enr::CombinedKey;
    use std::{fs::File, io::Read};

    /// Makes a record of a new node, at `1.1.1.1` with UDP port 9222 and TCP port 9223.
    fn make_enr() -> Enr {
        make_enr_of(&CombinedKey::generate_secp256k1(), 1, [1, 1, 1, 1], 9222, 9223)
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
        store.add_enr(make_enr());
        store.sync().unwrap();

        let files: Vec<_> = std::fs::read_dir(dir.path())
            .unwrap()
            .map(|entry| entry.unwrap().file_name())
            .collect();
        assert_eq!(files, ["bootstore.json"]);
    }

    /// Makes a record of the node with `key`, with `seq`, at IPv4 `ip` with these UDP and TCP
    /// ports.
    fn make_enr_of(key: &CombinedKey, seq: u64, ip: [u8; 4], udp: u16, tcp: u16) -> Enr {
        Enr::builder().ip4(ip.into()).udp4(udp).tcp4(tcp).seq(seq).build(key).unwrap()
    }

    /// The time `secs` seconds after the Unix epoch: the file keeps whole seconds.
    fn at(secs: u64) -> SystemTime {
        SystemTime::UNIX_EPOCH + std::time::Duration::from_secs(secs)
    }

    /// The records a store saves, and their evidence, are the ones it opens again.
    #[test]
    fn test_saved_records_open_again() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        store.add_enr(make_enr());
        store
            .records
            .add(Entry { udp_verified_at: Some(at(1_000)), ..Entry::candidate(make_enr()) });
        store.records.add(Entry {
            enr: make_enr(),
            udp_verified_at: Some(at(2_000)),
            tcp_verified_at: Some(at(3_000)),
        });
        store.sync().unwrap();

        let (opened, skipped) = BootStore::open(path).unwrap();

        assert_eq!(opened.records, store.records);
        assert!(skipped.is_empty());
    }

    /// The file lists the records in node ID order, whatever order they were added in, so that
    /// saving the same records writes the same file, and a diff of two files shows what changed.
    #[test]
    fn test_file_lists_records_in_node_id_order() {
        let records = [make_enr(), make_enr(), make_enr()];
        let mut forwards = BootStore::in_memory();
        let mut backwards = BootStore::in_memory();
        for record in &records {
            forwards.add_enr(record.clone());
        }
        for record in records.iter().rev() {
            backwards.add_enr(record.clone());
        }

        let json = forwards.records.to_json_vec().unwrap();

        assert_eq!(json, backwards.records.to_json_vec().unwrap());
        let contents = Contents::from_json_slice(&json).unwrap();
        assert_eq!(contents.version(), FILE_VERSION);
        let Contents::V1(listed) = contents else {
            panic!("the file isn't in format version 1");
        };
        let ids: Vec<_> = listed
            .iter()
            .map(|entry| Entry::deserialize(entry).unwrap().enr.node_id().raw())
            .collect();
        assert!(ids.is_sorted(), "the file doesn't list the records in node ID order");
    }

    /// A record with a higher `seq` replaces the stored one, and keeps each timestamp only if the
    /// sockets it proves are unchanged. The UDP and TCP sockets share the record's IP, so a new IP
    /// drops both.
    #[rstest::rstest]
    #[case::same_addresses([1, 1, 1, 1], 9222, 9223, true, true)]
    #[case::new_udp_port([1, 1, 1, 1], 9333, 9223, false, true)]
    #[case::new_tcp_port([1, 1, 1, 1], 9222, 9333, true, false)]
    #[case::new_ip([2, 2, 2, 2], 9222, 9223, false, false)]
    fn test_newer_record_keeps_evidence_of_unchanged_addresses(
        #[case] ip: [u8; 4],
        #[case] udp: u16,
        #[case] tcp: u16,
        #[case] keeps_udp: bool,
        #[case] keeps_tcp: bool,
    ) {
        let key = CombinedKey::generate_secp256k1();
        let mut records = Records::default();
        records.add(Entry {
            enr: make_enr_of(&key, 1, [1, 1, 1, 1], 9222, 9223),
            udp_verified_at: Some(at(1_000)),
            tcp_verified_at: Some(at(2_000)),
        });
        let newer = make_enr_of(&key, 2, ip, udp, tcp);

        let added = records.add(Entry::candidate(newer.clone()));

        assert!(matches!(added, Added::Replaced { .. }), "{added:?}");
        let expected = Entry {
            enr: newer,
            udp_verified_at: keeps_udp.then(|| at(1_000)),
            tcp_verified_at: keeps_tcp.then(|| at(2_000)),
        };
        assert_eq!(records.0.values().collect::<Vec<_>>(), [&expected]);
    }

    /// A session's newer evidence is a change to save, and its older evidence isn't. Adding the
    /// same record again, without evidence, keeps the evidence it has.
    #[test]
    fn test_session_evidence_only_moves_forward() {
        let enr = make_enr();
        let socket = SocketAddr::V4(enr.udp4_socket().unwrap());
        let mut store = BootStore::in_memory();
        store.records.add(Entry {
            udp_verified_at: Some(at(2_000)),
            tcp_verified_at: Some(at(2_000)),
            ..Entry::candidate(enr.clone())
        });
        store.sync().unwrap();

        store.set_udp_verified(enr.node_id(), socket, at(1_000));
        assert!(!store.sync().unwrap(), "older evidence was a change");
        store.add_enr(enr.clone());
        assert!(!store.sync().unwrap(), "the same record without evidence was a change");

        store.set_udp_verified(enr.node_id(), socket, at(3_000));
        assert!(store.sync().unwrap(), "newer evidence wasn't a change");
        let entries: Vec<_> = store.records.0.values().collect();
        assert_eq!(
            entries,
            [&Entry { enr, udp_verified_at: Some(at(3_000)), tcp_verified_at: Some(at(2_000)) }]
        );
    }

    /// A session confirms the UDP address of the stored record only if the record has a UDP
    /// address of the session's family, and the session's packets came from it. discv5 reports
    /// a session for a record without a UDP address of that family without checking one. A
    /// session from another socket, as for an unverifiable record, leaves the evidence as it is.
    /// An IPv4 sender that a socket listening on both families reports at an IPv4-mapped address
    /// is at the record's IPv4 address.
    #[rstest::rstest]
    #[case::matching("1.2.3.4:9222", true)]
    #[case::ipv4_mapped("[::ffff:1.2.3.4]:9222", true)]
    #[case::other_port("1.2.3.4:9000", false)]
    #[case::other_ip("5.6.7.8:9222", false)]
    #[case::no_udp_address_in_the_family("[::1]:9222", false)]
    fn test_session_confirms_only_the_record_udp_address(
        #[case] socket: SocketAddr,
        #[case] confirms: bool,
    ) {
        let enr = make_enr_of(&CombinedKey::generate_secp256k1(), 1, [1, 2, 3, 4], 9222, 9223);
        let mut store = BootStore::in_memory();
        store
            .records
            .add(Entry { udp_verified_at: Some(at(1_000)), ..Entry::candidate(enr.clone()) });
        store.sync().unwrap();

        store.set_udp_verified(enr.node_id(), socket, at(2_000));

        assert_eq!(store.sync().unwrap(), confirms);
        let expected = Entry {
            udp_verified_at: Some(at(if confirms { 2_000 } else { 1_000 })),
            ..Entry::candidate(enr)
        };
        assert_eq!(store.records.0.values().collect::<Vec<_>>(), [&expected]);
    }

    /// A session with a node the store doesn't hold adds nothing: its record comes from
    /// [`BootStore::add_enr`], which the table filter decides on.
    #[test]
    fn test_session_with_an_unknown_node_adds_nothing() {
        let enr = make_enr();
        let mut store = BootStore::in_memory();
        store.sync().unwrap();

        store.set_udp_verified(
            enr.node_id(),
            SocketAddr::V4(enr.udp4_socket().unwrap()),
            at(1_000),
        );

        assert!(store.is_empty());
        assert!(!store.sync().unwrap(), "a session with an unknown node was a change");
    }

    /// A full store evicts the record whose UDP address a session confirmed longest ago, the
    /// records no session confirmed first, so that the ones confirmed recently stay. Of those
    /// confirmed at the same time, the one with the lower node ID goes first, whatever the order
    /// they were added in.
    #[test]
    fn test_evicts_the_record_confirmed_longest_ago() {
        let mut never = [Entry::candidate(make_enr()), Entry::candidate(make_enr())];
        never.sort_by_key(|entry| entry.enr.node_id().raw());
        let [never_lower, never_higher] = never;
        let earlier = Entry { udp_verified_at: Some(at(1_000)), ..Entry::candidate(make_enr()) };
        let later = Entry { udp_verified_at: Some(at(2_000)), ..Entry::candidate(make_enr()) };
        let mut records = Records::default();
        for entry in [&later, &never_higher, &earlier, &never_lower] {
            records.add(entry.clone());
        }

        assert_eq!(records.evict(), Some(never_lower));
        assert_eq!(records.evict(), Some(never_higher));
        assert_eq!(records.evict(), Some(earlier));
        assert_eq!(records.evict(), Some(later));
    }

    /// A store holds one record per node, the one with the highest `seq`. A node that changes its
    /// record, such as for a new address, then takes one slot, and gossip dials only its current
    /// address. Otherwise one node could push the others out by bumping its `seq`. A new record
    /// with a lower `seq` than the stored one is older, and is ignored. So is a new record that
    /// differs from the stored one but has the same `seq`: the stored one is kept.
    #[rstest::rstest]
    #[case::higher_seq_replaces(1, 2, Kept::Second)]
    #[case::lower_seq_is_ignored(2, 1, Kept::First)]
    #[case::same_seq_is_ignored(1, 1, Kept::First)]
    fn test_one_record_per_node(
        #[case] first_seq: u64,
        #[case] second_seq: u64,
        #[case] kept: Kept,
    ) {
        let key = CombinedKey::generate_secp256k1();
        // Different UDP ports, so that the two records differ even with the same `seq`.
        let first = make_enr_of(&key, first_seq, [1, 1, 1, 1], 9222, 9223);
        let second = make_enr_of(&key, second_seq, [1, 1, 1, 1], 9000, 9223);
        let mut store = BootStore::in_memory();

        store.add_enr(first.clone());
        store.add_enr(second.clone());

        let expected = match kept {
            Kept::First => first,
            Kept::Second => second,
        };
        assert_eq!(store.iter().collect::<Vec<_>>(), [&expected]);
    }

    /// Which of two records of one node a store keeps, see [`test_one_record_per_node`].
    #[derive(Debug)]
    enum Kept {
        First,
        Second,
    }

    /// An entry that isn't a record is skipped, and returned with the reason, and the others are
    /// kept.
    #[test]
    fn test_open_skips_entries_that_are_not_records() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let kept = make_enr();
        let json = serde_json::json!({
            "version": 1,
            "records": [{"enr": "enr:-not-base64!"}, {"enr": kept.to_base64()}, 42],
        });
        std::fs::write(&path, json.to_string()).unwrap();

        let (mut store, skipped) = BootStore::open(path.clone()).unwrap();

        assert_eq!(store.iter().collect::<Vec<_>>(), [&kept]);
        assert!(skipped.iter().all(|skipped| matches!(skipped.reason, SkipReason::Invalid(_))));
        let skipped: Vec<_> = skipped.into_iter().map(|skipped| skipped.entry).collect();
        assert_eq!(
            skipped,
            [serde_json::json!({"enr": "enr:-not-base64!"}), serde_json::json!(42)]
        );

        // The next save writes the file without them.
        assert!(store.sync().unwrap(), "skipped entries weren't a change to save");
        let saved: serde_json::Value =
            serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
        assert_eq!(
            saved,
            serde_json::json!({"version": FILE_VERSION, "records": [{"enr": kept.to_base64()}]})
        );
    }

    /// Of the entries for one node, only the record with the highest `seq` is kept, and the others
    /// are returned as skipped, with the reason: an older record, another with the same `seq`, or
    /// the same record twice. An earlier version saved every record a node had.
    #[test]
    fn test_open_skips_records_it_does_not_keep() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let key = CombinedKey::generate_secp256k1();
        let older = make_enr_of(&key, 1, [1, 1, 1, 1], 9222, 9223);
        let newer = make_enr_of(&key, 2, [1, 1, 1, 1], 9222, 9223);
        let conflicting = make_enr_of(&key, 2, [2, 2, 2, 2], 9222, 9223);
        let json = serde_json::json!([older, newer, conflicting, newer]);
        std::fs::write(&path, json.to_string()).unwrap();

        let (store, skipped) = BootStore::open(path).unwrap();

        assert_eq!(store.iter().collect::<Vec<_>>(), [&newer]);
        let skipped: Vec<_> = skipped
            .into_iter()
            .map(|skipped| (skipped.entry, skipped.reason.to_string()))
            .collect();
        let reason = |reason: SkipReason| reason.to_string();
        assert_eq!(
            skipped,
            [
                (serde_json::json!(older), reason(SkipReason::Stale)),
                (serde_json::json!(conflicting), reason(SkipReason::Stale)),
                (serde_json::json!(newer), reason(SkipReason::Duplicate)),
            ]
        );
    }

    /// A file in the format earlier versions wrote, a list of records, opens as records without
    /// evidence, and the next save writes it in the current format.
    #[test]
    fn test_open_list_of_records() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let listed = make_enr();
        std::fs::write(&path, serde_json::json!([listed]).to_string()).unwrap();

        let (mut store, skipped) = BootStore::open(path.clone()).unwrap();

        assert!(skipped.is_empty());
        let entries: Vec<_> = store.records.0.values().collect();
        assert_eq!(entries, [&Entry::candidate(listed.clone())]);
        assert!(store.sync().unwrap(), "a file in an earlier format wasn't a change to save");
        let saved: serde_json::Value =
            serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
        assert_eq!(
            saved,
            serde_json::json!({"version": FILE_VERSION, "records": [{"enr": listed.to_base64()}]})
        );
    }

    /// A file in format version 1 opens with its evidence. Its contents are written out, not
    /// saved by the store, so that the store keeps reading version 1 once it writes a later one.
    #[test]
    fn test_open_version_1() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let key = CombinedKey::generate_secp256k1();
        let verified = make_enr_of(&key, 1, [1, 1, 1, 1], 9222, 9223);
        let candidate = make_enr();
        let contents = format!(
            r#"{{"version": 1, "records": [
                {{"enr": "{}", "udp_verified_at": 100, "tcp_verified_at": 200}},
                {{"enr": "{}"}}
            ]}}"#,
            verified.to_base64(),
            candidate.to_base64(),
        );
        std::fs::write(&path, contents).unwrap();

        let (store, skipped) = BootStore::open(path).unwrap();

        assert!(skipped.is_empty());
        let mut expected = Records::default();
        expected.add(Entry {
            enr: verified,
            udp_verified_at: Some(at(100)),
            tcp_verified_at: Some(at(200)),
        });
        expected.add(Entry::candidate(candidate));
        assert_eq!(store.records, expected);
    }

    /// A file in a format version after the ones this version reads, which a newer version wrote,
    /// fails to open, and is left as it is, for the version that wrote it.
    #[test]
    fn test_open_leaves_a_file_of_a_newer_version() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let contents = serde_json::json!({"version": FILE_VERSION + 1, "records": []}).to_string();
        std::fs::write(&path, &contents).unwrap();

        let err = BootStore::open(path.clone()).unwrap_err();

        assert!(
            matches!(err, BootStoreError::NewerVersion { version, .. } if version == FILE_VERSION + 1),
            "{err}"
        );
        assert_eq!(std::fs::read_to_string(path).unwrap(), contents);
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

    /// A file that isn't a bootstore fails to open, and is left as it is: it may be another file
    /// than the bootstore, given by mistake.
    #[rstest::rstest]
    #[case::not_json("not json")]
    #[case::object(r#"{"peers": []}"#)]
    #[case::no_version(r#"{"records": []}"#)]
    #[case::version_0(r#"{"version": 0, "records": []}"#)]
    #[case::version_1_without_records(r#"{"version": 1}"#)]
    #[case::binary("\u{0}\u{1}")]
    fn test_open_leaves_a_file_that_is_not_a_bootstore(#[case] contents: &str) {
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
        store.add_enr(make_enr());
        store.sync().unwrap();
        let before = std::fs::read(&path).unwrap();
        let mut reader = File::open(&path).unwrap();

        store.add_enr(make_enr());
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
        store.add_enr(make_enr());
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
        store.add_enr(make_enr());
        store.sync().unwrap();
        let inode = || std::fs::metadata(&path).unwrap().ino();

        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        let opened = inode();
        store.sync().unwrap();
        assert_eq!(inode(), opened, "a save rewrote the file it had just opened");

        store.add_enr(make_enr());
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

        let added = make_enr();
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
    /// Each save adds a record, so that the store has something to write each time. The reader
    /// stops at the first file it sees empty or cut off: one is enough.
    #[test]
    fn test_reader_never_sees_a_partial_file() {
        use std::sync::{
            Arc,
            atomic::{AtomicBool, Ordering},
        };

        const RECORDS: usize = 500;
        const SAVES: usize = 200;
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let (mut store, _) = BootStore::open(path.clone()).unwrap();
        for _ in 0..RECORDS {
            store.add_enr(make_enr());
        }
        store.sync().unwrap();

        let done = Arc::new(AtomicBool::new(false));
        let reader = std::thread::spawn({
            let done = Arc::clone(&done);
            move || {
                while !done.load(Ordering::Relaxed) {
                    let json = std::fs::read(&path).unwrap();
                    let whole = matches!(
                        Records::parse_json_slice(&json),
                        Ok(Parsed { records, .. }) if records.0.len() >= RECORDS
                    );
                    if !whole {
                        return true;
                    }
                }
                false
            }
        });
        for _ in 0..SAVES {
            store.add_enr(make_enr());
            store.sync().unwrap();
        }
        done.store(true, Ordering::Relaxed);

        assert!(!reader.join().unwrap(), "a reader saw the bootstore empty or cut off");
    }

    /// A directory can't be read as the bootstore.
    #[test]
    fn test_open_directory_fails() {
        let dir = tempfile::tempdir().unwrap();

        let err = BootStore::open(dir.path().to_path_buf()).unwrap_err();

        assert!(matches!(err, BootStoreError::Open { .. }), "{err}");
    }
}
