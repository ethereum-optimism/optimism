"""
Script to render a prompt instance by replacing placeholders with actual test and contract paths
from the first entry in the ranking JSON file.
"""

import json
from pathlib import Path


def load_ranking_data():
    """Load the ranking JSON file and return the first entry, stale entries, and run_id."""
    ranking_dir = Path(__file__).parent / "../tests_ranker" / "output"

    # Get the ranking file
    ranking_file = next(ranking_dir.glob("*_ranking.json"))

    # Extract run_id from filename
    run_id = ranking_file.stem.replace("_ranking", "")

    with open(ranking_file, "r") as f:
        data = json.load(f)

    if not data.get("entries"):
        raise ValueError(f"No entries found in {ranking_file.name}")

    stale_toml_entries = data.get("stale_toml_entries", [])

    return data["entries"][0], stale_toml_entries, run_id


def format_stale_entries(stale_entries):
    """Format stale TOML entries as markdown list.

    Args:
        stale_entries: List of dicts with test_path, contract_path, old_hash, new_hash

    Returns:
        Formatted markdown string with bullet list of stale entries, or "(none)" if empty
    """
    if not stale_entries:
        return "(none)"

    lines = []
    for entry in stale_entries:
        test_path = entry.get("test_path", "unknown")
        old_hash = entry.get("old_hash", "unknown")[:7]
        new_hash = entry.get("new_hash", "unknown")[:7]
        lines.append(f"- `{test_path}` (contract changed: {old_hash} → {new_hash})")

    return "\n".join(lines)


def load_prompt_template():
    """Load the prompt template markdown file."""
    prompt_file = Path(__file__).parent.parent.parent / "prompt" / "prompt.md"

    with open(prompt_file, "r") as f:
        return f.read()


def render_prompt(template, test_path, contract_path, stale_entries_list):
    """Replace the placeholders in the template with actual paths and stale entries.

    Args:
        template: The prompt template string
        test_path: Path to the test file
        contract_path: Path to the contract file
        stale_entries_list: Formatted markdown list of stale entries

    Returns:
        Rendered prompt with all placeholders replaced
    """
    return (
        template.replace("{TEST_PATH}", test_path)
        .replace("{CONTRACT_PATH}", contract_path)
        .replace("{{STALE_ENTRIES_LIST}}", stale_entries_list)
    )


def save_prompt_instance(rendered_prompt, run_id):
    """Save the rendered prompt to the output folder with run ID."""
    output_dir = Path(__file__).parent / "output"
    output_dir.mkdir(exist_ok=True)

    # Remove old prompt files
    for old_file in output_dir.glob("*_prompt.md"):
        old_file.unlink()

    filename = f"{run_id}_prompt.md"
    output_file = output_dir / filename

    with open(output_file, "w") as f:
        f.write(rendered_prompt)

    return output_file


def main():
    """Main function to render and save the prompt instance."""
    try:
        # Load ranking data and get run_id
        first_entry, stale_toml_entries, run_id = load_ranking_data()
        test_path = first_entry["test_path"]
        contract_path = first_entry["contract_path"]

        print(f"Using ranking from run {run_id}:")
        print(f"  Test path: {test_path}")
        print(f"  Contract path: {contract_path}")

        # Format stale entries for injection
        stale_entries_list = format_stale_entries(stale_toml_entries)
        if stale_toml_entries:
            print(f"  Stale TOML entries: {len(stale_toml_entries)}")

        # Load prompt template
        template = load_prompt_template()

        # Render the prompt with actual paths and stale entries
        rendered_prompt = render_prompt(template, test_path, contract_path, stale_entries_list)

        # Save the rendered prompt
        output_file = save_prompt_instance(rendered_prompt, run_id)

        print(f"Prompt instance saved to: {output_file}")

    except Exception as e:
        print(f"Error: {e}")
        return 1

    return 0


if __name__ == "__main__":
    exit(main())
