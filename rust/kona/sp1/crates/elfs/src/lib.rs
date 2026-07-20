//! Runtime loading of SP1 guest ELFs.
//!
//! ELFs are gitignored build artifacts loaded from `KONA_SP1_ELF_DIR` (for example,
//! `rust/kona/sp1/elf` after `just build-elfs`) rather than embedded in host binaries.

use std::{
    env, fs, io,
    path::{Path, PathBuf},
};

/// Environment variable naming the directory that holds built guest ELFs.
pub const ELF_DIR_ENV: &str = "KONA_SP1_ELF_DIR";

fn elf_dir() -> io::Result<PathBuf> {
    env::var(ELF_DIR_ENV).map(PathBuf::from).map_err(|_| {
        io::Error::new(
            io::ErrorKind::NotFound,
            format!(
                "{ELF_DIR_ENV} unset; build ELFs (cd rust/kona/sp1 && just build-elfs) and set \
                 it to rust/kona/sp1/elf"
            ),
        )
    })
}

/// Load ELF `name` from `dir`, failing if the artifact is missing or empty.
///
/// This is separate from [`load_elf`] so callers and tests can supply a directory without
/// mutating the process environment.
pub fn load_elf_from(dir: &Path, name: &str) -> io::Result<Vec<u8>> {
    let path = dir.join(name);
    let bytes = fs::read(&path).map_err(|err| {
        io::Error::new(err.kind(), format!("failed to read guest ELF {}: {err}", path.display()))
    })?;
    if bytes.is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            format!("guest ELF {} is empty", path.display()),
        ));
    }
    Ok(bytes)
}

/// Load a guest ELF by base file name (for example, `range-elf`) from `KONA_SP1_ELF_DIR`.
pub fn load_elf(name: &str) -> io::Result<Vec<u8>> {
    load_elf_from(&elf_dir()?, name)
}

/// Load the `range` guest ELF.
pub fn range_elf() -> io::Result<Vec<u8>> {
    load_elf("range-elf")
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_directory(name: &str) -> PathBuf {
        env::temp_dir().join(format!("kona-sp1-elfs-{name}-{}", std::process::id()))
    }

    #[test]
    fn missing_elf_is_an_error() {
        let directory = test_directory("missing");
        let _ = fs::remove_dir_all(&directory);

        let err = load_elf_from(&directory, "range-elf").expect_err("missing ELF must fail");
        assert_eq!(err.kind(), io::ErrorKind::NotFound);
    }

    #[test]
    fn empty_elf_is_an_error() {
        let directory = test_directory("empty");
        let _ = fs::remove_dir_all(&directory);
        fs::create_dir_all(&directory).expect("test directory is created");
        fs::write(directory.join("range-elf"), []).expect("empty test ELF is written");

        let err = load_elf_from(&directory, "range-elf").expect_err("empty ELF must fail");
        assert_eq!(err.kind(), io::ErrorKind::InvalidData);

        fs::remove_dir_all(directory).expect("test directory is removed");
    }
}
