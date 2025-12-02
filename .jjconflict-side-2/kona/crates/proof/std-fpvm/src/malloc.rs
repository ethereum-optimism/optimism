//! This module contains an implementation of a basic memory allocator for client programs in
//! running on an embedded device.
//!
//! The allocator is a linked list allocator based on the `dlmalloc` algorithm, which is a
//! well-known and widely used allocator software such as OS Kernels.

/// The global allocator for the program in embedded environments.
#[cfg(any(target_arch = "mips64", target_arch = "riscv64"))]
pub mod global_allocator {
    use buddy_system_allocator::LockedHeap;

    /// The maximum block size, as a power of two, for the buddy system allocator.
    const HEAP_ORDER: usize = 32;

    /// The global allocator for the program.
    #[global_allocator]
    static ALLOCATOR: LockedHeap<HEAP_ORDER> = LockedHeap::empty();

    /// Initialize the [SpinLockedAllocator] with the following parameters:
    /// * `heap_start_addr` is the starting address of the heap memory region,
    /// * `heap_size` is the size of the heap memory region in bytes.
    ///
    /// # Safety
    /// This function is unsafe because the caller must ensure:
    /// * The allocator has not already been initialized.
    /// * The provided memory region must be valid, non-null, and not used by anything else.
    /// * After aligning the start and end addresses, the size of the heap must be > 0, or the
    ///   function will panic.
    pub unsafe fn init_allocator(heap_start_addr: usize, heap_size: usize) {
        unsafe { ALLOCATOR.lock().init(heap_start_addr, heap_size) }
    }
}

/// Initialize heap memory for the `client` program with the given size.
///
/// # Safety
#[cfg_attr(
    any(target_arch = "mips64", target_arch = "riscv64"),
    doc = "See [global_allocator::init_allocator] safety comment."
)]
#[cfg_attr(
    not(any(target_arch = "mips64", target_arch = "riscv64")),
    doc = "This macro is entirely safe to invoke in non-MIPS and non-RISC-V64 profiles, and functions as a no-op."
)]
#[macro_export]
macro_rules! alloc_heap {
    () => {{
        #[cfg(any(target_arch = "mips64", target_arch = "riscv64"))]
        {
            use $crate::malloc::global_allocator::init_allocator;

            // The maximum heap size is configured to be an inordinate amount of memory (a
            // terabyte.) Fault proof VMs do not actually allocate pages when an `mmap`
            // is received, but instead allocate new pages on the fly. At startup, we
            // request the FPVM's heap pointer to be bumped to make room for any necessary
            // allocations throughout the lifecycle of the program.
            const MAX_HEAP_SIZE: usize = 1 << 40;

            // SAFETY: If the kernel fails to map the virtual memory, a panic is in order and we
            // should exit immediately. Program execution cannot continue.
            let region_start =
                $crate::io::mmap(MAX_HEAP_SIZE).expect("Kernel failed to map memory");

            // SAFETY: The memory region, at this point, is guaranteed to be valid and mapped by the
            // kernel.
            unsafe {
                init_allocator(region_start, MAX_HEAP_SIZE);
            }
        }
    }};
}
