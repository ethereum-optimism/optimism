//! Classification of deposit transaction source

use alloc::string::String;
use alloy_primitives::{B256, keccak256};

/// Source domain identifiers for deposit transactions.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Copy)]
#[repr(u8)]
pub enum DepositSourceDomainIdentifier {
    /// A user deposit source.
    User = 0,
    /// A L1 info deposit source.
    L1Info = 1,
    /// An upgrade deposit source.
    Upgrade = 2,
    /// An interop block replacement source.
    InteropBlockReplacement = 4,
}

/// Source domains for deposit transactions.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum DepositSourceDomain {
    /// A user deposit source.
    User(UserDepositSource),
    /// A L1 info deposit source.
    L1Info(L1InfoDepositSource),
    /// An upgrade deposit source.
    Upgrade(UpgradeDepositSource),
    /// An interop block replacement deposit source.
    InteropBlockReplacement(InteropBlockReplacementDepositSource),
}

impl DepositSourceDomain {
    /// Returns the source hash.
    pub fn source_hash(&self) -> B256 {
        match self {
            Self::User(ds) => ds.source_hash(),
            Self::L1Info(ds) => ds.source_hash(),
            Self::Upgrade(ds) => ds.source_hash(),
            Self::InteropBlockReplacement(ds) => ds.source_hash(),
        }
    }
}

/// A deposit transaction source.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Copy)]
pub struct UserDepositSource {
    /// The L1 block hash.
    pub l1_block_hash: B256,
    /// The log index.
    pub log_index: u64,
}

impl UserDepositSource {
    /// Creates a new [UserDepositSource].
    pub const fn new(l1_block_hash: B256, log_index: u64) -> Self {
        Self { l1_block_hash, log_index }
    }

    /// Returns the source hash.
    pub fn source_hash(&self) -> B256 {
        let mut input = [0u8; 32 * 2];
        input[..32].copy_from_slice(&self.l1_block_hash[..]);
        input[32 * 2 - 8..].copy_from_slice(&self.log_index.to_be_bytes());
        let deposit_id_hash = keccak256(input);
        let mut domain_input = [0u8; 32 * 2];
        let identifier_bytes: [u8; 8] = (DepositSourceDomainIdentifier::User as u64).to_be_bytes();
        domain_input[32 - 8..32].copy_from_slice(&identifier_bytes);
        domain_input[32..].copy_from_slice(&deposit_id_hash[..]);
        keccak256(domain_input)
    }
}

/// A L1 info deposit transaction source.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Copy)]
pub struct L1InfoDepositSource {
    /// The L1 block hash.
    pub l1_block_hash: B256,
    /// The sequence number.
    pub seq_number: u64,
}

impl L1InfoDepositSource {
    /// Creates a new [L1InfoDepositSource].
    pub const fn new(l1_block_hash: B256, seq_number: u64) -> Self {
        Self { l1_block_hash, seq_number }
    }

    /// Returns the source hash.
    pub fn source_hash(&self) -> B256 {
        let mut input = [0u8; 32 * 2];
        input[..32].copy_from_slice(&self.l1_block_hash[..]);
        input[32 * 2 - 8..].copy_from_slice(&self.seq_number.to_be_bytes());
        let deposit_id_hash = keccak256(input);
        let mut domain_input = [0u8; 32 * 2];
        let identifier_bytes: [u8; 8] =
            (DepositSourceDomainIdentifier::L1Info as u64).to_be_bytes();
        domain_input[32 - 8..32].copy_from_slice(&identifier_bytes);
        domain_input[32..].copy_from_slice(&deposit_id_hash[..]);
        keccak256(domain_input)
    }
}

/// An upgrade deposit transaction source.
///
/// This implements the translation of upgrade-tx identity information to a deposit source-hash,
/// which makes the deposit uniquely identifiable.
/// System-upgrade transactions have their own domain for source-hashes,
/// to not conflict with user-deposits or deposited L1 information.
/// The intent identifies the upgrade-tx uniquely, in a human-readable way.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct UpgradeDepositSource {
    /// The intent.
    pub intent: String,
}

impl UpgradeDepositSource {
    /// Creates a new [UpgradeDepositSource].
    pub const fn new(intent: String) -> Self {
        Self { intent }
    }

    /// Returns the source hash.
    pub fn source_hash(&self) -> B256 {
        let intent_hash = keccak256(self.intent.as_bytes());
        let mut domain_input = [0u8; 32 * 2];
        let identifier_bytes: [u8; 8] =
            (DepositSourceDomainIdentifier::Upgrade as u64).to_be_bytes();
        domain_input[32 - 8..32].copy_from_slice(&identifier_bytes);
        domain_input[32..].copy_from_slice(&intent_hash[..]);
        keccak256(domain_input)
    }
}

/// An interop block replacement deposit transaction source.
///
/// This implements the translation of the replaced output root information to a deposit
/// source-hash, which makes the deposit uniquely identifiable.
/// Interop block replacement transactions have their own domain for source-hashes,
/// to not conflict with user-deposits, system upgrades, or L1 information updates.
///
/// The output root in the source is the output root of the block that was replaced.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct InteropBlockReplacementDepositSource {
    /// The output root of the replaced block.
    pub output_root: B256,
}

impl InteropBlockReplacementDepositSource {
    /// Creates a new [InteropBlockReplacementDepositSource].
    pub const fn new(output_root: B256) -> Self {
        Self { output_root }
    }

    /// Returns the source hash.
    pub fn source_hash(&self) -> B256 {
        let mut domain_input = [0u8; 32 * 2];
        let identifier_bytes: [u8; 8] =
            (DepositSourceDomainIdentifier::InteropBlockReplacement as u64).to_be_bytes();
        domain_input[32 - 8..32].copy_from_slice(&identifier_bytes);
        domain_input[32..].copy_from_slice(&self.output_root[..]);
        keccak256(domain_input)
    }
}
