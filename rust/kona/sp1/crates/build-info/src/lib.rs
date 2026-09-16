//! Build provenance embedded in the SP1 guest ELFs.

#[cfg(test)]
mod sha;

/// The monorepo commit this guest was built from, recoverable from a guest ELF with
/// `grep -aoE 'KONA_SP1_BUILD\{git_sha=[^}]*\}' <elf>`.
///
/// [`concat!`] keeps the delimiters and the sha in one `.rodata` literal; a record assembled from
/// several literals is not guaranteed to land contiguously.
pub const BUILD_MARKER: &str = concat!("KONA_SP1_BUILD{git_sha=", env!("KONA_SP1_GIT_SHA"), "}");

#[cfg(test)]
mod tests {
    use super::{BUILD_MARKER, sha};

    /// Pins the marker against the shape the ELF scanners in the justfile and CI accept.
    #[test]
    fn marker_payload_is_a_shape_the_scanners_accept() {
        let payload = BUILD_MARKER
            .strip_prefix("KONA_SP1_BUILD{git_sha=")
            .and_then(|rest| rest.strip_suffix('}'))
            .expect("marker is delimited");
        assert!(sha::is_valid(payload), "marker payload {payload:?} is not an accepted shape");
    }

    #[test]
    fn accepts_every_shape_the_build_can_produce() {
        let hex = "0af3".repeat(10);
        assert!(sha::is_valid("unknown"));
        assert!(sha::is_valid(&hex));
        assert!(sha::is_valid(&format!("{hex}-dirty")));
        assert!(sha::is_valid(&format!("{hex}-custom")));
        assert!(sha::is_valid(&format!("{hex}-dirty-custom")));
    }

    #[test]
    fn rejects_shapes_the_scanners_would_miss() {
        assert!(!sha::is_valid(""));
        assert!(!sha::is_valid("-dirty"));
        assert!(!sha::is_valid(&"0".repeat(39)));
        assert!(!sha::is_valid(&"0".repeat(41)));
        assert!(!sha::is_valid(&"A".repeat(40)));
        assert!(!sha::is_valid(&format!("{}-custom-dirty", "0".repeat(40))));
    }
}
