//! Bootnode Store

use discv5::Enr;
use std::{
    collections::VecDeque,
    fs::File,
    io::{BufReader, Seek, SeekFrom},
    path::PathBuf,
};

/// The maximum number of peers that can be stored in the bootstore.
const MAX_PEERS: usize = 2048;

/// On-disk storage for [`Enr`]s.
///
/// The [`BootStore`] is a simple JSON file that holds the list of [`Enr`]s that have been
/// successfully peered.
///
/// When the number of peers within the [`BootStore`] exceeds `MAX_PEERS`, the oldest peers are
/// removed to make room for new ones.
#[derive(Debug, serde::Serialize, Default)]
pub struct BootStore {
    /// The file for the [`BootStore`].
    #[serde(skip)]
    pub file: Option<File>,
    /// [`Enr`]s for peers.
    pub peers: VecDeque<Enr>,
}

/// The bootstore caching policy.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BootStoreFile {
    /// Default path for the bootstore, ie `~/.kona/<chain_id>/bootstore.json`.
    Default {
        /// The l2 chain ID.
        chain_id: u64,
    },
    /// A custom bootstore path is used. This must be a valid path to a file.
    Custom(PathBuf),
}

impl From<File> for BootStore {
    fn from(file: File) -> Self {
        let peers = peers_from_file(&file);
        Self { file: Some(file), peers }
    }
}

impl TryInto<File> for BootStoreFile {
    type Error = std::io::Error;

    /// Returns a pointer to the bootstore file for the given combination of chain id and bootstore
    /// file type.
    fn try_into(self) -> Result<File, std::io::Error> {
        let path = TryInto::<PathBuf>::try_into(self)?;
        File::options().read(true).write(true).create(true).truncate(false).open(path)
    }
}

impl TryInto<BootStore> for BootStoreFile {
    type Error = std::io::Error;

    fn try_into(self) -> Result<BootStore, std::io::Error> {
        let file = TryInto::<File>::try_into(self)?;
        Ok(file.into())
    }
}

impl TryInto<PathBuf> for BootStoreFile {
    type Error = std::io::Error;

    fn try_into(self) -> Result<PathBuf, std::io::Error> {
        match self {
            Self::Default { chain_id } => {
                let mut path = dirs::home_dir()
                    .ok_or(std::io::Error::other("Failed to get home directory"))?;
                path.push(".kona");
                path.push(chain_id.to_string());
                path.push("bootstore.json");
                Ok(path)
            }
            Self::Custom(path) => Ok(path),
        }
    }
}

fn peers_from_file(file: &File) -> VecDeque<Enr> {
    debug!(target: "bootstore", "Reading boot store from disk: {:?}", file);
    let reader = BufReader::new(file);
    match serde_json::from_reader(reader).map(|s: BootStore| s.peers) {
        Ok(peers) => peers,
        Err(e) => {
            warn!(target: "bootstore", "Failed to read boot store from disk: {:?}", e);
            VecDeque::new()
        }
    }
}

// This custom implementation of `Deserialize` allows us to ignore
// enrs that have an invalid string format in the store.
impl<'de> serde::Deserialize<'de> for BootStore {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let peers: Vec<serde_json::Value> = serde::Deserialize::deserialize(deserializer)?;
        let mut store = Self { file: None, peers: VecDeque::new() };
        for peer in peers {
            match serde_json::from_value::<Enr>(peer) {
                Ok(enr) => {
                    store.peers.push_back(enr);
                }
                Err(e) => {
                    warn!(target: "peers_store", "Failed to deserialize ENR: {:?}", e);
                }
            }
        }
        Ok(store)
    }
}

impl BootStore {
    /// Adds an [`Enr`] to the store.
    ///
    /// This method will **note** panic on failure to write to disk. Instead, it is the
    /// responsibility of the caller to ensure the store is written to disk by calling
    /// [`BootStore::sync`] prior to dropping the store.
    pub fn add_enr(&mut self, enr: Enr) {
        self.add_rotate(enr);
    }

    /// Returns the number of peers in the bootstore that
    /// have the [`crate::OpStackEnr::OP_CL_KEY`] in the ENR.
    pub fn valid_peers(&self) -> Vec<&Enr> {
        self.peers
            .iter()
            .filter(|enr| enr.get_raw_rlp(crate::OpStackEnr::OP_CL_KEY.as_bytes()).is_some())
            .collect()
    }

    /// Returns the number of peers that contain the
    /// [`crate::OpStackEnr::OP_CL_KEY`] in the ENR *and*
    /// have the correct chain id and version.
    pub fn valid_peers_with_chain_id(&self, chain_id: u64) -> Vec<&Enr> {
        self.peers
            .iter()
            .filter(|enr| crate::EnrValidation::validate(enr, chain_id).is_valid())
            .collect()
    }

    /// Returns the number of peers in the in-memory store.
    pub fn len(&self) -> usize {
        self.peers.len()
    }

    /// Returns if the in-memory store is empty.
    pub fn is_empty(&self) -> bool {
        self.peers.is_empty()
    }

    /// Merges the given list of [`Enr`]s with the current list of peers.
    pub fn merge(&mut self, peers: impl IntoIterator<Item = Enr>) {
        peers.into_iter().for_each(|peer| self.add_rotate(peer));
    }

    /// Syncs the [`BootStore`] with the contents on disk.
    pub fn sync(&mut self) -> Result<(), std::io::Error> {
        if let Some(file) = &mut self.file {
            // Reset the file pointer to the beginning of the file to overwrite the file.
            // Reset file pointer AND truncate
            file.seek(SeekFrom::Start(0))?;
            file.set_len(0)?;

            serde_json::to_writer(file, &self.peers)?;
        }
        Ok(())
    }

    /// Returns all available bootstores for the given data directory.
    pub fn available(datadir: Option<PathBuf>) -> Vec<u64> {
        let mut bootstores = Vec::new();
        let path = datadir.unwrap_or_else(|| {
            let mut home = dirs::home_dir().expect("Failed to get home directory");
            home.push(".kona");
            home
        });
        if let Ok(entries) = std::fs::read_dir(path) {
            for entry in entries.flatten() {
                if let Ok(chain_id) = entry.file_name().to_string_lossy().parse::<u64>() {
                    bootstores.push(chain_id);
                }
            }
        }
        bootstores
    }

    /// Adds an [`Enr`] to the store, rotating the oldest peer out if necessary.
    fn add_rotate(&mut self, enr: Enr) {
        if self.peers.contains(&enr) {
            return;
        }

        debug!(target: "bootstore", "Adding enr to the boot store: {}", enr);
        self.peers.push_back(enr);

        // Prune the oldest peer if we exceed the maximum number of peers.
        if self.peers.len() > MAX_PEERS {
            debug!(target: "bootstore", "Boot store exceeded maximum peers, removing oldest peer");
            self.peers.pop_front();
        }
    }
}
