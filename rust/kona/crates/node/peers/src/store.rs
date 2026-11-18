//! Bootnode Store

use discv5::Enr;
use enr::NodeId;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use serde_with::{InspectError, VecSkipError};
use std::{
    cmp::{Ordering, Reverse},
    collections::BTreeMap,
    hash::{BuildHasher, RandomState},
    io::{self, Write},
    net::{SocketAddr, SocketAddrV4, SocketAddrV6},
    path::{Path, PathBuf},
    time::{Duration, SystemTime},
};
use tempfile::NamedTempFile;

/// The maximum number of peers that can be stored in the bootstore.
const MAX_PEERS: usize = 2048;

/// How long the evidence for an address is kept before new evidence replaces it. Gossip repeats
/// its evidence at each ping on a connection it dialed, every 15 seconds, and discovery at each
/// store interval for the nodes its table holds as connected: refreshing each time would make the
/// store save at every store interval.
const VERIFIED_REFRESH: Duration = Duration::from_secs(5 * 60);

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
/// When a new node would make the records exceed `max_peers`, one is evicted to make room for it,
/// see [`Records::evict`].
#[derive(Debug, Clone, PartialEq, Eq)]
struct Records {
    /// The records, by the bytes of their node IDs.
    entries: BTreeMap<[u8; 32], BootStoreEntry>,
    /// The most records there may be, [`MAX_PEERS`] by default.
    max_peers: usize,
}

impl Default for Records {
    fn default() -> Self {
        Self { entries: BTreeMap::new(), max_peers: MAX_PEERS }
    }
}

impl Records {
    /// Parses the contents of a bootstore file, keeping one record for each node, see
    /// [`Self::add`]. An entry that isn't a valid record is skipped, see [`Contents`].
    ///
    /// Empty contents hold no records: an earlier version truncated the file before it wrote it,
    /// and a crash in between left it empty.
    fn parse_json_slice(json: &[u8]) -> Result<Self, ParseError> {
        let mut records = Self::default();
        if json.iter().all(u8::is_ascii_whitespace) {
            return Ok(records);
        }
        let entries: Vec<BootStoreEntry> = match Contents::from_json_slice(json)? {
            Contents::V0(enrs) => enrs.into_iter().map(BootStoreEntry::candidate).collect(),
            Contents::V1(entries) => entries,
        };
        for entry in entries {
            records.add(entry);
        }
        Ok(records)
    }

    /// Returns the contents of a bootstore file that holds these records, in format version
    /// [`FILE_VERSION`], see [`Contents::V1`], in node ID order.
    fn to_json_vec(&self) -> Result<Vec<u8>, serde_json::Error> {
        let records: Vec<_> = self.entries.values().collect();
        serde_json::to_vec(&serde_json::json!({"version": FILE_VERSION, "records": records}))
    }

    /// Adds `entry`, keeping one record for each node, the one with the highest `seq`:
    ///
    /// - A record of a new node is added, evicting another if there are then too many.
    /// - A record with a higher `seq` replaces the stored one, and keeps its evidence for the
    ///   addresses it keeps, see [`BootStoreEntry::replaced_by`].
    /// - The same record replaces the stored one too, which takes only its newer evidence.
    /// - A record with a lower `seq`, or another with the same `seq`, is ignored.
    fn add(&mut self, entry: BootStoreEntry) -> Added {
        let key = entry.key();
        if let Some(stored) = self.entries.get_mut(&key) {
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
        let evicted = if self.entries.len() >= self.max_peers { self.evict() } else { None };
        self.entries.insert(key, entry);
        Added::New { evicted }
    }

    /// Removes and returns the record to evict to make room for a new one, by tier: a candidate,
    /// which only sessions confirmed, and a record gossip reached only once there is none.
    /// Anyone can start sessions, under fresh keys, so these alone shouldn't push out the peers
    /// gossip reached, unless the store holds nothing else.
    ///
    /// Of the candidates, the one whose UDP address a session confirmed longest ago goes first,
    /// those that no session confirmed before any. Of the reached records, the one gossip reached
    /// longest ago. Of those confirmed at the same time, or not at all, those without a session
    /// since the store was opened go first, and then the one whose session started last: the
    /// attacker picks the rate of fresh sessions, so evicting the oldest first would let a flood
    /// evict every session before it lasts long enough to confirm its record.
    ///
    /// Of those that still tie, any of them, drawn afresh at each eviction by hashing their node
    /// IDs with a random key. Not the first in node ID order: a node ID is the hash of a key, and
    /// an attacker could grind keys until its records sort after the others.
    fn evict(&mut self) -> Option<BootStoreEntry> {
        let tie_break = RandomState::new();
        let (&key, _) = self.entries.iter().min_by_key(|&(key, entry)| {
            let reached = entry.tcp_verified_at.is_some();
            let session =
                (entry.session_started_at.is_some(), entry.session_started_at.map(Reverse));
            let evidence = (entry.tcp_verified_at, entry.udp_verified_at, session);
            (reached, evidence, tie_break.hash_one(key))
        })?;
        self.entries.remove(&key)
    }
}

/// A record a bootstore holds, with the evidence that the node it names is at its addresses. A
/// record without `tcp_verified_at` is a candidate: nothing showed that gossip can reach it.
///
/// Only the store changes the evidence, see [`BootStore::set_udp_verified`].
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BootStoreEntry {
    /// The record.
    enr: Enr,
    /// When a discovery session last confirmed the record's UDP address.
    ///
    /// Ranks the candidates when a full store evicts one, and keeps the record when it expires.
    #[serde(default, skip_serializing_if = "Option::is_none", with = "unix_secs")]
    udp_verified_at: Option<SystemTime>,
    /// When a gossip connection we dialed last reached the record's TCP address, which
    /// authenticated the peer ID of its key.
    ///
    /// Sets the peers gossip reached above the candidates, which anyone can make under fresh
    /// keys: evicted last, seeded first, and redialed.
    #[serde(default, skip_serializing_if = "Option::is_none", with = "unix_secs")]
    tcp_verified_at: Option<SystemTime>,
    /// When the discovery session with the node at the record's UDP address started, if one did
    /// since the store was opened. Not saved: the session ends with the process.
    ///
    /// Lets a session confirm the record only once it has lasted, so that a flood of fresh
    /// sessions can't outrank the ones that last.
    #[serde(skip)]
    session_started_at: Option<SystemTime>,
    /// Until when the record is kept without evidence, see [`BootStore::expire`]: a while after
    /// the last discovery session with its node started, from any socket. Not saved, as
    /// [`Self::session_started_at`].
    ///
    /// Gives a fresh session the time to confirm the record, or gossip to reach it, before
    /// expiry drops it: discovery reports a session only once.
    #[serde(skip)]
    expiry_grace: Option<SystemTime>,
}

impl BootStoreEntry {
    /// Returns the record.
    pub const fn enr(&self) -> &Enr {
        &self.enr
    }

    /// Returns when a discovery session last confirmed the record's UDP address, if one did.
    pub const fn udp_verified_at(&self) -> Option<SystemTime> {
        self.udp_verified_at
    }

    /// Returns when a gossip connection we dialed last reached the record's TCP address, which
    /// authenticated the peer ID of its key, if one did.
    pub const fn tcp_verified_at(&self) -> Option<SystemTime> {
        self.tcp_verified_at
    }

