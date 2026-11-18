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
                SOURCE.verify('kona', metadata)
                self.assertEqual(SOURCE.ancestor('kona'), old)
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
                commit('ops/ci/runtime/new-helper.py', 'changed build helper')
                with self.assertRaises(ValueError): SOURCE.verify('kona', fresh)
