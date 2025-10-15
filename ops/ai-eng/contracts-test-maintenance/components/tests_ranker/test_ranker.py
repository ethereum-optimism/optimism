"""Consolidated test ranking system for contracts-bedrock test files.

This module combines all functionality from the tests_ranker package into a single file.
It provides utilities for discovering test files, mapping them to source contracts,
calculating staleness metrics, and generating ranked output.
"""

from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import time
import tomllib
from typing import Optional
import urllib.request
import urllib.error


# === Git Utilities ===


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


# === Scoring Utilities ===


def calculate_staleness_days(
    test_commit_ts: Optional[int], contract_commit_ts: Optional[int]
) -> Optional[float]:
    """Calculate staleness in days between test and contract commits.

    Args:
        test_commit_ts: Unix timestamp of test file's last commit.
        contract_commit_ts: Unix timestamp of contract file's last commit.

    Returns:
        Staleness in days (positive if contract is newer), or None if timestamps unavailable.
    """
    if test_commit_ts is not None and contract_commit_ts is not None:
        return (contract_commit_ts - test_commit_ts) / 86400
    return None


def calculate_test_score(
    staleness_days: Optional[float], test_commit_ts: Optional[int]
) -> Optional[float]:
    """Calculate test priority score using two-branch scoring algorithm.

    Args:
        staleness_days: Staleness in days (positive if contract is newer).
        test_commit_ts: Unix timestamp of test file's last commit.

    Returns:
        Priority score (higher means more urgent), or None if cannot calculate.
    """
    now_ts = int(time.time())

    if staleness_days is not None:
        if staleness_days > 0:
            # Case 1: Contract newer than test - use staleness_days
            return staleness_days
        elif test_commit_ts is not None:
            # Case 2: Test up to date or newer - use test age
            return (now_ts - test_commit_ts) / 86400
    elif test_commit_ts is not None:
        # Fallback: only test timestamp available - use test age
        return (now_ts - test_commit_ts) / 86400

    return None


# === Contract Mapping Utilities ===


def get_base_paths() -> tuple[Path, Path, Path]:
    """Get base paths for repository, contracts, and output directory.

    Returns:
        Tuple of (repo_root, contracts_bedrock, output_dir) paths.
    """
    repo_root = Path(__file__).parent.parents[4]
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


# === Exclusion Utilities ===


def _get_test_path_from_artifact(artifact_url: str, headers: dict) -> Optional[str]:
    """Download and parse log.json artifact to extract test path."""
    try:
        req = urllib.request.Request(artifact_url, headers=headers)
        with urllib.request.urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode())
            return data.get("selected_files", {}).get("test_path")
    except (urllib.error.URLError, json.JSONDecodeError, KeyError):
        return None


def _get_job_artifacts(project_slug: str, job_number: int, headers: dict) -> list[dict]:
    """Get artifacts list for a specific job."""
    try:
        artifacts_url = f"https://circleci.com/api/v2/project/{project_slug}/{job_number}/artifacts"
        req = urllib.request.Request(artifacts_url, headers=headers)
        with urllib.request.urlopen(req, timeout=10) as response:
            return json.loads(response.read().decode()).get("items", [])
    except urllib.error.HTTPError as e:
        print(f"Error fetching artifacts (job {job_number}): {e.code} {e.reason}")
        print(f"Tried URL: {artifacts_url}")
        return []


def _get_successful_job_number(workflow_id: str, headers: dict) -> Optional[int]:
    """Get job number for successful ai-contracts-test job in a workflow."""
    try:
        jobs_url = f"https://circleci.com/api/v2/workflow/{workflow_id}/job"
        req = urllib.request.Request(jobs_url, headers=headers)
        with urllib.request.urlopen(req, timeout=10) as response:
            jobs_data = json.loads(response.read().decode())

        for job in jobs_data.get("items", []):
            if job.get("name") == "ai-contracts-test" and job.get("status") == "success":
                return job["job_number"]
        return None
    except urllib.error.HTTPError as e:
        print(f"Error fetching jobs (workflow {workflow_id}): {e.code} {e.reason}")
        return None


