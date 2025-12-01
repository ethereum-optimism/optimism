"""
Script to render a prompt instance by replacing placeholders with actual test and contract paths
from the first entry in the ranking JSON file.
"""

import json
from pathlib import Path


def load_ranking_data():
    """Load the ranking JSON file and return the first entry and run_id."""
    ranking_dir = Path(__file__).parent / "../tests_ranker" / "output"

    # Get the ranking file
    ranking_file = next(ranking_dir.glob("*_ranking.json"))

    # Extract run_id from filename
    run_id = ranking_file.stem.replace("_ranking", "")

    with open(ranking_file, "r") as f:
        data = json.load(f)

    if not data.get("entries"):
        raise ValueError(f"No entries found in {ranking_file.name}")

    return data["entries"][0], run_id


def load_prompt_template():
    """Load the prompt template markdown file."""
    prompt_file = Path(__file__).parent.parent.parent / "prompt" / "prompt.md"

    with open(prompt_file, "r") as f:
        return f.read()


def render_prompt(template, test_path, contract_path):
    """Replace the placeholders in the template with actual paths."""
    return template.replace("{TEST_PATH}", test_path).replace(
        "{CONTRACT_PATH}", contract_path
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
        first_entry, run_id = load_ranking_data()
        test_path = first_entry["test_path"]
        contract_path = first_entry["contract_path"]

        print(f"Using ranking from run {run_id}:")
        print(f"  Test path: {test_path}")
        print(f"  Contract path: {contract_path}")

        # Load prompt template
        template = load_prompt_template()

        # Render the prompt with actual paths
        rendered_prompt = render_prompt(template, test_path, contract_path)

        # Save the rendered prompt
        output_file = save_prompt_instance(rendered_prompt, run_id)

        print(f"Prompt instance saved to: {output_file}")

    except Exception as e:
        print(f"Error: {e}")
        return 1

    return 0


if __name__ == "__main__":
    exit(main())
