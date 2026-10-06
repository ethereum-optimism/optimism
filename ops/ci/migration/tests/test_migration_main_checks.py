"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_main_checks as T
MAIN = T.MAIN
json = T.json
os = T.os
patch = T.patch
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
threading = T.threading


_SPEC=importlib.util.spec_from_file_location('l2', Path(__file__).resolve().parents[1] / 'l2-chains-sync-check.py')
L2=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(L2)


class MainTests(T._MainTestsFixtures, unittest.TestCase):

    def install_helpers(self, root):
        super().install_helpers(root)
        target=root/'ops/ci/migration'; target.mkdir()
        for name in ('l2-chains-sync-check.py','report-evidence.py'):
            shutil.copy2(Path(__file__).resolve().parents[1]/name,target/name)

    @unittest.skipUnless(os.environ.get('RWX_LIVE_MAIN_FIXTURE') == '1', 'Opt-in pinned original Main adapters')
    def test_real_l2_and_forge_adapters_preserve_nested_cwd_and_report_after_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.install_helpers(root)
            scripts = root / '.circleci/scripts'
            scripts.mkdir(parents=True)
            shutil.copyfile(MAIN.ROOT / '.circleci/scripts/check-l2-chains-sync.sh', scripts / 'check-l2-chains-sync.sh')
            (root / '.circleci/l2-rpcs.json').write_text('{"op-mainnet":{}}\n')
            (root / '.circleci/continue').mkdir()
            (root / '.circleci/continue/main.yml').write_text('workflows:\n  scheduled-daily-tests:\n    jobs:\n      - contracts-bedrock-tests-l2-fork:\n          matrix:\n            parameters:\n              fork_op_chain:\n                - op-mainnet\n              test_profile: liteci\n')
            version = root / 'op-deployer/pkg/deployer/forge/version.json'
            version.parent.mkdir(parents=True)
            version.write_text('{"forge":"v1.8.3"}\n')
            (root / 'mise.toml').write_text('[tools]\nforge = "1.8.3"\n')
            shutil.copyfile(MAIN.ROOT / 'op-deployer/justfile', root / 'op-deployer/justfile')
            self.install_justfiles(root)
            sha = self.commit(root)
            for job in ('l2-chains-sync-check', 'op-deployer-forge-version'):
                if job == 'l2-chains-sync-check':
                    result = subprocess.run([sys.executable, str(root/'ops/ci/migration/l2-chains-sync-check.py')], cwd=root/'op-deployer', env={**os.environ,'CI_COMMIT_SHA':sha,'CI_BRANCH':'codex/rwx-ci-pilot','CI_CHECK_PROVIDER':'circleci'}, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
                    report=root/'.ci/main-checks/l2-chains-sync-check'
                else:
                    result, report = self.run_adapter(root, job, sha)
                self.assertEqual(result.returncode, 0, result.stdout)
                stage = json.loads((report / 'check.stage.json').read_text())
                self.assertEqual(stage['cwd'], str(root / 'op-deployer') if job == 'op-deployer-forge-version' else str(root))
                self.assertEqual(stage['argv'], L2.COMMAND if job == 'l2-chains-sync-check' else MAIN.COMMANDS[job])
            version.write_text('{"forge":"v0.0.0"}\n')
            sha = self.commit(root)
            result, report = self.run_adapter(root, 'op-deployer-forge-version', sha)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(json.loads((report / 'check.stage.json').read_text())['exit_code'], 1)
            self.assertIn("Forge versions don't match", (report / 'check.log').read_text())
            self.assertNotEqual(json.loads((report / 'final.json').read_text())['exit_code'], 0)
            self.assertIn('<failure', (report / 'check.junit.xml').read_text())

    @unittest.skipUnless(os.environ.get('RWX_LIVE_MAIN_FIXTURE') == '1', 'Opt-in original Go/Mockery generation')
    def test_real_mock_generation_executes_freshly_and_detects_stale_committed_mock(self):
        self.exercise_mock_original_reports()

    def verify_mock_reports(self, originals, sha):
        spec=importlib.util.spec_from_file_location('comparison',Path(__file__).resolve().parents[1]/'compare-main-checks.py')
        comparison=importlib.util.module_from_spec(spec); spec.loader.exec_module(comparison)
        self.assertTrue(comparison.compare(originals,'check-generated-mocks-op-node',sha)['verified_parity'])

    def test_l2_selection_keeps_every_chain_and_rejects_missing_or_duplicate_matrix(self):
        config = {'workflows': {'scheduled-daily-tests': {'jobs': [{'contracts-bedrock-tests-l2-fork': {'matrix': {'parameters': {'fork_op_chain': ['op-mainnet', 'new-chain']}}}}]}}}
        self.assertEqual(L2.l2_selection(config, {'new-chain': {}, 'op-mainnet': {}})['matrix_chains'], ['new-chain', 'op-mainnet'])
        config['workflows']['scheduled-daily-tests']['jobs'][0]['contracts-bedrock-tests-l2-fork']['matrix']['parameters']['fork_op_chain'].append('new-chain')
        with self.assertRaises(ValueError):
            L2.l2_selection(config, {'op-mainnet': {}})
