//! This module contains the local types for the `kona-std-fpvm` crate.

/// File descriptors available to the `client` within the FPVM kernel.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FileDescriptor {
    /// Read-only standard input stream.
    StdIn,
    /// Write-only standard output stream.
    StdOut,
    /// Write-only standard error stream.
    StdErr,
    /// Read-only. Used to read the status of pre-image hinting.
    HintRead,
    /// Write-only. Used to provide pre-image hints
    HintWrite,
    /// Read-only. Used to read pre-images.
    PreimageRead,
    /// Write-only. Used to request pre-images.
    PreimageWrite,
}

impl From<FileDescriptor> for usize {
    fn from(fd: FileDescriptor) -> Self {
        match fd {
            FileDescriptor::StdIn => 0,
            FileDescriptor::StdOut => 1,
            FileDescriptor::StdErr => 2,
            FileDescriptor::HintRead => 3,
            FileDescriptor::HintWrite => 4,
            FileDescriptor::PreimageRead => 5,
            FileDescriptor::PreimageWrite => 6,
        }
    }
}

impl From<FileDescriptor> for i32 {
    fn from(fd: FileDescriptor) -> Self {
        usize::from(fd) as Self
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_file_descriptor_into_usize() {
        assert_eq!(usize::from(FileDescriptor::StdIn), 0);
        assert_eq!(usize::from(FileDescriptor::StdOut), 1);
        assert_eq!(usize::from(FileDescriptor::StdErr), 2);
        assert_eq!(usize::from(FileDescriptor::HintRead), 3);
        assert_eq!(usize::from(FileDescriptor::HintWrite), 4);
        assert_eq!(usize::from(FileDescriptor::PreimageRead), 5);
        assert_eq!(usize::from(FileDescriptor::PreimageWrite), 6);
    }

    #[test]
    fn test_file_descriptor_into_i32() {
        assert_eq!(i32::from(FileDescriptor::StdIn), 0);
        assert_eq!(i32::from(FileDescriptor::StdOut), 1);
        assert_eq!(i32::from(FileDescriptor::StdErr), 2);
        assert_eq!(i32::from(FileDescriptor::HintRead), 3);
        assert_eq!(i32::from(FileDescriptor::HintWrite), 4);
        assert_eq!(i32::from(FileDescriptor::PreimageRead), 5);
        assert_eq!(i32::from(FileDescriptor::PreimageWrite), 6);
    }
}
