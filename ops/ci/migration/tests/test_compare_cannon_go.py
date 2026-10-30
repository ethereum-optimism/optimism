"""Reject incomplete, stale, corrupt or deceptively green Cannon comparisons."""
import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('compare_cannon_go', Path(__file__).resolve().parents[2] / 'migration' / 'compare-cannon-go.py')
P = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(P)
G = P.G


class CompareTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(); self.addCleanup(temp.cleanup); self.root = Path(temp.name)
        self.sha = 'a' * 40; self.package = G.PREFIX + '/fixture'
        for provider, name in [('circleci', 'circle'), ('rwx', 'rwx')]: self.make(self.root / name, provider)

    def write(self, path, value): G.S.write(path, value)

    def seal(self, d):
        self.write(d / 'final.json', {'exit_code': 0, 'report_errors': [], 'original_sha256':
            {str(p.relative_to(d)): G.S.digest(p) for p in d.rglob('*') if p.is_file() and p.name != 'final.json'}})

    def make(self, d, provider):
        d.mkdir(); root = Path('/original/' + provider); effective = G.settings(provider, True, True, 16)
        self.write(d / 'settings.json', {'suite': 'cannon-go', 'source_sha': self.sha, 'provider': provider,
            'branch': 'codex/rwx-ci-pilot', 'workspace_root': str(root), 'settings': effective,
            'input_sha256': {'mise.toml': 'b' * 64}, 'environment': {'CI': 'true', 'SKIP_SLOW_TESTS': 'true', 'GOMAXPROCS': None},
            'go_environment': {'GOFLAGS': ''}, 'tool_versions': {'go': 'go fixture', 'forge': 'forge fixture'},
            'rwx_run_id': 'actual-fixture', 'rwx_task_attempt': '1'})
        raw = {'ImportPath': self.package, 'Dir': str(root / 'cannon/fixture')}
        (d / 'packages.json').write_text(json.dumps(raw))
        self.write(d / 'cpus.json', 16)
        rows = G.packages(json.dumps(raw), root, d / 'source')
        listing = [{'Package': self.package, 'Action': 'start'},
            {'Package': self.package, 'Action': 'output', 'Output': 'TestOne\n'},
            {'Package': self.package, 'Action': 'pass', 'Elapsed': 0}]
        original = [{'Package': self.package, 'Action': 'start'},
            {'Package': self.package, 'Test': 'TestOne', 'Action': 'run'},
            {'Package': self.package, 'Test': 'TestOne', 'Action': 'output', 'Output': 'original case output\n'},
            {'Package': self.package, 'Test': 'TestOne', 'Action': 'pass', 'Elapsed': .01},
            {'Package': self.package, 'Action': 'pass', 'Elapsed': .01}]
        for name, events in [('list', listing), ('original', original)]:
            (d / (name + '.json')).write_text(''.join(json.dumps(e) + '\n' for e in events))
        per_test = d / 'per-test' / self.package.replace('/', '.') / 'TestOne.log'
        per_test.parent.mkdir(parents=True); per_test.write_text('original case output\n')
        self.write(d / 'selection.json', {'packages': rows, 'initial_tests': {self.package: ['TestOne']},
                                         'authority': 'go list ./... and go test -list ' + G.LIST_PATTERN})
        suite = ET.Element('testsuite', name=self.package)
        ET.SubElement(suite, 'testcase', classname=self.package, name='TestOne', time='.01')
        ET.ElementTree(suite).write(d / 'junit.xml')
        report = root / '.ci/cannon-go/run'
        args = {'cpus': ['nproc'], 'lint': ['just', 'lint'], 'packages': ['go', 'list', '-e', '-json', './...'],
            'list': ['go', 'test', '-json', *G.go_flags(effective, list_tests=True)],
            'tests': [str(root / 'ops/scripts/gotestsum-split.sh'), '--format=testname',
                      '--junitfile=' + str(report / 'junit.xml'), '--jsonfile=' + str(report / 'original.json'), '--', *G.go_flags(effective)]}
        for name, argv in args.items():
            (d / (name + '.log')).write_text('')
            self.write(d / (name + '.stage.json'), {'argv': argv, 'cwd': str(root / 'cannon'), 'exit_code': 0,
                                                   'stdin': 'devnull', 'log_sha256': G.S.digest(d / (name + '.log'))})
        self.write(d / 'inputs-after.json', {'mise.toml': 'b' * 64})
        self.account(d)
        if provider == 'rwx':
            (d / 'dependencies').mkdir()
            for kind in ('go-modules', 'contracts'):
                self.write(d / 'dependencies' / (kind + '.json'), {'kind': kind, 'commit_sha': self.sha,
                    'settings': G.ARTIFACTS.SETTINGS[kind], 'mise_sha256': 'b' * 64, 'files': {'original': 'c' * 64},
                    'tool_versions': {'go': 'go fixture', 'forge': 'forge fixture'}})
        self.seal(d)

    def account(self, d):
        selected = G.read(d / 'selection.json')
        self.write(d / 'coverage.json', G.observed(d, selected['packages'], selected['initial_tests']))
        G.PROJECT.project(d / 'original.json', d / 'native.json')

    def compare(self): return P.compare(self.root, self.sha)

    def test_complete_original_report_matches(self):
        report = self.compare(); self.assertTrue(report['verified_parity']); self.assertEqual(report['case_identities'], 1)
        self.assertEqual(report['outcomes'], {'pass': 1})

    def test_missing_corrupt_or_unsealed_original_cannot_compare(self):
        for name in ('original.json', 'packages.json', 'tests.stage.json', 'junit.xml'):
            d = self.root / 'rwx'; path = d / name; original = path.read_bytes()
            path.unlink()
            with self.subTest(name=name), self.assertRaises((ValueError, OSError)): self.compare()
            path.write_bytes(original + b'corrupt')
            with self.assertRaises((ValueError, OSError)): self.compare()
            path.write_bytes(original)
        (self.root / 'rwx/unsealed').write_text('extra')
        with self.assertRaisesRegex(ValueError, 'unsealed'): self.compare()

    def test_stale_source_settings_or_dependency_cannot_compare(self):
        d = self.root / 'rwx'; original = G.read(d / 'settings.json')
        for change in ({'source_sha': 'd' * 40}, {'settings': original['settings'] | {'count': None}},
                       {'settings': original['settings'] | {'tags': ['ci']}}, {'rwx_task_attempt': '2'},
                       {'go_environment': {'GOFLAGS': '-run=TestOne'}}):
            self.write(d / 'settings.json', original | change); self.seal(d)
            with self.subTest(change=change), self.assertRaises(ValueError): self.compare()
        self.write(d / 'settings.json', original)
        path = d / 'dependencies/contracts.json'; value = G.read(path)
        self.write(path, value | {'commit_sha': 'f' * 40}); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'dependency'): self.compare()

    def test_changed_selection_and_duplicate_initial_execution_cannot_compare(self):
        d = self.root / 'rwx'; original = G.read(d / 'selection.json')
        for names in ([], ['TestOne', 'TestOne'], ['TestMissing']):
            value = copy.deepcopy(original); value['initial_tests'][self.package] = names
            self.write(d / 'selection.json', value); self.seal(d)
            with self.subTest(names=names), self.assertRaises(ValueError): self.compare()
        self.write(d / 'selection.json', original)
        path = d / 'original.json'; events = path.read_text(); path.write_text(events + events); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'retry'): self.compare()

    def test_foreign_package_and_failed_testmain_cannot_hide_behind_passing_cases(self):
        d = self.root / 'rwx'; path = d / 'original.json'; original = path.read_text()
        for events in ([{'Package': 'outside/suite', 'Action': 'start'}, {'Package': 'outside/suite', 'Action': 'pass', 'Elapsed': 0}],
                       [{'Package': self.package, 'Action': 'fail', 'Elapsed': 0}]):
            path.write_text(original + ''.join(json.dumps(e) + '\n' for e in events)); self.seal(d)
            with self.subTest(events=events), self.assertRaises(ValueError): self.compare()

    def test_fabricated_native_report_or_altered_command_cannot_compare(self):
        d = self.root / 'rwx'; path = d / 'native.json'; original = path.read_bytes()
        path.write_text(''); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'projection'): self.compare()
        path.write_bytes(original); path = d / 'tests.stage.json'; value = G.read(path)
        value['argv'].insert(-1, '-run=TestOne'); self.write(path, value); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'stage'): self.compare()

    def test_circle_manifest_can_restore_only_original_sealed_empty_bytes(self):
        d = self.root / 'circle'; (d / 'packages.log').unlink()
        result = self.compare()
        self.assertEqual(result['circle_manifest_declared_empty'], [{'path': 'packages.log', 'sha256': P.EMPTY}])

    def test_corrupt_per_test_output_cannot_compare_even_with_a_new_seal(self):
        d = self.root / 'rwx'
        next((d / 'per-test').rglob('*.log')).write_text('fabricated case output\n'); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'per-test'): self.compare()

    def test_different_original_skip_reasons_are_not_normalized_away(self):
        for name, reason in [('circle', 'original reason'), ('rwx', 'different reason')]:
            d = self.root / name; events = [json.loads(line) for line in (d / 'original.json').read_text().splitlines()]
            for event in events:
                if event.get('Test') and event['Action'] == 'pass': event['Action'] = 'skip'
                if event.get('Test') and event['Action'] == 'output': event['Output'] = '    fixture_test.go:1: ' + reason + '\n'
            (d / 'original.json').write_text(''.join(json.dumps(e) + '\n' for e in events))
            next((d / 'per-test').rglob('*.log')).write_text('    fixture_test.go:1: ' + reason + '\n')
            tree = ET.parse(d / 'junit.xml'); ET.SubElement(next(tree.iter('testcase')), 'skipped', message=reason)
            tree.write(d / 'junit.xml'); self.account(d); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'skips'): self.compare()

    def test_actual_nproc_difference_is_retained_without_changing_selection(self):
        d = self.root / 'rwx'; path = d / 'settings.json'; value = G.read(path)
        value['settings']['parallel'] = 8; self.write(path, value); self.write(d / 'cpus.json', 8)
        for name in ('list', 'tests'):
            path = d / (name + '.stage.json'); value = G.read(path)
            value['argv'] = [arg.replace('-parallel=16', '-parallel=8') for arg in value['argv']]
            self.write(path, value)
        self.seal(d); result = self.compare()
        self.assertEqual(result['settings_differences'][0]['circle'], 16)
        self.assertEqual(result['settings_differences'][0]['rwx'], 8)
        self.write(d / 'cpus.json', 32); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'nproc'): self.compare()


if __name__ == '__main__': unittest.main()
