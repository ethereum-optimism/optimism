# AI Contract Test Maintenance System - Runbook

> **Operational guide** for using and maintaining the system.

## Overview

The AI Contract Test Maintenance System is an automated CI workflow that identifies stale Solidity test files in the `contracts-bedrock` package and uses the Devin AI API to improve them. It ranks tests based on staleness metrics by comparing git commit timestamps between test files and their corresponding source contracts.

The system uses a **two-branch scoring algorithm**:
- Tests whose contracts have moved ahead receive priority based on staleness days
- Up-to-date tests are ranked by age to ensure continuous coverage rotation

### Key Features

- **Automated CI Integration**: Runs twice weekly on schedule (Monday/Thursday) or on-demand
- **Smart Prioritization**: Focuses on tests that are most out of sync with their contracts
- **Duplicate Prevention**: Automatically excludes recently processed files (last 2 weeks)
- **No-Changes Tracking**: Tests with comprehensive coverage tracked in TOML file to avoid redundant work
- **Stale Entry Detection**: Automatically identifies when tracked tests need re-analysis due to contract changes
- **Resilient Monitoring**: Handles long-running Devin sessions with retry logic
- **Full Audit Trail**: All runs logged with complete traceability

## Architecture

### System Components

```
contracts-test-maintenance/
├── VERSION                          # System version
├── exclusion.toml                   # Static exclusions configuration
├── no-need-changes.toml             # Tests with comprehensive coverage (auto-managed)
├── log.jsonl                        # Execution history and results
├── prompt/
│   └── prompt.md                   # AI instruction template (~2000 lines)
├── components/
│   ├── tests_ranker/               # Stage 1: Test ranking
│   │   ├── test_ranker.py
│   │   └── output/{run_id}_ranking.json
│   ├── prompt-renderer/            # Stage 2: Prompt generation
│   │   ├── render.py
│   │   └── output/{run_id}_prompt.md
│   └── devin-api/                  # Stage 3: AI execution
│       └── devin_client.py
└── docs/
    └── runbook.md                  # This document
```

### Three-Stage Pipeline

1. **Test Ranking** (`test_ranker.py`): Discovers tests, maps to contracts, calculates staleness
2. **Prompt Rendering** (`render.py`): Fills template with highest-priority test/contract paths
3. **Devin Execution** (`devin_client.py`): Creates AI session, monitors progress, logs results

## CI Integration

### CircleCI Workflow

The system is integrated into CircleCI via the `ai-contracts-test-workflow` workflow:

**Trigger Conditions** (from `.circleci/config.yml`):
- **Scheduled**: Runs twice weekly on `build_mon_thu` schedule (Monday/Thursday)
- **Manual Dispatch**: Can be triggered via CircleCI UI or API (see Usage section for details)
  - Set `ai_contracts_test_dispatch=true` to enable this workflow
  - Set `main_dispatch=false` to prevent the main workflow from also running

**Job Configuration**:
```yaml
ai-contracts-test:
  resource_class: medium
  docker:
    - image: cimg/base:2024.01
  steps:
    - utils/checkout-with-mise
    - run: just ai-contracts-test
    - store_artifacts: log.json
```

**Required Contexts**:
- `circleci-repo-readonly-authenticated-github-token`: For git operations
- `circleci-api-token`: For fetching previous run artifacts
- `devin-api`: Contains Devin API credentials

### Viewing Results

**Slack Notifications**:
- Automatic notification posted to #evm-safety Slack channel when PR is created
- Two notification types:
  - **Test improvements**: Includes PR URL, test file info, and reviewer guide link
  - **No changes needed**: Notifies team that test has comprehensive coverage with PR for TOML tracking update
- Helps expedite review process by alerting reviewers immediately

**In CircleCI**:
1. Navigate to the `ai-contracts-test` job
2. Check "Artifacts" tab for `log.json`
3. Review console output for run details

**In GitHub**:
- Completed sessions create PRs with branch name: `ai/improve-[contract-name]-coverage`
- PRs include comprehensive test improvements and documentation

## Usage

### Automatic Execution (CI)

