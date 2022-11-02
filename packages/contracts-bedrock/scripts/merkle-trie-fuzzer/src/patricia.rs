#[doc = include_str!("../README.md")]
use clap::Parser;
use core::fmt;
use eth_trie::MemoryDB;
use eth_trie::{EthTrie, Trie, TrieError};
use ethers::abi::Token;
use ethers::types::H256;
use ethers::utils::hex;
use rand::Rng;
use std::sync::Arc;
use yansi::Paint;

#[derive(Parser, Debug)]
#[clap(name = "patricia")]
struct Patricia {
    /// Pretty print the [TrieTestCase]
    #[clap(short)]
    pub pretty_print: bool,

    /// Select a different mode for the fuzz input generator
    #[command(subcommand)]
    pub mode: Option<Mode>,
}

/// The mode setting for the fuzzer
///
/// TODO: Other modes
#[derive(clap::Subcommand, Debug)]
enum Mode {
    /// Generate a valid test case
    Valid,
}

/// Represents data for a Trie Test Case
#[derive(Default, Debug)]
struct TrieTestCase {
    /// The root of the merkle trie.
    pub root: Vec<u8>,
    /// The key that the `proof` is generated for.
    pub key: Vec<u8>,
    /// The value mapped to `key`.
    pub val: Vec<u8>,
    /// The proof of inclusion for `key`.
    pub proof: Vec<Vec<u8>>,
}

impl TrieTestCase {
    /// ABI encode the [TrieTestCase] as the tuple (root, key, val, proof).
    pub fn encode(self) -> Result<String, TrieError> {
        Ok(hex::encode(ethers::abi::encode(&[
            Token::FixedBytes(self.root),
            Token::Bytes(self.key),
            Token::Bytes(self.val),
            Token::Array(
                self.proof
                    .into_iter()
                    .map(|elem| Token::Bytes(elem))
                    .collect(),
            ),
        ])))
    }
}

/// Pretty print a [TrieTestCase]
impl fmt::Display for TrieTestCase {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_fmt(format_args!(
            "{} 0x{}\n",
            Paint::green("Root:"),
            hex::encode(&self.root)
        ))?;
        f.write_fmt(format_args!(
            "{} 0x{}\n",
            Paint::green("Key:"),
            hex::encode(&self.key)
        ))?;
        f.write_fmt(format_args!(
            "{} 0x{}\n",
            Paint::green("Val:"),
            hex::encode(&self.val)
        ))?;
        f.write_fmt(format_args!(
            "{}\n{}",
            Paint::green("Proof:"),
            self.proof
                .iter()
                .enumerate()
                .map(|(i, p)| format!("{} 0x{}", Paint::yellow(format!("{})", i)), hex::encode(p)))
                .collect::<Vec<_>>()
                .join("\n\n")
        ))
    }
}

fn main() -> Result<(), TrieError> {
    let args = Patricia::parse();

    // Spawn a threaded random number generator.
    let mut thread_rng = rand::thread_rng();

    // Generate a random *even* value between 2 and 1024.
    let rand_n = {
        let mut i = thread_rng.gen_range(2..=1024);
        if i % 2 == 0 {
            i += 1;
        }
        i
    };
    // Choose a random element to grab an inclusion proof for.
    let rand_elem = thread_rng.gen_range(0..=rand_n);
    // Create a blank test case
    let mut test_case = TrieTestCase::default();
    // Spawn a new in-memory DB for our trie.
    let memdb = Arc::new(MemoryDB::new(true));
    // Create a new trie and populate it with a random, even number
    // of key-value pairs.
    let mut trie = EthTrie::new(memdb);
    (0..=rand_n).for_each(|i| {
        let (a, b) = (H256::random(), H256::random());

        if i == rand_elem {
            test_case.key = a.as_bytes().to_vec();
            test_case.val = b.as_bytes().to_vec();
        }

        trie.insert(a.as_bytes(), b.as_bytes()).unwrap();
    });

    // Assign the [TrieTestCase]'s root and proof
    test_case.root = trie.root_hash()?.as_bytes().to_vec();
    test_case.proof = trie.get_proof(test_case.key.as_slice()).unwrap();

    if args.pretty_print {
        print!("{}", test_case);
    } else {
        if let Ok(encoded) = test_case.encode() {
            print!("{}", encoded);
        } else {
            eprint!("Error encoding test case!");
        }
    }

    Ok(())
}
