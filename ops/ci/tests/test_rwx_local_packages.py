"""Check the chosen DAG and the native package input boundary caught by the probe."""
import json
import os
import importlib.util
import re
from pathlib import Path
import subprocess
import shutil
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]


def definition(name):
    return json.loads(subprocess.check_output(
        ['yq', '-o=json', '.', str(ROOT / '.rwx' / name)], text=True))


def tasks(name):
    return {task['key']: task for task in definition(name)['tasks']}


class LocalPackageTests(unittest.TestCase):
    def test_sp1_filtered_toolchain_imports_and_binds_every_helper(self):
        selected = tasks('sp1-guest.yml')['toolchain']['filter']
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in selected:
                target = root / name
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(ROOT / name, target)
            # Load only the bytes that native RWX transfers, without installing
            # tools or invoking the workload. A missing import fails here.
            spec = importlib.util.spec_from_file_location('sp1_filtered', root / 'ops/ci/runtime/sp1-guest.py')
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            self.assertEqual(set(selected), set(module.TOOL_INPUTS))

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
            self.assertTrue(all(name == 'mise.toml' or name.startswith('ops/ci/runtime/')
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

    def test_filtered_common_go_and_foundry_setup_run_from_component_directories(self):
        # Execute the declared scripts against command stubs. No system package
        # writes or tool downloads occur; missing helper bytes still fail.
        for name,mode in [('common',''),('go','go'),('foundry','compiler')]:
            with self.subTest(package=name), tempfile.TemporaryDirectory() as tmp:
                root=Path(tmp)/'scratch/project';root.mkdir(parents=True)
                task=tasks('packages/toolchain-'+name+'.yml')['prepare']
                selected=task.get('filter',{}).get('workspace',tasks('packages/toolchain-common.yml')['prepare']['filter']['workspace'])
                for filename in selected:
                    target=root/filename;target.parent.mkdir(parents=True,exist_ok=True)
                    shutil.copy2(ROOT/filename,target)
                binaries=root/'bin';binaries.mkdir()
                log=root/'commands.log'
                for binary in ('mise','sudo','svm'):
                    path=binaries/binary
                    path.write_text('#!/bin/bash\nset -eu\nprintf "%s:%s:%s\\n" "$(basename "$0")" "$PWD" "$*" >> "$BOOTSTRAP_LOG"\nif [[ "$(basename "$0")" == mise && "${1:-}" == bin-paths ]]; then printf "%s\\n" "$(dirname "$0")"; fi\n')
                    path.chmod(0o755)
                component=root/'component';component.mkdir()
                env=dict(os.environ,PATH=str(binaries)+os.pathsep+os.environ['PATH'],RWX_ENV=str(root/'exports'),BOOTSTRAP_LOG=str(log),GO_TOOL_MODE=mode,FOUNDRY_TOOL_MODE=mode)
                result=subprocess.run(['bash','-e','-o','pipefail','-c',task['run']],cwd=component,env=env,capture_output=True,text=True)
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                self.assertTrue((root/'exports/PATH').is_file())
                self.assertTrue(all(str(root.resolve()) in line for line in log.read_text().splitlines()))
                self.assertFalse((root/'ops/ci/migration').exists())
