"""Exclusion handling for test files based on TOML configuration."""

import tomllib
from pathlib import Path


def load_exclusions(contracts_bedrock: Path) -> tuple[list[Path], set[Path]]:
    """Load and normalize exclusion paths from TOML configuration.

    Args:
        contracts_bedrock: Path to the contracts-bedrock directory.

    Returns:
        Tuple of (excluded_dirs, excluded_files) as normalized Path objects.

    Raises:
        FileNotFoundError: If exclusions.toml file is not found.
        tomllib.TOMLDecodeError: If TOML file is malformed.
    """
    exclusions_file = (
        contracts_bedrock / "scripts" / "checks" / "test-validation" / "exclusions.toml"
    )

    with exclusions_file.open("rb") as f:
        exclusions = tomllib.load(f)

    excluded_dirs: list[Path] = []
    excluded_files: set[Path] = set()

    # Flatten all exclusion paths from different categories
    all_excluded_paths = [
        path for paths in exclusions["excluded_paths"].values() for path in paths
    ]

    for path in all_excluded_paths:
        if path.endswith("/"):
            # Directory exclusion - store as Path object without trailing slash
            excluded_dirs.append(Path(path.rstrip("/")))
        else:
            # File exclusion - store as Path object in set for O(1) lookup
            excluded_files.add(Path(path))

    return excluded_dirs, excluded_files


def is_path_excluded(
    relative_path: Path, excluded_dirs: list[Path], excluded_files: set[Path]
) -> bool:
    """Check if a path should be excluded based on exclusion rules.

    Args:
        relative_path: Path relative to contracts-bedrock directory.
        excluded_dirs: List of excluded directory paths.
        excluded_files: Set of excluded file paths.

    Returns:
        True if the path should be excluded, False otherwise.
    """
    return relative_path in excluded_files or any(
        relative_path.is_relative_to(excluded_dir) for excluded_dir in excluded_dirs
    )
