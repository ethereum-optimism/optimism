# AI Contract Test Maintenance System

## Overview

The AI Contract Test Maintenance System analyzes Solidity test files in the `contracts-bedrock` package and ranks them based on staleness metrics. It compares git commit timestamps between test files and their corresponding source contracts to identify which tests need attention most urgently.

The system uses a two-branch scoring algorithm: tests whose contracts have moved ahead receive priority based on staleness days, while up-to-date tests are ranked by age to ensure continuous coverage.

## Usage

```bash
# From the ai-eng directory
just ai-contracts-test
```

Individual steps (for debugging):
```bash
just rank    # Rank tests by staleness
just render  # Generate prompt for highest-priority test
just devin   # Execute with Devin API
```

## Output

### Test Ranking Output

The `just rank` command generates `components/tests_ranker/output/{run_id}_ranking.json`:

```json
{
  "run_id": "20250922_143052",
  "generated_at": "2025-09-22T14:30:52.517107+00:00",
  "entries": [
    {
      "test_path": "test/L1/ProtocolVersions.t.sol",
      "contract_path": "src/L1/ProtocolVersions.sol",
      "test_commit_ts": 1746564380,
      "contract_commit_ts": 1738079001,
      "staleness_days": -98.21,
      "score": 135.84
    }
  ]
}
```

**Entry fields:**

- `run_id` - Unique identifier for this ranking run (YYYYMMDD_HHMMSS format)
- `generated_at` - ISO timestamp when the ranking was generated
- `test_path` - Relative path to test file from contracts-bedrock
- `contract_path` - Relative path to source contract from contracts-bedrock
- `test_commit_ts` - Unix timestamp of test file's last commit
- `contract_commit_ts` - Unix timestamp of contract file's last commit
- `staleness_days` - Calculated staleness (positive = contract newer)
- `score` - Priority score (higher = more urgent)

### Prompt Renderer Output

The `just render` command generates a markdown file in `components/prompt-renderer/output/` with the name format `{run_id}_prompt.md`. This file contains the AI prompt template with the highest-priority test and contract paths filled in, ready to be used for test maintenance analysis.

For example, a run with ID `20250922_143052` will generate `20250922_143052_prompt.md`. The system automatically links prompts to their corresponding ranking runs through the shared run ID.

### Devin API Client

The Devin API client (`components/devin-api/devin_client.py`) automatically:

1. **Finds the latest prompt** from the prompt renderer output
2. **Creates a Devin session** with the generated prompt
3. **Monitors the session** until completion ("blocked", "expired", or "finished")
4. **Logs results** to `log.jsonl` in the project root

#### Prerequisites

Devin API credentials in `components/devin-api/.env`

#### Session Monitoring

The client monitors Devin sessions with resilient error handling:
- **30-second request timeout** to prevent hanging
- **Exponential backoff retry** for server errors (1min → 2min → 4min → 8min)
- **Patient monitoring** for long-running sessions (30+ minutes for CI completion)

#### Session Logging

All Devin sessions are automatically logged to `log.jsonl` with:

```json
{
  "run_id": "20250924_160648",
  "run_time": "2025-09-24 16:06:48",
  "devin_session_id": "sess_abc123",
  "selected_files": {
    "test_path": "test/libraries/Storage.t.sol",
    "contract_path": "src/libraries/Storage.sol"
  },
  "status": "finished",
  "pull_request_url": "https://github.com/ethereum-optimism/optimism/pull/12345"
}
```

**Log fields:**
- `run_id` - Links to the ranking run that generated this session
- `run_time` - Human-readable timestamp of the run
- `devin_session_id` - Unique Devin session identifier
- `selected_files` - The test-contract pair that was worked on
- `status` - Final session status ("finished", "blocked", "expired", "failed")
- `pull_request_url` - GitHub PR URL (only present if status is "finished")

#### Duplicate Prevention

The ranking system automatically excludes files processed in the **last 7 days** to prevent duplicate work:
- Files with status `finished`, `blocked`, or `failed` are temporarily excluded
- After 7 days, files become available for ranking again (aligns with PR auto-close policy)
- This prevents immediate re-ranking of files still under review
