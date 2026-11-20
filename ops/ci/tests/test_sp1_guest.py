from contextlib import redirect_stderr
import importlib.util
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('sp1_guest', Path(__file__).resolve().parents[1] / 'runtime' / 'sp1-guest.py')
G = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(G)


class GuestTests(unittest.TestCase):
    def test_occupied_canonical_workspace_fails_without_reading_or_changing_its_files(self):
        native=G.helper('sp1-guest-native-build')
        with tempfile.TemporaryDirectory() as temp:
            base=Path(temp);root=base/'source';root.mkdir()
            occupied=base/'occupied';occupied.mkdir();sentinel=occupied/'original.txt';sentinel.write_bytes(b'original occupied data\n')
            with patch.object(native,'ROOT',root),patch.object(native,'WORKSPACE',occupied),patch.object(native,'CARGO',base/'cargo'):
                self.assertEqual(native.execute(),1)
            self.assertEqual(sentinel.read_bytes(),b'original occupied data\n')
            self.assertFalse((root/'.ci/sp1-guest/dependency/files').exists())

    def test_uncommitted_change_fails_before_the_clone_that_would_not_contain_it(self):
        native=G.helper('sp1-guest-native-build')
        with tempfile.TemporaryDirectory() as temp:
            # Resolved, so that the absent canonical paths pass the first precondition.
            base=Path(temp).resolve();root=base/'source';root.mkdir();tracked=root/'tracked.txt';tracked.write_text('committed\n')
            for argv in (['git','init','-q'],['git','add','.'],
                         ['git','-c','user.name=CI fixture','-c','user.email=ci-fixture@example.invalid','commit','-qm','fixture']):
                subprocess.run(argv,cwd=root,check=True,stdout=subprocess.DEVNULL)
            tracked.write_text('uncommitted\n')
            prepare=AssertionError('The build must not prepare the clone')
            with patch.object(native,'ROOT',root),patch.object(native,'WORKSPACE',base/'workspace'),patch.object(native,'CARGO',base/'cargo'),patch.object(native.subprocess,'run',side_effect=prepare),redirect_stderr(io.StringIO()) as stderr:
                self.assertEqual(native.execute(),1)
            self.assertIn('uncommitted',stderr.getvalue())
            self.assertFalse((base/'workspace').exists())

    def test_guest_feature_markers_and_source_suffix_bind_actual_elf_bytes(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp); sha = 'a'*40; programs = list(G.PROGRAMS)
            def write(features, source, extra=b''):
                (path/'vkeys.toml').write_text('git_sha="'+source+'"\nsp1_toolchain_tag="v6.0.0"\n'+
                    ''.join(n+'="0x'+'b'*64+'"\n' for n in programs))
                for n in programs:
                    data = ('KONA_SP1_BUILD{git_sha='+source+'}').encode() + extra
                    if features and n=='super-range': data += G.TEST_CONFIG_MARKER
                    (path/(n+'-elf')).write_bytes(data)
            write(True, sha+'-test')
            G.check_elfs(path, sha, 'test-config-fallback')
            write(False, sha+'-test')
            with self.assertRaisesRegex(ValueError, 'test-config'): G.check_elfs(path, sha, 'test-config-fallback')
            write(True, sha)
            with self.assertRaisesRegex(ValueError, 'source'): G.check_elfs(path, sha, 'test-config-fallback')
            with self.assertRaisesRegex(ValueError, 'test-config'): G.check_elfs(path, sha, '')
            write(False, sha)
            G.check_elfs(path, sha, '')
            # A second, different marker fails: CircleCI compares every distinct marker.
            write(False, sha, extra=b'KONA_SP1_BUILD{git_sha=unknown}')
            with self.assertRaisesRegex(ValueError, 'source'): G.check_elfs(path, sha, '')
            # CircleCI greps the manifest for the names its consumers read.
            (path/'log-commitment-elf').unlink(); programs.remove('log-commitment')
            write(False, sha)
            with self.assertRaisesRegex(ValueError, 'consumers'): G.check_elfs(path, sha, '')

    def test_checks_receive_the_built_elfs_and_run_the_circleci_steps_in_order(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp); files = base/'artifact/files'; files.mkdir(parents=True); recipes = []
            for name in ('vkeys.toml', 'super-range-elf'): (files/name).write_text(name)
            def run(*argv, cwd=G.RUST):
                recipes.append((argv, cwd, os.environ['RUSTFLAGS'], sorted(p.name for p in (base/'sp1/elf').iterdir())))
            with patch.object(G,'SP1',base/'sp1'), patch.object(G,'cache_start') as start, patch.object(G,'cache_finish',side_effect=lambda status: status), patch.object(G,'run',side_effect=run), patch.dict(os.environ):
                (base/'sp1').mkdir()
                self.assertEqual(G.checks(base/'artifact', 'test-config-fallback'), 0)
            start.assert_called_once_with('checks', 'test-config-fallback')
            self.assertEqual(recipes, [(('just', recipe), G.RUST, '-Dwarnings', ['super-range-elf', 'vkeys.toml'])
                                       for recipe in ('test-sp1-guest', 'lint-sp1-guest', 'check-vkey-crates')])

    def test_failing_recipe_keeps_its_exit_status_and_the_cache_still_finishes(self):
        for returncode, expected in ((101, 101), (-15, 143)):
            def steps(): raise subprocess.CalledProcessError(returncode, ['just', 'test-sp1-guest'])
            with self.subTest(returncode=returncode), patch.object(G,'cache_start'), patch.object(G,'cache_finish',side_effect=lambda status: status) as finish:
                self.assertEqual(G.cached('checks', '', steps), expected)
                finish.assert_called_once_with(expected)

    def test_pin_checks_reject_manifest_and_lock_drift_before_installation(self):
        tools = [shutil.which(name) for name in ('yq', 'jq', 'just')]
        if not all(tools): self.skipTest('The pin checks need yq, jq and just')
        script = 'ops/ci/runtime/sp1-guest-toolchain.sh'
        # Only the files that the toolchain task transfers: its filter must suffice for the pin checks.
        inputs = json.loads(subprocess.check_output(
            ['yq', '-o=json', '.tasks[] | select(.key == "toolchain") | .filter', str(G.ROOT/'.rwx/sp1-guest.yml')], text=True))
        self.assertIn(script, inputs)
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)/'source'; home = Path(temp)/'home'; home.mkdir()
            for name in inputs:
                target = root/name; target.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(G.ROOT/name, target)
            # Without cargo and rustup, a run that passes the pin checks cannot install or relink a toolchain.
            path = os.pathsep.join([*dict.fromkeys(str(Path(tool).parent) for tool in tools), '/usr/bin', '/bin'])
            if shutil.which('cargo', path=path) or shutil.which('rustup', path=path):
                self.skipTest('yq, jq or just share a directory with cargo or rustup')
            def run():
                return subprocess.run(['bash', script], cwd=root, env={'PATH': path, 'HOME': str(home)}, capture_output=True, text=True)
            self.assertNotIn('SP1 version drift', run().stderr)
            manifest = root/'rust/Cargo.toml'; original = manifest.read_text()
            changed, count = re.subn('(sp1-sdk\\s*=\\s*\\{[^}\\n]*version\\s*=\\s*")[^"]+', '\\g<1>0.0.0', original)
            self.assertEqual(count, 1)
            manifest.write_text(changed)
            result = run()
            self.assertEqual(result.returncode, 1)
            self.assertIn('ERROR: SP1 version drift', result.stderr)
            self.assertIn('rust/Cargo.toml sp1-sdk: 0.0.0', result.stderr)
            manifest.write_text(original)
            (root/'rust/kona/sp1/programs/Cargo.lock').write_text('invalid original TOML = [\n')
            result = run()
            self.assertEqual(result.returncode, 1)
            self.assertIn('ERROR: SP1 version drift', result.stderr)
            self.assertIn('resolved sp1-lib: <lockfile parse failed>', result.stderr)


if __name__ == '__main__': unittest.main()
