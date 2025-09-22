# AI Contract Test Maintenance System

## Overview

The AI Contract Test Maintenance System analyzes Solidity test files in the `contracts-bedrock` package and ranks them based on staleness metrics. It compares git commit timestamps between test files and their corresponding source contracts to identify which tests need attention most urgently.

The system uses a two-branch scoring algorithm: tests whose contracts have moved ahead receive priority based on staleness days, while up-to-date tests are ranked by age to ensure continuous coverage.

## Usage

```bash
# From the ai-eng directory
just rank
```

## Output

The tool generates `tests_ranker/output/ranking.json`:

```json
{
  "generated_at": "2025-09-19T16:49:56.517107+00:00",
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
- `test_path` - Relative path to test file from contracts-bedrock
- `contract_path` - Relative path to source contract from contracts-bedrock
- `test_commit_ts` - Unix timestamp of test file's last commit
- `contract_commit_ts` - Unix timestamp of contract file's last commit
- `staleness_days` - Calculated staleness (positive = contract newer)
- `score` - Priority score (higher = more urgent)
