"""
Script to create and monitor Devin AI sessions for contract test maintenance.
Loads prompt from the prompt renderer output and sends it to the Devin API,
then monitors the session until completion while logging the results.
"""

from datetime import datetime, timezone
import glob
import json
import os
from pathlib import Path
import sys
import time
import urllib.request


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


def write_log(session_id, status, session_data):
    """Write log with full session information."""
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

    # Add PR link if status is finished or no_changes_needed (both create PRs)
    if status in ["finished", "no_changes_needed"] and session_data:
        pr_data = session_data.get("pull_request") or {}
        pr_url = pr_data.get("url")
        if pr_url:
            log_entry["pull_request_url"] = pr_url

    with open("../../log.json", "w") as f:
        json.dump(log_entry, f)


def _make_request(url, headers, data=None, method="GET"):
    """Make HTTP request to Devin API and return JSON response."""
    try:
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        if e.code in [502, 504]:
            print(f"Server error ({e.code}) - will retry")
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

    retry_delay = 60
    while True:
        response_data = _make_request(f"{base_url}/sessions", headers, data, "POST")

        if response_data is None:
            print(f"Session creation timed out, retrying in {retry_delay}s...")
            time.sleep(retry_delay)
            retry_delay = min(retry_delay * 2, 480)
            continue

        session_id = response_data["session_id"]
        print(f"Created session: {session_id}")
        return session_id


def monitor_session(session_id):
    """Monitor session status until completion."""
    api_key, base_url = _validate_environment()
    headers = _create_headers(api_key)
    last_status_enum = None
    retry_delay = 60  # Start with 1 minute
    setup_printed = False
    timeout_count = 0
    blocked_start_time = None  # Track when we first entered blocked state
    blocked_timeout = 300  # 5 minutes timeout for blocked state without outcome

    while True:
        try:
            api_response = _make_request(f"{base_url}/sessions/{session_id}", headers)

            # Handle server timeout (no response) - retry with backoff
            if api_response is None:
                timeout_count += 1
                # Only print after 3rd consecutive timeout to reduce noise
                if timeout_count >= 3:
                    print(f"API slow to respond, still monitoring session... (retry in {retry_delay}s)")
                time.sleep(retry_delay)
                retry_delay = min(retry_delay * 2, 480)  # Cap at 8 minutes
                continue

            # Reset retry delay and timeout count on successful request
            retry_delay = 60
            if timeout_count > 0:
                timeout_count = 0

            status_enum = api_response.get("status_enum")

            # Handle Devin setup phase (status_enum is None but we got a response)
            if status_enum is None:
                if not setup_printed:
                    print("Devin is setting up...")
                    setup_printed = True
                time.sleep(5)
                continue

            # Print setup completion message once
            if setup_printed and status_enum:
                print("Devin finished setup")
                setup_printed = False

            # Only print when status changes and is meaningful
            if status_enum and status_enum != last_status_enum:
                print(f"Status: {status_enum}")
                last_status_enum = status_enum

            # Stop monitoring for terminal statuses (only if we have valid status data)
            if api_response and status_enum in ["blocked", "finished", "expired", "suspend_requested", "suspend_requested_frontend"]:
                # Handle user stopping the session
                if status_enum in ["suspend_requested", "suspend_requested_frontend"]:
                    print("Session stopped by user")
                    return

                # Blocked or finished - check for outcome
                if status_enum in ["blocked", "finished"]:
                    # Ensure we have valid status data before accessing nested fields
                    if api_response is None:
                        print("Warning: Terminal status reached but no status data available, retrying...")
                        time.sleep(retry_delay)
                        continue

                    # Check structured output and PR (both should be populated when blocked)
                    # Note: Devin API nests structured_output twice: {structured_output: {structured_output: {...}}}
                    # The outer structured_output can be null, so we use `or {}` to handle that case
                    structured = (api_response.get("structured_output") or {}).get("structured_output") or {}
                    analysis_complete = structured.get("analysis_complete", False)
                    changes_needed = structured.get("changes_needed")

                    pr_data = api_response.get("pull_request") or {}
                    pr_url = pr_data.get("url")

                    # Case 1: Structured output indicates no changes needed
                    if analysis_complete and changes_needed is False:
                        reason = structured.get("reason", "Not provided")
                        print(f"Session completed - no changes needed")
                        print(f"Reason: {reason}")
                        if pr_url:
                            print(f"PR created for TOML tracking: {pr_url}")

                        write_log(session_id, "no_changes_needed", api_response)
                        return

                    # Case 2: PR created with test improvements (only if no structured output yet)
                    # We need to wait for structured_output to determine if this is a no-changes case
                    if pr_url and analysis_complete:
                        # We have both PR and completed analysis, and changes_needed != False
                        # This means actual test improvements were made
                        print(f"Session completed successfully - PR created: {pr_url}")
                        write_log(session_id, "finished", api_response)
                        return

                    # If blocked without complete data, keep waiting briefly
                    # Devin may still be populating the data
                    if status_enum == "blocked":
                        if blocked_start_time is None:
                            blocked_start_time = time.time()
                            print("Devin is blocked - waiting for complete outcome data...")

                        elapsed = time.time() - blocked_start_time
                        if elapsed > blocked_timeout:
                            print(f"Timeout: Devin blocked for {int(elapsed)}s without outcome - check Devin web interface")
                            sys.exit(1)

                        time.sleep(5)
                        continue

                    # Reset blocked timer if we move out of blocked state
                    blocked_start_time = None

                    # Finished without PR and no structured output = error
                    print(f"Session finished without PR or clear outcome - check Devin web interface")
                    sys.exit(1)

                # Expired = session timed out
                if status_enum == "expired":
                    print(f"Session expired")
                    write_log(session_id, "expired", api_response)
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
