#!/usr/bin/env bash
# Regenerates KonaSp1/Code/{Types,Funs}.lean from kona-sp1-client-utils::super_root.
# Requires CHARON and AENEAS to point at binaries built from the commits in this script.
# Charon: AeneasVerif/charon@798016509139eb51f0d764645f76d068d34ba5f6
# Aeneas: AeneasVerif/aeneas@73b7cf9f7442c5aba550c224bd7fd212d39ba32f
set -euo pipefail

: "${CHARON:?set CHARON to the charon binary}"
: "${AENEAS:?set AENEAS to the aeneas binary}"

VERIF_DIR=$(cd "$(dirname "$0")/.." && pwd)
CRATE_DIR="$VERIF_DIR/../crates/client"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT
LLBC="$WORK_DIR/kona_sp1_client_utils.llbc"

charon_args=(
  --preset=aeneas
  --start-from 'crate::super_root'
  --opaque 'crate::super_root::hash_super_root_proof'
  --opaque 'crate::super_root::encode_super_root_proof'
  --opaque 'crate::super_root::_::zk_dispute_game_public_values'
  # Range- and consolidation-program input checks, outside the aggregation proof.
  --opaque 'crate::super_root::ensure_range_previous_super_root_coverage'
  --opaque 'crate::super_root::ensure_consolidation_input_coverage'
)
# Macro-generated impls that Aeneas cannot translate and the proofs do not use.
for tr in 'serde_core::ser::Serialize' 'serde_core::de::Deserialize' 'serde_core::de::Visitor' \
          'core::fmt::Debug' 'core::fmt::Display' 'core::error::Error' \
          'alloy_sol_types::private::SolTypeValue<_>' 'alloy_sol_types::types::event::topic::EventTopic' \
          'alloy_sol_types::types::struct::SolStruct' 'alloy_sol_types::types::ty::SolType' \
          'alloy_sol_types::types::value::SolValue'; do
  charon_args+=(--exclude "{impl $tr for _}")
done

# The release guest wraps on overflow; extract with the same arithmetic.
(cd "$CRATE_DIR" && CARGO_PROFILE_DEV_OVERFLOW_CHECKS=false \
  "$CHARON" cargo "${charon_args[@]}" --dest-file "$LLBC")

"$AENEAS" -backend lean "$LLBC" \
  -dest "$VERIF_DIR" -subdir /KonaSp1/Code -split-files -namespace KonaSp1 \
  -filter-trait-methods
