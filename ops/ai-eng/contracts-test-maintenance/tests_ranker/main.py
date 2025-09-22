"""Main entry point for test ranking generation."""

from pathlib import Path

import contract_mapping
import exclusions
import file_discovery
import git_utils
import output
import scoring


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
    test_commit_ts = git_utils.get_file_commit_timestamp(test_file, repo_root)
    contract_commit_ts = git_utils.get_file_commit_timestamp(source_contract, repo_root)

    # Calculate metrics
    staleness_days = scoring.calculate_staleness_days(
        test_commit_ts, contract_commit_ts
    )
    score = scoring.calculate_test_score(staleness_days, test_commit_ts)

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
    test_files = file_discovery.find_test_files(contracts_bedrock)
    filtered_files = file_discovery.filter_excluded_files(
        test_files, contracts_bedrock, excluded_dirs, excluded_files
    )

    entries = []
    for test_file in filtered_files:
        # Find corresponding source contract
        source_contract = contract_mapping.find_source_contract(
            test_file, contracts_bedrock
        )

        if source_contract:
            entry = create_test_entry(
                test_file, source_contract, contracts_bedrock, repo_root
            )
            entries.append(entry)

    return entries


def main() -> None:
    """Main function to generate test ranking JSON."""
    try:
        # Get base paths
        repo_root, contracts_bedrock, output_dir = contract_mapping.get_base_paths()

        # Load exclusions
        excluded_dirs, excluded_files = exclusions.load_exclusions(contracts_bedrock)

        # Collect test entries
        entries = collect_test_entries(
            contracts_bedrock, excluded_dirs, excluded_files, repo_root
        )

        # Generate ranking JSON
        output_file = output.generate_ranking_json(entries, output_dir)

        print(f"Generated {output_file} with {len(entries)} entries")

    except Exception as e:
        print(f"Error generating test ranking: {e}")
        raise


if __name__ == "__main__":
    main()
