#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

#[global_allocator]
static ALLOC: reth_cli_util::allocator::Allocator = reth_cli_util::allocator::new_allocator();

#[cfg(all(feature = "jemalloc-prof", unix))]
#[unsafe(export_name = "_rjem_malloc_conf")]
static MALLOC_CONF: &[u8] = b"prof:true,prof_active:true,lg_prof_sample:19\0";

fn main() {
    if let Err(err) = op_reth::main() {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
