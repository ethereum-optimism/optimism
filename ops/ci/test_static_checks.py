#!/usr/bin/env python3
"""Run actual pinned static tools, original exclusions and intentional failures."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SPEC=importlib.util.spec_from_file_location('static',Path(__file__).with_name('static-checks.py'))
C=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(C)


class Reports(unittest.TestCase):
    def test_shell_discovery_and_original_authority(self):
        self.assertEqual(C.digest(C.ROOT/C.ORB),C.ORB_SHA256)
        self.assertEqual(C.shell_files('./folder with spaces/a.sh\n./b.sh\n'),['./b.sh','./folder with spaces/a.sh'])
        for bad in ('','./a.sh\n./a.sh','../a.sh','./a.sh/../b.sh'):
            with self.subTest(bad=bad),self.assertRaises(ValueError):C.shell_files(bad)

    def test_rule_test_reports_preserve_original_missing_fixture_and_full_matches(self):
        data={'config_missing_tests':['.semgrep/rules/missing.yaml'],'config_missing_fixtests':[],
              'config_with_errors':[],'fixtest_results':{},'results':{'.semgrep/rules/test.yaml':{'checks':{'rule':{
                'passed':True,'errors':[],'matches':{'/work/.semgrep/tests/test.py':{'expected_lines':[2],'reported_lines':[2]}}}}}}}
        result=C.machine_report('semgrep-test',data,'/work')
        self.assertIn('<repo>/.semgrep/tests/test.py',result['results']['.semgrep/rules/test.yaml']['checks']['rule']['matches'])
        self.assertEqual(result['config_missing_tests'],data['config_missing_tests'])
        data['results']['.semgrep/rules/test.yaml']['checks']['rule']['matches']['/work/.semgrep/tests/test.py']['reported_lines']=[]
        with self.assertRaisesRegex(ValueError,'annotations'):C.machine_report('semgrep-test',data,'/work')


class LiveTools(unittest.TestCase):
    def fixture(self):
        tmp=tempfile.TemporaryDirectory();self.addCleanup(tmp.cleanup);root=Path(tmp.name)/'repo'
        (root/'ops/ci').mkdir(parents=True);(root/'packages/contracts-bedrock/lib').mkdir(parents=True)
        (root/'docs/public-docs').mkdir(parents=True)
        for source in Path(C.__file__).parent.glob('*.py'):
            if not source.name.startswith('test_'):shutil.copy2(source,root/'ops/ci'/source.name)
        shutil.copy2(C.ROOT/C.ORB,root/C.ORB)
        (root/'.gitignore').write_text('.ci/\nshellcheck.log\n')
        subprocess.run(['git','init','-q'],cwd=root,check=True)
        self.commit(root)
        return root

    def commit(self,root):
        subprocess.run(['git','add','.'],cwd=root,check=True)
        subprocess.run(['git','-c','user.name=Fixture','-c','user.email=fixture@example.invalid','commit','-qm','original fixture'],cwd=root,check=True)

    def run_check(self,root,job,provider='rwx',evidence_label=None):
        sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
        env=dict(os.environ,CI_BRANCH='pilot',CI_COMMIT_SHA=sha,CI_CHECK_PROVIDER=provider,RWX_RUN_ID='a'*32,RWX_TASK_ATTEMPT_NUMBER='1')
        child=subprocess.run(['python3',str(root/'ops/ci/static-checks.py'),job],cwd=root,env=env,
                             capture_output=True,text=True,timeout=90)
        d=root/'.ci/static-checks'/job
        if child.returncode and os.environ.get('RWX_STATIC_FIXTURE_RETAIN_DIR'):
            target=Path(os.environ['RWX_STATIC_FIXTURE_RETAIN_DIR'])/(evidence_label or job)
            shutil.rmtree(target,ignore_errors=True);shutil.copytree(d,target)
        return child,d,sha

    @unittest.skipUnless(os.environ.get('RWX_LIVE_SHELL_FIXTURE')=='1','Pinned original ShellCheck fixture')
    def test_original_orb_exclusions_spaces_fresh_failure_and_complete_parity(self):
        root=self.fixture();(root/'folder with spaces').mkdir()
        for name in ('check.sh','folder with spaces/check.sh'):
            (root/name).write_text('#!/bin/bash\nset -euo pipefail\nvalue=original\nprintf "%s\\n" "$value"\n')
        for name in ('packages/contracts-bedrock/lib/ignored.sh','docs/public-docs/ignored.sh'):
            (root/name).write_text('#!/bin/bash\necho $unquoted\n')
        self.commit(root)
        first,d,sha=self.run_check(root,'shell-check');self.assertEqual(first.returncode,0,first.stderr+first.stdout)
        selected=json.loads((d/'selection.json').read_text());self.assertEqual(len(selected['files']),2)
        native=root/'.ci/native';shutil.copytree(d,native)
        second,d,_=self.run_check(root,'shell-check','circleci');self.assertEqual(second.returncode,0,second.stderr+second.stdout)
        compared=subprocess.run(['python3',str(root/'ops/ci/compare-static-checks.py'),'--circle',str(d),'--rwx',str(native),
            '--sha',sha,'--job','shell-check','--output',str(root/'.ci/parity.json')],capture_output=True,text=True,timeout=30)
        self.assertEqual(compared.returncode,0,compared.stderr)
        (root/'check.sh').write_text('#!/bin/bash\nvalue="$1"\necho $value\n');self.commit(root)
        failed,d,_=self.run_check(root,'shell-check',evidence_label='shell-initial-failure');self.assertNotEqual(failed.returncode,0)
        self.assertIn('SC2086',(d/'original.shellcheck.log').read_text())
        self.assertNotEqual(json.loads((d/'final.json').read_text())['exit_code'],0)
        self.assertEqual(json.loads((d/'check.stage.json').read_text())['exit_code'],1)
        corrupt=d/'check.log';corrupt.write_text(corrupt.read_text()+'corrupt')
        # Every original stays sealed even on failure; failed reports cannot be
        # converted into passing comparison inputs by changing the reporting view.
        self.assertNotEqual(C.digest(corrupt),json.loads((d/'final.json').read_text())['original_sha256']['check.log'])

    @unittest.skipUnless(os.environ.get('RWX_LIVE_SEMGREP_FIXTURE')=='1','Pinned Semgrep execution fixture')
    def test_actual_rule_annotations_full_scan_and_initial_failure(self):
        root=self.fixture();(root/'.semgrep/rules').mkdir(parents=True);(root/'.semgrep/tests').mkdir()
        (root/'.semgrep/rules/fixture.yaml').write_text('rules:\n- id: original-rule\n  languages: [python]\n  message: original diagnostic\n  severity: ERROR\n  pattern: forbidden(...)\n')
        (root/'.semgrep/tests/fixture.py').write_text('# ruleid: original-rule\nforbidden(1)\n# ok: original-rule\nallowed(1)\n')
        (root/'.semgrepignore').write_text('.semgrep/\nops/\n')
        (root/'source.py').write_text('allowed(0)\n');(root/'known-issue.py').write_text('forbidden(0)\n');self.commit(root)
        subprocess.run(['git','branch','develop'],cwd=root,check=True)
        subprocess.run(['git','checkout','-qb','pilot'],cwd=root,check=True)
        (root/'source.py').write_text('allowed(1)\n');self.commit(root)
        for job in ('semgrep-test','semgrep-scan-local'):
            first,d,sha=self.run_check(root,job);self.assertEqual(first.returncode,0,first.stderr+first.stdout)
            if job=='semgrep-scan-local':
                report=json.loads((d/'coverage.json').read_text())
                self.assertEqual(report['paths']['scanned'],['source.py'])
                self.assertEqual(json.loads((d/'settings.json').read_text())['baseline']['changed_files'],{'source.py':'M'})
            native=root/'.ci'/('native-'+job);shutil.copytree(d,native)
            second,d,_=self.run_check(root,job,'circleci');self.assertEqual(second.returncode,0,second.stderr+second.stdout)
            compared=subprocess.run(['python3',str(root/'ops/ci/compare-static-checks.py'),'--circle',str(d),'--rwx',str(native),
                '--sha',sha,'--job',job,'--output',str(root/'.ci'/('parity-'+job+'.json'))],capture_output=True,text=True,timeout=30)
            self.assertEqual(compared.returncode,0,compared.stderr)
        (root/'source.py').write_text('forbidden(2)\n');self.commit(root)
        failed,d,_=self.run_check(root,'semgrep-scan-local',evidence_label='semgrep-initial-failure');self.assertNotEqual(failed.returncode,0)
        self.assertEqual(json.loads((d/'check.json').read_text())['results'][0]['check_id'],'semgrep.rules.original-rule')
        self.assertEqual(json.loads((d/'check.stage.json').read_text())['exit_code'],1)
        self.assertNotEqual(json.loads((d/'final.json').read_text())['exit_code'],0)
        subprocess.run(['git','branch','-D','develop'],cwd=root,check=True,capture_output=True)
        unavailable,d,_=self.run_check(root,'semgrep-scan-local',evidence_label='semgrep-missing-baseline')
        self.assertNotEqual(unavailable.returncode,0)
        self.assertIn('develop',' '.join(json.loads((d/'final.json').read_text())['report_errors']))
        self.assertFalse((d/'check.stage.json').exists())


if __name__=='__main__':unittest.main()
