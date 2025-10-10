# AI Engineering Tools

> Collection of AI-driven automation systems for Optimism engineering workflows

## Overview

This directory contains AI-powered tools designed to automate and improve engineering processes across the Optimism monorepo. Each project uses AI to handle repetitive tasks, maintain code quality, and enhance developer productivity.

## Projects

### 🧪 [Contract Test Maintenance](contracts-test-maintenance/)

Automated system for maintaining and improving Solidity test coverage in `contracts-bedrock`.

- **Purpose**: Identify stale tests and improve coverage using AI
- **Status**: ✅ Active - Runs twice weekly
- **Tech**: Python + Devin AI API
- **Docs**:
  - [README](contracts-test-maintenance/README.md) - Overview and quick start
  - [Design Doc](https://www.notion.so/oplabs/AI-Contracts-Test-Maintenance-System-Design-Doc-24ff153ee1628035a3fccb0fa3e3b157) (Notion) - Business context and vision
  - [Tech Spec](https://www.notion.so/oplabs/AI-Contracts-Test-Maintenance-System-Technical-Specification-25af153ee162807895a9ffdb0452cfa2) (Notion) - Technical architecture
  - [Runbook](contracts-test-maintenance/docs/runbook.md) - Operational guide

**Quick Start**:
```bash
just ai-contracts-test
```

### 💎 [Graphite Code Review](graphite/)

AI-powered code review rules for Solidity files in pull requests.

- **Purpose**: Automated PR reviews following project standards
- **Status**: 🔧 Configuration
- **Tech**: Graphite + Diamond
- **Docs**: [graphite/rules.md](graphite/rules.md)

## Available Commands

Commands available via `just` in this directory:

```bash
# Contract Test Maintenance
just rank                  # Rank tests by staleness
just render                # Generate AI prompt
just devin                 # Execute with Devin
just ai-contracts-test     # Full pipeline
```

> See [justfile](justfile) for complete command list

## Adding New Projects

When adding a new AI-driven engineering tool:

1. Create a new directory: `ai-eng/your-project/`
2. Add project documentation: `your-project/README.md`
3. Update this file with project details
4. Add relevant commands to [justfile](justfile)
5. Follow existing patterns for CI integration

## Philosophy

These tools are designed to:
- ✅ **Automate repetitive tasks** that don't require human creativity
- ✅ **Maintain quality standards** consistently across the codebase
- ✅ **Free up engineering time** for high-value work
- ✅ **Run primarily in CI** with optional local execution for testing

## Support

Each project has its own documentation and support channels. See individual project READMEs for details.

**General Questions**: Contact EVM Safety Team

---

**Maintainer**: EVM Safety Team
**Projects**: 2 active

