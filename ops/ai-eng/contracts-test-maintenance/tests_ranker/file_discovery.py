"""File discovery utilities for finding and filtering test files."""

from pathlib import Path

from exclusions import is_path_excluded


def find_test_files(contracts_bedrock: Path) -> list[Path]:
    """Find all test files in the contracts-bedrock test directory.

    Args:
        contracts_bedrock: Path to the contracts-bedrock directory.

    Returns:
        Sorted list of test file paths.
    """
    return sorted((contracts_bedrock / "test").rglob("*.t.sol"))


def filter_excluded_files(
    test_files: list[Path],
    contracts_bedrock: Path,
    excluded_dirs: list[Path],
    excluded_files: set[Path],
) -> list[Path]:
    """Filter out excluded test files based on exclusion rules.

    Args:
        test_files: List of test file paths.
        contracts_bedrock: Path to the contracts-bedrock directory.
        excluded_dirs: List of excluded directory paths.
        excluded_files: Set of excluded file paths.

    Returns:
        List of test files that are not excluded.
    """
    filtered_files = []
    for file_path in test_files:
        relative_path = file_path.relative_to(contracts_bedrock)
        if not is_path_excluded(relative_path, excluded_dirs, excluded_files):
            filtered_files.append(file_path)
    return filtered_files
