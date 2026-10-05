"""Check the chosen DAG and the native package input boundary caught by the probe."""
import json
import re
from pathlib import Path
import subprocess
import shutil
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


def definition(name):
    return json.loads(subprocess.check_output(
        ['yq', '-o=json', '.', str(ROOT / '.rwx' / name)], text=True))


def tasks(name):
    return {task['key']: task for task in definition(name)['tasks']}


class LocalPackageTests(unittest.TestCase):
    def test_each_of_twelve_go_verdicts_waits_for_only_its_own_compilation(self):
        graph = tasks('go-tests.yml')
        for phase in ('compile', 'verdict'):
            self.assertEqual({key for key in graph if key.startswith(phase + '-')},
                             {phase + '-' + str(index) for index in range(12)})
        for index in range(12):
            compile_task = graph['compile-' + str(index)]
            self.assertEqual(compile_task['use'], ['source', 'go-build-tools', 'discovery'])
            verdict = graph['verdict-' + str(index)]
            self.assertEqual([key for key in verdict['use'] if key.startswith('compile-')],
                             ['compile-' + str(index) + '.build'])
            self.assertTrue({'go.build', 'contracts.build', 'kona.build', 'op-reth.build',
                             'prestate.build', 'discovery', 'source'} <= set(verdict['use']))
        leaf = tasks('packages/go-test-verdict.yml')['run']
        self.assertIs(leaf['cache'], False)
        self.assertTrue({'!.ci/go-cache', '!.ci/rust-cache', '!rust/target'} <= set(leaf['filter']))
        self.assertEqual(leaf['outputs']['filesystem']['filter']['workspace'], ['.ci/go-cache/full/run'])

    def test_acceptance_keeps_both_variants_and_excludes_discovery_compiler_state(self):
        graph = tasks('acceptance.yml')
        self.assertEqual({key for key in graph if key.startswith('verdict-')},
                         {'verdict-' + variant + '-' + str(index)
                          for variant in ('opn', 'kona') for index in range(8)})
        for variant, cl in [('opn', 'op-node'), ('kona', 'kona-node')]:
            self.assertEqual(graph['discovery-' + variant]['with']['cl-kind'], cl)
            for index in range(8):
                verdict = graph['verdict-' + variant + '-' + str(index)]
                self.assertEqual([key for key in verdict['use'] if key.startswith('discovery-')],
                                 ['discovery-' + variant + '.build'])
        leaf = tasks('packages/acceptance-verdict.yml')['run']
        self.assertIs(leaf['cache'], False)
        self.assertIn('!.ci/go-cache', leaf['filter'])
        self.assertIn('!.ci/rust-cache', leaf['filter'])
        self.assertIn('!rust/target', leaf['filter'])

    def test_artifact_mounts_and_credentials_are_not_package_value_arguments(self):
        # Native RWX rejected an artifact expression in with even though lint
        # accepted it. Bytes now arrive through filtered package.use inputs.
        for path in sorted((ROOT / '.rwx').glob('*.yml')):
            for task in tasks(path.name).values():
                if task.get('call', '').startswith('${{ run.dir }}/packages/'):
                    values = json.dumps(task.get('with', {}))
                    with self.subTest(path=path.name, task=task['key']):
                        self.assertNotIn('.artifacts.', values)
                        self.assertNotIn('vaults.', values)

    def test_output_filesystem_filters_retain_static_preparation_roots(self):
        # Native output filters silently omitted parameterized paths. Each
        # producer is isolated, so its static family root contains preparation
        # evidence without including verdict results.
        for name, root in [('contracts', '.ci/contract-suites'),
                           ('contract-upgrades', '.ci/contract-upgrades'),
                           ('contract-coverage', 'project/.ci/contract-coverage')]:
            leaf = tasks('packages/' + name + '-compile.yml')['build']
            filters = leaf['outputs']['filesystem']['filter']['workspace']
            self.assertIn(root, filters)
            self.assertNotIn('${{', json.dumps(filters))

    def test_nested_references_are_bound_to_calls_with_the_declared_child(self):
        # The first hosted run caught a reference to a .build child on an
        # unchanged Rust E2E command. Lint did not reject that reference.
        for path in sorted((ROOT / '.rwx').glob('*.yml')):
            graph = tasks(path.name)
            for parent, child in set(re.findall(r'tasks\.([\w-]+)\.tasks\.([\w-]+)', path.read_text())):
                with self.subTest(path=path.name, parent=parent, child=child):
                    self.assertIn('call', graph[parent])
                    called = graph[parent]['call'].removeprefix('${{ run.dir }}/')
                    self.assertIn(child, tasks(called))

    def test_shared_tools_use_narrow_files_without_checkout_history(self):
        for path in sorted((ROOT / '.rwx').glob('*.yml')):
            graph = tasks(path.name)
            if graph.get('tools', {}).get('call') != '${{ run.dir }}/packages/toolchain-common.yml':
                continue
            self.assertEqual(graph['tools']['use'], ['mise', 'bootstrap-inputs'])
            files = graph['bootstrap-inputs']['outputs']['filesystem']['filter']['workspace']
            self.assertIn('mise.toml', files)
            self.assertTrue(all(name == 'mise.toml' or name.startswith(('ops/ci/', '.circleci/scripts/'))
                                for name in files))
            self.assertNotIn('.git', files)
        for name in ('common', 'go', 'foundry', 'rust'):
            self.assertNotIn('vaults.', json.dumps(definition('packages/toolchain-' + name + '.yml')))

    def test_coverage_normalizes_tool_inputs_without_moving_the_project_checkout(self):
        graph = tasks('contract-coverage.yml')
        self.assertEqual(graph['code']['with']['path'], 'project')
        bootstrap = graph['bootstrap-inputs']
        files = bootstrap['outputs']['filesystem']['filter']['workspace']
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name in files:
                destination = root / 'project' / name
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / name, destination)
            subprocess.run(['bash', '-e', '-o', 'pipefail', '-c', bootstrap['run']], cwd=root, check=True)
            for name in files:
                self.assertEqual((root / name).read_bytes(), (ROOT / name).read_bytes())
                self.assertEqual((root / 'project' / name).read_bytes(), (ROOT / name).read_bytes())


if __name__ == '__main__':
    unittest.main()
