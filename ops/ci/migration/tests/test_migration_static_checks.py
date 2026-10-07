"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_static_checks as T
C = T.C
json = T.json
os = T.os
shutil = T.shutil
subprocess = T.subprocess
tempfile = T.tempfile


class LiveTools(T._LiveToolsFixtures, unittest.TestCase):
    def fixture(self):
        root=super().fixture()
        target=root/'ops/ci/migration'; target.mkdir()
        for source in Path(__file__).resolve().parents[1].glob('*.py'):
            shutil.copy2(source,target/source.name)
        return root


    @unittest.skipUnless(os.environ.get('RWX_LIVE_SHELL_FIXTURE') == '1', 'Pinned original ShellCheck fixture')
    def test_original_orb_exclusions_spaces_fresh_failure_and_complete_parity(self):
        root = self.fixture()
        (root / 'folder with spaces').mkdir()
        for name in ('check.sh', 'folder with spaces/check.sh'):
            (root / name).write_text('#!/bin/bash\nset -euo pipefail\nvalue=original\nprintf "%s\\n" "$value"\n')
        for name in ('packages/contracts-bedrock/lib/ignored.sh', 'docs/public-docs/ignored.sh'):
            (root / name).write_text('#!/bin/bash\necho $unquoted\n')
        self.commit(root)
        first, d, sha = self.run_check(root, 'shell-check')
        self.assertEqual(first.returncode, 0, first.stderr + first.stdout)
        selected = json.loads((d / 'selection.json').read_text())
        self.assertEqual(len(selected['files']), 2)
        native = root / '.ci/native'
        shutil.copytree(d, native)
        second, d, _ = self.run_check(root, 'shell-check', 'circleci')
        self.assertEqual(second.returncode, 0, second.stderr + second.stdout)
        compared = subprocess.run(['python3', str(root / 'ops/ci/migration/compare-static-checks.py'), '--circle', str(d), '--rwx', str(native), '--sha', sha, '--job', 'shell-check', '--output', str(root / '.ci/parity.json')], capture_output=True, text=True, timeout=30)
        self.assertEqual(compared.returncode, 0, compared.stderr)
        (root / 'check.sh').write_text('#!/bin/bash\nvalue="$1"\necho $value\n')
        self.commit(root)
        failed, d, _ = self.run_check(root, 'shell-check', evidence_label='shell-initial-failure')
        self.assertNotEqual(failed.returncode, 0)
        self.assertIn('SC2086', (d / 'original.shellcheck.log').read_text())
        self.assertNotEqual(json.loads((d / 'final.json').read_text())['exit_code'], 0)
        self.assertEqual(json.loads((d / 'check.stage.json').read_text())['exit_code'], 1)
        corrupt = d / 'check.log'
        corrupt.write_text(corrupt.read_text() + 'corrupt')
        self.assertNotEqual(C.digest(corrupt), json.loads((d / 'final.json').read_text())['original_sha256']['check.log'])

    @unittest.skipUnless(os.environ.get('RWX_LIVE_SEMGREP_FIXTURE') == '1', 'Pinned Semgrep execution fixture')
    def test_actual_rule_annotations_full_scan_and_initial_failure(self):
        root = self.fixture()
        (root / '.semgrep/rules').mkdir(parents=True)
        (root / '.semgrep/tests').mkdir()
        (root / '.semgrep/rules/fixture.yaml').write_text('rules:\n- id: original-rule\n  languages: [python]\n  message: original diagnostic\n  severity: ERROR\n  pattern: forbidden(...)\n')
        (root / '.semgrep/tests/fixture.py').write_text('# ruleid: original-rule\nforbidden(1)\n# ok: original-rule\nallowed(1)\n')
        (root / '.semgrepignore').write_text('.semgrep/\nops/\n')
        (root / 'source.py').write_text('allowed(0)\n')
        (root / 'known-issue.py').write_text('forbidden(0)\n')
        self.commit(root)
        subprocess.run(['git', 'branch', 'develop'], cwd=root, check=True)
        subprocess.run(['git', 'checkout', '-qb', 'pilot'], cwd=root, check=True)
        (root / 'source.py').write_text('allowed(1)\n')
        self.commit(root)
        for job in ('semgrep-test', 'semgrep-scan-local'):
            first, d, sha = self.run_check(root, job)
            self.assertEqual(first.returncode, 0, first.stderr + first.stdout)
            if job == 'semgrep-scan-local':
                report = json.loads((d / 'coverage.json').read_text())
                self.assertEqual(report['paths']['scanned'], ['source.py'])
                self.assertEqual(json.loads((d / 'settings.json').read_text())['baseline']['changed_files'], {'source.py': 'M'})
            native = root / '.ci' / ('native-' + job)
            shutil.copytree(d, native)
            second, d, _ = self.run_check(root, job, 'circleci')
            self.assertEqual(second.returncode, 0, second.stderr + second.stdout)
            compared = subprocess.run(['python3', str(root / 'ops/ci/migration/compare-static-checks.py'), '--circle', str(d), '--rwx', str(native), '--sha', sha, '--job', job, '--output', str(root / '.ci' / ('parity-' + job + '.json'))], capture_output=True, text=True, timeout=30)
            self.assertEqual(compared.returncode, 0, compared.stderr)
        (root / 'source.py').write_text('forbidden(2)\n')
        self.commit(root)
        failed, d, _ = self.run_check(root, 'semgrep-scan-local', evidence_label='semgrep-initial-failure')
        self.assertNotEqual(failed.returncode, 0)
        self.assertEqual(json.loads((d / 'check.json').read_text())['results'][0]['check_id'], 'semgrep.rules.original-rule')
        self.assertEqual(json.loads((d / 'check.stage.json').read_text())['exit_code'], 1)
        self.assertNotEqual(json.loads((d / 'final.json').read_text())['exit_code'], 0)
        subprocess.run(['git', 'branch', '-D', 'develop'], cwd=root, check=True, capture_output=True)
        unavailable, d, _ = self.run_check(root, 'semgrep-scan-local', evidence_label='semgrep-missing-baseline')
        self.assertNotEqual(unavailable.returncode, 0)
        self.assertIn('develop', ' '.join(json.loads((d / 'final.json').read_text())['report_errors']))
        self.assertFalse((d / 'check.stage.json').exists())
