# AI Contract Test Maintenance System

> Automated system for maintaining and improving Solidity test coverage in `contracts-bedrock` using AI-powered analysis.

## What It Does

Automatically identifies stale test files and uses Devin AI to improve test quality, coverage, and organization. Runs twice weekly (Monday/Thursday) in CircleCI.

**Three-Stage Pipeline**:
1. **Test Ranking** → Analyzes ~100 tests, calculates staleness scores
2. **Prompt Rendering** → Generates AI instructions for highest-priority test
3. **Devin Execution** → AI improves tests and creates PR

## Quick Start

### Running the System

**Automatic (CI)** - No action needed, runs on schedule

**Manual Trigger** - CircleCI UI:
1. Trigger Pipeline → ethereum-optimism/optimism
2. Set: `ai_contracts_test_dispatch: true`, `main_dispatch: false`

**Local Testing** - For development only:
```bash
cd ops/ai-eng
just ai-contracts-test  # Full pipeline
just rank               # Test ranking only
```

> 📖 **Full usage instructions**: [Runbook - Usage](docs/runbook.md#usage)

### Output

- **PRs**: Branch `ai/improve-[contract-name]-coverage` with test improvements
- **Logs**: `log.json` with execution details (also in CircleCI artifacts)
- **Artifacts**: CircleCI stores logs for 30 days

## Key Features

- ✅ Smart prioritization using staleness scoring
- ✅ Duplicate prevention (2-week cooldown)
- ✅ Resilient session monitoring with retry logic
- ✅ Full audit trail in CI artifacts

## Documentation

| Document | Purpose | Location |
|----------|---------|----------|
| **[🛠️ Runbook](docs/runbook.md)** | Operational guide and troubleshooting | Repository |
| **[🎯 Prompt Template](prompt/prompt.md)** | AI instructions (~2000 lines) | Repository |
| **[⚙️ Exclusion Config](exclusion.toml)** | Configure excluded tests | Repository |

### Quick Links

**Operational Guide**:
- [CI Integration](docs/runbook.md#ci-integration) - How it runs in CircleCI
- [Configuration](docs/runbook.md#configuration) - Exclusions and scoring
- [Monitoring](docs/runbook.md#monitoring-and-debugging) - Check system health
- [Troubleshooting](docs/runbook.md#troubleshooting-guide) - Common issues
- [Maintenance](docs/runbook.md#maintenance) - Update prompts/exclusions

## Configuration

**Exclude Tests** - Edit `exclusion.toml`:
```toml
[exclusions]
directories = ["test/invariants/", "test/scripts/"]
files = ["test/vendor/Initializable.t.sol"]
```

**Environment** - CI contexts provide credentials automatically. For local testing, see [Runbook - Prerequisites](docs/runbook.md#prerequisites).

## Monitoring

**Latest Run**:
```bash
cat log.json | jq .
```

**CircleCI**: Job `ai-contracts-test` → Check artifacts and console output

**GitHub PRs**: Search `is:pr author:app/devin ai/improve`

> 📊 **Detailed monitoring**: [Runbook - Monitoring](docs/runbook.md#monitoring-and-debugging)

## Troubleshooting

**Common Issues**:
- No tests ranked → Check `exclusion.toml`, see [runbook](docs/runbook.md#no-tests-ranked)
- Session stuck → Check Devin dashboard, see [runbook](docs/runbook.md#devin-session-stuck-in-running)
- Tests fail after improvements → Review PR diff, see [runbook](docs/runbook.md#tests-fail-after-devin-improvements)

> 🔧 **Full troubleshooting guide**: [Runbook - Troubleshooting](docs/runbook.md#troubleshooting-guide)

## Support

1. Check the [runbook](docs/runbook.md)
2. Review `log.json` and CircleCI artifacts
3. Contact EVM Safety Team

---

**Status**: ✅ Active
**Schedule**: Monday & Thursday
**Maintainer**: EVM Safety Team
**Version**: 1.0.0

