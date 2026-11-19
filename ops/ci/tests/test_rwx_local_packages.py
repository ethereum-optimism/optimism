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
        ['yq', '-o=json', 'explode(.)', str(ROOT / '.rwx' / name)], text=True))


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
            # The seed uses the tools, so RWX applies it above them. `discovery` uses `source`,
            # so RWX applies it above `source`.
            self.assertEqual(compile_task['use'], ['go-build-tools', 'compiler-seed-' + str(index), 'source', 'discovery'])
            verdict = graph['verdict-' + str(index)]
            self.assertEqual([key for key in verdict['use'] if key.startswith('compile-')],
                             ['compile-' + str(index) + '.build'])
            self.assertTrue({'go.build', 'contracts.build', 'kona.build', 'op-reth.build',
                             'prestate.build', 'discovery', 'source'} <= set(verdict['use']))
        leaf = tasks('packages/go-test-verdict.yml')['run']
        self.assertIs(leaf['cache'], False)
        self.assertTrue({'!.ci/go-cache/full/build', '!.ci/go-cache/full/dependencies', '!.ci/rust-cache', '!rust/target'} <= set(leaf['filter']))
        self.assertEqual(leaf['outputs']['filesystem']['filter']['workspace'], ['.ci/go-cache/full/run'])

    def test_every_workload_builds_a_dependency_at_the_planned_commit_with_the_same_tools(self):
        # A dependency build has one cache key for all workloads and for later commits only if
        # each caller passes the planned commit and the same tool tasks, and nothing else.
        tools = {'go': ['go'], 'contracts': ['go', 'anvil'], 'kona': ['release'], 'op-reth': ['release'],
                 'sp1-executor': ['release'], 'prestate': ['go'], 'modules': ['go']}
        calls = 0
        for path in sorted((ROOT / '.rwx').glob('*.yml')):
            graph = tasks(path.name)
            for key, task in graph.items():
                match = re.fullmatch(r'\$\{\{ run\.dir \}\}/packages/go-(' + '|'.join(tools) + r')\.yml', task.get('call', ''))
                if not match: continue
                kind, calls = match.group(1), calls + 1
                planned = {'modules': 'go-modules'}.get(kind, kind)
                self.assertEqual(task['with']['commit-sha'], '${{ tasks.dependency-plan.values.' + planned + ' }}', (path.name, key))
                used = task['use'] if isinstance(task['use'], list) else [task['use']]
                self.assertEqual([graph[tool]['with']['mode'] for tool in used], tools[kind], (path.name, key))
        self.assertEqual(calls, 24)
        for kind in tools:
            build = tasks('packages/go-' + kind + '.yml')['build']
            self.assert_starts_without_a_checkout(build, build['outputs']['filesystem']['filter']['workspace'])
            self.assertEqual((build['use'], build['env']['CI_COMMIT_SHA']), ('package.use', '${{ params.commit-sha }}'))
        # The Go compiler seeds follow the same contract.
        graph = tasks('go-tests.yml')
        prepared = graph['compiler-preparation']['outputs']['filesystem']['filter']['workspace']
        self.assert_starts_without_a_checkout(graph['compiler-preparation'], ['.ci/go-tests/dependencies/contracts'])
        self.assertEqual(graph['discovery']['outputs']['filesystem']['filter']['workspace'], prepared)
        for index in range(12):
            seed = graph['compiler-seed-' + str(index)]
            self.assertEqual(seed['use'], 'compiler-preparation')
            self.assert_starts_without_a_checkout(seed, prepared)
            self.assertEqual(seed['outputs']['filesystem']['filter']['workspace'], ['.ci/go-cache/full/build'])

    def test_contract_compiles_name_the_files_that_they_read(self):
        # A compile with `.git` among its inputs has a new cache key for each commit. RWX keys a
        # task on file contents only for an include list, so each compile names what it reads.
        shared = tasks('packages/contracts-compile.yml')['build']
        fork = tasks('contract-l2-fork.yml')['compile']
        scripts = lambda task: {name for name in task['filter'] if name.startswith('ops/ci/runtime/')}
        self.assertEqual(scripts(shared), {'ops/ci/runtime/ci-report.py', 'ops/ci/runtime/contract-upgrades.py'})
        self.assertEqual(scripts(fork), scripts(shared) | {'ops/ci/runtime/contract-l2-fork.py', 'ops/ci/runtime/rpc-pacer.py'})
        self.assertEqual(set(shared['env']), {'CI_BRANCH'})
        # The FFI helper is Go. Follow its imports: each package must be under a listed tree.
        prefix, seen, pending = 'github.com/ethereum-optimism/optimism/', set(), ['packages/contracts-bedrock/scripts/go-ffi']
        while pending:
            directory = pending.pop()
            if directory in seen: continue
            seen.add(directory)
            for source in (ROOT / directory).glob('*.go'):
                if source.name.endswith('_test.go'): continue
                for found in re.findall('"' + re.escape(prefix) + '([^"]+)"', source.read_text()):
                    if (ROOT / found).is_dir(): pending.append(found)
        self.assertGreater(len(seen), 10)
        for task in (shared, fork):
            self.assertFalse([name for name in task['filter'] if name.startswith('!')])
            self.assertTrue({'mise.toml', 'go.mod', 'go.sum', 'packages/contracts-bedrock', '.ci/go-cache/pr-checks/modules'} <= set(task['filter']))
            for directory in seen:
                top = directory.split('/')[0]
                self.assertTrue(top in task['filter'] or (top.startswith('op-') and 'op-*' in task['filter'])
                                or directory.startswith('packages/contracts-bedrock'), directory)
        verdicts = 0
        for path in ('contracts.yml', 'contract-upgrades.yml'):
            graph = tasks(path)
            self.assertEqual(graph['compile']['call'], '${{ run.dir }}/packages/contracts-compile.yml')
            for key, task in graph.items():
                if key.startswith('verdict-') and '-modified-' not in key:
                    verdicts += 1
                    self.assertEqual(task['use'][-1], 'compile', (path, key))
        self.assertEqual(verdicts, 13)

    def assert_starts_without_a_checkout(self, task, inputs):
        """The task sees mise.toml and `inputs` only, checks out CI_COMMIT_SHA itself and filters its output.

        A task passes on every layer that it uses, and the tool tasks carry the checkout of the
        run. Without the input filter, that checkout is in the task's cache key and under its
        own checkout. Without the output filter, the task's checkout reaches each consumer.
        """
        flat = [name for entry in task['filter'] for name in (entry if isinstance(entry, list) else [entry])]
        self.assertEqual(flat, ['mise.toml', *inputs])
        self.assertIn('git fetch -q origin "$CI_COMMIT_SHA"\ngit checkout -qf FETCH_HEAD\n', task['run'])
        self.assertEqual(task['outputs']['filesystem']['filter']['system'], [])

    def test_acceptance_keeps_both_variants_and_excludes_compiler_state(self):
        graph = tasks('acceptance.yml')
        # Like Circle's memory-all jobs: 4 nodes each, on 32 GiB (2xlarge.gen2).
        self.assertEqual({key for key in graph if key.startswith('verdict-')},
                         {'verdict-' + variant + '-' + str(index)
                          for variant in ('opn', 'kona') for index in range(4)})
        for variant, cl in [('opn', 'op-node'), ('kona', 'kona-node')]:
            for index in range(4):
                verdict = graph['verdict-' + variant + '-' + str(index)]
                # The circleci shim gives each shard index its own share of the tests.
                self.assertEqual((verdict['with']['cl-kind'], verdict['with']['shard-index']), (cl, str(index)))
        leaf = tasks('packages/acceptance-verdict.yml')['run']
        self.assertIs(leaf['cache'], False)
        self.assertEqual(leaf['runner']['memory'], '32gb')
        # Compiler state stays out; the producer's module cache stays in, as in Circle.
        self.assertTrue({'!.ci/go-cache/full/build', '!.ci/go-cache/full/dependencies',
                         '!.ci/go-cache/acceptance/run'} <= set(leaf['filter']))
        self.assertFalse([rule for rule in leaf['filter']
                          if '.ci/go-cache/full/modules'.startswith(rule.lstrip('!').rstrip('/'))])
        self.assertIn('!.ci/rust-cache', leaf['filter'])
        self.assertIn('!rust/target', leaf['filter'])
        # The task does not reset its checkout, so each excluded path must hold no tracked file.
        # A hidden tracked file is a modified build input to go-artifacts.py.
        hidden = subprocess.check_output(['git', 'ls-files', '--', *(rule.lstrip('!') for rule in leaf['filter'])], cwd=ROOT, text=True)
        self.assertEqual(hidden, '')

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

    def test_output_filesystem_filters_and_test_results_have_no_parameterized_paths(self):
        # RWX does not expand an expression in these paths, and gives no warning. A filter then
        # omits the path, and a task reports no tests. RWX does expand an artifact path.
        for name in ('contracts-compile', 'contracts-compile-modified'):
            leaf = tasks('packages/' + name + '.yml')['build']
            filters = leaf['outputs']['filesystem']['filter']['workspace']
            self.assertNotIn('${{', json.dumps(filters))
        declared = 0
        for path in sorted((ROOT / '.rwx').rglob('*.yml')):
            for task in tasks(path.relative_to(ROOT / '.rwx').as_posix()).values():
                results = task.get('outputs', {}).get('test-results', [])
                declared += len(results)
                with self.subTest(path=path.name, task=task['key']):
                    self.assertNotIn('${{', json.dumps(results))
        self.assertGreater(declared, 20)

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
            # A tool task without this filter has the checkout in its cache key. It then runs
            # for each commit, and gives every task that uses it a new cache key.
            files = tasks('packages/toolchain-' + name + '.yml')['prepare']['filter']['workspace']
            self.assertIn('mise.toml', files)
            self.assertTrue(all(name == 'mise.toml' or name.startswith('ops/ci/runtime/') for name in files))
        # The same holds for a tool task that a workload defines itself.
        inline = {'acceptance.yml': ['acceptance-tools'], 'rust-e2e.yml': ['runtime-tools'],
                  'rust.yml': ['wasm-tools', 'lint-tools', 'rust-fmt-tools'],
                  'pr-checks.yml': ['main-tools', 'static-tools', 'go-lint-tools']}
        for path, keys in inline.items():
            graph = tasks(path)
            for key in keys:
                files = graph[key]['filter']
                self.assertIn('mise.toml', files, (path, key))
                self.assertTrue(all(name == 'mise.toml' or name.startswith('ops/ci/runtime/') for name in files), (path, key))


    def test_tool_preparation_retries_downloads_and_exports_the_policy(self):
        # Tool downloads time out intermittently. Each install sees mise's retry
        # policy, and the exported environment carries it to later installs.
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            binaries = root / 'bin'; binaries.mkdir()
            log = root / 'mise.log'
            stub = binaries / 'mise'
            stub.write_text('#!/bin/bash\nprintf "%s retries=%s timeout=%s\\n" "$*" "${MISE_HTTP_RETRIES:-}" "${MISE_HTTP_TIMEOUT:-}" >> "$STUB_LOG"\n')
            stub.chmod(0o755)
            exports = root / 'exports'
            env = dict(os.environ, PATH=str(binaries) + os.pathsep + os.environ['PATH'], RWX_ENV=str(exports), STUB_LOG=str(log))
            env.pop('MISE_HTTP_RETRIES', None); env.pop('MISE_HTTP_TIMEOUT', None)
            subprocess.run(['bash', str(ROOT / 'ops/ci/runtime/rwx-prepare.sh'), 'go'], env=env, check=True)
            self.assertIn('install go gotestsum retries=5 timeout=90s', log.read_text().splitlines())
            self.assertEqual((exports / 'MISE_HTTP_RETRIES').read_text(), '5\n')
            self.assertEqual((exports / 'MISE_HTTP_TIMEOUT').read_text(), '90s\n')


if __name__ == '__main__':
    unittest.main()
