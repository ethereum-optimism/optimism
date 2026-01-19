#![doc = include_str!("../README.md")]
#![warn(missing_debug_implementations, missing_docs, unreachable_pub, rustdoc::all)]
#![deny(unused_must_use, rust_2018_idioms)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![no_std]
#![cfg_attr(any(target_arch = "mips64", target_arch = "riscv64"), no_main)]

extern crate alloc;

use alloc::string::String;
use kona_preimage::{HintWriter, OracleReader};
use kona_std_fpvm::{FileChannel, FileDescriptor};
use kona_std_fpvm_proc::client_entry;

/// The global preimage oracle reader pipe.
static ORACLE_READER_PIPE: FileChannel =
    FileChannel::new(FileDescriptor::PreimageRead, FileDescriptor::PreimageWrite);

/// The global hint writer pipe.
static HINT_WRITER_PIPE: FileChannel =
    FileChannel::new(FileDescriptor::HintRead, FileDescriptor::HintWrite);

/// The global preimage oracle reader.
static ORACLE_READER: OracleReader<FileChannel> = OracleReader::new(ORACLE_READER_PIPE);

/// The global hint writer.
static HINT_WRITER: HintWriter<FileChannel> = HintWriter::new(HINT_WRITER_PIPE);

#[client_entry]
fn main() -> Result<(), String> {
    #[cfg(feature = "client-tracing")]
    {
        use kona_std_fpvm::tracing::FpvmTracingSubscriber;

        let subscriber = FpvmTracingSubscriber::new(tracing::Level::INFO);
        tracing::subscriber::set_global_default(subscriber)
            .expect("Failed to set tracing subscriber");
    }

    kona_proof::block_on(kona_client::interop::run(ORACLE_READER, HINT_WRITER))
}
