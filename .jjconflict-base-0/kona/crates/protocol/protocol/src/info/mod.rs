//! Module containing L1 Attributes types (aka the L1 block info transaction).
//!
//! # Developer notes
//!
//! The structs implemented throughout this module form three chains of
//! embedding to emulate inheritance.  By `a < b` we denote that the fields of
//! struct `a` are a subset of the fields of struct `b`.  Delegation is
//! implemented through accessors and by help of the `ambassador` crate.  The
//! hardforks `Bedrock` and `Ecotone` each contain both fields that are used by
//! all later hardforks and some that are not.  They are implemented by
//! splitting them in two, e.g.  `L1BlockInfoBedrockBase` and
//! `L1BlockInfoBedrock`, where the former contains exactly the fields are used
//! by later hardforks and the latter embeds the former and then adds some
//! fields.
//!
//! The chains of embedding are:
//!
//! 1. L1BlockInfoBedrockBase < L1BlockInfoEcotoneBase < L1BlockInfoIsthmus < L1BlockInfoJovian
//! 2. L1BlockInfoBedrockBase < L1BlockInfoBedrock
//! 3. L1BlockInfoEcotoneBase < L1BlockInfoEcotone

mod variant;
pub use variant::L1BlockInfoTx;

mod bedrock;
pub use bedrock::{L1BlockInfoBedrock, L1BlockInfoBedrockFields, L1BlockInfoBedrockOnlyFields};

mod bedrock_base;
pub use bedrock_base::L1BlockInfoBedrockBaseFields;

mod ecotone;
pub use ecotone::{L1BlockInfoEcotone, L1BlockInfoEcotoneFields, L1BlockInfoEcotoneOnlyFields};

mod ecotone_base;
pub use ecotone_base::L1BlockInfoEcotoneBaseFields;

mod isthmus;
pub use isthmus::{L1BlockInfoIsthmus, L1BlockInfoIsthmusBaseFields, L1BlockInfoIsthmusFields};

mod jovian;
pub use jovian::{L1BlockInfoJovian, L1BlockInfoJovianBaseFields, L1BlockInfoJovianFields};

mod errors;
pub use errors::{BlockInfoError, DecodeError};