The system runs automatically on schedule. No manual intervention needed.

### Manual Execution

**Local Testing** (from `ops/ai-eng/` directory):

⚠️ **Note**: Local execution is for testing/debugging only. The system is designed to run in CI.

```bash
# Prerequisites: Create components/devin-api/.env with credentials
# (See Prerequisites section above)

# Full pipeline
just ai-contracts-test

# Individual stages (for debugging)
just rank    # Stage 1: Rank tests by staleness
just render  # Stage 2: Generate prompt for highest-priority test
just devin   # Stage 3: Execute with Devin API (creates real session!)
```

**Manual CI Trigger**:

**Option 1: CircleCI UI** (Recommended)
1. Navigate to CircleCI → ethereum-optimism/optimism
2. Click "Trigger Pipeline" (top right)
3. Select branch (usually `develop`)
4. Add parameters:
   - `ai_contracts_test_dispatch`: `true`
   - `main_dispatch`: `false` (⚠️ Important: prevents main workflow from running)
5. Click "Trigger Pipeline"

**Option 2: CircleCI API**
```bash
curl -X POST https://circleci.com/api/v2/project/gh/ethereum-optimism/optimism/pipeline \
  -H "Circle-Token: $CIRCLE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "branch": "develop",
    "parameters": {
      "ai_contracts_test_dispatch": true,
      "main_dispatch": false
    }
  }'
```

