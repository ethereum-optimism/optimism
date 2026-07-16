# Optimism AI Hooks

Repository hook bundle for AI coding agents.

Claude loads `.claude/settings.json` directly. Codex discovers this bundle through the workspace marketplace at `.agents/plugins/marketplace.json`; if the marketplace is not listed, install it once with:

```bash
codex plugin marketplace add .
codex plugin add optimism-ai-hooks --marketplace optimism-workspace
```

The reminder text and trigger logic live in `.agents/hooks/remind-prepush-checks.sh`, which is shared with Claude via symlink. The Codex plugin includes only a small bootstrap script so the installed plugin cache can find and invoke the shared repo script at runtime.
