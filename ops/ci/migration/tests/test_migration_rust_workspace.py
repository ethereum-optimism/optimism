"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_rust_workspace as T
REPORT = T.REPORT
SCRIPTS = T.SCRIPTS
json = T.json
os = T.os
patch = T.patch
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
time = T.time


class ConfigurationTests(unittest.TestCase):

    @unittest.skipUnless(shutil.which('yq'), 'requires the pinned yq tool')
    def test_circle_adapters_preserve_generic_fallbacks(self):
        definition = Path(__file__).resolve().parents[4] / '.circleci/continue/rust-ci.yml'
        config = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(definition)], text=True))
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / 'rust').mkdir()
            (root / 'other').mkdir()
            (root / 'ops/ci/runtime').mkdir(parents=True)
            helper = root / 'ops/ci/runtime/rust-workspace.sh'
            helper.write_text('#!/usr/bin/env bash\nprintf "shared:%s\\n" "$1"\n')
            binaries = root / 'bin'
            binaries.mkdir()
            for name in ('cargo', 'rustup', 'typos', 'zepter'):
                path = binaries / name
                path.write_text(f'#!/usr/bin/env bash\nprintf "{name}:%s\\n" "$*"\n')
                path.chmod(493)
            env = os.environ | {'PATH': str(binaries) + os.pathsep + os.environ['PATH']}

            def run(job, params):
                step = next((s['run'] for s in config['jobs'][job]['steps'] if isinstance(s, dict) and 'run' in s))
                command = step['command']
                for key, value in params.items():
                    command = command.replace(f'<<parameters.{key}>>', value)
                result = subprocess.run(['bash', '-e', '-c', command], cwd=root / params['directory'], env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                return result.stdout
            for job, params, expected in (('rust-ci-typos', {'directory': 'rust'}, 'shared:typos'), ('rust-ci-typos', {'directory': 'other'}, 'typos:'), ('rust-ci-zepter', {'directory': 'rust', 'command': 'zepter run check'}, 'shared:zepter'), ('rust-ci-zepter', {'directory': 'rust', 'command': 'zepter run custom'}, 'zepter:run custom')):
                with self.subTest(job=job, params=params):
                    self.assertEqual(run(job, params).strip(), expected)
            invocations = config['workflows']['rust-ci']['jobs']
            for mode, name in (('wasm-unknown', 'rust-wasm-unknown'), ('wasm-wasi', 'rust-wasm-wasi')):
                params = next((i['rust-ci-cargo-hack-build'] for i in invocations if isinstance(i, dict) and i.get('rust-ci-cargo-hack-build', {}).get('name') == name))
                params = {k: params[k] for k in ('directory', 'target', 'flags')}
                self.assertEqual(run('rust-ci-cargo-hack-build', params).strip(), 'shared:' + mode)
                params['flags'] = '--workspace'
                self.assertEqual(run('rust-ci-cargo-hack-build', params).splitlines(), ['rustup:target add ' + params['target'], 'cargo:hack build --target ' + params['target'] + ' --workspace'])