    /// Returns when the discovery session with the node at the record's UDP address started, if
    /// one did since the store was opened.
    pub const fn session_started_at(&self) -> Option<SystemTime> {
        self.session_started_at
    }

    /// Returns an entry for `enr` without evidence.
    const fn candidate(enr: Enr) -> Self {
        Self {
            enr,
            udp_verified_at: None,
            tcp_verified_at: None,
            session_started_at: None,
            expiry_grace: None,
        }
    }

    /// Returns the entry's key in [`Records`]: the bytes of its node ID, as `NodeId` isn't ordered.
    const fn key(&self) -> [u8; 32] {
        self.enr.node_id().raw()
    }

    /// Returns `newer`, a record of the same node with a higher `seq`, or the same record, with the
    /// evidence of this entry for the addresses that `newer` keeps, where it is newer: a timestamp
    /// is kept only if the sockets it proves are unchanged. A record has one IP for each family,
    /// which UDP and TCP share, so a new IP drops both, and a new TCP port only `tcp_verified_at`.
    /// The same record keeps all its sockets, so it takes the newer of each timestamp. The later
    /// `expiry_grace` is kept whatever changed, as it is for the node, not for an address.
    fn replaced_by(&self, mut newer: Self) -> Self {
        newer.expiry_grace = newer.expiry_grace.max(self.expiry_grace);
        if udp_sockets(&self.enr) == udp_sockets(&newer.enr) {
            newer.udp_verified_at = newer.udp_verified_at.max(self.udp_verified_at);
            newer.session_started_at = newer.session_started_at.max(self.session_started_at);
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
        evicted: Option<BootStoreEntry>,
    },
    /// It replaced an older record of its node.
    Replaced {
        /// The entry it replaced.
        old: BootStoreEntry,
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

/// The contents of a bootstore file, in each format version.
///
/// An entry that isn't a valid record is skipped, and logged, rather than failing the file: the
/// records come from other nodes, and one that a later version can't read shouldn't stop the node
/// from starting. Without them, the node finds its peers again through discovery, as op-node does
/// when geth drops its node database for another version.
#[derive(Debug)]
enum Contents {
    /// Format version 0, which has no version field: a list of records, without evidence.
    V0(Vec<Enr>),
    /// Format version 1, `{"version": 1, "records": [...]}`.
    V1(Vec<BootStoreEntry>),
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

        /// The fields of format version 1 besides the version.
        #[serde_with::serde_as]
        #[derive(Deserialize)]
        struct V1 {
            #[serde_as(as = "VecSkipError<_, LogInvalid>")]
            records: Vec<BootStoreEntry>,
        }

        let json: Value = serde_json::from_slice(json).map_err(ParseError::Json)?;
        if json.is_array() {
            return serde_with::As::<VecSkipError<serde_with::Same, LogInvalid>>::deserialize(json)
                .map(Self::V0)
                .map_err(ParseError::Json);
        }
        let Versioned { version } = Versioned::deserialize(&json).map_err(ParseError::Json)?;
        match version {
            0 => Err(ParseError::Json(serde::de::Error::custom(
                "format version 0 is a list of records, without a version",
            ))),
            1 => V1::deserialize(json)
                .map(|V1 { records }| Self::V1(records))
                .map_err(ParseError::Json),
            FIRST_NEWER_VERSION.. => Err(ParseError::Newer(version)),
        }
    }
}

/// Logs an entry of a bootstore file that isn't a valid record, which [`Contents`] skips. Only a
/// hand edit, or a record that this version can't read, makes one, so it is a WARN. The next save
/// leaves it out.
struct LogInvalid;

impl InspectError for LogInvalid {
    fn inspect_error(error: impl serde::de::Error) {
        warn!(target: "bootstore", %error, "Dropping a bootstore entry that isn't a node record");
    }
}

/// Why the contents of a file can't be read as a bootstore.
#[derive(Debug)]
enum ParseError {
    /// It isn't in any format version up to [`FILE_VERSION`].
    Json(serde_json::Error),
    /// It is in a format version after [`FILE_VERSION`].
    Newer(u64),
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
    /// Whether the file differs from what a save would write: the records changed since they were
    /// last saved, or the file isn't as the store writes it, see [`Self::open`].
    dirty: bool,
    /// How long the evidence for an address is kept before new evidence replaces it,
    /// [`VERIFIED_REFRESH`] by default.
    verified_refresh: Duration,
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
        Self {
            path: None,
            records: Records::default(),
            dirty: false,
            verified_refresh: VERIFIED_REFRESH,
        }
    }

    /// Opens the store saved at `path`. It only reads the file, so that others can read it while
    /// the node saves it. A missing file holds no records.
    ///
    /// Of the records of a node, it keeps the one with the highest `seq`. A list of records, the
    /// format of earlier versions, is read as records without evidence. An entry that isn't a
    /// valid record is skipped, and logged at WARN. If the file isn't what a save would write, the
    /// next [`Self::sync`] writes it, and creates its directory if needed.
    ///
    /// Fails if the file can't be read, isn't a bootstore, or is in a format version a newer
    /// version of the node wrote. Such a file is left as it is.
    pub fn open(path: PathBuf) -> Result<Self, BootStoreError> {
        let json = match std::fs::read(&path) {
            Ok(json) => json,
            Err(err) if err.kind() == io::ErrorKind::NotFound => Vec::new(),
            Err(source) => return Err(BootStoreError::Open { path, source }),
        };
        let records = Records::parse_json_slice(&json).map_err(|err| match err {
            ParseError::Json(source) => BootStoreError::Parse { path: path.clone(), source },
            ParseError::Newer(version) => {
                BootStoreError::NewerVersion { path: path.clone(), version }
            }
        })?;
        let dirty = records.to_json_vec().map_or(true, |saved| saved != json);
        Ok(Self { path: Some(path), records, dirty, verified_refresh: VERIFIED_REFRESH })
    }

    /// Sets the most records the store holds, 2048 by default, which should be at least one: a new
    /// record always gets in. If it holds more, it evicts them, as it would to make room for new
    /// ones, and the next [`Self::sync`] saves the file without them.
    pub fn with_max_peers(mut self, max: usize) -> Self {
        self.records.max_peers = max;
        while self.records.entries.len() > max {
            if self.records.evict().is_none() {
                break;
            }
            self.dirty = true;
        }
        self
    }

    /// Sets how long the evidence for an address is kept before new evidence replaces it, five
    /// minutes by default. Discovery and gossip repeat their evidence, and refreshing it each time
    /// would make the store save at every store interval.
    pub const fn with_verified_refresh(mut self, refresh: Duration) -> Self {
        self.verified_refresh = refresh;
        self
    }

    /// Adds `enr` without evidence, as a candidate. The store keeps one record for each node, the
    /// one with the highest `seq`, so `enr` replaces the stored record of its node only if its
    /// `seq` is higher. A new node evicts another if there are then too many. The file changes at
    /// the next [`Self::sync`].
    pub fn add_enr(&mut self, enr: Enr) {
        self.add(BootStoreEntry::candidate(enr));
    }

    /// Notes that a discovery session with the node `node_id`, whose packets came from `socket`,
    /// started at `at`, if the record's UDP address of the family of `socket` is `socket`. discv5
    /// doesn't check that for a record without a UDP address of that family, nor for an
    /// unverifiable record. A session from another socket says nothing about the record's
    /// address, and ends the count for the session before it: the node now talks to discovery
    /// from elsewhere, so that session lasting no longer shows the record's address works. Does
    /// nothing for a node the store doesn't hold.
    ///
    /// The session confirms the address only once it has lasted, see [`Self::set_udp_verified`].
    /// That keeps the peers whose sessions last from being crowded out of a full store: anyone
    /// can start sessions under fresh keys, and if each confirmed its record at once, a flood of
    /// them would always be the most recently confirmed.
    ///
    /// A session from any socket keeps the record without evidence until `grace` after `at`, see
    /// [`Self::expire`], so that it has the time to confirm the record, or gossip to reach it.
    pub fn session_started(
        &mut self,
        node_id: NodeId,
        socket: SocketAddr,
        at: SystemTime,
        grace: Duration,
    ) {
        if let Some(entry) = self.records.entries.get_mut(&node_id.raw()) {
            entry.session_started_at = holds_socket(udp_sockets(&entry.enr), socket).then_some(at);
            entry.expiry_grace = entry.expiry_grace.max(at.checked_add(grace));
        }
    }

    /// Notes that discovery still holds the session with the node `node_id` at `socket` at `at`,
    /// so that the session confirms the UDP address of its stored record, if it started at that
    /// address, see [`Self::session_started`], at least as long ago as
    /// [`Self::with_verified_refresh`] sets, five minutes by default. geth likewise saves a node
    /// only once it has answered several pings.
    ///
    /// Does nothing for a node the store doesn't hold, nor until the evidence it has is as old as
    /// that either. A newer time changes the file at the next [`Self::sync`].
    pub fn set_udp_verified(&mut self, node_id: NodeId, socket: SocketAddr, at: SystemTime) {
        let Some(entry) = self.records.entries.get_mut(&node_id.raw()) else {
            return;
        };
        let lasted = entry
            .session_started_at
            .and_then(|started| started.checked_add(self.verified_refresh))
            .is_some_and(|due| due <= at);
        if lasted && holds_socket(udp_sockets(&entry.enr), socket) {
            self.dirty |= refresh(&mut entry.udp_verified_at, at, self.verified_refresh);
        }
    }

    /// Notes that gossip reached the node `node_id` at `addr` at `at`, over a connection it
    /// dialed, which authenticated the node's key: it confirmed the TCP address of its stored
    /// record, if the record's TCP address of the family of `addr` is `addr`. A connection to
    /// another address leaves the record's evidence as it is.
    ///
    /// Does nothing for a node the store doesn't hold, nor until the evidence it has is as old as
    /// [`Self::with_verified_refresh`] sets. A newer time changes the file at the next
    /// [`Self::sync`].
    pub fn set_tcp_verified(&mut self, node_id: NodeId, addr: SocketAddr, at: SystemTime) {
        let Some(entry) = self.records.entries.get_mut(&node_id.raw()) else {
            return;
        };
        if holds_socket(tcp_sockets(&entry.enr), addr) {
            self.dirty |= refresh(&mut entry.tcp_verified_at, at, self.verified_refresh);
        }
    }

    /// Adds `entry`, see [`Records::add`].
    fn add(&mut self, entry: BootStoreEntry) {
        let added = self.records.add(entry);
        match &added {
            Added::New { evicted: Some(evicted) } => {
                debug!(target: "bootstore", enr = %evicted.enr, "Bootstore full, evicted a record");
            }
            Added::Replaced { old } => {
                if let Some(new) = self.records.entries.get(&old.key()) {
                    debug!(target: "bootstore", old = ?old.enr, new = ?new.enr, "Replaced a record with its node's newer one");
                }
            }
            Added::New { evicted: None } | Added::Same { .. } | Added::Stale => {}
        }
        self.dirty |= added.changed();
    }

    /// Returns the entry of the node `node_id`, if the store holds one.
    pub fn get(&self, node_id: NodeId) -> Option<&BootStoreEntry> {
        self.records.entries.get(&node_id.raw())
    }

    /// Returns the entries, the records with their evidence, in node ID order.
    pub fn entries(&self) -> impl Iterator<Item = &BootStoreEntry> {
        self.records.entries.values()
    }

    /// Returns the records, in node ID order.
    pub fn iter(&self) -> impl Iterator<Item = &Enr> {
        self.records.entries.values().map(|entry| &entry.enr)
    }

    /// Returns the entries of peers of the chain `chain_id`: those whose records have an
    /// [`crate::OpStackEnr::OP_CL_KEY`] of that chain and a version this node knows, in node ID
    /// order.
    pub fn valid_peers_with_chain_id(
        &self,
        chain_id: u64,
    ) -> impl Iterator<Item = &BootStoreEntry> {
        self.entries()
            .filter(move |entry| crate::EnrValidation::validate(&entry.enr, chain_id).is_valid())
    }

    /// Returns the number of peers in the in-memory store.
    pub fn len(&self) -> usize {
        self.records.entries.len()
    }

    /// Returns if the in-memory store is empty.
    pub fn is_empty(&self) -> bool {
        self.records.entries.is_empty()
    }

    /// Dates the evidence dated after `at`, which a clock that ran ahead wrote, `at`, so that it
    /// ages from then on: once the clock is set right, a peer that never answers again would
    /// otherwise keep it until the clock catches up. The file changes at the next [`Self::sync`].
    pub fn cap_evidence(&mut self, at: SystemTime) {
        for entry in self.records.entries.values_mut() {
            for evidence in [&mut entry.udp_verified_at, &mut entry.tcp_verified_at] {
                if let Some(evidence) = evidence.as_mut().filter(|evidence| **evidence > at) {
                    *evidence = at;
                    self.dirty = true;
                }
            }
        }
    }

    /// Forgets the evidence older than `ttl` at `now`, one timestamp at a time, and drops the
    /// records it leaves without any, unless they are still in the grace a session gave them, see
    /// [`Self::session_started`]. Returns how many it dropped. The file changes at the next
    /// [`Self::sync`].
    ///
    /// Evidence dated after `now` is dated `now`, see [`Self::cap_evidence`].
    pub fn expire(&mut self, now: SystemTime, ttl: Duration) -> usize {
        self.cap_evidence(now);
        let before = now.checked_sub(ttl);
        let len = self.records.entries.len();
        let mut changed = false;
        self.records.entries.retain(|_, entry| {
            for evidence in [&mut entry.udp_verified_at, &mut entry.tcp_verified_at] {
                if evidence.take_if(|at| before.is_some_and(|before| *at < before)).is_some() {
                    changed = true;
                }
            }
            entry.udp_verified_at.is_some() ||
                entry.tcp_verified_at.is_some() ||
                entry.expiry_grace.is_some_and(|until| until > now)
        });
        let dropped = len - self.records.entries.len();
        self.dirty |= changed || dropped > 0;
        dropped
    }

    /// Keeps only the records `keep` accepts, and returns how many it removed. The file changes at
    /// the next [`Self::sync`].
    pub fn retain(&mut self, mut keep: impl FnMut(&Enr) -> bool) -> usize {
        let len = self.records.entries.len();
        self.records.entries.retain(|_, entry| keep(&entry.enr));
        let removed = len - self.records.entries.len();
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

/// Sets `evidence` to `at`, which is now, unless the time it holds is less than `delay` before
/// `at`. Returns whether it changed. A time after `at` is replaced: it can't be true, as the file
/// was written while the clock ran ahead, or edited.
fn refresh(evidence: &mut Option<SystemTime>, at: SystemTime, delay: Duration) -> bool {
    let fresh = |last: SystemTime| last <= at && last.checked_add(delay).is_none_or(|due| due > at);
    if evidence.is_some_and(fresh) {
        return false;
    }
    *evidence = Some(at);
    true
}

/// Returns whether `addr` is the socket of its family in `sockets`, a record's
/// [`udp_sockets`] or [`tcp_sockets`]. A socket that listens on both families reports an IPv4
/// peer at an IPv4-mapped IPv6 address, which a record holds as its IPv4 one.
fn holds_socket((v4, v6): (Option<SocketAddrV4>, Option<SocketAddrV6>), addr: SocketAddr) -> bool {
    match SocketAddr::new(addr.ip().to_canonical(), addr.port()) {
        SocketAddr::V4(addr) => v4 == Some(addr),
        SocketAddr::V6(addr) => v6 == Some(addr),
    }
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

    /// The TTL of the expiry tests.
    const TTL: Duration = Duration::from_secs(1_000);

    /// The time the expiry tests expire at, in seconds after the epoch, well after [`TTL`].
    const NOW: u64 = 10_000;

    /// A record of a new node, at `1.1.1.1` with UDP port 9222 and TCP port 9223.
    fn enr_of_new_node() -> Enr {
        enr_of(&CombinedKey::generate_secp256k1(), 1, [1, 1, 1, 1], 9222, 9223)
    }

    /// A store opened at a path whose directory doesn't exist is empty, and writes nothing until
    /// it saves, which creates the directory and the file.
    #[test]
    fn test_save_creates_the_directory() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("datadir").join("bootstore.json");

        let mut store = BootStore::open(path.clone()).unwrap();
        assert!(store.is_empty());
        assert!(!path.parent().unwrap().exists(), "opening the store wrote to the disk");

        assert!(store.sync().unwrap(), "a save didn't create the missing file");
        assert!(path.is_file());
    }

    /// Opening and saving leave no temporary file behind, only the bootstore.
    #[test]
    fn test_open_and_save_leave_no_temporary_file() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");

        let mut store = BootStore::open(path).unwrap();
        store.add_enr(enr_of_new_node());
        store.sync().unwrap();

        let files: Vec<_> = std::fs::read_dir(dir.path())
            .unwrap()
            .map(|entry| entry.unwrap().file_name())
            .collect();
        assert_eq!(files, ["bootstore.json"]);
    }

    /// A record of the node with `key`, with `seq`, at IPv4 `ip` with these UDP and TCP ports.
    fn enr_of(key: &CombinedKey, seq: u64, ip: [u8; 4], udp: u16, tcp: u16) -> Enr {
        Enr::builder().ip4(ip.into()).udp4(udp).tcp4(tcp).seq(seq).build(key).unwrap()
    }

    /// Notes in `store` a session with the node `node_id` from `socket` that started
    /// [`VERIFIED_REFRESH`] before `at`, and that discovery still held at `at`, so that it confirms
    /// the record's UDP address at `at`, if `socket` is that address.
    fn confirm_session(store: &mut BootStore, node_id: NodeId, socket: SocketAddr, at: SystemTime) {
        store.session_started(node_id, socket, at - VERIFIED_REFRESH, Duration::ZERO);
        store.set_udp_verified(node_id, socket, at);
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
        let mut store = BootStore::open(path.clone()).unwrap();
        store.add_enr(enr_of_new_node());
        store.records.add(BootStoreEntry {
            udp_verified_at: Some(at(1_000)),
            ..BootStoreEntry::candidate(enr_of_new_node())
        });
        store.records.add(BootStoreEntry {
            enr: enr_of_new_node(),
            udp_verified_at: Some(at(2_000)),
            tcp_verified_at: Some(at(3_000)),
            session_started_at: None,
            expiry_grace: None,
        });
        store.sync().unwrap();

        let opened = BootStore::open(path).unwrap();

        assert_eq!(opened.records, store.records);
        assert!(!opened.dirty, "a file the store saved wasn't as it writes it");
    }

    /// The file lists the records in node ID order, whatever order they were added in, so that
    /// saving the same records writes the same file, and a diff of two files shows what changed.
    #[test]
    fn test_file_lists_records_in_node_id_order() {
        let records = [enr_of_new_node(), enr_of_new_node(), enr_of_new_node()];
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
        let saved: Value = serde_json::from_slice(&json).unwrap();
        assert_eq!(saved["version"], FILE_VERSION);
        let Contents::V1(listed) = Contents::from_json_slice(&json).unwrap() else {
            panic!("the file isn't in format version 1");
        };
        let ids: Vec<_> = listed.iter().map(BootStoreEntry::key).collect();
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
        let stored = BootStoreEntry {
            enr: enr_of(&key, 1, [1, 1, 1, 1], 9222, 9223),
            udp_verified_at: Some(at(1_000)),
            tcp_verified_at: Some(at(2_000)),
            session_started_at: None,
            expiry_grace: None,
        };
        records.add(stored.clone());
        let newer = enr_of(&key, 2, ip, udp, tcp);

        let added = records.add(BootStoreEntry::candidate(newer.clone()));

        assert!(matches!(&added, Added::Replaced { old } if *old == stored), "{added:?}");
        let expected = BootStoreEntry {
            enr: newer,
            udp_verified_at: keeps_udp.then(|| at(1_000)),
            tcp_verified_at: keeps_tcp.then(|| at(2_000)),
            session_started_at: None,
            expiry_grace: None,
        };
        assert_eq!(records.entries.values().collect::<Vec<_>>(), [&expected]);
    }

    /// A session's newer evidence is a change to save. Adding the same record again, without
    /// evidence, keeps the evidence it has.
    #[test]
    fn test_session_evidence_only_moves_forward() {
        let enr = enr_of_new_node();
        let socket = SocketAddr::V4(enr.udp4_socket().unwrap());
        let mut store = BootStore::in_memory();
        store.records.add(BootStoreEntry {
            udp_verified_at: Some(at(2_000)),
            tcp_verified_at: Some(at(2_000)),
            ..BootStoreEntry::candidate(enr.clone())
        });
        store.sync().unwrap();

        store.add_enr(enr.clone());
        assert!(!store.sync().unwrap(), "the same record without evidence was a change");

        confirm_session(&mut store, enr.node_id(), socket, at(3_000));
        assert!(store.sync().unwrap(), "newer evidence wasn't a change");
        let entries: Vec<_> = store.records.entries.values().collect();
        assert_eq!(
            entries,
            [&BootStoreEntry {
                enr,
                udp_verified_at: Some(at(3_000)),
                tcp_verified_at: Some(at(2_000)),
                session_started_at: Some(at(3_000) - VERIFIED_REFRESH),
                expiry_grace: Some(at(3_000) - VERIFIED_REFRESH),
            }]
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
        let enr = enr_of(&CombinedKey::generate_secp256k1(), 1, [1, 2, 3, 4], 9222, 9223);
        let mut store = BootStore::in_memory();
        store.records.add(BootStoreEntry {
            udp_verified_at: Some(at(1_000)),
            ..BootStoreEntry::candidate(enr.clone())
        });
        store.sync().unwrap();

        confirm_session(&mut store, enr.node_id(), socket, at(2_000));

        assert_eq!(store.sync().unwrap(), confirms);
        let expected = BootStoreEntry {
            udp_verified_at: Some(at(if confirms { 2_000 } else { 1_000 })),
            session_started_at: confirms.then(|| at(2_000) - VERIFIED_REFRESH),
            // Whatever the socket, see `test_expire_keeps_a_record_in_its_grace`.
            expiry_grace: Some(at(2_000) - VERIFIED_REFRESH),
            ..BootStoreEntry::candidate(enr)
        };
        assert_eq!(store.records.entries.values().collect::<Vec<_>>(), [&expected]);
    }

    /// Evidence for a node the store doesn't hold, from a session or a gossip connection, adds
    /// nothing: its record comes from [`BootStore::add_enr`], which the table filter decides on.
    #[test]
    fn test_evidence_for_an_unknown_node_adds_nothing() {
        let enr = enr_of_new_node();
        let mut store = BootStore::in_memory();
        store.sync().unwrap();

        confirm_session(
            &mut store,
            enr.node_id(),
            SocketAddr::V4(enr.udp4_socket().unwrap()),
            at(1_000),
        );
        store.set_tcp_verified(
            enr.node_id(),
            SocketAddr::V4(enr.tcp4_socket().unwrap()),
            at(1_000),
        );

        assert!(store.is_empty());
        assert!(!store.sync().unwrap(), "evidence for an unknown node was a change");
    }

    /// A gossip connection we dialed confirms the TCP address of the stored record only if the
    /// record has a TCP address of the connection's family, and the connection is to it: not to
    /// its UDP port, say. It leaves the UDP evidence as it is. A connection to another address
    /// leaves the TCP evidence as it is too.
    #[rstest::rstest]
    #[case::matching("1.2.3.4:9223", true)]
    #[case::ipv4_mapped("[::ffff:1.2.3.4]:9223", true)]
    #[case::udp_port("1.2.3.4:9222", false)]
    #[case::other_ip("5.6.7.8:9223", false)]
    #[case::no_tcp_address_in_the_family("[::1]:9223", false)]
    fn test_gossip_confirms_only_the_record_tcp_address(
        #[case] addr: SocketAddr,
        #[case] confirms: bool,
    ) {
        let enr = enr_of(&CombinedKey::generate_secp256k1(), 1, [1, 2, 3, 4], 9222, 9223);
        let mut store = BootStore::in_memory();
        store.records.add(BootStoreEntry {
            udp_verified_at: Some(at(1_000)),
            tcp_verified_at: Some(at(1_000)),
            ..BootStoreEntry::candidate(enr.clone())
        });
        store.sync().unwrap();

        // Long enough after the stored evidence to replace it.
        store.set_tcp_verified(enr.node_id(), addr, at(2_000));

        assert_eq!(store.sync().unwrap(), confirms);
        let expected = BootStoreEntry {
            udp_verified_at: Some(at(1_000)),
            tcp_verified_at: Some(at(if confirms { 2_000 } else { 1_000 })),
            ..BootStoreEntry::candidate(enr)
        };
        assert_eq!(store.records.entries.values().collect::<Vec<_>>(), [&expected]);
    }

    /// Discovery and gossip repeat their evidence, so the store takes it only once the evidence
    /// it has is [`VERIFIED_REFRESH`] old. Otherwise every store interval would save the file.
    #[rstest::rstest]
    #[case::session(
        |enr: &Enr| SocketAddr::V4(enr.udp4_socket().unwrap()),
        confirm_session,
        BootStoreEntry::udp_verified_at
    )]
    #[case::gossip(
        |enr: &Enr| SocketAddr::V4(enr.tcp4_socket().unwrap()),
        BootStore::set_tcp_verified,
        BootStoreEntry::tcp_verified_at
    )]
    fn test_evidence_is_refreshed_once_it_is_old(
        #[case] addr_of: fn(&Enr) -> SocketAddr,
        #[case] set: fn(&mut BootStore, NodeId, SocketAddr, SystemTime),
        #[case] verified_at: fn(&BootStoreEntry) -> Option<SystemTime>,
    ) {
        let enr = enr_of_new_node();
        let addr = addr_of(&enr);
        let refresh = VERIFIED_REFRESH.as_secs();
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.sync().unwrap();

        set(&mut store, enr.node_id(), addr, at(1_000));
        assert!(store.sync().unwrap(), "the first evidence wasn't a change");
        set(&mut store, enr.node_id(), addr, at(1_000 + refresh - 1));
        assert!(!store.sync().unwrap(), "evidence before the refresh was a change");
        set(&mut store, enr.node_id(), addr, at(1_000 + refresh));
        assert!(store.sync().unwrap(), "evidence at the refresh wasn't a change");

        assert_eq!(verified_at(store.get(enr.node_id()).unwrap()), Some(at(1_000 + refresh)));
    }

    /// Sessions under fresh keys, which anyone can start, shouldn't push out a peer that gossip
    /// reached at its TCP address: a full store should evict a record nothing but a session
    /// confirmed first. Each such session confirms the record's UDP address later than the
    /// reached peer's last one, so evicting by that alone forgets the reached peer first.
    #[test]
    fn test_sessions_under_fresh_keys_keep_a_reached_peer() {
        let reached = enr_of_new_node();
        let mut store = BootStore::in_memory().with_max_peers(2);
        store.add_enr(reached.clone());
        let (udp, tcp) = (reached.udp4_socket().unwrap(), reached.tcp4_socket().unwrap());
        confirm_session(&mut store, reached.node_id(), SocketAddr::V4(udp), at(1_000));
        store.set_tcp_verified(reached.node_id(), SocketAddr::V4(tcp), at(1_000));

        // One more than fits beside the reached peer, so that one record must go.
        for _ in 0..2 {
            let fresh = enr_of_new_node();
            store.add_enr(fresh.clone());
            let udp = SocketAddr::V4(fresh.udp4_socket().unwrap());
            confirm_session(&mut store, fresh.node_id(), udp, at(2_000));
        }

        assert_eq!(store.len(), 2);
        let entry = store
            .get(reached.node_id())
            .expect("sessions under fresh keys pushed out a reached peer");
        assert_eq!(entry.tcp_verified_at(), Some(at(1_000)));
    }

    /// A session confirms its record's UDP address only once it has lasted the refresh delay:
    /// anyone can start one, under a fresh key. Without a session since the store was opened, as
    /// for a record from the file, discovery holding the node confirms nothing.
    #[test]
    fn test_a_session_confirms_only_once_it_has_lasted() {
        let enr = enr_of_new_node();
        let socket = SocketAddr::V4(enr.udp4_socket().unwrap());
        let refresh = VERIFIED_REFRESH.as_secs();
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());

        store.set_udp_verified(enr.node_id(), socket, at(1_000));
        assert_eq!(store.get(enr.node_id()).unwrap().udp_verified_at(), None);

        store.session_started(enr.node_id(), socket, at(1_000), Duration::ZERO);
        store.set_udp_verified(enr.node_id(), socket, at(1_000 + refresh - 1));
        assert_eq!(store.get(enr.node_id()).unwrap().udp_verified_at(), None);
        store.set_udp_verified(enr.node_id(), socket, at(1_000 + refresh));
        assert_eq!(store.get(enr.node_id()).unwrap().udp_verified_at(), Some(at(1_000 + refresh)));
    }

    /// Evidence older than the TTL should be forgotten, each timestamp on its own, so that a peer
    /// gossip last reached long ago no longer outranks the candidates when the store evicts,
    /// while its UDP address, which discovery still confirms, keeps the record.
    #[rstest::rstest]
    #[case::tcp_old(true, false)]
    #[case::udp_old(false, true)]
    fn test_expire_forgets_old_evidence(#[case] tcp_old: bool, #[case] udp_old: bool) {
        let enr = enr_of_new_node();
        let udp = SocketAddr::V4(enr.udp4_socket().unwrap());
        let tcp = SocketAddr::V4(enr.tcp4_socket().unwrap());
        // Just older than the TTL, or just as old.
        let dated = |old: bool| at(NOW - TTL.as_secs() - u64::from(old));
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.set_tcp_verified(enr.node_id(), tcp, dated(tcp_old));
        confirm_session(&mut store, enr.node_id(), udp, dated(udp_old));

        assert_eq!(store.expire(at(NOW), TTL), 0, "a record with recent evidence was dropped");

        let entry = store.get(enr.node_id()).expect("the record is kept");
        assert_eq!(entry.tcp_verified_at().is_none(), tcp_old, "tcp_verified_at");
        assert_eq!(entry.udp_verified_at().is_none(), udp_old, "udp_verified_at");
    }

    /// A record left without evidence younger than the TTL for either address should be dropped,
    /// and the file saved without it: otherwise a store that never fills up keeps dead records
    /// for good, and the seeds are drawn from them.
    #[test]
    fn test_expire_drops_a_record_without_recent_evidence() {
        let enr = enr_of_new_node();
        let udp = SocketAddr::V4(enr.udp4_socket().unwrap());
        let tcp = SocketAddr::V4(enr.tcp4_socket().unwrap());
        let old = at(NOW - TTL.as_secs() - 1);
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.set_tcp_verified(enr.node_id(), tcp, old);
        confirm_session(&mut store, enr.node_id(), udp, old);
        store.sync().unwrap();

        assert_eq!(store.expire(at(NOW), TTL), 1);
        assert!(store.get(enr.node_id()).is_none(), "a record without recent evidence was kept");
        assert!(store.sync().unwrap(), "the store didn't save its file without the record");
    }

    /// A record without evidence should be kept while the grace of its last session lasts,
    /// whatever socket the session came from: the session confirms the record only once it has
    /// lasted, or gossip once it has dialed it, and discovery reports a session only when it
    /// starts, so nothing would add the record again. A session from another socket than the
    /// record's UDP address, as behind a NAT that rewrites the source port, can't confirm it, but
    /// gossip can.
    #[rstest::rstest]
    #[case::record_socket("1.1.1.1:9222")]
    #[case::another_socket("1.1.1.1:9000")]
    fn test_expire_keeps_a_record_in_its_grace(#[case] socket: SocketAddr) {
        let enr = enr_of_new_node();
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.session_started(enr.node_id(), socket, at(NOW - 10), Duration::from_secs(11));

        assert_eq!(store.expire(at(NOW), TTL), 0);
        assert!(store.get(enr.node_id()).is_some(), "a record in its grace was dropped");
    }

    /// A record whose session's grace ended without evidence should be dropped, although the
    /// session started within the TTL: a session that neither lasted nor led gossip to the node,
    /// as under one of a flood of fresh keys, shouldn't keep its record for the TTL.
    #[test]
    fn test_expire_drops_a_record_past_its_grace() {
        let enr = enr_of_new_node();
        let udp = SocketAddr::V4(enr.udp4_socket().unwrap());
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.session_started(enr.node_id(), udp, at(NOW - 10), Duration::from_secs(10));

        assert_eq!(store.expire(at(NOW), TTL), 1);
        assert!(store.get(enr.node_id()).is_none(), "a record past its grace was kept");
    }

    /// A record that nothing ever confirmed, and that no session gave a grace, should be dropped
    /// too, as a record from a file of an earlier version, or a bootnode that doesn't answer: the
    /// configured bootnodes are added again at the next startup.
    #[test]
    fn test_expire_drops_a_record_never_confirmed() {
        let enr = enr_of_new_node();
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());

        assert_eq!(store.expire(at(NOW), TTL), 1);
        assert!(store.get(enr.node_id()).is_none(), "a record never confirmed was kept");
    }

    /// Evidence dated after now, which a clock that ran ahead wrote, should age from now, so that
    /// it expires a TTL later. Otherwise, once the clock is set right, a peer that never answers
    /// again stays in the first tier of the seeds and the redials until the clock catches up
    /// with it, and a store saved by a clock years ahead keeps such records for years.
    #[test]
    fn test_expire_ages_evidence_dated_after_now() {
        let enr = enr_of_new_node();
        let tcp = SocketAddr::V4(enr.tcp4_socket().unwrap());
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.set_tcp_verified(enr.node_id(), tcp, at(NOW + 1_000_000));

        assert_eq!(store.expire(at(NOW), TTL), 0);
        let entry = store.get(enr.node_id()).expect("the record is kept");
        assert_eq!(entry.tcp_verified_at(), Some(at(NOW)), "evidence dated after now isn't now");

        assert_eq!(store.expire(at(NOW + TTL.as_secs() + 1), TTL), 1, "the evidence didn't age");
    }

    /// A session from another socket than the record's UDP address ends the count for the
    /// session before it, which then doesn't confirm the record however long ago it started: the
    /// node talks to discovery from elsewhere now.
    #[test]
    fn test_session_from_another_socket_ends_the_count() {
        let enr = enr_of_new_node();
        let socket = SocketAddr::V4(enr.udp4_socket().unwrap());
        let refresh = VERIFIED_REFRESH.as_secs();
        let mut store = BootStore::in_memory();
        store.add_enr(enr.clone());
        store.session_started(enr.node_id(), socket, at(1_000), Duration::ZERO);

        store.session_started(
            enr.node_id(),
            "1.1.1.1:9000".parse().unwrap(),
            at(1_100),
            Duration::ZERO,
        );
        store.set_udp_verified(enr.node_id(), socket, at(1_000 + refresh));

        let entry = store.get(enr.node_id()).unwrap();
        assert_eq!(entry.session_started_at(), None);
        assert_eq!(entry.udp_verified_at(), None);
    }

    /// Sessions under fresh keys shouldn't push out a candidate whose session has lasted: they
    /// confirm nothing until they have lasted too, so a full store evicts one of them first.
    #[test]
    fn test_sessions_under_fresh_keys_keep_a_lasting_candidate() {
        let lasting = enr_of_new_node();
        let refresh = VERIFIED_REFRESH.as_secs();
        let mut store = BootStore::in_memory().with_max_peers(2);
        store.add_enr(lasting.clone());
        let socket = SocketAddr::V4(lasting.udp4_socket().unwrap());
        store.session_started(lasting.node_id(), socket, at(1_000), Duration::ZERO);
        store.set_udp_verified(lasting.node_id(), socket, at(1_000 + refresh));

        // One more than fits beside the lasting candidate, so that one record must go.
        for _ in 0..2 {
            let fresh = enr_of_new_node();
            store.add_enr(fresh.clone());
            let udp = SocketAddr::V4(fresh.udp4_socket().unwrap());
            store.session_started(fresh.node_id(), udp, at(1_000 + refresh + 10), Duration::ZERO);
            store.set_udp_verified(fresh.node_id(), udp, at(1_000 + refresh + 10));
        }

        assert_eq!(store.len(), 2);
        assert!(
            store.get(lasting.node_id()).is_some(),
            "sessions under fresh keys pushed out a candidate whose session lasted"
        );
    }

    /// Fresh sessions shouldn't push out an unconfirmed session that started before them, which
    /// may be about to be confirmed: their rate is the attacker's to choose, so evicting the oldest
    /// first would let a flood evict every session before it lasts long enough. A full store
    /// should evict the youngest instead, as Kademlia prefers the nodes that have been around
    /// longer. A new record still gets in, pushing out the one that came before it.
    #[test]
    fn test_fresh_sessions_keep_an_older_unconfirmed_session() {
        let older = enr_of_new_node();
        let mut store = BootStore::in_memory().with_max_peers(2);
        store.add_enr(older.clone());
        store.session_started(
            older.node_id(),
            SocketAddr::V4(older.udp4_socket().unwrap()),
            at(1_000),
            Duration::ZERO,
        );

        let mut fresh = Vec::new();
        for secs in [1_100, 1_200] {
            let enr = enr_of_new_node();
            store.add_enr(enr.clone());
            store.session_started(
                enr.node_id(),
                SocketAddr::V4(enr.udp4_socket().unwrap()),
                at(secs),
                Duration::ZERO,
            );
            fresh.push(enr);
        }

        assert_eq!(store.len(), 2);
        assert!(store.get(older.node_id()).is_some(), "fresh sessions pushed out an older one");
        assert!(store.get(fresh[1].node_id()).is_some(), "the newest record didn't get in");
    }

    /// Of the candidates no session has confirmed, a full store evicts the one without a session
    /// since it was opened first, as from the file, and then the one whose session started last.
    #[test]
    fn test_evicts_the_unconfirmed_candidate_whose_session_started_last() {
        let session_at = |secs| BootStoreEntry {
            session_started_at: Some(at(secs)),
            ..BootStoreEntry::candidate(enr_of_new_node())
        };
        let (earlier, later) = (session_at(1_000), session_at(2_000));
        let from_file = BootStoreEntry::candidate(enr_of_new_node());
        let mut records = Records::default();
        for entry in [&later, &from_file, &earlier] {
            records.add(entry.clone());
        }

        assert_eq!(records.evict(), Some(from_file));
        assert_eq!(records.evict(), Some(later));
        assert_eq!(records.evict(), Some(earlier));
    }

    /// When a session started isn't saved: the session ends with the process, so a record opened
    /// from the file has had no session yet.
    #[test]
    fn test_session_start_is_not_saved() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let enr = enr_of_new_node();
        let mut store = BootStore::open(path.clone()).unwrap();
        store.add_enr(enr.clone());
        store.session_started(
            enr.node_id(),
            SocketAddr::V4(enr.udp4_socket().unwrap()),
            at(1_000),
            Duration::ZERO,
        );
        store.sync().unwrap();

        let opened = BootStore::open(path).unwrap();

        assert_eq!(opened.get(enr.node_id()).unwrap().session_started_at(), None);
    }

    /// Evidence dated after now, as in a file written while the clock ran ahead, or edited by
    /// hand, should be replaced by the next evidence rather than kept: it can't be true, and it
    /// would otherwise stay until the clock catches up, and rank as the most recent. Adding the
    /// refresh delay to the latest time a `SystemTime` holds would panic the discovery task.
    #[rstest::rstest]
    #[case::ahead(at(1_000_000))]
    #[case::latest(SystemTime::UNIX_EPOCH + Duration::from_secs(i64::MAX as u64))]
    fn test_evidence_after_now_is_replaced(#[case] future: SystemTime) {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let enr = enr_of_new_node();
        let secs = future.duration_since(SystemTime::UNIX_EPOCH).unwrap().as_secs();
        let contents = format!(
            r#"{{"version": 1, "records": [{{"enr": "{}", "tcp_verified_at": {secs}}}]}}"#,
            enr.to_base64()
        );
        std::fs::write(&path, contents).unwrap();
        let mut store = BootStore::open(path).unwrap();

        store.set_tcp_verified(
            enr.node_id(),
            SocketAddr::V4(enr.tcp4_socket().unwrap()),
            at(2_000),
        );

        assert_eq!(store.get(enr.node_id()).unwrap().tcp_verified_at(), Some(at(2_000)));
    }

    /// A refresh delay too long to add to a session's start, set by mistake, makes the session
    /// never last, rather than panic.
    #[test]
    fn test_longest_refresh_delay_never_confirms() {
        let enr = enr_of_new_node();
        let socket = SocketAddr::V4(enr.udp4_socket().unwrap());
        let mut store = BootStore::in_memory().with_verified_refresh(Duration::MAX);
        store.add_enr(enr.clone());
        store.session_started(enr.node_id(), socket, at(1_000), Duration::ZERO);

        store.set_udp_verified(enr.node_id(), socket, at(2_000));

        assert_eq!(store.get(enr.node_id()).unwrap().udp_verified_at(), None);
    }

    /// A store given a lower limit than it holds evicts the records over it, as it would to make
    /// room for new ones, and saves the file without them.
    #[test]
    fn test_lower_limit_evicts_the_records_over_it() {
        let mut store = BootStore::in_memory();
        let enrs = [enr_of_new_node(), enr_of_new_node(), enr_of_new_node()];
        for (secs, enr) in [1_000, 2_000, 3_000].into_iter().zip(&enrs) {
            store.add_enr(enr.clone());
            confirm_session(
                &mut store,
                enr.node_id(),
                SocketAddr::V4(enr.udp4_socket().unwrap()),
                at(secs),
            );
        }
        store.sync().unwrap();

        let mut store = store.with_max_peers(2);

        assert!(store.get(enrs[0].node_id()).is_none(), "the record confirmed longest ago stayed");
        assert_eq!(store.len(), 2);
        assert!(store.sync().unwrap(), "evicting records wasn't a change to save");

        // A new record still gets in.
        store.add_enr(enr_of_new_node());
        assert_eq!(store.len(), 2);
    }

    /// A store with no refresh delay takes all newer evidence.
    #[test]
    fn test_no_refresh_delay_takes_all_newer_evidence() {
        let enr = enr_of_new_node();
        let addr = SocketAddr::V4(enr.tcp4_socket().unwrap());
        let mut store = BootStore::in_memory().with_verified_refresh(Duration::ZERO);
        store.add_enr(enr.clone());
        store.set_tcp_verified(enr.node_id(), addr, at(1_000));
        store.sync().unwrap();

        store.set_tcp_verified(enr.node_id(), addr, at(1_001));

        assert!(store.sync().unwrap(), "newer evidence wasn't a change");
        assert_eq!(store.get(enr.node_id()).unwrap().tcp_verified_at(), Some(at(1_001)));
    }

    /// Of the candidates, a full store evicts the one whose UDP address a session confirmed
    /// longest ago, those no session confirmed first, so that the ones confirmed recently stay.
    /// Those that tie go in any order, see [`test_evicts_records_that_tie_in_no_fixed_order`].
    #[test]
    fn test_evicts_the_candidate_confirmed_longest_ago() {
        let never = [
            BootStoreEntry::candidate(enr_of_new_node()),
            BootStoreEntry::candidate(enr_of_new_node()),
        ];
        let [never_1, never_2] = never.clone();
        let earlier = BootStoreEntry {
            udp_verified_at: Some(at(1_000)),
            ..BootStoreEntry::candidate(enr_of_new_node())
        };
        let later = BootStoreEntry {
            udp_verified_at: Some(at(2_000)),
            ..BootStoreEntry::candidate(enr_of_new_node())
        };
        let mut records = Records::default();
        for entry in [&later, &never_1, &earlier, &never_2] {
            records.add(entry.clone());
        }

        let mut evicted_first = [records.evict().unwrap(), records.evict().unwrap()];
        evicted_first.sort_by_key(BootStoreEntry::key);
        let mut never = never;
        never.sort_by_key(BootStoreEntry::key);
        assert_eq!(evicted_first, never);
        assert_eq!(records.evict(), Some(earlier));
        assert_eq!(records.evict(), Some(later));
    }

    /// Which of two records that tie a full store evicts shouldn't follow from their node IDs: a
    /// node ID is the hash of a key, and an attacker can grind keys until its records sort after
    /// the others, so that they outlast them. So the order should vary from one eviction to the
    /// next. With 64 evictions, the chance that one of the records is never evicted is 2⁻⁶³.
    #[test]
    fn test_evicts_records_that_tie_in_no_fixed_order() {
        let mut records = Records::default();
        records.add(BootStoreEntry::candidate(enr_of_new_node()));
        records.add(BootStoreEntry::candidate(enr_of_new_node()));

        let evicted: std::collections::HashSet<_> = (0..64)
            .map(|_| records.clone().evict().expect("the records aren't empty").key())
            .collect();

        assert_eq!(evicted.len(), 2, "the same one of two records that tie was always evicted");
    }

    /// A full store evicts a record that gossip reached only once it holds no candidate, however
    /// recently a session confirmed the candidates. Of the reached records, it evicts the one
    /// gossip reached longest ago.
    #[test]
    fn test_evicts_a_reached_peer_last() {
        let reached = |secs| BootStoreEntry {
            udp_verified_at: Some(at(1_000)),
            tcp_verified_at: Some(at(secs)),
            ..BootStoreEntry::candidate(enr_of_new_node())
        };
        let (reached_earlier, reached_later) = (reached(1_000), reached(2_000));
        let candidate = BootStoreEntry {
            udp_verified_at: Some(at(3_000)),
            ..BootStoreEntry::candidate(enr_of_new_node())
        };
        let mut records = Records::default();
        for entry in [&reached_later, &candidate, &reached_earlier] {
            records.add(entry.clone());
        }

        assert_eq!(records.evict(), Some(candidate));
        assert_eq!(records.evict(), Some(reached_earlier));
        assert_eq!(records.evict(), Some(reached_later));
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
        let first = enr_of(&key, first_seq, [1, 1, 1, 1], 9222, 9223);
        let second = enr_of(&key, second_seq, [1, 1, 1, 1], 9000, 9223);
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

    /// An entry that isn't a record is skipped, and the others are kept. The next save leaves it
    /// out.
    #[test]
    fn test_open_skips_entries_that_are_not_records() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let kept = enr_of_new_node();
        let json = serde_json::json!({
            "version": 1,
            "records": [{"enr": "enr:-not-base64!"}, {"enr": kept.to_base64()}, 42],
        });
        std::fs::write(&path, json.to_string()).unwrap();

        let mut store = BootStore::open(path.clone()).unwrap();

        assert_eq!(store.iter().collect::<Vec<_>>(), [&kept]);
        assert!(store.sync().unwrap(), "skipped entries weren't a change to save");
        let saved: serde_json::Value =
            serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
        assert_eq!(
            saved,
            serde_json::json!({"version": FILE_VERSION, "records": [{"enr": kept.to_base64()}]})
        );
    }

    /// Of the entries for one node, only the record with the highest `seq` is kept: not an older
    /// record, another with the same `seq`, or the same record twice. An earlier version saved
    /// every record a node had. The next save leaves the others out.
    #[test]
    fn test_open_keeps_one_record_per_node() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let key = CombinedKey::generate_secp256k1();
        let older = enr_of(&key, 1, [1, 1, 1, 1], 9222, 9223);
        let newer = enr_of(&key, 2, [1, 1, 1, 1], 9222, 9223);
        let conflicting = enr_of(&key, 2, [2, 2, 2, 2], 9222, 9223);
        let json = serde_json::json!([older, newer, conflicting, newer]);
        std::fs::write(&path, json.to_string()).unwrap();

        let mut store = BootStore::open(path).unwrap();

        assert_eq!(store.iter().collect::<Vec<_>>(), [&newer]);
        assert!(store.sync().unwrap(), "the records it didn't keep weren't a change to save");
    }

    /// A file in the format earlier versions wrote, a list of records, opens as records without
    /// evidence, and the next save writes it in the current format.
    #[test]
    fn test_open_list_of_records() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let listed = enr_of_new_node();
        std::fs::write(&path, serde_json::json!([listed]).to_string()).unwrap();

        let mut store = BootStore::open(path.clone()).unwrap();

        let entries: Vec<_> = store.records.entries.values().collect();
        assert_eq!(entries, [&BootStoreEntry::candidate(listed.clone())]);
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
        let verified = enr_of(&key, 1, [1, 1, 1, 1], 9222, 9223);
        let candidate = enr_of_new_node();
        let contents = format!(
            r#"{{"version": 1, "records": [
                {{"enr": "{}", "udp_verified_at": 100, "tcp_verified_at": 200}},
                {{"enr": "{}"}}
            ]}}"#,
            verified.to_base64(),
            candidate.to_base64(),
        );
        std::fs::write(&path, contents).unwrap();

        let store = BootStore::open(path).unwrap();

        let mut expected = Records::default();
        expected.add(BootStoreEntry {
            enr: verified,
            udp_verified_at: Some(at(100)),
            tcp_verified_at: Some(at(200)),
            session_started_at: None,
            expiry_grace: None,
        });
        expected.add(BootStoreEntry::candidate(candidate));
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

        let store = BootStore::open(path).unwrap();

        assert!(store.is_empty());
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
        let mut store = BootStore::open(path.clone()).unwrap();
        store.add_enr(enr_of_new_node());
        store.sync().unwrap();
        let before = std::fs::read(&path).unwrap();
        let mut reader = File::open(&path).unwrap();

        store.add_enr(enr_of_new_node());
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
        let mut store = BootStore::open(path.clone()).unwrap();
        store.add_enr(enr_of_new_node());
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
        let mut store = BootStore::open(path.clone()).unwrap();
        store.add_enr(enr_of_new_node());
        store.sync().unwrap();
        let inode = || std::fs::metadata(&path).unwrap().ino();

        let mut store = BootStore::open(path.clone()).unwrap();
        let opened = inode();
        store.sync().unwrap();
        assert_eq!(inode(), opened, "a save rewrote the file it had just opened");

        store.add_enr(enr_of_new_node());
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
            BootStore::open(dir.path().join("bootstore.json")).unwrap()
        } else {
            BootStore::in_memory()
        };
        // A missing file is a change to save, so that it gets created.
        assert_eq!(store.sync().unwrap(), file);

        let added = enr_of_new_node();
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
        let mut store = BootStore::open(path.clone()).unwrap();
        for _ in 0..RECORDS {
            store.add_enr(enr_of_new_node());
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
                        Ok(records) if records.entries.len() >= RECORDS
                    );
                    if !whole {
                        return true;
                    }
                }
                false
            }
        });
        for _ in 0..SAVES {
            store.add_enr(enr_of_new_node());
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
