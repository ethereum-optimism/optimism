"""Run the contract producer with real Forge and verify its Go artifact handoff."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[3]


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_DEPENDENCIES_FIXTURE') == '1',
                     'Opt-in pinned Forge/Go contract producer fixture')
class ContractDependenciesTests(unittest.TestCase):
    def test_skips_executable_solidity_tests_but_preserves_helpers_and_bound_artifacts(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / 'project'; root.mkdir()
            def run(*argv, cwd=root):
                return subprocess.run(argv, cwd=cwd, text=True, capture_output=True, check=True)
            run('git', 'init', '-q')
            contracts = root / 'packages/contracts-bedrock'; contracts.mkdir(parents=True)
            # A real initialized submodule exercises the production update command.
            module = Path(temporary) / 'module'; module.mkdir()
            run('git', 'init', '-q', cwd=module); (module / 'library.txt').write_text('fixture')
            run('git', 'add', '.', cwd=module)
            run('git', '-c', 'user.name=CI fixture', '-c', 'user.email=ci@example.invalid',
                'commit', '-qm', 'Fixture library', cwd=module)
            run('git', '-c', 'protocol.file.allow=always', 'submodule', 'add', '-q',
                str(module), 'packages/contracts-bedrock/lib')
            recipe = (ROOT / 'packages/contracts-bedrock/justfile').read_text()
            recipe = recipe[recipe.index('forge-build *ARGS:'):recipe.index('# Developer build command')]
            files = {
                'mise.toml': '# Pinned tools are inherited from the prepared worker.\n',
                '.gitignore': '.ci/\ntmp/\n**/forge-artifacts/\n**/cache/\n**/artifacts/\n',
                'packages/contracts-bedrock/justfile': recipe,
                'packages/contracts-bedrock/foundry.toml': '[profile.default]\nsrc="src"\nscript="scripts"\nout="forge-artifacts"\nsolc="0.8.28"\nbuild_info=true\nbuild_info_path="artifacts/build-info"\n',
                'packages/contracts-bedrock/src/Protocol.sol': '// SPDX-License-Identifier: MIT\npragma solidity 0.8.28;\ncontract Protocol {}\n',
                # The alias must retain .sol helpers inside test/, not exclude the directory.
                'packages/contracts-bedrock/test/dispute/zk/MockSP1Verifier.sol': '// SPDX-License-Identifier: MIT\npragma solidity 0.8.28;\ncontract MockSP1Verifier { function verify() external pure returns (bool) { return true; } }\n',
                # An intentionally invalid executable test proves the producer excludes .t.sol.
                'packages/contracts-bedrock/test/Unneeded.t.sol': 'this file must not be compiled\n',
                'op-deployer/justfile': 'copy-contract-artifacts:\n  go run copy.go\n',
                'op-deployer/copy.go': 'package main\nimport("encoding/json";"os")\nfunc main(){\n data,err:=os.ReadFile("../packages/contracts-bedrock/forge-artifacts/MockSP1Verifier.sol/MockSP1Verifier.json");if err!=nil{panic(err)}\n var artifact struct{Bytecode struct{Object string}};if err=json.Unmarshal(data,&artifact);err!=nil{panic(err)};if len(artifact.Bytecode.Object)<4{panic("missing deployable mock bytecode")}\n if err=os.MkdirAll("pkg/deployer/artifacts/forge-artifacts",0755);err!=nil{panic(err)};if err=os.WriteFile("pkg/deployer/artifacts/forge-artifacts/mock.json",data,0644);err!=nil{panic(err)}\n}\n',
            }
            for name, text in files.items():
                path = root / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_text(text)
            for name in ('go-dependencies.sh', 'go-artifacts.py'):
                path = root / 'ops/ci/runtime' / name; path.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / 'ops/ci/runtime' / name, path)
            run('git', 'add', '.')
            run('git', '-c', 'user.name=CI fixture', '-c', 'user.email=ci@example.invalid',
                'commit', '-qm', 'Contract dependency fixture')
            sha = run('git', 'rev-parse', 'HEAD').stdout.strip()
            env = {**os.environ, 'CI_COMMIT_SHA': sha}
            result = subprocess.run(['bash', 'ops/ci/runtime/go-dependencies.sh', 'contracts'],
                                    cwd=root, env=env, text=True, capture_output=True, timeout=120)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            artifact = root / '.ci/go-tests/dependencies/contracts'
            metadata = json.loads((artifact / 'metadata.json').read_text())
            self.assertEqual(metadata['settings'], {'profile': 'ci', 'build_args': ['--deny-warnings', '--skip', 'test']})
            self.assertTrue(any('MockSP1Verifier.sol/MockSP1Verifier.json' in name for name in metadata['files']))
            self.assertFalse(any('Unneeded.t.sol' in name for name in metadata['files']))
            # Cold restoration must supply the same deployable helper and reject old settings.
            spec = importlib.util.spec_from_file_location('artifact_fixture', root / 'ops/ci/runtime/go-artifacts.py')
            module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
            with patch.dict(os.environ, CI_COMMIT_SHA=sha):
                shutil.rmtree(contracts / 'forge-artifacts'); module.restore('contracts', artifact)
                retained = (artifact / 'metadata.json').read_bytes()
                (contracts / 'src/Protocol.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.28;\ncontract Protocol { function value(uint256 unused) external pure returns(uint256) { return 42; } }\n')
                failure = subprocess.run(['bash', 'ops/ci/runtime/go-dependencies.sh', 'contracts'],
                                         cwd=root, env=env, text=True, capture_output=True, timeout=120)
                self.assertNotEqual(failure.returncode, 0, failure.stdout + failure.stderr)
                self.assertIn('unused', (failure.stdout + failure.stderr).lower())
                self.assertEqual((artifact / 'metadata.json').read_bytes(), retained)
                metadata['settings'] = {'profile': 'ci'}
                (artifact / 'metadata.json').write_text(json.dumps(metadata))
                with self.assertRaisesRegex(ValueError, 'settings'): module.restore('contracts', artifact)


if __name__ == '__main__': unittest.main()
