"""Real Forge/Just/Kontrol fixtures for complete fresh builds and original failures."""
import json
import copy
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tarfile
import tempfile
import time
import unittest

import importlib.util


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-','_'), Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


K = helper('kontrol-build'); C = helper('compare-kontrol-build'); SCRIPTS = Path(__file__).parent


@unittest.skipUnless(os.environ.get('RWX_LIVE_KONTROL_FIXTURE') == '1','Opt-in actual Forge, Just and pinned Kontrol Docker workloads')
class LiveTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name); self.contracts = self.root / 'packages/contracts-bedrock'
        scripts = self.root / 'ops/ci'; scripts.mkdir(parents=True)
        for path in SCRIPTS.glob('*.py'): shutil.copy2(path,scripts / path.name)
        shutil.copy2(SCRIPTS / 'kontrol-image.json',scripts / 'kontrol-image.json')
        self.env = {**os.environ,'CI':'true','CI_BRANCH':'codex/rwx-ci-pilot',
            'RWX_RUN_ID':os.environ.get('RWX_RUN_ID','isolated-kontrol-fixture'),'RWX_TASK_ATTEMPT_NUMBER':'1'}
        for name in (*K.ENVIRONMENT,'KONTROL_CI_REPORT_DIR'):
            self.env.pop(name,None)
        files = {'mise.toml':(K.ROOT/'mise.toml').read_text(),
            '.gitignore':'.ci/\ntmp/\npackages/contracts-bedrock/forge-artifacts/\npackages/contracts-bedrock/cache/\npackages/contracts-bedrock/artifacts/\npackages/contracts-bedrock/deployments/\npackages/contracts-bedrock/snapshots/\nop-deployer/pkg/deployer/artifacts/forge-artifacts/\n',
            'packages/contracts-bedrock/foundry.toml':'''[profile.default]
src="src"
script="scripts"
out="forge-artifacts"
solc="0.8.15"
evm_version="london"
optimizer=true
optimizer_runs=200
ast=true
build_info_path="artifacts/build-info"
extra_output=["metadata","storageLayout"]
remappings=["forge-std/=lib/forge-std/src/"]
fs_permissions=[{access="read-write",path="."},{access="read-write",path="../../.ci"}]
[profile.ci.fuzz]
runs=128
''',
            'packages/contracts-bedrock/src/FixtureStorage.sol':'''// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;
contract FixtureStorage { uint256 public value; constructor(uint256 initial) { value=initial; } }
''',
            'packages/contracts-bedrock/scripts/deploy/Deploy.s.sol':'''// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;
import {Vm} from "forge-std/Vm.sol";
import {VmSafe} from "forge-std/Vm.sol";
import {StateDiff} from "../libraries/StateDiff.sol";
import {FixtureStorage} from "../../src/FixtureStorage.sol";
contract Deploy {
 Vm private constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
 function runWithStateDiff() public {
  bool fp=vm.envOr("KONTROL_FP_DEPLOYMENT",false);
  vm.startStateDiffRecording();
  FixtureStorage deployed=new FixtureStorage(fp ? 84 : 42);
  VmSafe.AccountAccess[] memory accesses=vm.stopAndReturnStateDiff();
  string memory json=StateDiff.encodeAccountAccesses(accesses);
  vm.writeJson(json,string.concat(vm.projectRoot(),"/snapshots/state-diff/31337.json"));
  string memory names=vm.serializeAddress("contracts","FixtureStorage",address(deployed));
  vm.writeJson(names,vm.envString("DEPLOYMENT_OUTFILE"));
  vm.writeLine("../../.ci/fresh-marker",fp ? "fault-proofs" : "default");
 }
}
''',
            K.PROOFS+'/FutureProof.sol':'''// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;
import {DeploymentSummary} from "./utils/DeploymentSummary.sol";
contract FutureProof is DeploymentSummary { function futureProof() public pure returns(uint256) {return 42;} }
'''}
        original = (K.CONTRACTS / 'justfile').read_text()
        build = original[original.index('forge-build *ARGS:'):original.index('# Developer build command')]
        summaries = original[original.index('# Generates default Kontrol summary.'):original.index('# Generates ABI snapshots')]
        files['packages/contracts-bedrock/justfile'] = build + '\n' + summaries
        for variant,names in K.GENERATED.items():
            for name in names:
                files[name]='// SPDX-License-Identifier: MIT\npragma solidity 0.8.15;\ncontract '+Path(name).stem+' {}\n'
        files['packages/contracts-bedrock/scripts/Artifacts.s.sol']='// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\ncontract Artifacts {}\n'
        deployer=(K.ROOT/'op-deployer/justfile').read_text()
        files['op-deployer/justfile']=deployer[deployer.index('copy-contract-artifacts:\n'):deployer.index('_LDFLAGSSTRING :=')]
        for name in ('go.mod','go.sum','op-deployer/pkg/deployer/artifacts/cmd/mktar/main.go'):
            files[name]=(K.ROOT/name).read_text()
        for name,content in files.items():
            path=self.root/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_text(content)
        for relative in ('test/kontrol/scripts/make-summary-deployment.sh','test/kontrol/scripts/common.sh',
                         'test/kontrol/scripts/json/clean_json.py','test/kontrol/scripts/json/reverse_key_values.py','scripts/libraries/StateDiff.sol'):
            target=self.contracts/relative;target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(K.CONTRACTS/relative,target)
        (self.contracts/'deployments/hardhat').mkdir(parents=True)
        (self.contracts/'snapshots/state-diff').mkdir(parents=True)
        self.run_command(['git','init','-q'],self.root)
        for name,value in [('user.name','CI fixture'),('user.email','ci-fixture@example.invalid')]:
            self.run_command(['git','config',name,value],self.root)
        # File transport copies objects; a local-path clone uses hardlinks that
        # RWX's snapshot filesystem cannot provide reliably. Keep the same gitlink.
        self.run_command(['git','-c','protocol.file.allow=always','submodule','add','-q',(K.CONTRACTS/'lib/forge-std').as_uri(),
                          'packages/contracts-bedrock/lib/forge-std'],self.root)
        self.commit()
        self.image=self.root/'.ci/kontrol-build/image'
        shutil.copytree(K.ROOT/'.ci/kontrol-build/image',self.image)
        self.run_command([sys.executable,'ops/ci/kontrol-contracts.py'],self.root)
        self.artifact=self.root/'.ci/go-tests/dependencies/contracts-kontrol'
        self.report=self.root/'.ci/kontrol-build/run'

    def run_command(self,argv,cwd,**extra):
        result=subprocess.run(argv,cwd=cwd,env={**self.env,**extra},capture_output=True,text=True,timeout=180)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr);return result.stdout.strip()

    def commit(self):
        self.run_command(['git','add','.'],self.root);self.run_command(['git','commit','-qm','Kontrol original fixture input'],self.root)
        self.sha=self.run_command(['git','rev-parse','HEAD'],self.root);self.env['CI_COMMIT_SHA']=self.sha

    def execute(self,provider='circleci'):
        argv=[sys.executable,'ops/ci/kontrol-build.py','--provider',provider]
        if provider=='rwx': argv+=['--contract-artifact',str(self.artifact)]
        return subprocess.run(argv,cwd=self.root,env=self.env,capture_output=True,text=True,timeout=240)

    def retain(self,name):
        target=K.ROOT/'.ci/kontrol-build/helper-fixtures'/name
        shutil.rmtree(target,ignore_errors=True);shutil.copytree(self.report,target)

    def reset_runtime(self):
        # This checkout is an isolated fixture owned by this test.
        self.run_command(['git','restore','--',*sum(K.GENERATED.values(),[])],self.root)
        for name in ('forge-artifacts','cache','artifacts'):
            shutil.rmtree(self.contracts/name,ignore_errors=True)
        self.run_command([sys.executable,'ops/ci/go-artifacts.py','restore','contracts-kontrol',str(self.artifact)],self.root)

    def test_actual_two_variants_fresh_build_future_proof_discovery_and_strict_parity(self):
        for label in ('first','reused','changed'):
            if label=='changed':
                source=self.contracts/'src/FixtureStorage.sol'
                source.write_text(source.read_text()+'\ncontract FutureCompilerInput {function fresh() external pure returns(uint256) {return 7;}}\n')
                self.commit()
            if label!='first':self.run_command([sys.executable,'ops/ci/kontrol-contracts.py'],self.root)
            cache=K.G.read(self.artifact/'preparation/cache.json')
            self.assertEqual(cache['reused'],label=='reused')
            target=K.ROOT/'.ci/kontrol-build/helper-fixtures'/('producer-'+label)
            shutil.rmtree(target,ignore_errors=True);shutil.copytree(self.artifact,target)
        pair=self.root/'.ci/paired'
        for provider,key in [('circleci','circle'),('rwx','rwx')]:
            self.reset_runtime();result=self.execute(provider);self.assertEqual(result.returncode,0,result.stdout+result.stderr)
            shutil.copytree(self.report,pair/key);self.retain('complete-'+key)
        self.assertEqual((self.root/'.ci/fresh-marker').read_text().splitlines(),['default','fault-proofs']*2)
        report=C.compare(pair,self.sha);self.assertTrue(report['verified_parity'])
        self.assertIn(K.PROOFS+'/FutureProof.sol',report['selection']['proof_sources'])
        self.assertEqual(report['coverage']['tests'],0)
        original=C.compiler(pair/'rwx/initial-compiler')
        summaries=C.compiler(pair/'rwx/summaries-compiler')
        retained=report['compiler_phases']['summaries']['verified_retained_previous_artifacts']['rwx']
        self.assertTrue(retained)
        retained_path=retained[0]['path']
        self.assertEqual(original[retained_path],summaries[retained_path])
        histories=([original],[original])
        # Old payloads remain mandatory and cannot acquire a new identity.
        changed=dict(summaries);value=json.loads(changed[retained_path]);value['bytecode']['object']='0xdeadbeef'
        changed[retained_path]=json.dumps(value).encode()
        with self.assertRaisesRegex(ValueError,'retained'):
            C.compare_compiler(summaries,changed,[str(self.root)]*2,histories)
        added=dict(summaries);added[retained_path.removesuffix('.json')+'.extra.json']=summaries[retained_path]
        with self.assertRaisesRegex(ValueError,'retained'):
            C.compare_compiler(summaries,added,[str(self.root)]*2,histories)
        missing=dict(summaries);missing.pop(retained_path)
        with self.assertRaisesRegex(ValueError,'inventories'):
            C.compare_compiler(summaries,missing,[str(self.root)]*2,histories)
        with self.assertRaisesRegex(ValueError,'unbound'):
            C.compare_compiler(summaries,summaries,[str(self.root)]*2)
        cache_name='packages/contracts-bedrock/cache/solidity-files-cache.json'
        cache=json.loads(original[cache_name])
        binding=cache['files']['scripts/Artifacts.s.sol']['artifacts']['Artifacts']['0.8.28']['default']
        old='packages/contracts-bedrock/forge-artifacts/'+binding['path']
        alternate='Artifacts.s.sol/Artifacts.0.8.28.default.json'
        if binding['path']==alternate:alternate='Artifacts.s.sol/Artifacts.0.8.28.json'
        new='packages/contracts-bedrock/forge-artifacts/'+alternate
        renamed=dict(original);renamed[new]=renamed.pop(old);binding['path']=alternate
        renamed[cache_name]=json.dumps(cache).encode()
        aliases=C.compare_compiler(original,renamed,[str(self.root)]*2)['verified_artifact_path_aliases']
        self.assertEqual(len(aliases),1)
        # Every original artifact and reference remains mandatory under aliases.
        missing=dict(renamed);missing.pop(new)
        with self.assertRaises(ValueError):C.compare_compiler(original,missing,[str(self.root)]*2)
        extra=dict(renamed);extra[old]=renamed[new]
        with self.assertRaisesRegex(ValueError,'unbound'):C.compare_compiler(original,extra,[str(self.root)]*2)
        corrupt=dict(renamed);value=json.loads(corrupt[new]);value['bytecode']['object']='0xdeadbeef';corrupt[new]=json.dumps(value).encode()
        with self.assertRaisesRegex(ValueError,'runtime'):C.compare_compiler(original,corrupt,[str(self.root)]*2)
        wrong=copy.deepcopy(cache);artifact=wrong['files']['scripts/Artifacts.s.sol']['artifacts']['Artifacts']
        artifact['0.8.29']=artifact.pop('0.8.28')
        mismatch=dict(renamed);mismatch[cache_name]=json.dumps(wrong).encode()
        with self.assertRaises(ValueError):C.compare_compiler(original,mismatch,[str(self.root)]*2)
        duplicate=copy.deepcopy(cache)
        duplicate['files']['scripts/Artifacts.s.sol']['artifacts']['OtherContract']={'0.8.28':{'default':binding}}
        mismatch=dict(renamed);mismatch[cache_name]=json.dumps(duplicate).encode()
        with self.assertRaises(ValueError):C.compare_compiler(original,mismatch,[str(self.root)]*2)
        # The full hosted build leaves a qualified old summary beside its newly
        # compiled payload under the same source/compiler/profile identity.
        # Model those filenames with actual pre/post-generation Forge outputs.
        old_cache=json.loads(original[cache_name])
        source='test/kontrol/proofs/utils/DeploymentSummary.sol';name='DeploymentSummary'
        versions=old_cache['files'][source]['artifacts'][name]
        version=next(iter(versions));old_binding=versions[version]['default']
        old_path='packages/contracts-bedrock/forge-artifacts/'+old_binding['path']
        qualified='DeploymentSummary.sol/DeploymentSummary.'+version+'.json'
        retained_path='packages/contracts-bedrock/forge-artifacts/'+qualified
        prior=dict(original);prior[retained_path]=prior.pop(old_path);old_binding['path']=qualified
        prior[cache_name]=json.dumps(old_cache).encode()
        rebuilt=C.compiler(pair/'rwx/proofs-compiler')
        self.assertNotEqual(rebuilt[old_path],prior[retained_path])
        rebuilt[retained_path]=prior[retained_path]
        historical=C.compare_compiler(rebuilt,rebuilt,[str(self.root)]*2,([prior],[prior]))
        self.assertEqual(historical['contract_artifacts'],len(C.compiler_bindings(rebuilt,[prior])['bindings'])+1)
        self.assertEqual(historical['verified_retained_previous_artifacts']['rwx'][0]['bound_phase'],0)
        continued=C.compare_compiler(rebuilt,rebuilt,[str(self.root)]*2,([prior,rebuilt],[prior,rebuilt]))
        self.assertEqual(continued['verified_retained_previous_artifacts']['rwx'][0]['bound_phase'],0)
        bad=dict(rebuilt);bad[retained_path]=rebuilt[old_path]
        with self.assertRaisesRegex(ValueError,'retained'):
            C.compare_compiler(rebuilt,bad,[str(self.root)]*2,([prior],[prior]))
        bad=dict(rebuilt);bad.pop(retained_path)
        with self.assertRaisesRegex(ValueError,'retained compiler inventories'):
            C.compare_compiler(rebuilt,bad,[str(self.root)]*2,([prior],[prior]))
        bad=dict(rebuilt);bad[retained_path.removesuffix('.json')+'.extra.json']=bad[retained_path]
        with self.assertRaisesRegex(ValueError,'retained'):
            C.compare_compiler(rebuilt,bad,[str(self.root)]*2,([prior],[prior]))
        # Both supported Docker stores identify exactly the same pinned image.
        path=pair/'rwx/runtime-image.json';original=path.read_bytes()
        final_path=pair/'rwx/final.json';original_final=final_path.read_bytes()
        image=json.loads(original)
        selected=K.IMAGE.selection()
        for identity in (selected['config_digest'],selected['image'].split('@',1)[1], 'sha256:'+'f'*64):
            image[0]['Id']=identity;K.S.write(path,image)
            final=K.G.read(final_path);final['original_sha256']['runtime-image.json']=K.S.digest(path);K.S.write(final_path,final)
            if identity.endswith('f'*64):
                with self.assertRaises(ValueError):C.compare(pair,self.sha)
            else:
                self.assertTrue(C.compare(pair,self.sha)['verified_parity'])
        path.write_bytes(original);final_path.write_bytes(original_final)
        for name in ('settings.json','selection.json','coverage.json','proofs.stage.json','dependencies/contracts-kontrol/metadata.json'):
            path=pair/'rwx'/name;original=path.read_bytes();final_path=pair/'rwx/final.json';original_final=final_path.read_bytes()
            value=K.G.read(path)
            if name=='settings.json':value['source_sha']='f'*40
            elif name=='selection.json':value['proof_sources'].pop(K.PROOFS+'/FutureProof.sol')
            elif name=='coverage.json':value['variants']=['default']
            elif name=='proofs.stage.json':value['argv']=['true']
            else:value['settings']={'profile':'default'}
            K.S.write(path,value);final=K.G.read(final_path);final['original_sha256'][name]=K.S.digest(path);K.S.write(final_path,final)
            with self.subTest(name=name),self.assertRaises(ValueError):C.compare(pair,self.sha)
            path.write_bytes(original);final_path.write_bytes(original_final)
        path=pair/'rwx/variants/fault-proofs/generated/files'/K.GENERATED['fault-proofs'][0]
        original=path.read_bytes();path.write_bytes(original+b'// actual changed generated file\n')
        with self.assertRaises(ValueError):C.compare(pair,self.sha)

    def test_real_kontrol_failure_keeps_original_state_and_stops_before_other_variant(self):
        path=self.contracts/'scripts/deploy/Deploy.s.sol';path.write_text(path.read_text().replace('StateDiff.encodeAccountAccesses(accesses)',"'{\"broken\":true}'"))
        self.commit();result=self.execute();self.assertNotEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertNotEqual(K.G.read(self.report/'summaries.stage.json')['exit_code'],0)
        self.assertTrue((self.report/'variants/default/load-inputs/manifest.json').is_file())
        self.assertFalse((self.report/'variants/fault-proofs').exists());self.assertFalse((self.report/'proofs.stage.json').exists())
        self.assertTrue(K.G.read(self.report/'final-compiler/files.json'));self.retain('original-kontrol-failure')

    def test_real_proof_compiler_failure_keeps_both_original_generated_summaries(self):
        path=self.contracts/'justfile';text=path.read_text().replace('  forge build {{ARGS}}',
            '  printf "invalid original proof compiler input" >test/kontrol/proofs/UntrackedBad.sol\n  forge build {{ARGS}}')
        path.write_text(text);self.commit();result=self.execute();self.assertNotEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertEqual(K.G.read(self.report/'summaries.stage.json')['exit_code'],0)
        self.assertNotEqual(K.G.read(self.report/'proofs.stage.json')['exit_code'],0)
        for variant in ('default','fault-proofs'):self.assertTrue((self.report/'variants'/variant/'generated/manifest.json').is_file())
        self.assertFalse((self.report/'coverage.json').exists());self.retain('original-proof-compiler-failure')

    def test_actual_just_cancellation_retains_first_summary_and_signal(self):
        path=self.contracts/'justfile';text=path.read_text().replace('kontrol-summary-fp:\n',
            'kontrol-summary-fp:\n  touch ../../.ci/cancel-ready\n  sleep 30\n');path.write_text(text);self.commit()
        process=subprocess.Popen([sys.executable,'ops/ci/kontrol-build.py','--provider','circleci'],cwd=self.root,env=self.env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
        try:
            deadline=time.monotonic()+120
            while not (self.root/'.ci/cancel-ready').exists() and process.poll() is None and time.monotonic()<deadline:time.sleep(.05)
            self.assertTrue((self.root/'.ci/cancel-ready').exists());process.send_signal(signal.SIGTERM)
            output=process.communicate(timeout=20)[0].decode();self.assertNotEqual(process.returncode,0,output)
            self.assertIn(K.G.read(self.report/'summaries.stage.json')['exit_code'],(-signal.SIGTERM,128+signal.SIGTERM))
            self.assertTrue((self.report/'variants/default/generated/manifest.json').is_file())
            self.assertFalse((self.report/'proofs.stage.json').exists());self.retain('original-cancellation')
        finally:
            if process.poll() is None:process.kill();process.communicate()

    def test_stale_and_corrupt_real_contract_inputs_fail_before_verdicts(self):
        path=self.artifact/'metadata.json';original=path.read_bytes();value=K.G.read(path);value['commit_sha']='0'*40;K.S.write(path,value)
        result=self.execute('rwx');self.assertNotEqual(result.returncode,0);self.assertFalse((self.report/'summaries.stage.json').exists());self.retain('stale-contract-input')
        path.write_bytes(original);archive=self.artifact/'files.tar.gz';archive.write_bytes(archive.read_bytes()+b'corrupt original archive')
        result=self.execute('rwx');self.assertNotEqual(result.returncode,0);self.assertFalse((self.report/'summaries.stage.json').exists());self.retain('corrupt-contract-input')

    def test_corrupt_image_preparation_fails_before_expensive_original_commands(self):
        path=self.image/'inspect.json';path.write_bytes(path.read_bytes()+b'corrupt original image report')
        result=self.execute('rwx');self.assertNotEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertFalse((self.report/'summaries.stage.json').exists());self.retain('corrupt-image-input')


if __name__=='__main__':unittest.main()
