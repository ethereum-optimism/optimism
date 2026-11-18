"""Check the gate policy's wiring in the RWX definitions, and the receipt and status verdicts."""
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location('pr_gate', ROOT / 'ops/ci/runtime/pr-gate.py')
G = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(G)
RECEIPT = re.compile(r'python3 ops/ci/runtime/pr-gate\.py receipt (\S+)')


def definition(path):
    return json.loads(subprocess.check_output(['yq', '-o=json', 'explode(.)', str(path)], text=True))


def tasks(path):
    return {task['key']: task for task in definition(path)['tasks']}


def terminal(key):
    return f'({key}.succeeded || {key}.failed || {key}.skipped)'


def groups(gate):
    return sorted({row['group'] for row in G.gates()[gate]['dependencies']})


class WiringTests(unittest.TestCase):
    """pr-gates.json and the definitions must name the same tasks. A hosted run would fail on a difference."""

    @classmethod
    def setUpClass(cls):
        # {group: (definition file name, receipt task)}
        cls.receipts = {}
        for path in sorted((ROOT / '.rwx').glob('*.yml')):
            for task in tasks(path).values():
                match = RECEIPT.fullmatch(task['run'].strip()) if isinstance(task.get('run'), str) else None
                if match:
                    assert match[1] not in cls.receipts, match[1]
                    cls.receipts[match[1]] = (path.name, task)

    def test_each_group_has_one_receipt_that_reads_every_task_of_the_group(self):
        self.assertEqual(set(self.receipts), {group for gate in G.gates() for group in groups(gate)})
        for group, (name, receipt) in self.receipts.items():
            with self.subTest(group=group):
                names = G.group_tasks(group)
                workload = tasks(ROOT / '.rwx' / name)
                self.assertLessEqual(set(names), set(workload))
                wanted = {G.variable('TASK', key, state): '${{ tasks.%s.%s }}' % (key, state) for key in names for state in G.STATES}
                self.assertEqual({key: value for key, value in receipt['env'].items() if key.startswith('TASK_')}, wanted)
                # The receipt waits for every task, and runs whatever their states are.
                self.assertEqual(sorted(receipt['after'].removeprefix('${{ ').removesuffix(' }}').split(' && ')),
                                 sorted(terminal(key) for key in names))
                self.assertRegex(receipt['env']['GROUP_SELECTED'], r'^\$\{\{ tasks\.route\.values\.run-[a-z0-9-]+ \}\}$')
                self.assertIs(receipt['cache'], False)

    def test_each_gate_has_one_status_task_that_reads_every_receipt_of_the_gate(self):
        coordinator = definition(ROOT / '.rwx/pr-gates.yml')
        graph = {task['key']: task for task in coordinator['tasks']}
        calls = {task['call'].removeprefix('${{ run.dir }}/'): key for key, task in graph.items()
                 if str(task.get('call', '')).startswith('${{ run.dir }}/')}
        checks = coordinator['on']['github']['push']['status-checks']['custom']
        for gate in G.gates():
            with self.subTest(gate=gate):
                found = [task for task in graph.values() if task.get('run') == 'python3 ops/ci/runtime/pr-gate.py status ' + gate]
                self.assertEqual(len(found), 1)
                status = found[0]
                wanted, waits = {}, set()
                for group in groups(gate):
                    name, receipt = self.receipts[group]
                    waits.add(terminal(calls[name]))
                    for state in G.STATES:
                        wanted[G.variable('GROUP', group, state)] = '${{ tasks.%s.tasks.%s.%s }}' % (calls[name], receipt['key'], state)
                self.assertEqual({key: value for key, value in status['env'].items() if key.startswith('GROUP_')}, wanted)
                self.assertEqual(set(status['after'].removeprefix('${{ ').removesuffix(' }}').split(' && ')), waits)
                # RWX fails a custom status check when one of its tasks is skipped.
                self.assertNotIn('if', status)
                self.assertIs(status['cache'], False)
                self.assertEqual([check['tasks'] for check in checks].count(status['key']), 1)

    def test_each_workload_has_one_status_check_that_names_its_call(self):
        # `rwx lint` accepts a check that names no task, and a run from the CLI reports no checks.
        coordinator = definition(ROOT / '.rwx/pr-gates.yml')
        checks = coordinator['on']['github']['push']['status-checks']['custom']
        named = [check['tasks'] for check in checks]
        self.assertLessEqual(set(named), {task['key'] for task in coordinator['tasks']})
        calls = [task['key'] for task in coordinator['tasks'] if str(task.get('call', '')).startswith('${{ run.dir }}/') and 'init' in task]
        self.assertEqual(sorted(key for key in named if key.startswith('native-')), sorted(calls))
        self.assertEqual(len({check['name'] for check in checks}), len(checks))
        # The coordinator is the one definition that a push starts.
        for path in sorted((ROOT / '.rwx').glob('*.yml')):
            self.assertEqual('github' in definition(path)['on'], path.name == 'pr-gates.yml', path.name)

    def test_each_gated_task_runs_in_every_run(self):
        # RWX reuses the result of a task whose inputs it has seen. A gated task
        # must run anyway: it sets `cache: false`, or it includes the run id and
        # the attempt number in its cache key, which keeps a tool cache usable.
        for group, (name, _) in self.receipts.items():
            workload = tasks(ROOT / '.rwx' / name)
            for key in G.group_tasks(group):
                task = workload[key]
                if 'call' in task:
                    package = definition(ROOT / '.rwx' / task['call'].removeprefix('${{ run.dir }}/'))
                    leaves = [leaf for leaf in package['tasks'] if 'run' in leaf]
                    self.assertEqual(len(leaves), 1, key)
                    task = leaves[0]
                fresh = task.get('cache') is False or all(
                    task.get('env', {}).get(variable) == {'cache-key': 'included'} for variable in ('RWX_RUN_ID', 'RWX_TASK_ATTEMPT_NUMBER'))
                self.assertTrue(fresh, f'{name}: {key}')


class VerdictTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        patcher = patch.object(G, 'ROOT', self.root)
        patcher.start()
        self.addCleanup(patcher.stop)

    def run_command(self, command, name, prefix, states, **flags):
        """Run a command with one state per task. Return its exit status and its record."""
        environment = {G.variable(prefix, key, state): str(state == actual).lower()
                       for key, actual in states.items() for state in G.STATES}
        environment.update(flags)
        with patch.dict(os.environ, environment, clear=True):
            status = command(name)
        kind = 'groups' if command is G.receipt else 'status'
        return status, json.loads((self.root / '.ci/pr-gates' / kind / name / 'verdict.json').read_text())

    def test_receipt_passes_when_selected_tasks_succeeded_or_unselected_tasks_were_skipped(self):
        group = 'rust-workspace'
        names = G.group_tasks(group)
        self.assertGreater(len(names), 20)
        for selected, state, expected in [('true', 'succeeded', 0), ('true', 'failed', 1), ('true', 'skipped', 1),
                                          ('false', 'skipped', 0), ('false', 'succeeded', 1), ('false', 'failed', 1)]:
            with self.subTest(selected=selected, state=state):
                status, record = self.run_command(G.receipt, group, 'TASK', dict.fromkeys(names, state), GROUP_SELECTED=selected)
                self.assertEqual(status, expected)
                self.assertEqual(record['passed'], expected == 0)
                self.assertEqual(record['selected'], selected == 'true')
                self.assertTrue(all(row[state] for row in record['tasks'].values()))

    def test_receipt_fails_when_one_task_of_many_differs(self):
        group = 'main-go'
        names = G.group_tasks(group)
        for selected, good, bad in [('true', 'succeeded', 'skipped'), ('true', 'succeeded', 'failed'), ('false', 'skipped', 'succeeded')]:
            with self.subTest(selected=selected, bad=bad):
                states = {**dict.fromkeys(names, good), names[-1]: bad}
                status, record = self.run_command(G.receipt, group, 'TASK', states, GROUP_SELECTED=selected)
                self.assertEqual(status, 1)
                self.assertFalse(record['passed'])
                self.assertTrue(record['tasks'][names[-1]][bad])

    def test_receipt_rejects_states_that_rwx_did_not_supply_as_one_terminal_state(self):
        group = 'main-kontrol'
        environment = {'GROUP_SELECTED': 'true', 'TASK_CHECK_SUCCEEDED': 'true', 'TASK_CHECK_FAILED': 'false', 'TASK_CHECK_SKIPPED': 'false'}
        with patch.dict(os.environ, environment, clear=True):
            self.assertEqual(G.receipt(group), 0)
        # A task that has not finished has no true state: the receipt did not wait for it.
        for change in ({'TASK_CHECK_SUCCEEDED': 'false'}, {'TASK_CHECK_SKIPPED': 'true'}, {'TASK_CHECK_FAILED': ''},
                       {'TASK_CHECK_FAILED': 'True'}, {'GROUP_SELECTED': 'yes'}):
            with self.subTest(change=change), patch.dict(os.environ, {**environment, **change}, clear=True):
                with self.assertRaises(ValueError):
                    G.receipt(group)
        # A receipt without the variable of a task fails: the definition must pass every state.
        del environment['TASK_CHECK_SKIPPED']
        with patch.dict(os.environ, environment, clear=True), self.assertRaises(KeyError):
            G.receipt(group)

    def test_status_passes_only_when_every_receipt_succeeded(self):
        for gate in G.gates():
            names = groups(gate)
            requires = [row['name'] for row in G.gates()[gate]['dependencies']]
            for selected, outcome in [('true', 'passed'), ('false', 'safe_skip')]:
                with self.subTest(gate=gate, selected=selected):
                    status, record = self.run_command(G.status, gate, 'GROUP', dict.fromkeys(names, 'succeeded'), GATE_SELECTED=selected)
                    self.assertEqual(status, 0)
                    self.assertTrue(record['passed'])
                    # The record names each CircleCI prerequisite once, in the policy's order.
                    self.assertEqual([row['name'] for row in record['dependencies']], requires)
                    self.assertEqual({row['outcome'] for row in record['dependencies']}, {outcome})
            for state in ('failed', 'skipped'):
                with self.subTest(gate=gate, state=state):
                    states = {**dict.fromkeys(names, 'succeeded'), names[0]: state}
                    status, record = self.run_command(G.status, gate, 'GROUP', states, GATE_SELECTED='true')
                    self.assertEqual(status, 1)
                    self.assertFalse(record['passed'])
                    outcomes = {row['group']: row['outcome'] for row in record['dependencies']}
                    self.assertEqual(outcomes[names[0]], 'failed')
                    self.assertEqual({outcomes[name] for name in names[1:]} or {'passed'}, {'passed'})

    def test_policy_names_each_circleci_job_and_each_task_once(self):
        seen = {}
        for gate, policy in G.gates().items():
            # `also` names the other CircleCI jobs that the tasks of a row mirror.
            names = [name for row in policy['dependencies'] for name in (row['name'], *row.get('also', []))]
            self.assertEqual(len(set(names)), len(names))
            for row in policy['dependencies']:
                self.assertTrue(row['tasks'])
                for task in row['tasks']:
                    self.assertNotIn((row['group'], task), seen)
                    seen[row['group'], task] = row['name']
        with self.assertRaisesRegex(ValueError, 'no task'):
            G.group_tasks('unknown-group')


if __name__ == '__main__':
    unittest.main()
