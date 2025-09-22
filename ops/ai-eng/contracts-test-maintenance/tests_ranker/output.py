"""Output generation utilities for creating ranking JSON files."""

import json
from datetime import datetime, timezone
from pathlib import Path


def generate_ranking_json(
    entries: list[dict[str, str | int | float | None]], output_dir: Path
) -> Path:
    """Generate the ranking JSON file.

    Args:
        entries: List of test-to-contract mappings with scores.
        output_dir: Directory to write the output file.

    Returns:
        Path to the generated JSON file.
    """
    # Ensure output directory exists
    output_dir.mkdir(parents=True, exist_ok=True)

    # Sort entries by score (descending), with None scores at the end
    sorted_entries = sorted(
        entries, key=lambda x: (x["score"] is None, -(x["score"] or 0))
    )

    # Create ranking JSON
    ranking = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "entries": sorted_entries,
    }

    # Write to output file
    output_file = output_dir / "ranking.json"
    with output_file.open("w") as f:
        json.dump(ranking, f, indent=2)

    return output_file