def _get_workflow_id(pipeline_id: str, headers: dict) -> Optional[str]:
    """Get workflow ID from pipeline if successful."""
    try:
        workflows_url = f"https://circleci.com/api/v2/pipeline/{pipeline_id}/workflow"
        req = urllib.request.Request(workflows_url, headers=headers)
        with urllib.request.urlopen(req, timeout=10) as response:
            workflows_data = json.loads(response.read().decode())

        workflows = workflows_data.get("items", [])
        if workflows and workflows[0].get("status") == "success":
            return workflows[0]["id"]
        return None
    except urllib.error.HTTPError as e:
        print(f"Error fetching workflow (pipeline {pipeline_id}): {e.code} {e.reason}")
        return None


def fetch_last_processed_from_circleci() -> list[Path]:
    """Fetch recently processed test files from CircleCI artifacts.

    Returns:
        List of test file paths from the last 3 successful runs.
    """
    circleci_token = os.getenv("CIRCLE_API_TOKEN")
    if not circleci_token:
        print("CIRCLE_API_TOKEN not found - skipping artifact check")
        return []

    print("Checking CircleCI for previous run artifacts...")
    excluded_paths = []

    try:
        headers = {"Circle-Token": circleci_token}
        project_slug = "gh/ethereum-optimism/optimism"
        # Always check develop branch for previous successful runs
        branch = "develop"
        two_weeks_ago = time.time() - (14 * 24 * 3600)

        # Get recent pipelines
        pipelines_url = f"https://circleci.com/api/v2/project/{project_slug}/pipeline?branch={branch}"
        req = urllib.request.Request(pipelines_url, headers=headers)
        with urllib.request.urlopen(req, timeout=10) as response:
            pipelines = json.loads(response.read().decode()).get("items", [])

        if not pipelines:
            print("No previous pipelines found")
            return []

        # Process recent pipelines (within 2 weeks)
        from datetime import datetime as dt
        for pipeline in pipelines:
            # Check age
            if pipeline.get("created_at"):
                pipeline_time = dt.fromisoformat(pipeline["created_at"].replace("Z", "+00:00")).timestamp()
                if pipeline_time < two_weeks_ago:
                    print("Reached pipelines older than 2 weeks, stopping search")
                    break

            # Get workflow → job → artifacts → test path
            workflow_id = _get_workflow_id(pipeline["id"], headers)
            if not workflow_id:
                continue

            job_number = _get_successful_job_number(workflow_id, headers)
            if not job_number:
                continue

            artifacts = _get_job_artifacts(project_slug, job_number, headers)
            for artifact in artifacts:
                if artifact["path"].endswith("log.json"):
                    test_path = _get_test_path_from_artifact(artifact["url"], headers)
                    if test_path:
                        print(f"Excluding recently processed file: {test_path}")
                        excluded_paths.append(Path(test_path))
                    break

        if excluded_paths:
            print(f"Excluded {len(excluded_paths)} recently processed file(s)")
        else:
            print("No recent successful runs found")

        return excluded_paths

    except (urllib.error.URLError, json.JSONDecodeError, ValueError, KeyError) as e:
        print(f"Could not fetch CircleCI artifacts: {e}")
        return []


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
    exclusions_file = Path(__file__).parent.parent.parent / "exclusion.toml"

    with exclusions_file.open("rb") as f:
        exclusions = tomllib.load(f)

    excluded_dirs: list[Path] = []
    excluded_files: set[Path] = set()

    # Get exclusion directories and files
    exclusion_config = exclusions.get("exclusions", {})
    exclusion_directories = exclusion_config.get("directories", [])
    exclusion_files = exclusion_config.get("files", [])

    # Process directory exclusions
    for directory in exclusion_directories:
        # Directory exclusion - store as Path object without trailing slash
        excluded_dirs.append(Path(directory.rstrip("/")))

    # Process file exclusions
    for file_path in exclusion_files:
        # File exclusion - store as Path object in set for O(1) lookup
        excluded_files.add(Path(file_path))

    # Add recently processed files from CircleCI artifacts (avoid immediate duplicates)
    last_processed_files = fetch_last_processed_from_circleci()
    for test_file in last_processed_files:
        excluded_files.add(test_file)

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


# === File Discovery Utilities ===


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


# === Output Generation Utilities ===


