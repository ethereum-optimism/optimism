"""Git utility functions for retrieving file commit information."""

import subprocess
from pathlib import Path
from typing import Optional


def get_file_commit_timestamp(file_path: Path, repo_root: Path) -> Optional[int]:
    """Get the timestamp of the last commit that modified a file.

    Args:
        file_path: Path to the file.
        repo_root: Path to the git repository root.

    Returns:
        Unix timestamp of the last commit, or None if unable to determine.
    """
    try:
        # Get relative path from repo root
        relative_path = file_path.relative_to(repo_root)

        # Run git log to get the last commit timestamp for this file
        result = subprocess.run(
            ["git", "log", "-1", "--format=%ct", "--", str(relative_path)],
            cwd=repo_root,
            capture_output=True,
            text=True,
            check=True,
        )

        if result.stdout.strip():
            return int(result.stdout.strip())

    except (subprocess.CalledProcessError, ValueError, OSError):
        pass

    return None
