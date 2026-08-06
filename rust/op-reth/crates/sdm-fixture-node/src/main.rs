//! `op-reth-sdm-fixture` test-only fixture producer binary.

#[global_allocator]
static ALLOC: reth_cli_util::allocator::Allocator = reth_cli_util::allocator::new_allocator();

fn main() {
    reth_cli_util::sigsegv_handler::install();
    op_reth_sdm_fixture::run()
}
