#!/usr/bin/env python3
"""Check that RWX's gate policy names the prerequisites of CircleCI's required gates.

For each gate, CircleCI's `requires:` list must name the same jobs as RWX's
policy, ops/ci/runtime/pr-gates.json, each as `terminal`. A row of the policy
names one job. The order of the list has no meaning in CircleCI, so the check
ignores it. The check prints each difference and exits with 1.
"""
import collections
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
MANIFEST = 'ops/ci/runtime/pr-gates.json'
# Gate: (CircleCI config fragment, workflow that runs the gate with its prerequisites).
GATES = {
    'ci-gate': ('.circleci/continue/main.yml', 'main'),
    'required-contracts-ci': ('.circleci/continue/main.yml', 'contracts-feature-tests'),
    'required-rust-ci': ('.circleci/continue/rust-ci.yml', 'rust-ci'),
    'required-rust-e2e': ('.circleci/continue/rust-e2e.yml', 'rust-e2e-ci'),
}
HINT = 'After a CircleCI gate change, update ops/ci/runtime/pr-gates.json.'


def rwx_prerequisites():
    """Return {gate: [name of each CircleCI job that RWX's policy mirrors]}."""
    manifest = json.loads((ROOT / MANIFEST).read_text())
    return {gate: [row['name'] for row in policy['dependencies']]
            for gate, policy in manifest['gates'].items()}


def circle_prerequisites(gate):
    """Return [(job, requirement)] from the `requires:` list of the gate in CircleCI.

    A workflow lists a job as `key` or as `{key: parameters}`, and `name:` in the
    parameters renames the job. The gate is the one job whose resulting name is
    `gate`. A `requires:` entry is `job`, which requires `success`, or
    `{job: requirement}`. Raises ValueError if the workflow does not run exactly
    one such job, or runs it with `always-succeed`.
    """
    if gate not in GATES:
        raise ValueError(f'{gate}: RWX has a policy for this gate, but GATES names no CircleCI workflow for it')
    config, workflow = GATES[gate]
    document = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(ROOT / config)], text=True))
    found = []
    for job in document.get('workflows', {}).get(workflow, {}).get('jobs', []):
        key, parameters = next(iter(job.items())) if isinstance(job, dict) else (job, {})
        if parameters.get('name', key) == gate:
            found.append(parameters)
    if len(found) != 1:
        raise ValueError(f'{gate}: workflow {workflow} of {config} has {len(found)} jobs with this name; expected 1')
    if found[0].get('always-succeed', False):
        raise ValueError(f'{gate}: workflow {workflow} of {config} runs the gate with always-succeed, so it has no prerequisites')
    return [next(iter(entry.items())) if isinstance(entry, dict) else (entry, 'success') for entry in found[0].get('requires', [])]


def problems():
    """Return one line for each difference; an empty list means that every gate is aligned."""
    lines = []
    for gate, wanted in rwx_prerequisites().items():
        try:
            circle = circle_prerequisites(gate)
        except ValueError as error:
            lines.append(str(error))
            continue
        names, wanted = collections.Counter(job for job, _ in circle), collections.Counter(wanted)
        found = [f'  only in CircleCI: {job}' for job in sorted((names - wanted).elements())]
        found += [f'  only in RWX: {job}' for job in sorted((wanted - names).elements())]
        found += [f'  not `terminal` in CircleCI: {job} ({requirement})' for job, requirement in circle if requirement != 'terminal']
        if found:
            config, workflow = GATES[gate]
            lines += [f"{gate}: `requires:` in workflow {workflow} of {config} differs from RWX's policy:", *found]
    return lines


if __name__ == '__main__':
    found = problems()
    if found:
        print('\n'.join([*found, HINT]), file=sys.stderr)
    sys.exit(1 if found else 0)
