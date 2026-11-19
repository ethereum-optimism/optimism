"""Check when a dependency that an ancestor commit built may be reused."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('go_dependency_source', Path(__file__).resolve().parents[1] / 'runtime' / 'go-dependency-source.py')
SOURCE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SOURCE)


class SourceIdentityTests(unittest.TestCase):
    def test_go_only_edit_reuses_rust_helper_but_source_and_build_inputs_invalidate(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(['git', *args], cwd=root, text=True).strip()
            git('init', '-q')
            def commit(path, text):
                p=root/path;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text)
                git('add', '.');git('-c','user.name=Fixture','-c','user.email=fixture@example.invalid','commit','-qm','fixture')
                return git('rev-parse','HEAD')
            commit('mise.toml', '# toolchain')
            old=commit('rust/kona/src/lib.rs', 'old rust')
            with patch.object(SOURCE, 'ROOT', root):
                for kind in SOURCE.SCOPES:
                    self.assertEqual(len(SOURCE.fingerprint(kind)), 64)
                metadata={'commit_sha':old,'build_inputs_sha256':SOURCE.fingerprint('kona')}
                commit('op-service/errutil/errors.go', 'new go')
                # RWX's own cache key covers a task definition, so a .rwx change keeps the ancestor.
                commit('.rwx/go-tests.yml', 'changed task definition')
                SOURCE.verify('kona', metadata)
                for kind in ('kona', 'op-reth', 'sp1-executor'):
                    self.assertEqual(SOURCE.ancestor(kind), old)
                prestate = SOURCE.fingerprint('prestate')
                commit('cannon/mipsevm/state.go', 'changed cannon')
                self.assertNotEqual(prestate, SOURCE.fingerprint('prestate'))
                # Matching content alone cannot authorize an unrelated revision.
                with self.assertRaises((ValueError, subprocess.CalledProcessError)):
                    SOURCE.verify('kona', {**metadata,'commit_sha':'0'*40})
                commit('rust/kona/src/lib.rs', 'new rust')
                with self.assertRaises(ValueError): SOURCE.verify('kona', metadata)
                fresh={'commit_sha':git('rev-parse','HEAD'),'build_inputs_sha256':SOURCE.fingerprint('kona')}
                (root/'mise.toml').write_text('uncommitted tool change')
                with self.assertRaises(ValueError): SOURCE.verify('kona',fresh)
                git('checkout','--','mise.toml')
                commit('ops/ci/runtime/pr-gate.py', 'a script that no build runs')
                SOURCE.verify('kona', fresh)
                commit('ops/ci/runtime/go-dependencies.sh', 'changed build script')
                with self.assertRaises(ValueError): SOURCE.verify('kona', fresh)

    def test_scopes_list_every_script_that_a_build_reaches(self):
        runtime = Path(SOURCE.__file__).parent
        names = [path.name for path in runtime.iterdir() if path.is_file()]

        def reached(roots):
            """Return the scripts that `roots` name, directly or through another script."""
            seen, pending = set(roots), list(roots)
            while pending:
                current = pending.pop()
                # These two declare the lists under test. They name scripts as data and run none.
                if current in ('go-dependency-source.py', 'rust-target-cache.py'): continue
                text = (runtime / current).read_text()
                for name in names:
                    stem = name.rsplit('.', 1)[0]
                    if name not in seen and (name in text or f"helper('{stem}')" in text):
                        seen.add(name); pending.append(name)
            return seen

        listed = lambda paths: {path.removeprefix(SOURCE.RUNTIME) for path in paths}
        # .rwx/packages/go-*.yml run go-dependencies.sh; the compiler seeds also run go-full-tests.sh.
        self.assertLessEqual(reached(['go-dependencies.sh']), listed(SOURCE.BUILD_SCRIPTS))
        self.assertLessEqual(reached(['go-dependencies.sh', 'go-full-tests.sh']),
                             listed(SOURCE.BUILD_SCRIPTS + SOURCE.COMPILER_SCRIPTS))
        for kind, scope in SOURCE.SCOPES.items():
            self.assertLessEqual(set(SOURCE.BUILD_SCRIPTS), set(scope), kind)
        self.assertLessEqual(set(SOURCE.COMPILER_SCRIPTS), set(SOURCE.SCOPES['compiler']))
        # .rwx/rust.yml: the baselines, and the tool tasks of their own checkout.
        spec = importlib.util.spec_from_file_location('rust_target_cache', runtime / 'rust-target-cache.py')
        cache = importlib.util.module_from_spec(spec); spec.loader.exec_module(cache)
        self.assertLessEqual(reached(['rust-compiler-baseline.py', 'rust-dependencies.py', 'rwx-prepare.sh', 'rwx-rust-prepare.sh']),
                             set(cache.BASELINE_SCRIPTS))
