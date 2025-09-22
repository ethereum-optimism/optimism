# AI Contract Test Maintenance System - Operational Runbook

This runbook covers the operation and maintenance of the AI Contract Test Maintenance System, which automates test maintenance workflows for Optimism smart contracts.

---

## 1. System Overview

The AI Contract Test Maintenance System is designed to continuously reduce technical debt by automating the maintenance of Solidity/Foundry tests in the `contracts-bedrock` monorepo. The system integrates with AI coding agent to generate safe, reviewable pull requests that improve test quality and coverage.

### Current Implementation Status (v0.1.0)

**✅ Currently Available:**
- **Test Ranking System** - Intelligent test selection based on staleness metrics
- **Manual Operation** - Command-line interface for generating rankings

**🚧 Planned Components (MVP):**
- Prompt renderer with template processing
- Devin API client integration
- CI scaffolding and testing framework
- Scheduled CI workflow

### Core Principle

Each test–contract pair receives a **score** based on staleness and age metrics. The system prioritizes tests whose contracts have diverged ahead, ensuring maintenance efforts focus on the most critical areas first.

---

## 2. Current System Architecture

### 2.1 Test Ranking System (v0.1.0)

The foundational component follows a modular architecture:

```
tests_ranker/
├── main.py                     # Entry point and orchestration
├── git_utils.py               # Git timestamp operations
├── exclusions.py              # TOML loading and exclusion logic
├── file_discovery.py          # Test file finding and filtering
├── contract_mapping.py        # Test-to-contract mapping logic
├── scoring.py                 # Staleness and score calculations
├── output.py                  # JSON generation and formatting
└── output/                    # Generated ranking files
    └── ranking.json
```

### Module Responsibilities

- **`git_utils.py`**: Retrieves commit timestamps for files using git commands
- **`exclusions.py`**: Loads exclusion rules from TOML configuration and checks paths
- **`file_discovery.py`**: Finds test files and filters out excluded ones
- **`contract_mapping.py`**: Maps test files to their corresponding source contracts
- **`scoring.py`**: Calculates staleness and priority scores using the two-branch algorithm
- **`output.py`**: Generates and formats the final ranking JSON file
- **`main.py`**: Orchestrates the entire workflow

### 2.2 Planned Full System Architecture

The complete system (future releases) will extend the current foundation:

```
AI Contract Test Maintenance System
├── prompt/                    # 🚧 Planned: Canonical AI prompt templates
├── tests_ranker/              # ✅ Current: Test selection & ranking
├── prompt_renderer/           # 🚧 Planned: Template processing & context injection
├── devin_client/              # 🚧 Planned: Devin API integration
└── ci/                        # 🚧 Planned: CI workflows & scheduling
```

---

## 3. Prerequisites

### System Requirements

- **Python 3.11+** - Required for all system components
- **Git access** - Must have read access to the Optimism repository
- **Repository setup** - Must be run from within the Optimism monorepo

### Permissions

- **Read access** to `packages/contracts-bedrock/` directory
- **Git history access** for commit timestamp retrieval
- **Write access** to `tests_ranker/output/` for ranking generation

---

## 4. Scoring Algorithm

### Staleness Calculation

```python
staleness_days = (contract_commit_ts - test_commit_ts) / 86400
```

**Interpretation:**

- Positive: Contract was updated more recently than the test
- Zero/Negative: Test is up to date or newer than the contract

### Two-Branch Scoring

The scoring system ensures continuous coverage with two cases:

**Case 1 – Contract newer than test:**

```python
if staleness_days > 0:
    score = staleness_days
```

→ Prioritizes contracts that have moved ahead of their tests

**Case 2 – Test is up to date or newer:**

```python
else:
    score = (now_ts - test_commit_ts) / 86400
```

→ Falls back to test age, so older tests are revisited first

### Fallback Handling

If only test timestamp is available:

```python
score = (now_ts - test_commit_ts) / 86400
```

---

## 5. Current Operations

### 5.1 Running the Test Ranker

```bash
# From the ai-eng directory
just rank

# Or directly
cd contracts-test-maintenance/tests_ranker && python3 main.py
```

### 5.2 Interpreting Results

The tool generates `tests_ranker/output/ranking.json` with the following structure:

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

### Entry Fields

- **`test_path`**: Relative path to test file from contracts-bedrock
- **`contract_path`**: Relative path to source contract from contracts-bedrock
- **`test_commit_ts`**: Unix timestamp of test file's last commit
- **`contract_commit_ts`**: Unix timestamp of contract file's last commit
- **`staleness_days`**: Calculated staleness (positive = contract newer)
- **`score`**: Priority score (higher = more urgent)

---

## 6. Configuration

### 6.1 Exclusion System

The tool respects exclusion rules defined in:

```
packages/contracts-bedrock/scripts/checks/test-validation/exclusions.toml
```

### Exclusion Types

- **File exclusions**: Specific test files to skip
- **Directory exclusions**: Entire directories to skip (paths ending with `/`)

### Example Configuration

```toml
[excluded_paths]
legacy_tests = [
    "test/legacy/",
    "test/specific-file.t.sol"
]
```

---

## 7. Technical Details

### 7.1 Selection Workflow

1. **Discovery**: Find all `*.t.sol` files in test directory
2. **Filtering**: Remove excluded files based on TOML configuration
3. **Mapping**: Match test files to corresponding source contracts
4. **Timestamping**: Get git commit timestamps for both files
5. **Scoring**: Calculate staleness and priority scores
6. **Ranking**: Sort by score (descending) and generate JSON

### 7.2 Expected Behavior

- **Priority**: Tests whose contracts are drifting ahead are handled first
- **Coverage**: Once caught up, system revisits tests by age (oldest first)
- **Continuity**: Every test will eventually be selected, ensuring full coverage
- **Scalability**: Handles hundreds of test files efficiently

---

## 8. Development Notes

### 8.1 Basic Troubleshooting

**If ranking.json is not generated:**
- Verify you're running from the correct directory (Optimism monorepo root)
- Ensure `packages/contracts-bedrock/test/` directory exists with `*.t.sol` files

**If git errors occur:**
- Ensure full git history is available: `git fetch --unshallow`

### 8.2 Runbook Evolution

This runbook focuses on the current v0.1.0 implementation (test ranker). Comprehensive operational procedures, performance monitoring, and maintenance workflows will be added as the system grows with additional components in future releases.

---

## 9. Future Roadmap

### 9.1 MVP Roadmap

**v0.2.0 - Prompt Renderer**
- Template processing system for AI prompts
- Context injection (test paths, contract paths, run metadata)
- Text format output for Devin API consumption

**v0.3.0 - Devin API Integration**
- API client for Devin session management
- Automated instruction sending and status polling
- Response handling and error management

**v0.4.0 - CI Scaffolding**
- CI pipeline testing framework
- Devin integration validation
- Local testing capabilities

**v0.5.0 - MVP Release**
- Scheduled CI workflow
- End-to-end automated test maintenance pipeline
- Basic monitoring and error handling

### 9.2 Integration Points

The current test ranker serves as the foundation for:
- **AI Agent Selection** - Provides ranked inputs for automated improvements
- **Pipeline Orchestration** - Enables targeted maintenance scheduling
- **Quality Metrics** - Tracks technical debt over time
- **Resource Planning** - Guides human reviewer workload distribution
