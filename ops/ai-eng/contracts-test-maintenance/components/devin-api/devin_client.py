"""
Script to create and monitor Devin AI sessions for contract test maintenance.
Loads prompt from the prompt renderer output and sends it to the Devin API,
then monitors the session until completion while logging the results.
"""

from datetime import datetime
import glob
import json
import os
from pathlib import Path
import time
import urllib.request

# Load .env file
if os.path.exists(".env"):
    with open(".env") as f:
        for line in f:
            if "=" in line and not line.strip().startswith("#"):
                key, value = line.strip().split("=", 1)
                os.environ[key] = value.strip("\"'").strip()


def find_prompt_file():
    """Find the latest generated prompt file from the prompt renderer output."""
    output_dir = "../prompt-renderer/output"
    prompt_files = glob.glob(f"{output_dir}/*_prompt.md")

    if not prompt_files:
        raise FileNotFoundError(f"No prompt files found in {output_dir}")

    if len(prompt_files) > 1:
        raise ValueError(f"Multiple prompt files found in {output_dir}: {prompt_files}")

    return prompt_files[0]


def load_prompt_from_file(file_path):
    """Load and return the contents of a prompt file."""
    with open(file_path, "r", encoding="utf-8") as f:
        return f.read().strip()


def log_session(session_id, status, session_data):
    """Log PR link and final status to JSONL file."""
    # Extract run_id and selected files from existing data
    try:
        prompt_file = find_prompt_file()
        run_id = os.path.basename(prompt_file).replace("_prompt.md", "")
        run_time = datetime.strptime(run_id, "%Y%m%d_%H%M%S").strftime(
            "%Y-%m-%d %H:%M:%S"
        )

        ranking_file = f"../tests_ranker/output/{run_id}_ranking.json"
        with open(ranking_file, "r") as f:
            data = json.load(f)
        selected_files = {
            "test_path": data["entries"][0]["test_path"],
            "contract_path": data["entries"][0]["contract_path"],
        }
    except Exception as e:
        print(f"Error retrieving run data: {e}")
        run_id = None
        run_time = None
        selected_files = {}

    # Read system version
    version_file = Path(__file__).parent.parent.parent / "VERSION"
    try:
        with open(version_file, "r") as f:
            system_version = f.read().strip()
    except (FileNotFoundError, IOError):
        system_version = "unknown"

    log_entry = {
        "system_version": system_version,
        "run_id": run_id,
        "run_time": run_time,
        "devin_session_id": session_id,
        "selected_files": selected_files,
        "status": status,
    }

    # Only add PR link if status is finished
    if status == "finished" and session_data:
        pr_url = session_data.get("pull_request", {}).get("url")
        if pr_url:
            log_entry["pull_request_url"] = pr_url

    with open("../../log.jsonl", "a") as f:
        f.write(json.dumps(log_entry) + "\n")


def _make_request(url, headers, data=None, method="GET"):
    """Make HTTP request to Devin API and return JSON response."""
    try:
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        if e.code == 504:
            print(f"Server timeout (504) - will retry")
            return None
        else:
            print(f"Request failed: {method} {url}")
            print(f"Error: {e}")
            raise
    except TimeoutError as e:
        print(f"Request timeout - will retry")
        return None
    except Exception as e:
        print(f"Request failed: {method} {url}")
        print(f"Error: {e}")
        raise


def _validate_environment():
    """Validate required environment variables."""
    api_key = os.getenv("DEVIN_API_KEY")
    base_url = os.getenv("DEVIN_API_BASE_URL")

    if not api_key:
        raise ValueError("DEVIN_API_KEY environment variable not set")
    if not base_url:
        raise ValueError("DEVIN_API_BASE_URL environment variable not set")

    return api_key, base_url


def _create_headers(api_key, content_type=None):
    """Create HTTP headers with authorization and optional content type."""
    headers = {"Authorization": f"Bearer {api_key}"}
    if content_type:
        headers["Content-Type"] = content_type
    return headers


def create_session(prompt):
    """Create a new Devin session with the given prompt."""
    api_key, base_url = _validate_environment()

    print(f"Creating session at: {base_url}/sessions")
    headers = _create_headers(api_key, "application/json")
    data = json.dumps({"prompt": prompt}).encode("utf-8")

    response_data = _make_request(f"{base_url}/sessions", headers, data, "POST")
    session_id = response_data["session_id"]

    print(f"Created session: {session_id}")
    return session_id


def monitor_session(session_id):
    """Monitor session status until completion."""
    api_key, base_url = _validate_environment()
    headers = _create_headers(api_key)
    last_status = None
    retry_delay = 60  # Start with 1 minute

    while True:
        try:
            status = _make_request(f"{base_url}/sessions/{session_id}", headers)

            # Handle server timeout (no response) - retry with backoff
            if status is None:
                print(f"Retrying in {retry_delay} seconds...")
                time.sleep(retry_delay)
                retry_delay = min(retry_delay * 2, 480)  # Cap at 8 minutes
                continue

            # Reset retry delay on successful request
            retry_delay = 60
            current_status = status.get("status_enum")

            # Handle Devin setup phase (status_enum is None but we got a response)
            if current_status is None:
                print("Devin is setting up...")
                time.sleep(5)
                continue

            # Only print when status changes and is meaningful
            if current_status and current_status != last_status:
                print(f"Status: {current_status}")
                last_status = current_status

            # Stop monitoring for non-working statuses
            if current_status in ["blocked", "expired", "finished"]:
                print(f"Session finished with status: {current_status}")
                log_session(session_id, current_status, status)
                return

            time.sleep(5)
        except KeyboardInterrupt:
            print(
                f"\nSession {session_id} is still running. Check Devin web interface for progress."
            )
            return


def send_prompt(prompt):
    """Create a session and monitor it until completion."""
    session_id = create_session(prompt)
    monitor_session(session_id)


if __name__ == "__main__":
    try:
        prompt_file = find_prompt_file()
        prompt = load_prompt_from_file(prompt_file)
        print(f"Using prompt from: {prompt_file}")
        send_prompt(prompt)
    except (FileNotFoundError, ValueError) as e:
        print(f"Error: {e}")
        exit(1)
