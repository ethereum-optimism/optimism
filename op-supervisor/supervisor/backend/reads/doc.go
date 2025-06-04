// Package reads implements chain read-handles.
//
// These protect read results. The actual reads may still be disturbed,
// but the handle can be inspected at any time, to abort if needed.
// And the final result can inspect the read-handle, to determine if the combined reading result is valid.
//
// Append-only writes to the state do not need any invalidation.
// And updates have parent-hash checks, and thus do not require any rewind-locking on the parent either.
//
// But rewinds, replacements, etc. that do not append do need invalidation, to make sure other reads are not affected.
//
// This approach was chosen over simpler global read-write ChainsDB locking for two main reasons:
// 1. Fine-grained invalidation: Only operations depending on rewound blocks are affected
// 2. Non-blocking reads: Rewinds don't block unrelated read operations
package reads
