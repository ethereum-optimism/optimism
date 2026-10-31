#!/usr/bin/env python3
"""Exercise the Circle cache setup against an offline Go consumer."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[2]


class ModuleCacheTests(unittest.TestCase):
    def test_machine_gopath_uses_restored_docker_cache_in_later_shells(self):
        helpers = json.loads(subprocess.check_output(
            ['yq', '-o=json', '.', str(ROOT / '.circleci/continue/helpers.yml')], text=True))
        steps = helpers['commands']['restore-go-mod-cache']['steps']
        setup = next(step['run']['command'] for step in steps if 'run' in step)
        save = helpers['commands']['save-go-mod-cache']['steps'][0]['save_cache']
        restore = next(step['restore_cache'] for step in steps if 'restore_cache' in step)
        self.assertEqual(save['key'], restore['keys'][0])

        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            home = root / 'home with spaces'
            home.mkdir()
            project = root / 'project'
            project.mkdir()
            proxy = root / 'proxy/example.invalid/dependency/@v'
            proxy.mkdir(parents=True)
            module = 'module example.invalid/dependency\n\ngo 1.22.0\n'
            (proxy / 'v1.0.0.mod').write_text(module)
            (proxy / 'v1.0.0.info').write_text('{"Version":"v1.0.0","Time":"2024-01-01T00:00:00Z"}')
            with zipfile.ZipFile(proxy / 'v1.0.0.zip', 'w') as archive:
                archive.writestr('example.invalid/dependency@v1.0.0/go.mod', module)
                archive.writestr('example.invalid/dependency@v1.0.0/value.go',
                                 'package dependency\nconst Value = 42\n')
            (project / 'go.mod').write_text(
                'module example.invalid/consumer\n\ngo 1.22.0\n\nrequire example.invalid/dependency v1.0.0\n')
            (project / 'consumer.go').write_text(
                'package consumer\nimport "example.invalid/dependency"\nvar Value = dependency.Value\n')
            env = {k: v for k, v in os.environ.items() if k not in ('GOROOT', 'GOMODCACHE', 'GOFLAGS', 'GOWORK')}
            env.update(HOME=str(home), GOPATH=str(home / '.go_workspace'),
                       GOTOOLCHAIN='local', GOSUMDB='off', GOPROXY=(root / 'proxy').as_uri(),
                       GOCACHE=str(root / 'build-cache'), BASH_ENV=str(root / 'bash-env'))
            self.assertEqual(len(save['paths']), 1)
            canonical = home / Path(save['paths'][0]).relative_to('~')
            # Model the producer's cache snapshot before the machine restores it.
            subprocess.run(['go', 'mod', 'download', 'all'], cwd=project,
                           env={**env, 'GOMODCACHE': str(canonical)}, check=True, capture_output=True)
            env['GOPROXY'] = 'off'
            cold = subprocess.run(['go', 'list', '-deps', './...'], cwd=project,
                                  env=env, capture_output=True, text=True)
            self.assertNotEqual(cold.returncode, 0)
            self.assertIn('module lookup disabled', cold.stderr)
            subprocess.run(['bash', '-euc', setup], cwd=project, env=env, check=True, capture_output=True)
            # Circle sources BASH_ENV afresh for each subsequent run step.
            effective = subprocess.check_output(['bash', '-euc', 'go env GOMODCACHE'],
                                                cwd=project, env=env, text=True).strip()
            self.assertEqual(effective, str(canonical))
            nested = project / 'nested'
            nested.mkdir()
            result = subprocess.run(['bash', '-euc', 'go test -mod=readonly ../...'],
                                    cwd=nested, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
