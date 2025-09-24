# AI Contract Test Maintenance System

## Overview

The AI Contract Test Maintenance System analyzes Solidity test files in the `contracts-bedrock` package and ranks them based on staleness metrics. It compares git commit timestamps between test files and their corresponding source contracts to identify which tests need attention most urgently.

The system uses a two-branch scoring algorithm: tests whose contracts have moved ahead receive priority based on staleness days, while up-to-date tests are ranked by age to ensure continuous coverage.

## Usage

```bash
# From the ai-eng directory

# Option 1: Run both steps in one command (recommended)
just prompt

# Option 2: Run steps individually
# Step 1: Rank tests by staleness
just rank

# Step 2: Generate AI prompt for the highest-priority test
just render
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
