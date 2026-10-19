#!/usr/bin/env python3
"""Select unchanged production job definitions for three benchmark-only PRs."""
import copy
import json
import os
from pathlib import Path
import subprocess

BRANCHES = {f"codex/rwx-bench-{language}-20261019": language for language in ("go", "rust", "solidity")}


def identity(item):
    return (item, {}) if isinstance(item, str) else next(iter(item.items()))


def select(config, language):
    original = copy.deepcopy(config)
    workflow, targets, gate = {
        "go": ("main", ["go-tests"], "ci-gate"),
        "rust": ("rust-ci", ["rust-tests", "rust-cargo-hack"], "required-rust-ci"),
        "solidity": ("contracts-feature-tests", ["contracts-bedrock-tests main"], "required-contracts-ci"),
    }[language]
    items = copy.deepcopy(config["workflows"][workflow]["jobs"])
    if language == "solidity":
        for item in items:
            kind, params = identity(item)
            if params.get("name") == "contracts-bedrock-tests <<matrix.features>>":
                assert params["matrix"]["parameters"]["features"].count("main") == 1
                params.pop("matrix")
                params.update(name="contracts-bedrock-tests main", features="main")
    by_name = {params.get("name", kind): item for item in items for kind, params in [identity(item)]}
    names = set()

    def add(name):
        if name in names:
            return
        item = by_name[name]  # A missing prerequisite must fail closed.
        names.add(name)
        for dep in identity(item)[1].get("requires", []):
            add(dep if isinstance(dep, str) else next(iter(dep)))

    for target in targets:
        add(target)
    jobs = [item for item in items if identity(item)[1].get("name", identity(item)[0]) in names]
    # Keep the normal four required names. Only the selected workload's gate
    # queries dependencies; the other three use the existing skip companion.
    for name in ("ci-gate", "required-contracts-ci", "required-rust-ci", "required-rust-e2e"):
        params = {"context": ["circleci-api-token"]}
        if name == gate:
            params["requires"] = [{target: "terminal"} for target in targets]
        else:
            params["always-succeed"] = True
        jobs.append({name: params})
    config["workflows"] = {"benchmark-" + language: {"jobs": jobs}}
    assert all(config[k] == original[k] for k in original if k != "workflows")
    return config


def main():
    language = BRANCHES.get(os.environ.get("CIRCLE_BRANCH", ""))
    if language is None:
        print("Regular CI routing retained.")
        return
    config_path = Path("/tmp/merged-config.yml")
    params_path = Path("/tmp/pipeline-parameters.json")
    config = json.loads(subprocess.check_output(["yq", "-o=json", str(config_path)], text=True))
    selected = select(config, language)
    params = json.loads(params_path.read_text())
    if language == "go":
        # Match RWX's fresh verdicts; do not change compiler caches or grants.
        params["c-go_fresh_tests_effective"] = True
    config_path.write_text(json.dumps(selected, indent=2) + "\n")
    params_path.write_text(json.dumps(params, indent=2) + "\n")
    print("Selected benchmark-" + language + "; production job definitions and cache permissions retained.")


if __name__ == "__main__":
    main()
