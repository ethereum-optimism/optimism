"""Contract mapping utilities for linking test files to source contracts."""

from pathlib import Path
from typing import Optional


def get_base_paths() -> tuple[Path, Path, Path]:
    """Get base paths for repository, contracts, and output directory.

    Returns:
        Tuple of (repo_root, contracts_bedrock, output_dir) paths.
    """
    repo_root = Path(__file__).parent.parent.parent.parent.parent
    contracts_bedrock = repo_root / "packages" / "contracts-bedrock"
    output_dir = Path(__file__).parent / "output"
    return repo_root, contracts_bedrock, output_dir


def find_source_contract(
    test_file_path: Path, contracts_bedrock: Path
) -> Optional[Path]:
    """Map a test file to its corresponding source contract.

    Args:
        test_file_path: Path to the test file (.t.sol).
        contracts_bedrock: Path to the contracts-bedrock directory.

    Returns:
        Path to the corresponding source contract, or None if not found.
    """
    # Get the test file name without .t.sol extension
    test_name = test_file_path.stem.replace(".t", "")

    # Get the relative directory structure from test/
    test_relative = test_file_path.relative_to(contracts_bedrock / "test")
    test_dir = test_relative.parent

    # Try to find source contract in src/ with same directory structure
    potential_source = contracts_bedrock / "src" / test_dir / f"{test_name}.sol"

    if potential_source.exists():
        return potential_source

    # Try without directory structure in src/
    for src_subdir in (contracts_bedrock / "src").rglob("*.sol"):
        if src_subdir.name == f"{test_name}.sol":
            return src_subdir

    return None
