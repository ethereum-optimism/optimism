"""Circle alignment is removable; native policy is independently executable."""
import importlib.util
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[2]/'tests'))
import test_pr_gate as fixtures
_SPEC=importlib.util.spec_from_file_location('alignment',Path(__file__).resolve().parents[1]/'circle-alignment.py')
A=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(A)


class AlignmentTests(unittest.TestCase):
    def setUp(self):
        original=fixtures.G.ROOT
        fixtures.GateTests.setUp(self)
        mapping=A.G.read(original/'ops/ci/migration/circle-gates.json')
        for row in [*mapping['gates'].values(),mapping['e2e_gate']]:
            target=self.root/row['circle_config'];target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes((original/row['circle_config']).read_bytes())
        path=self.root/'circle-gates.json';path.write_text(json.dumps(mapping))
        for obj,name,value in [(A,'G',fixtures.G),(A,'MAPPING',path)]:
            handle=patch.object(obj,name,value);handle.start();self.addCleanup(handle.stop)

    def test_alignment_rejects_renamed_and_nonterminal_circle_prerequisites(self):
        A.configuration()
        path=self.root/'.circleci/continue/rust-ci.yml';source=path.read_text()
        for before,after in [('            - rust-fmt: terminal','            - future-required-job: terminal'),('            - rust-fmt: terminal','            - rust-fmt: success')]:
            path.write_text(source.replace(before,after,1))
            with self.assertRaisesRegex(ValueError,'Circle gate dependency'):A.configuration('required-rust-ci')
            fixtures.G.configuration('required-rust-ci')
        path.unlink()
        fixtures.G.configuration('required-rust-ci')

    def test_e2e_alignment_rejects_added_removed_renamed_and_nonterminal_jobs(self):
        native=A.e2e_configuration()
        for job in ('kona-proof-action-single','rust-e2e-binaries','rust-workspace-release'):self.assertIn(job,native)
        path=self.root/'.circleci/continue/rust-e2e.yml';source=path.read_text()
        line='            - rust-e2e-restart: terminal\n'
        self.assertIn(line,source)
        for changed in [source.replace(line,line+'            - rust-e2e-future: terminal\n',1),
                        source.replace(line,'',1),
                        source.replace(line,'            - rust-e2e-restarts: terminal\n',1),
                        source.replace(line,'            - rust-e2e-restart: success\n',1)]:
            path.write_text(changed)
            with self.assertRaisesRegex(ValueError,'Circle E2E gate dependency'):A.e2e_configuration()
        path.write_text(source)
        A.e2e_configuration()
