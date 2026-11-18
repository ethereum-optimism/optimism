#!/usr/bin/env python3
"""Run the E2E release build script with stub tools and check what it hands to the verdicts."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
BINARIES = ('kona-host', 'kona-node', 'op-reth', 'op-reth-test-engine')

STUB = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys
command, args = Path(sys.argv[0]).name, sys.argv[1:]
with open(os.environ['CALL_LOG'], 'a') as log:
    log.write(json.dumps({'command': command, 'args': args, 'cwd': os.getcwd(),
                          'lto': os.environ.get('CARGO_PROFILE_RELEASE_LTO')}) + '\n')
if command == 'mold':
    os.execvp(args[1], args[1:])
elif command == 'sccache':
    if args[:1] == ['--show-stats']: print('{}')
elif args == ['--version']:
    print('pinned tool')
elif command == 'cargo' and args[:1] == ['metadata']:
    names = os.environ['BUILT_BINARIES'].split()
    print(json.dumps({'packages': [{'targets': [{'name': name, 'kind': ['bin']} for name in names]
                                                + [{'name': 'library', 'kind': ['lib']}]}]}))
elif command == 'cargo' and args[:1] == ['build']:
    release = Path(os.environ['CARGO_TARGET_DIR']) / 'release'
    release.mkdir(parents=True, exist_ok=True)
    for name in os.environ['BUILT_BINARIES'].split():
        (release / name).write_text('compiled ' + name)
    sys.exit(int(os.environ.get('BUILD_EXIT', '0')))
else:
    sys.exit('Unexpected command: ' + command + ' ' + repr(args))
'''


class ReleaseBuildTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve() / 'repo'
        for name in ('rust-e2e-release.sh', 'op-reth-report.py', 'rust-target-cache.py', 'go-artifacts.py'):
            path = self.root / 'ops/ci/runtime' / name; path.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(SCRIPTS / name, path)
        pin = self.root / 'rust/op-reth/crates/chainspec/res/superchain-configs.tar.sha256'
        pin.parent.mkdir(parents=True); pin.write_text('0' * 64 + '  superchain-configs.tar\n')
        (self.root / 'rust/Cargo.lock').write_text('pinned dependencies\n')
        (self.root / 'mise.toml').write_text('# Pinned tools are stubbed.\n')
        (self.root / '.gitignore').write_text('.ci/\nrust/target/\n')
        for args in (['init', '-q'], ['config', 'user.email', 'fixture@example.invalid'], ['config', 'user.name', 'CI Fixture'], ['add', '.'], ['commit', '-qm', 'fixture']):
            subprocess.run(['git', *args], cwd=self.root, check=True)
        tools = self.root.parent / 'bin'; tools.mkdir()
        for name in ('cargo', 'mold', 'rustc', 'sccache'):
            (tools / name).write_text(STUB); (tools / name).chmod(0o755)
        self.log = self.root.parent / 'calls.jsonl'
        sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=self.root, text=True).strip()
        self.env = {**os.environ, 'PATH': str(tools) + os.pathsep + os.environ['PATH'], 'CALL_LOG': str(self.log),
                    'CI_COMMIT_SHA': sha, 'BUILT_BINARIES': ' '.join(BINARIES + ('utility',))}
        self.metadata = self.root / '.ci/go-tests/dependencies/rust-e2e-release/metadata.json'

    def build(self, part='binaries', **env):
        return subprocess.run(['bash', 'ops/ci/runtime/rust-e2e-release.sh', part], cwd=self.root, env={**self.env, **env},
                              text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    def builds(self):
        return [call for call in map(json.loads, self.log.read_text().splitlines())
                if call['command'] == 'cargo' and call['args'][:1] == ['build']]

    WORKSPACE = ['build', '--locked', '--profile', 'release', '--workspace', '--features', 'default', '--lib']

    def test_binaries_half_builds_and_packs_the_four_runtime_binaries(self):
        result = self.build()
        self.assertEqual(result.returncode, 0, result.stdout)
        builds = self.builds()
        # CircleCI's rust-e2e-binaries: every library, and the binaries of its `binary` parameter.
        self.assertEqual([call['args'] for call in builds],
                         [self.WORKSPACE + ['--bin', 'op-reth', '--bin', 'op-reth-test-engine', '--bin', 'kona-node', '--bin', 'kona-host']])
        self.assertEqual((builds[0]['cwd'], builds[0]['lto']), (str(self.root / 'rust'), 'off'))
        metadata = json.loads(self.metadata.read_text())
        self.assertEqual((metadata['kind'], metadata['commit_sha']), ('rust-e2e-release', self.env['CI_COMMIT_SHA']))
        self.assertEqual(set(metadata['files']), {'rust/target/release/' + name for name in BINARIES})

    def test_workspace_half_builds_every_other_binary_and_packs_nothing(self):
        result = self.build('workspace')
        self.assertEqual(result.returncode, 0, result.stdout)
        # CircleCI's rust-workspace-release: every library, and each binary that the other half leaves out.
        self.assertEqual([call['args'] for call in self.builds()], [self.WORKSPACE + ['--bin', 'utility']])
        self.assertFalse(self.metadata.exists())
        self.assertNotEqual(self.build('everything').returncode, 0)

    def test_failed_build_keeps_its_status_and_packs_nothing(self):
        result = self.build(BUILD_EXIT='23')
        self.assertEqual(result.returncode, 23, result.stdout)
        self.assertFalse(self.metadata.exists())

    def test_missing_runtime_binary_fails(self):
        for missing in BINARIES:
            shutil.rmtree(self.root / 'rust/target', ignore_errors=True)
            result = self.build(BUILT_BINARIES=' '.join(name for name in BINARIES if name != missing))
            with self.subTest(missing=missing):
                self.assertNotEqual(result.returncode, 0, result.stdout)
                self.assertFalse(self.metadata.exists())


if __name__ == '__main__': unittest.main()
