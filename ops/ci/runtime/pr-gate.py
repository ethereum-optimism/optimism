#!/usr/bin/env python3
"""Report the RWX mirror of a CircleCI gate from the states of its prerequisite tasks.

ops/ci/runtime/pr-gates.json names, for each CircleCI gate, the jobs that the
gate requires and the RWX tasks that mirror each job. Tasks of one workload
definition form a group.

`receipt GROUP` runs in the workload. RWX passes it the state of each task of
the group and whether the route selected the workload. The receipt passes if
the workload is selected and every task succeeded, or the workload is not
selected and every task was skipped.

`status GATE` runs in the coordinator. RWX passes it the state of each receipt
of the gate. The status passes if every receipt succeeded. RWX fails a custom
status check when one of its tasks is skipped, so this task has no `if:`.

Each command writes its inputs and its result under .ci/pr-gates/ and exits 0
only on a pass.
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import sys

ROOT = Path(__file__).resolve().parents[3]
MANIFEST = ROOT / 'ops/ci/runtime/pr-gates.json'
STATES = ('succeeded', 'failed', 'skipped')


def variable(prefix, name, state):
    """Return the environment variable that carries one engine state of one task."""
    return '_'.join((prefix, name.replace('-', '_'), state)).upper()


def flag(name):
    value = os.environ[name]
    if value not in ('true', 'false'):
        raise ValueError(f'{name} is {value!r}; expected true or false')
    return value == 'true'


def states(prefix, names):
    """Return {name: {state: bool}} from the environment.

    Raises ValueError unless each task has exactly one true state. A task that
    has not finished has none, so a task that does not wait for it fails here.
    """
    result = {}
    for name in names:
        result[name] = {state: flag(variable(prefix, name, state)) for state in STATES}
        if sum(result[name].values()) != 1:
            raise ValueError(f'{name} is not in exactly one of the states {", ".join(STATES)}')
    return result


def gates():
    return json.loads(MANIFEST.read_text())['gates']


def group_tasks(group):
    """Return the tasks of a group. A task mirrors one CircleCI job, so no task repeats."""
    tasks = [task for gate in gates().values() for row in gate['dependencies'] if row['group'] == group for task in row['tasks']]
    if not tasks or len(tasks) != len(set(tasks)):
        raise ValueError(f'pr-gates.json has no task, or a repeated task, for group {group}')
    return sorted(tasks)


def report(kind, name, record, passed):
    """Write the record of one command and return the exit status."""
    directory = ROOT / '.ci/pr-gates' / kind / name
    shutil.rmtree(directory, ignore_errors=True)
    directory.mkdir(parents=True)
    (directory / 'verdict.json').write_text(json.dumps({**record, 'passed': passed}, indent=2, sort_keys=True) + '\n')
    return 0 if passed else 1


def receipt(group):
    selected = flag('GROUP_SELECTED')
    found = states('TASK', group_tasks(group))
    wanted = 'succeeded' if selected else 'skipped'
    unexpected = sorted(task for task, row in found.items() if not row[wanted])
    if unexpected:
        print(f'{group}: the route {"selected" if selected else "did not select"} this workload, '
              f'but these tasks are not {wanted}: {", ".join(unexpected)}', file=sys.stderr)
    return report('groups', group, {'group': group, 'selected': selected, 'tasks': found}, not unexpected)


def status(gate):
    dependencies = gates()[gate]['dependencies']
    selected = flag('GATE_SELECTED')
    found = states('GROUP', sorted({row['group'] for row in dependencies}))
    failed = sorted(group for group, row in found.items() if not row['succeeded'])
    if failed:
        print(f'{gate}: these receipts did not succeed: {", ".join(failed)}', file=sys.stderr)
    # A receipt succeeds for a selected workload whose tasks passed, and for an
    # unselected workload whose tasks were all skipped.
    outcome = 'passed' if selected else 'safe_skip'
    record = {'gate': gate, 'selected': selected, 'receipts': found,
              'dependencies': [{**row, 'outcome': outcome if found[row['group']]['succeeded'] else 'failed'} for row in dependencies]}
    return report('status', gate, record, not failed)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest='command', required=True)
    commands.add_parser('receipt').add_argument('name', metavar='GROUP')
    commands.add_parser('status').add_argument('name', metavar='GATE')
    arguments = parser.parse_args()
    sys.exit({'receipt': receipt, 'status': status}[arguments.command](arguments.name))