**Note**: Setting `main_dispatch: false` is important to run ONLY the AI contracts test workflow. Otherwise, the main workflow will also execute (since `main_dispatch` defaults to `true`).

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
  ],
  "stale_toml_entries": [
    {
      "test_path": "test/L1/SystemConfig.t.sol",
      "contract_path": "src/L1/SystemConfig.sol",
      "old_hash": "abc1234",
      "new_hash": "def5678"
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
- `stale_toml_entries` - Array of no-need-changes.toml entries that need removal (contract hash changed)

### Prompt Renderer Output

The `just render` command generates a markdown file in `components/prompt-renderer/output/` with the name format `{run_id}_prompt.md`. This file contains the AI prompt template with:
- The highest-priority test and contract paths filled in
- List of stale no-need-changes.toml entries (if any) that Devin should remove before starting analysis
- Instructions for adding new entries when no changes are needed

For example, a run with ID `20250922_143052` will generate `20250922_143052_prompt.md`. The system automatically links prompts to their corresponding ranking runs through the shared run ID.

### Devin API Client

The Devin API client (`components/devin-api/devin_client.py`) automatically:

1. **Finds the latest prompt** from the prompt renderer output
2. **Creates a Devin session** with the generated prompt
3. **Monitors the session** until terminal state ("blocked", "expired", or "suspend_requested")
4. **Logs results** to `log.json` in the project root

#### Prerequisites

**For CI (automatic execution)**:
- Credentials are provided via CircleCI context (`devin-api`)
- No local configuration needed

**For local testing only**:
- Create `components/devin-api/.env` with:
  ```
  DEVIN_API_KEY=your_key_here
  DEVIN_API_BASE_URL=https://api.devin.ai/v1
  ```

#### Session Monitoring

The client monitors Devin sessions with resilient error handling:
- **30-second request timeout** to prevent hanging
- **Exponential backoff retry** for server errors (1min → 2min → 4min → 8min)
- **Patient monitoring** for long-running sessions (30+ minutes for CI completion)

#### Session Logging

All Devin sessions are automatically logged to `log.json` with:

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
- `status` - Final session status ("finished", "no_changes_needed", "blocked", "expired", "failed")
- `pull_request_url` - GitHub PR URL (present for both "finished" and "no_changes_needed" statuses)

#### Duplicate Prevention

The ranking system uses two complementary strategies to prevent duplicate work:

**1. No-Changes Tracking (with automatic reintegration)**:
- Tests with comprehensive coverage tracked in `no-need-changes.toml`
- Each entry includes contract git hash for validation
- Automatically excluded from ranking while contract remains unchanged
- **Automatically reintegrated** when contract changes (hash mismatch detected)
- Stale entries flagged in ranking output for Devin to remove

**2. Recent Processing Exclusion (2-week cooldown)**:
- Queries CircleCI API for recent successful runs
- Extracts test paths from stored `log.json` artifacts
- Temporarily excludes these files from ranking
- Files become available again after 2 weeks
- Prevents immediate re-ranking of files still under review

## Configuration

### Exclusion Rules (`exclusion.toml`)

Configure which test files should be permanently excluded from ranking:

```toml
[exclusions]
# Exclude entire directories
directories = [
    "test/invariants/",
    "test/opcm/",
    "test/scripts/",
    "test/setup/"
]

# Exclude specific files
files = [
    "test/L1/OPContractsManagerContractsContainer.t.sol",
    "test/dispute/lib/LibClock.t.sol",
    # ... more files
]
```

**Why exclude files?**:
- **Invariant tests**: Complex fuzzing tests requiring specialized approach
- **Scripts**: Deployment/utility scripts, not traditional unit tests
- **Setup helpers**: Infrastructure code without test coverage needs
- **OPCM tests**: Large integration tests handled separately

### Scoring Algorithm

The two-branch scoring algorithm prioritizes work:

```python
if contract_is_newer_than_test:
    score = staleness_days  # Higher staleness = higher priority
else:
    score = test_age_days   # Older tests need refresh
```

**Example Scenarios**:
- Contract updated 30 days after test → `score = 30` (high priority)
- Test updated yesterday, contract older → `score = 1` (low priority)
- Test last touched 200 days ago → `score = 200` (rotation priority)

### Environment Variables

**In CI (from CircleCI contexts)**:
- `CIRCLE_API_TOKEN`: For fetching artifacts from previous runs (from `circleci-api-token` context)
- `DEVIN_API_KEY`: Devin API authentication (from `devin-api` context)
- `DEVIN_API_BASE_URL`: Devin API endpoint (from `devin-api` context)

**For local testing only**:
- Set above variables in `components/devin-api/.env` file
- `CIRCLE_BRANCH` (optional): Branch to check for artifacts (defaults to `develop`)

## Workflow Details

### Stage 1: Test Ranking

**Process**:
1. Scan `packages/contracts-bedrock/test/` for `*.t.sol` files
2. For each test file, find corresponding source contract:
   - Try matching directory structure: `test/L1/Foo.t.sol` → `src/L1/Foo.sol`
   - Fall back to recursive search in `src/` directory
3. Get git commit timestamps using `git log -1 --format=%ct`
4. Calculate staleness: `contract_commit_ts - test_commit_ts`
5. Calculate priority score using two-branch algorithm
6. Apply exclusions:
   - Static exclusions from `exclusion.toml`
   - Dynamic exclusions from CircleCI artifacts (2-week window)
   - **No-changes tracking from `no-need-changes.toml`** (excluded while contract hash matches)
7. Detect stale TOML entries (contract hash changed since entry was added)
8. Sort by score (descending) and output to JSON with stale entries

**Output Fields**:
- `test_path`: Relative path from `contracts-bedrock/`
- `contract_path`: Relative path from `contracts-bedrock/`
- `test_commit_ts`: Unix timestamp (seconds since epoch)
- `contract_commit_ts`: Unix timestamp (seconds since epoch)
- `staleness_days`: Float, positive = contract is newer
- `score`: Float, higher = more urgent attention needed
- `stale_toml_entries`: Array of entries that need removal (contract changed)

### Stage 2: Prompt Rendering

**Process**:
1. Load ranking JSON from Stage 1 output
2. Extract first entry (highest priority test) and stale TOML entries
3. Format stale entries as markdown bullet list
4. Load prompt template from `prompt/prompt.md`
5. Replace placeholders:
   - `{TEST_PATH}` → test file path
   - `{CONTRACT_PATH}` → contract file path
   - `{{STALE_ENTRIES_LIST}}` → formatted list of stale entries (or "(none)")
6. Save rendered prompt to `output/` with same `run_id`

**The Prompt Template** contains:
- Role definition and task instructions
- **Stale entries cleanup instructions** (if any entries flagged)
- **No-changes tracking instructions** (how to add entries when no improvements needed)
- Comprehensive testing methodology (4 phases)
- Naming conventions for test contracts and functions
- Fuzz testing decision trees
- Organization rules (function-specific vs uncategorized)
- Error handling patterns and semgrep rules
- Validation requirements (`just pre-pr` must pass)
- PR submission guidelines

### Stage 3: Devin Execution

**Process**:
1. Find latest prompt file from Stage 2
2. Create Devin session via POST to `/sessions` endpoint with session creation retry logic
3. Monitor session with polling:
   - Poll every 30 seconds for status updates
   - Check for blocked state with 5-minute timeout
   - Parse `structured_output` for completion signal
   - Implement exponential backoff for server errors (502/504)
   - Continue monitoring until terminal state reached
4. Determine final status based on Devin state and structured output
5. Log results to `log.json` with full session details

**Devin API Terminal States**:
- `blocked`: Devin reached a blocking state (e.g., needs approval, PR created, or waiting)
- `expired`: Session timeout reached
- `suspend_requested` / `suspend_requested_frontend`: User manually stopped session

**Status Detection Logic**:
The client distinguishes between different outcomes by examining both Devin API status and structured output:

1. **`finished`**: Devin blocked + PR created for test improvements
2. **`no_changes_needed`**: Devin blocked + structured_output indicates comprehensive coverage + PR created for TOML entry
3. **`blocked`**: Devin blocked but no PR or unclear state
4. **`expired`**: Session timeout
5. User-stopped sessions are not logged

**Note**: The `status` field in `log.json` represents our interpretation of what happened, not the raw Devin API status.

**Error Handling**:
- 30-second timeout per HTTP request
- Retry with exponential backoff: 1min → 2min → 4min → 8min
- Patient monitoring for long-running CI operations (30+ minutes)

## Monitoring and Debugging

### Checking System Health

**Via Slack**:
- Monitor #evm-safety Slack channel for PR creation notifications
- Each notification includes PR URL and test file information
- Successful runs will always post to Slack when PR is created

**Via CircleCI**:
```bash
# View recent runs
circleci job list gh/ethereum-optimism/optimism

# Check specific workflow
circleci workflow get WORKFLOW_ID
```

**Via Log Files**:
```bash
# View latest execution
cat ops/ai-eng/contracts-test-maintenance/log.json | jq .

# Note: Only the latest run is stored (file is overwritten each time)
# For historical data, check CircleCI artifacts
```

### Common Issues

#### No tests ranked

**Symptoms**: Ranking JSON has empty `entries` array

**Causes**:
- All tests excluded via `exclusion.toml` (misconfiguration)
- No test files found in `packages/contracts-bedrock/test/`
- Test-to-contract mapping failed (contract files moved/renamed)

**Resolution**:
1. Check exclusion rules in `exclusion.toml` for overly broad patterns
2. Verify `packages/contracts-bedrock/test/` contains `*.t.sol` files
3. Run `just rank` locally to see detailed output and errors
4. Check if contracts were recently reorganized/renamed

#### Devin session stuck in "running"

**Symptoms**: Session never reaches terminal state

**Causes**:
- Devin internal issue
- Test improvements taking longer than expected
- Network connectivity problems

**Resolution**:
1. Check Devin dashboard for session status
2. Review session console output for progress
3. Wait for automatic timeout (typically 2 hours)
4. Contact Devin support if persistent

#### Tests fail after Devin improvements

**Symptoms**: PR shows failing CI checks

**Causes**:
- Test improvements introduced syntax errors
- Fuzz test constraints too restrictive
- Contract behavior misunderstood

**Resolution**:
1. Review PR diff for obvious issues
2. Check test output logs in CI
3. Devin will automatically attempt fixes
4. Manual intervention may be needed for edge cases

#### Duplicate work on same test

**Symptoms**: Same test file processed multiple times in short period

**Causes**:
- CircleCI artifact API failures
- Recent runs not yet stored artifacts
- Manual runs bypassing exclusion logic

**Resolution**:
1. Verify `CIRCLE_API_TOKEN` is set correctly
2. Check CircleCI artifacts are being stored
3. Wait for cooldown period (2 weeks)

### Debug Mode

Run individual stages locally to debug:

```bash
cd ops/ai-eng/contracts-test-maintenance

# Stage 1: See what tests are ranked
cd components/tests_ranker
python3 test_ranker.py
cat output/*_ranking.json | jq '.entries[0:5]'  # Top 5

# Stage 2: Check prompt rendering
cd ../prompt-renderer
python3 render.py
head -50 output/*_prompt.md  # Preview prompt

# Stage 3: Test Devin client (dry-run mode not available)
# Only run if you want to create a real session
cd ../devin-api
# python3 devin_client.py  # Creates real session!
```

### Logs and Artifacts

**Location of outputs**:
```
ops/ai-eng/contracts-test-maintenance/
├── log.json                               # Latest execution log (overwritten)
├── components/tests_ranker/output/
│   └── {run_id}_ranking.json             # Latest test ranking
└── components/prompt-renderer/output/
    └── {run_id}_prompt.md                # Latest generated prompt
```

**In CircleCI**:
- Job artifacts include `log.json` for each run
- Stored for 30 days by default
- Accessible via CircleCI API or web UI

## Maintenance

### Updating the Prompt Template

The prompt template (`prompt/prompt.md`) defines how Devin improves tests:

1. Edit the template file
2. Test changes locally: `just rank && just render`
3. Review generated prompt in `components/prompt-renderer/output/`
4. Commit changes to the repository
5. **Test the changes**:
   - **Option A**: Manually trigger a CI run (see "Manual CI Trigger" in Usage section)
   - **Option B**: Wait for next scheduled run (Monday/Thursday)
6. Monitor first few PRs to validate improvements

**Key sections to update**:
- Testing methodology phases
- Naming conventions
- Fuzz testing criteria
- Validation requirements

### Adding Exclusions

To permanently exclude tests from ranking:

1. Edit `exclusion.toml`
2. Add to `directories` array (for entire directories)
3. Or add to `files` array (for specific files)
4. Commit changes - next run will use updated exclusions

### Updating System Version

When making significant changes:

1. Update `VERSION` file
2. System version is logged in `log.json`
3. Helps track which system version processed each test

## Troubleshooting Guide

### System Not Running

**Check**: Is the schedule active?
```bash
# View CircleCI schedules
circleci schedule list gh/ethereum-optimism/optimism develop
```

**Verify**: Are contexts configured?
- `devin-api` context must exist with API credentials
- `circleci-api-token` context must have valid token

### No PRs Being Created

**Possible causes**:
1. Devin sessions blocked/failed (check Devin dashboard)
2. All high-priority tests recently processed
3. Devin API credentials expired

**Investigation**:
```bash
# Check latest run
cat log.json | jq .

# For historical data, check CircleCI artifacts
# Navigate to past builds and download log.json artifacts
```

### Understanding Score Values

**High scores (>100)**:
- Contract significantly newer than test (multiple months)
- Test hasn't been touched in a long time
- **Action**: Urgent attention needed

**Medium scores (10-100)**:
- Recent changes to contract
- Test moderately out of date
- **Action**: Normal priority

**Low scores (<10)**:
- Test recently updated
- Contract stable
- **Action**: Routine maintenance rotation

## Related Documentation

**Repository Files**:
- **Prompt Template**: `prompt/prompt.md` - Complete AI instructions (~2000 lines)
- **CI Configuration**: `.circleci/config.yml` - Workflow definition
- **Exclusion Config**: `exclusion.toml` - Static exclusion rules
- **Justfile**: `ops/ai-eng/justfile` - Command definitions

## Support

For issues or questions:
1. Check this runbook first
2. Review latest `log.json` entry
3. Check CircleCI job output and artifacts for historical data
4. Contact EVM Safety Team via standard channels

## Notes

- The system overwrites `log.json` on each run (only latest execution is stored locally)
- For execution history, access CircleCI artifacts (stored for 30 days)
- Old output files in `components/*/output/` are automatically cleaned up on each run
