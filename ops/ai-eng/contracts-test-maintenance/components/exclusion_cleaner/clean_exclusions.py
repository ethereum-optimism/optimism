"""
Clean expired entries from exclusion.toml.

This module removes temporary exclusions that don't have open PRs.
Works in-memory and returns cleaned data without writing to file.
"""

import json
import os
from pathlib import Path
import sys
import tomllib
import urllib.request


def has_open_pr(file_path: str) -> bool:
    """Check if a file has an open PR using GitHub API.

    Args:
        file_path: Relative path to the test file (e.g., "test/L1/Portal.t.sol")

    Returns:
        True if file has an open PR, False otherwise
    """
    try:
        # Search for open PRs first
        repo = "ethereum-optimism/optimism"
        search_url = f'https://api.github.com/search/issues?q=repo:{repo}+is:pr+is:open&per_page=100'

        headers = {'Accept': 'application/vnd.github+json'}

        # Add auth token if available (check multiple env var names)
        github_token = (
            os.environ.get('GITHUB_TOKEN') or
            os.environ.get('GH_TOKEN') or
            os.environ.get('GITHUB_ACCESS_TOKEN') or
            os.environ.get('GITHUB_TOKEN_READONLY') or
            os.environ.get('GITHUB_TOKEN_GOVERNANCE')
        )
        if github_token:
            headers['Authorization'] = f'token {github_token}'

        req = urllib.request.Request(search_url, headers=headers)

        with urllib.request.urlopen(req, timeout=10) as response:
            search_data = json.loads(response.read().decode('utf-8'))

        # For each open PR, check if it modifies our file
        full_path = f'packages/contracts-bedrock/{file_path}'

        for pr in search_data.get('items', []):
            pr_number = pr['number']

            # Get files changed in this PR
            files_url = f'https://api.github.com/repos/{repo}/pulls/{pr_number}/files?per_page=100'
            req = urllib.request.Request(files_url, headers=headers)

            with urllib.request.urlopen(req, timeout=10) as response:
                files_data = json.loads(response.read().decode('utf-8'))

            # Check if our file is in the changed files
            for file in files_data:
                if file.get('filename') == full_path:
                    print(f"Found open PR #{pr_number} for {file_path}", file=sys.stderr)
                    return True

        return False

    except Exception as e:
        print(f"Warning: Could not check PR status for {file_path}: {e}", file=sys.stderr)
        # On error, assume file has open PR (safer to keep in exclusions)
        return True


def load_exclusions() -> dict:
    """Load exclusions from TOML file.

    Returns:
        Dictionary containing exclusions data
    """
    exclusions_file = Path(__file__).parent.parent.parent / "exclusion.toml"

    with exclusions_file.open("rb") as f:
        return tomllib.load(f)


def clean_temporary_exclusions_in_memory(data: dict) -> dict:
    """Remove temporary exclusions that don't have open PRs.

    Args:
        data: Exclusions data loaded from TOML

    Returns:
        Updated exclusions data (modified in-place and returned)
    """
    if "opened_PRs" not in data:
        print("No opened_PRs section found", file=sys.stderr)
        return data

    to_remove = []

    # Check each file for open PRs
    for file_path in data["opened_PRs"]:
        if not has_open_pr(file_path):
            to_remove.append(file_path)
            print(f"No open PR found for {file_path}, will remove from exclusions", file=sys.stderr)

    # Remove files without open PRs
    for file_path in to_remove:
        data["opened_PRs"].remove(file_path)

    if not to_remove:
        print("All files in opened_PRs have open PRs", file=sys.stderr)
    else:
        print(f"Cleaned {len(to_remove)} files without open PRs from opened_PRs", file=sys.stderr)

    return data


def get_cleaned_exclusions() -> dict:
    """Load and clean exclusions, returning ready-to-use dictionary.

    Returns:
        Dictionary with cleaned exclusions data
    """
    exclusions_data = load_exclusions()
    return clean_temporary_exclusions_in_memory(exclusions_data)


if __name__ == "__main__":
    # Get cleaned exclusions as a reusable dictionary
    cleaned_data = get_cleaned_exclusions()

    # Output as JSON for easy consumption by other tools
    print(json.dumps(cleaned_data, indent=2))
