# Changelog

All notable changes to the AI Contract Test Maintenance System will be documented in this file.

## [Unreleased]

## [0.1.0] - 2025-09-22

### Added

- **Test Ranking System** - Automated selector that ranks test files based on staleness metrics
  - Git-based staleness calculation using commit timestamps between tests and contracts
  - Two-branch scoring algorithm (staleness vs test age) for priority determination
  - Contract mapping system to link test files with corresponding source contracts
  - File discovery with exclusion support for filtered analysis
  - JSON output format (`ranking.json`) for ranked test results
- **Modular Python Architecture** - Clean separation of concerns across modules
  - `git_utils.py` - Git timestamp operations
  - `exclusions.py` - TOML-based exclusion rule loading
  - `file_discovery.py` - Test file discovery and filtering
  - `contract_mapping.py` - Test-to-contract mapping logic
  - `scoring.py` - Staleness and score calculations
  - `output.py` - JSON generation and formatting
  - `main.py` - Orchestration and entry point
- **Exclusion System Integration** - Respects existing test validation exclusions
  - Reads from `packages/contracts-bedrock/scripts/checks/test-validation/exclusions.toml`
  - Supports both file-level and directory-level exclusions
- **Command Line Interface** - Simple execution via justfile integration
  - `just rank` command for manual test ranking execution
  - Outputs comprehensive ranking to `tests_ranker/output/ranking.json`
- **Comprehensive Documentation** - Full operational and technical documentation
  - Operational runbook with architecture overview and usage instructions
  - Technical specification for planned AI system integration