def generate_ranking_json(
    entries: list[dict[str, str | int | float | None]], output_dir: Path, run_id: str
) -> Path:
    """Generate the ranking JSON file.

    Args:
        entries: List of test-to-contract mappings with scores.
        output_dir: Directory to write the output file.
        run_id: Timestamp-based run identifier.

    Returns:
        Path to the generated JSON file.
    """
    # Ensure output directory exists
    output_dir.mkdir(parents=True, exist_ok=True)

    # Remove old ranking files
    for old_file in output_dir.glob("*_ranking.json"):
        old_file.unlink()

    # Sort entries by score (descending), with None scores at the end
    sorted_entries = sorted(
        entries, key=lambda x: (x["score"] is None, -(x["score"] or 0))
    )

    # Create ranking JSON
    ranking = {
        "run_id": run_id,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "entries": sorted_entries,
    }

    # Write to output file with run_id
    output_file = output_dir / f"{run_id}_ranking.json"
    with output_file.open("w") as f:
        json.dump(ranking, f, indent=2)

    return output_file


# === Main Application Logic ===


def create_test_entry(
    test_file: Path,
    source_contract: Path,
    contracts_bedrock: Path,
    repo_root: Path,
) -> dict[str, str | int | float | None]:
    """Create a single test entry with all calculated metrics.

    Args:
        test_file: Path to the test file.
        source_contract: Path to the corresponding source contract.
        contracts_bedrock: Path to the contracts-bedrock directory.
        repo_root: Path to the git repository root.

    Returns:
        Dictionary with test metrics and scores.
    """
    test_rel = str(test_file.relative_to(contracts_bedrock))
    source_rel = str(source_contract.relative_to(contracts_bedrock))

    # Get commit timestamps
    test_commit_ts = get_file_commit_timestamp(test_file, repo_root)
    contract_commit_ts = get_file_commit_timestamp(source_contract, repo_root)

    # Calculate metrics
    staleness_days = calculate_staleness_days(test_commit_ts, contract_commit_ts)
    score = calculate_test_score(staleness_days, test_commit_ts)

    return {
        "test_path": test_rel,
        "contract_path": source_rel,
        "test_commit_ts": test_commit_ts,
        "contract_commit_ts": contract_commit_ts,
        "staleness_days": staleness_days,
        "score": score,
    }


def collect_test_entries(
    contracts_bedrock: Path,
    excluded_dirs: list[Path],
    excluded_files: set[Path],
    repo_root: Path,
) -> list[dict[str, str | int | float | None]]:
    """Collect test file entries and map them to source contracts.

    Args:
        contracts_bedrock: Path to the contracts-bedrock directory.
        excluded_dirs: List of excluded directory paths.
        excluded_files: Set of excluded file paths.
        repo_root: Path to the git repository root.

    Returns:
        List of dictionaries with test_path, contract_path, commit timestamps, staleness_days, and score.
    """
    # Find and filter test files
    test_files = find_test_files(contracts_bedrock)
    filtered_files = filter_excluded_files(
        test_files, contracts_bedrock, excluded_dirs, excluded_files
    )

    entries = []
    for test_file in filtered_files:
        # Find corresponding source contract
        source_contract = find_source_contract(test_file, contracts_bedrock)

        if source_contract:
            entry = create_test_entry(
                test_file, source_contract, contracts_bedrock, repo_root
            )
            entries.append(entry)

    return entries


def main() -> None:
    """Main function to generate test ranking JSON."""
    try:
        # Generate unique run ID
        run_id = datetime.now().strftime("%Y%m%d_%H%M%S")
        print(f"Starting ranking run: {run_id}")

        # Get base paths
        repo_root, contracts_bedrock, output_dir = get_base_paths()

        # Load exclusions
        excluded_dirs, excluded_files = load_exclusions(contracts_bedrock)

        # Collect test entries
        entries = collect_test_entries(
            contracts_bedrock, excluded_dirs, excluded_files, repo_root
        )

        # Generate ranking JSON with run_id
        output_file = generate_ranking_json(entries, output_dir, run_id)

        print(f"Generated {output_file} with {len(entries)} entries")
        print(f"Run ID: {run_id}")

    except Exception as e:
        print(f"Error generating test ranking: {e}")
        raise


if __name__ == "__main__":
    main()
