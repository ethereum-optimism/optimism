"""Change copies of the real CircleCI gate fragments and of RWX's gate policy."""
import importlib.util
import json
from pathlib import Path
import re
import shutil
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('alignment', Path(__file__).resolve().parents[1] / 'circle-alignment.py')
A = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(A)


class AlignmentTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        for name in {A.MANIFEST, *(config for config, _ in A.GATES.values())}:
            (self.root / name).parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(A.ROOT / name, self.root / name)
        handle = patch.object(A, 'ROOT', self.root)
        handle.start()
        self.addCleanup(handle.stop)

    def test_added_removed_renamed_and_nonterminal_prerequisites_are_named(self):
        self.assertEqual(A.problems(), [])
        for gate, (config, workflow) in A.GATES.items():
            job = A.rwx_prerequisites()[gate][0]
            path = self.root / config
            source = path.read_text()
            line = re.search(rf'^ *- {re.escape(job)}: terminal\n', source, re.MULTILINE).group()
            indent = line[:len(line) - len(line.lstrip())]
            header = f"{gate}: `requires:` in workflow {workflow} of {config} differs from RWX's policy:"
            for change, expected in ((line + indent + '- future-job: terminal\n', ['  only in CircleCI: future-job']),
                                     ('', [f'  only in RWX: {job}']),
                                     (line.replace(job, job + '-renamed'), [f'  only in CircleCI: {job}-renamed', f'  only in RWX: {job}']),
                                     (line.replace(': terminal', ''), [f'  not `terminal` in CircleCI: {job} (success)'])):
                path.write_text(source.replace(line, change, 1))
                with self.subTest(gate=gate, change=change):
                    self.assertEqual(A.problems(), [header, *expected])
            path.write_text(source)

    def test_order_is_ignored_and_a_gate_without_a_circleci_workflow_is_named(self):
        path = self.root / A.MANIFEST
        manifest = json.loads(path.read_text())
        for policy in manifest['gates'].values():
            policy['dependencies'].reverse()
        path.write_text(json.dumps(manifest))
        self.assertEqual(A.problems(), [])
        manifest['gates']['future-gate'] = {'dependencies': [{'name': 'future-job'}]}
        path.write_text(json.dumps(manifest))
        self.assertEqual(A.problems(), ['future-gate: RWX has a policy for this gate, but GATES names no CircleCI workflow for it'])

    def test_gate_is_the_one_job_with_the_gate_name_after_a_rename(self):
        gate = 'required-rust-ci'
        config, workflow = A.GATES[gate]

        def prerequisites(jobs):
            (self.root / config).write_text(f'workflows:\n  {workflow}:\n    jobs:\n{jobs}')
            return A.circle_prerequisites(gate)
        other = f'      - {gate}:\n          name: fan-in\n          requires: [{{build: terminal}}]\n'
        renamed = f'      - utils-gate:\n          name: {gate}\n          requires: [lint, {{test: terminal}}]\n'
        self.assertEqual(prerequisites('      - build\n' + other + renamed), [('lint', 'success'), ('test', 'terminal')])
        for message, jobs in (('has 0 jobs with this name', other),
                              ('has 2 jobs with this name', renamed + f'      - {gate}:\n          requires: []\n'),
                              ('always-succeed', f'      - {gate}:\n          always-succeed: true\n')):
            with self.subTest(message=message), self.assertRaisesRegex(ValueError, message):
                prerequisites(jobs)
        self.assertIn(f'{gate}: workflow {workflow} of {config} runs the gate with always-succeed, so it has no prerequisites', A.problems())
        # A gate that requires another job of its own kind waits for that job's prerequisites.
        inner = f'      - {gate}:\n          name: inner\n          requires: [{{build: terminal}}, lint]\n'
        outer = f'      - {gate}:\n          requires: [{{inner: terminal}}, {{test: terminal}}]\n'
        self.assertEqual(prerequisites(inner + outer), [('build', 'terminal'), ('lint', 'success'), ('test', 'terminal')])
        # Without `terminal`, the gate would not run after a failure of the inner gate: name it.
        self.assertEqual(prerequisites(inner + outer.replace('{inner: terminal}', 'inner')), [('inner', 'success'), ('test', 'terminal')])
        with self.assertRaisesRegex(ValueError, 'inner: .* always-succeed'):
            prerequisites(inner.replace('requires:', 'always-succeed: true\n          requires:') + outer)
