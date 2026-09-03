//! Typed handles of the compiled circuit.
//!
//! `sql-to-dbsp --handles` returns one flat tuple: every table's input handle in declaration
//! order, then every non-local view's output handle in declaration order, then the
//! compiler-injected `error_view`. This module names each position and pins its type, so a
//! change to `chainview.sql` that reorders or retypes a relation fails to compile here rather
//! than silently feeding a table through the wrong handle.

use dbsp::{
    DBSPHandle, OutputHandle, ZSetHandle,
    circuit::CircuitConfig,
    typed_batch::{OrdZSet, SpineSnapshot},
    utils::{Tup2, Tup3, Tup4, Tup5, Tup8, Tup10},
};
use feldera_sqllib::{ByteArray, SqlString};

use crate::generated::circuit;

/// Row of `l1_blocks`: `(number, hash, parent_hash, ts)`.
pub type L1BlockRow = Tup4<i64, ByteArray, ByteArray, i64>;
/// Row of `l1_status`: `(kind, number, hash, parent_hash, ts)`.
pub type L1StatusRow = Tup5<SqlString, i64, ByteArray, ByteArray, i64>;
/// Row of `l2_status`:
/// `(kind, number, hash, parent_hash, ts, l1_origin_number, l1_origin_hash, seq_num)`.
pub type L2StatusRow = Tup8<SqlString, i64, ByteArray, ByteArray, i64, i64, ByteArray, i64>;
/// Row of `l2_safe_blocks` and `l2_safe_canonical`:
/// `(seq, l2_number, l2_hash, l2_parent_hash, l2_ts, l1_origin_number, l1_origin_hash, seq_num,
/// derived_from_number, derived_from_hash)`.
pub type L2SafeRow =
    Tup10<i64, i64, ByteArray, ByteArray, i64, i64, ByteArray, i64, i64, ByteArray>;
/// Row of `unsafe_block_signer`: `(l1_number, l1_hash, signer)`.
pub type UnsafeBlockSignerRow = Tup3<i64, ByteArray, ByteArray>;
/// Row of `safe_head_updates`: `(l1_number, l1_hash, l2_number, l2_hash)`.
pub type SafeHeadUpdateRow = Tup4<i64, ByteArray, i64, ByteArray>;
/// Row of `finalized_l2`: `(l2_number, l2_hash, derived_from_number, derived_from_hash)`.
pub type FinalizedL2Row = Tup4<Option<i64>, Option<ByteArray>, Option<i64>, Option<ByteArray>>;
/// Row of `current_signer`: `(signer, l1_number)`.
pub type CurrentSignerRow = Tup2<ByteArray, i64>;
/// Row of the compiler-injected `error_view`: `(table_or_view_name, message, metadata)`.
pub type ErrorRow = Tup3<SqlString, SqlString, SqlString>;

/// An accumulated output: the coalesced delta of a view for one transaction.
///
/// `sql-to-dbsp` spells the batch type `WSet<Row>`, which `feldera_sqllib` defines as
/// exactly [`OrdZSet<Row>`].
pub type ViewOutput<Row> = OutputHandle<SpineSnapshot<OrdZSet<Row>>>;

/// The circuit's input and output handles, by relation name.
#[derive(Clone)]
pub struct Handles {
    /// `l1_blocks` input.
    pub l1_blocks: ZSetHandle<L1BlockRow>,
    /// `l1_status` input.
    pub l1_status: ZSetHandle<L1StatusRow>,
    /// `l2_status` input.
    pub l2_status: ZSetHandle<L2StatusRow>,
    /// `l2_safe_blocks` input.
    pub l2_safe_blocks: ZSetHandle<L2SafeRow>,
    /// `unsafe_block_signer` input.
    pub unsafe_block_signer: ZSetHandle<UnsafeBlockSignerRow>,
    /// `l2_safe_canonical` output.
    pub l2_safe_canonical: ViewOutput<L2SafeRow>,
    /// `safe_head_updates` output.
    pub safe_head_updates: ViewOutput<SafeHeadUpdateRow>,
    /// `finalized_l2` output.
    pub finalized_l2: ViewOutput<FinalizedL2Row>,
    /// `current_signer` output.
    pub current_signer: ViewOutput<CurrentSignerRow>,
    /// `error_view` output (LATENESS violations and other runtime errors).
    pub error_view: ViewOutput<ErrorRow>,
}

impl core::fmt::Debug for Handles {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.write_str("Handles { .. }")
    }
}

/// Builds the circuit and names its handles.
///
/// The destructuring pattern is the single place that depends on the tuple order emitted by
/// the compiler; each element's type is pinned by the field it is assigned to.
pub fn build(config: CircuitConfig) -> Result<(DBSPHandle, Handles), dbsp::Error> {
    let (
        dbsp,
        (
            l1_blocks,
            l1_status,
            l2_status,
            l2_safe_blocks,
            unsafe_block_signer,
            l2_safe_canonical,
            safe_head_updates,
            finalized_l2,
            current_signer,
            error_view,
        ),
    ) = circuit::circuit(config)?;
    Ok((
        dbsp,
        Handles {
            l1_blocks,
            l1_status,
            l2_status,
            l2_safe_blocks,
            unsafe_block_signer,
            l2_safe_canonical,
            safe_head_updates,
            finalized_l2,
            current_signer,
            error_view,
        },
    ))
}
