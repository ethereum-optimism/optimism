#!/usr/bin/env python3
"""Exercise unmodified Forge uploads against a private official Sourcify service."""
import argparse
import ctypes
import hashlib
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import ssl
import subprocess
import sys
import tempfile
import threading
import time
import uuid

SOURCIFY_SHA = '9282528d8c210f22287761c26c4232c15f66585d'
SOURCIFY_TAG = 'sourcify-4byte@1.1.16'
REGISTRY_IMAGE = 'optimism-rwx-selector-registry:9282528'
POSTGRES_IMAGE = 'postgres:15-alpine@sha256:3d0f7584ed7d04e27fa050d6683a74746608faf21f202be78460d679cc56461f'
HOST = 'api.4byte.sourcify.dev'
IMPORT = '/signature-database/v1/import'
LOOKUP = '/signature-database/v1/lookup'
SCHEMA = 'services/database/sourcify-database.sql'


def _source_library():
    import importlib.util
    path = Path(__file__).resolve().parents[1] / 'runtime' / 'ci-source.py'
    spec = importlib.util.spec_from_file_location('ci_source', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


SOURCE = _source_library()
check, digest, write, command = SOURCE.check, SOURCE.digest, SOURCE.write, SOURCE.command


def source_inputs(source, revision=SOURCIFY_SHA, allow_gitlinks=False):
    return SOURCE.source_inputs(source, revision, allow_gitlinks)


def build_images(directory):
    """Build unmodified pinned images; never start a writable destination here."""
    directory = Path(directory).resolve()
    check(not directory.exists(), 'Registry image preparation already exists')
    directory.mkdir(parents=True)
    source = directory / 'sourcify'
    stages = []
    def retry(name, argv, cleanup=None):
        for attempt in range(1, 6):
            started = time.time()
            stdout, stderr = directory / f'{name}-{attempt}.log', directory / f'{name}-{attempt}.stderr.log'
            with stdout.open('wb') as out, stderr.open('wb') as err:
                result = subprocess.run(argv, stdout=out, stderr=err)
            stages.append({'name': name, 'attempt': attempt, 'argv': argv, 'started_at': started,
                           'elapsed_seconds': time.time() - started, 'exit_code': result.returncode,
                           'stdout_sha256': digest(stdout), 'stderr_sha256': digest(stderr)})
            write(directory / 'stages.json', stages)
            if result.returncode == 0:
                return
            if cleanup:
                cleanup()
            check(attempt != 5, 'Test registry dependency failed: ' + name)
            time.sleep(2 ** attempt)
    try:
        retry('source', ['git', 'clone', '--depth', '1', '--branch', SOURCIFY_TAG,
                        'https://github.com/argotorg/sourcify.git', str(source)],
              lambda: shutil.rmtree(source, ignore_errors=True))
        check(command(['git', '-C', str(source), 'rev-parse', 'HEAD']) == SOURCIFY_SHA,
              'Registry release tag differs from pinned source')
        inputs = source_inputs(source)
        write(directory / 'source-inputs.json', {'source_sha': SOURCIFY_SHA, 'inputs': inputs})
        retry('registry-image', ['docker', 'build', '-f', str(source / 'services/4byte/Dockerfile'),
                                '-t', REGISTRY_IMAGE, str(source)])
        retry('postgres-image', ['docker', 'pull', POSTGRES_IMAGE])
        images = {image: command(['docker', 'image', 'inspect', '--format', '{{.Id}}', image])
                  for image in (REGISTRY_IMAGE, POSTGRES_IMAGE)}
        check(all(re.fullmatch(r'sha256:[0-9a-f]{64}', value) for value in images.values()),
              'Invalid produced image identity')
        check(source_inputs(source) == inputs, 'Registry build changed its committed inputs')
        write(directory / 'images.json', images)
        write(directory / 'final.json', {'state': 'ready', 'tests': 0, 'uploads': 0, 'exit_code': 0})
    except BaseException as error:
        write(directory / 'final.json', {'state': 'failed', 'tests': 0, 'uploads': 0,
                                        'exit_code': 1, 'error': str(error) or type(error).__name__})
        raise


class Registry:
    """Private containers; the client joins their otherwise unroutable namespace."""
    def __init__(self, source, output, expected_images=None):
        self.source, self.output = Path(source).resolve(), Path(output).resolve()
        self.output.mkdir(parents=True, exist_ok=True)
        self.prefix = 'optimism-selector-shadow-' + uuid.uuid4().hex
        self.pg, self.api = self.prefix + '-pg', self.prefix + '-api'
        self.containers = []
        self.private = tempfile.TemporaryDirectory(prefix='selector-private-')
        self.private_path = Path(self.private.name)
        self.closed = False
        self.expected_images = expected_images

    def docker(self, *args, **kwargs):
        return command(['docker', *args], **kwargs)

    def sql(self, text):
        return self.docker('exec', '-i', self.pg, 'psql', '-X', '-v', 'ON_ERROR_STOP=1',
                           '-U', 'postgres', '-d', 'selector_shadow', '-At', input=text.encode())

    def rows(self):
        text = self.sql("SELECT coalesce(json_agg(row_to_json(t)), '[]'::json) FROM "
                        "(SELECT signature, '0x'||encode(signature_hash_4,'hex') AS hash4, "
                        "'0x'||encode(signature_hash_32,'hex') AS hash32, created_at "
                        "FROM public.signatures ORDER BY signature) t;")
        return json.loads(text)

    def start(self):
        check(command(['git', '-C', str(self.source), 'rev-parse', 'HEAD']) == SOURCIFY_SHA,
              'Foreign signature registry source')
        check((self.source / SCHEMA).read_bytes() == subprocess.check_output(
              ['git', '-C', str(self.source), 'show', SOURCIFY_SHA + ':' + SCHEMA]), 'Changed registry schema')
        write(self.output / 'registry-source.json', {'repository': 'https://github.com/argotorg/sourcify',
              'source_sha': SOURCIFY_SHA, 'schema_sha256': digest(self.source / SCHEMA),
              'dockerfile_sha256': digest(self.source / 'services/4byte/Dockerfile'),
              'lockfile_sha256': digest(self.source / 'package-lock.json'), 'helper_sha256': digest(__file__),
              'inputs': source_inputs(self.source)})
        image_ids = {image: self.docker('image', 'inspect', '--format', '{{.Id}}', image)
                     for image in (REGISTRY_IMAGE, POSTGRES_IMAGE)}
        write(self.output / 'image-ids.json', image_ids)
        check(all(re.fullmatch(r'sha256:[0-9a-f]{64}', value) for value in image_ids.values()),
              'Invalid test registry image identity')
        if self.expected_images is not None:
            check(image_ids == self.expected_images, 'Restored Docker images differ from their producer')
        self.docker('run', '-d', '--name', self.pg, '--network', 'none',
                    '--tmpfs', '/var/lib/postgresql/data:rw,noexec,nosuid,size=512m',
                    '-e', 'POSTGRES_DB=selector_shadow', '-e', 'POSTGRES_HOST_AUTH_METHOD=trust', POSTGRES_IMAGE)
        self.containers.append(self.pg)
        for _ in range(120):
            result = subprocess.run(['docker', 'exec', self.pg, 'pg_isready', '-h', '127.0.0.1',
                                     '-U', 'postgres', '-d', 'selector_shadow'],
                                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if result.returncode == 0:
                break
            time.sleep(.5)
        else:
            raise ValueError('Private database never became ready')
        # Load the committed schema into this fresh private DB. No migration
        # command or repository .env file can select an external destination.
        self.sql((self.source / SCHEMA).read_text())
        self.sql("CREATE ROLE selector_writer LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION; "
                 "GRANT CONNECT ON DATABASE selector_shadow TO selector_writer; "
                 "GRANT USAGE ON SCHEMA public TO selector_writer; "
                 "GRANT SELECT,INSERT ON public.signatures TO selector_writer; "
                 "GRANT SELECT ON public.compiled_contracts_signatures TO selector_writer;")
        check(self.rows() == [], 'Test registry contains restored upload results')
        write(self.output / 'initial-rows.json', [])
        self.docker('run', '-d', '--name', self.api, '--network', 'container:' + self.pg,
                    '--user', '1000:1000', '--read-only', '--cap-drop', 'ALL',
                    '--security-opt', 'no-new-privileges', '--tmpfs', '/tmp:rw,noexec,nosuid,size=64m',
                    '-e', 'PORT=4444', '-e', 'FOURBYTES_POSTGRES_HOST=127.0.0.1',
                    '-e', 'FOURBYTES_POSTGRES_DB=selector_shadow', '-e', 'FOURBYTES_POSTGRES_USER=selector_writer',
                    '-e', 'FOURBYTES_POSTGRES_PASSWORD=', REGISTRY_IMAGE)
        self.containers.append(self.api)
        for _ in range(120):
            result = subprocess.run(['docker', 'exec', self.api, 'node', '-e',
                                    "fetch('http://127.0.0.1:4444/health').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"],
                                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if result.returncode == 0:
                break
            time.sleep(.5)
        else:
            raise ValueError('Official private registry never became ready')
        inspect = json.loads(self.docker('inspect', self.pg, self.api))
        check(inspect[0]['HostConfig']['NetworkMode'] == 'none'
              and inspect[1]['HostConfig']['NetworkMode'].startswith('container:')
              and all(not row['HostConfig']['PortBindings'] for row in inspect), 'Published or routable test registry')
        self.pid = inspect[0]['State']['Pid']
        check(type(self.pid) is int and self.pid > 0, 'Missing private database namespace')
        write(self.output / 'containers.json', inspect)
        # Bind the running service's actual source and generated JavaScript,
        # separately from non-reproducible Docker creation timestamps.
        script = """const fs=require('fs'),path=require('path'),crypto=require('crypto');
        const root='/home/app/services/4byte',out={};
        function walk(dir){for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
        const p=path.join(dir,entry.name);if(entry.isDirectory()){if(entry.name!=='node_modules')walk(p);}
        else if(entry.isFile())out[path.relative(root,p)]=crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
        else throw Error('Unexpected service file type');}}
        walk(root);console.log(JSON.stringify(out));"""
        write(self.output / 'service-content.json', json.loads(self.docker('exec', self.api, 'node', '-e', script)))
        self.certificate()
        return self

    def certificate(self):
        p = self.private_path
        self.hosts = p / 'hosts'
        self.hosts.write_text('127.0.0.1 localhost ' + HOST + '\n::1 localhost\n')
        self.ca, self.cert, self.key, self.bundle = (p / name for name in ('ca.pem', 'server.pem', 'server.key', 'trust.pem'))
        commands = [
            ['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1', '-subj', '/CN=RWX selector test root',
             '-keyout', str(p / 'ca.key'), '-out', str(self.ca), '-addext', 'basicConstraints=critical,CA:TRUE'],
            ['openssl', 'req', '-newkey', 'rsa:2048', '-nodes', '-subj', '/CN=' + HOST,
             '-keyout', str(self.key), '-out', str(p / 'server.csr')],
        ]
        extensions = p / 'extensions'
        extensions.write_text('basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\n'
                              'extendedKeyUsage=serverAuth\nsubjectAltName=DNS:' + HOST + '\n')
        commands += [['openssl', 'x509', '-req', '-in', str(p / 'server.csr'), '-CA', str(self.ca), '-CAkey', str(p / 'ca.key'),
                      '-CAcreateserial', '-days', '1', '-out', str(self.cert), '-extfile', str(extensions)]]
        for argv in commands:
            subprocess.run(argv, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.bundle.write_bytes(Path('/etc/ssl/certs/ca-certificates.crt').read_bytes() + b'\n' + self.ca.read_bytes())
        for name, path in [('ca.pem', self.ca), ('server.pem', self.cert)]:
            (self.output / name).write_bytes(path.read_bytes())

    def run(self, name, argv, cwd, timeout=2400):
        check(not self.closed and name.replace('-', '').isalnum(), 'Invalid or closed private registry attempt')
        output = self.output / name
        output.mkdir()
        # sudo deliberately resets PATH/HOME. Restore only the caller's build
        # environment inside the isolated client; never forward CI credentials.
        environment = {key: os.environ[key] for key in (
            'PATH', 'HOME', 'USER', 'LOGNAME', 'TMPDIR', 'TMP', 'TEMP', 'XDG_CACHE_HOME',
            'XDG_CONFIG_HOME', 'SVM_HOME', 'FOUNDRY_PROFILE', 'FOUNDRY_SOLC', 'FOUNDRY_OFFLINE')
            if key in os.environ}
        environment_file = self.private_path / (name + '-environment.json')
        write(environment_file, environment)
        environment_file.chmod(0o600)
        executable = shutil.which(argv[0])
        check(executable is not None, 'Selector command is unavailable before isolation')
        isolated_argv = [executable, *argv[1:]]
        args = ['sudo', '-n', 'nsenter', '--net=/proc/' + str(self.pid) + '/ns/net',
                'unshare', '--mount', '--pid', '--fork', '--mount-proc', sys.executable, str(Path(__file__).resolve()),
                'child', str(output), str(self.hosts), str(self.bundle), str(self.cert), str(self.key),
                str(os.getuid()), str(os.getgid()), str(Path(cwd).resolve()), str(environment_file), *isolated_argv]
        started, cancelled = time.time(), []
        row = {'argv': argv, 'cwd': str(Path(cwd).resolve()), 'started_at': started,
               'exit_code': None, 'timed_out': False, 'signals': cancelled}
        write(output / 'execution.json', row)
        with (output / 'stdout.log').open('wb') as stdout, (output / 'stderr.log').open('wb') as stderr:
            process = subprocess.Popen(args, stdout=stdout, stderr=stderr, start_new_session=True)
            def stop(signum, _):
                cancelled.append(signum)
                subprocess.run(['sudo', '-n', 'kill', '-' + str(signum), '--', '-' + str(process.pid)],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            previous = {signum: signal.signal(signum, stop) for signum in (signal.SIGTERM, signal.SIGINT)}
            try:
                try:
                    status = process.wait(timeout=timeout)
                except subprocess.TimeoutExpired:
                    row['timed_out'] = True
                    stop(signal.SIGTERM, None)
                    try:
                        process.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        stop(signal.SIGKILL, None)
                        process.wait(timeout=10)
                    status = 124
            finally:
                for signum, handler in previous.items():
                    signal.signal(signum, handler)
        status = status if status >= 0 else 128 - status
        row.update(exit_code=status, elapsed_seconds=time.time() - started,
                   stdout_sha256=digest(output / 'stdout.log'), stderr_sha256=digest(output / 'stderr.log'))
        write(output / 'execution.json', row)
        write(output / 'database-rows.json', self.rows())
        return status

    def close(self):
        if self.closed:
            return
        self.closed = True
        for name in reversed(self.containers):
            with (self.output / ('api.log' if name == self.api else 'postgres.log')).open('wb') as log:
                subprocess.run(['docker', 'logs', name], stdout=log, stderr=subprocess.STDOUT)
            subprocess.run(['docker', 'rm', '-f', '-v', name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.private.cleanup()


def child(output, hosts, bundle, cert, key, uid, gid, cwd, environment_file, argv):
    output = Path(output)
    check(os.geteuid() == 0 and os.getpid() == 1, 'Client did not enter a private PID/mount namespace')
    subprocess.run(['mount', '--make-rprivate', '/'], check=True)
    for source, target in [(hosts, '/etc/hosts'), (bundle, '/etc/ssl/certs/ca-certificates.crt')]:
        subprocess.run(['mount', '--bind', source, target], check=True)
    check([name for _, name in socket.if_nameindex()] == ['lo'], 'Client has an external network interface')
    blocked = []
    for address in ('1.1.1.1', '2606:4700:4700::1111'):
        try:
            connection = socket.create_connection((address, 443), 2)
            connection.close()
            raise ValueError('Client can reach an external network')
        except OSError as error:
            check(error.errno == 101, 'External connection was not blocked by routing')
            blocked.append({'address': address, 'errno': error.errno})
    check(socket.gethostbyname(HOST) == '127.0.0.1', 'Client did not resolve the isolated destination')
    lock, counter = threading.Lock(), [0]

    class Proxy(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def forward(self):
            check(self.headers.get('Host') == HOST and (self.path == IMPORT or self.path.startswith(LOOKUP + '?')),
                  'Unexpected selector destination')
            length = int(self.headers.get('Content-Length', '0'))
            check(0 <= length <= 102400, 'Oversized selector import')
            body = self.rfile.read(length)
            if self.command == 'POST':
                check(self.path == IMPORT, 'Unexpected write destination')
                data = json.loads(body)
                check(set(data) == {'function', 'event'} and all(isinstance(values, list) and len(values) <= 1000
                      and all(isinstance(value, str) for value in values) for values in data.values()), 'Invalid selector import schema')
            backend = http.client.HTTPConnection('127.0.0.1', 4444, timeout=15)
            try:
                backend.request(self.command, self.path, body=body, headers={'Content-Type': 'application/json'})
                response = backend.getresponse()
                raw = response.read()
                with lock:
                    number = counter[0]
                    counter[0] += 1
                    stem = 'http-' + str(number)
                    (output / (stem + '-request.json')).write_bytes(body)
                    (output / (stem + '-response.json')).write_bytes(raw)
                    write(output / (stem + '.json'), {'method': self.command, 'path': self.path, 'host': self.headers['Host'],
                          'status': response.status, 'request_sha256': hashlib.sha256(body).hexdigest(),
                          'response_sha256': hashlib.sha256(raw).hexdigest()})
                self.send_response(response.status)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)
            finally:
                backend.close()

        do_POST = forward
        do_GET = forward

    server = ThreadingHTTPServer(('127.0.0.1', 443), Proxy)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(cert, key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    env = json.loads(Path(environment_file).read_text())
    check(isinstance(env, dict) and all(isinstance(key, str) and isinstance(value, str)
          for key, value in env.items()), 'Invalid isolated build environment')
    # Client and proxy have no privilege to add routes or recover setuid powers.
    check(ctypes.CDLL(None, use_errno=True).prctl(38, 1, 0, 0, 0) == 0, 'Could not prevent new privileges')
    os.setgroups([])
    os.setgid(int(gid))
    os.setuid(int(uid))
    check(os.geteuid() != 0, 'Selector client retained root privilege')
    write(output / 'isolation.json', {'pid': os.getpid(), 'uid': os.getuid(), 'gid': os.getgid(),
          'network_namespace': os.readlink('/proc/self/ns/net'), 'mount_namespace': os.readlink('/proc/self/ns/mnt'),
          'interfaces': socket.if_nameindex(), 'external_blocked': blocked, 'no_new_privileges': True,
          'hosts_sha256': digest('/etc/hosts'), 'trust_sha256': digest('/etc/ssl/certs/ca-certificates.crt'),
          'destination': 'https://' + HOST + IMPORT})
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    env['SSL_CERT_FILE'] = bundle
    try:
        process = subprocess.Popen(argv, cwd=cwd, env=env, start_new_session=True)
        def stop(signum, _):
            try:
                os.killpg(process.pid, signum)
            except ProcessLookupError:
                pass
        previous = {signum: signal.signal(signum, stop) for signum in (signal.SIGTERM, signal.SIGINT)}
        try:
            status = process.wait()
            return status if status >= 0 else 128 - status
        finally:
            for signum, handler in previous.items():
                signal.signal(signum, handler)
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


def fixture(source, output):
    output = Path(output).resolve()
    project = output / 'project'
    (project / 'src').mkdir(parents=True)
    (project / 'foundry.toml').write_text('[profile.default]\nsrc="src"\nsolc_version="0.8.28"\n')
    (project / 'src/Fixture.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.28;\n'
        'contract Fixture { struct Pair { uint256 x; address recipient; }\n'
        'error Problem(uint256 value); event Pong(uint256 value);\n'
        'function ping(uint256 value) external pure returns (uint256) { if (value==0) revert Problem(value); return value; }\n'
        'function transform(Pair calldata pair) external pure returns (uint256) { return pair.x; } }\n')
    signatures = ['ping(uint256)', 'transform((uint256,address))', 'Problem(uint256)', 'Pong(uint256)']
    expected = {name: command(['cast', 'keccak', name]) for name in signatures}
    registry = Registry(source, output / 'registry')
    try:
        registry.start()
        for name in ('first-upload', 'duplicate-upload'):
            check(registry.run(name, ['forge', 'selectors', 'up', '--all'], project) == 0, 'Real Forge upload fixture failed')
            rows = registry.rows()
            check({row['signature']: row['hash32'] for row in rows} == expected, 'Official database lacks complete real Forge signatures')
        registry.sql('REVOKE INSERT ON public.signatures FROM selector_writer;')
        check(registry.run('denied-write', ['forge', 'selectors', 'up', '--all'], project) != 0,
              'Real Forge ignored denied database writes')
        check({row['signature']: row['hash32'] for row in registry.rows()} == expected, 'Denied upload changed test database')
        check(registry.run('cancelled-client', ['sleep', '60'], project, timeout=1) == 124,
              'Private client cancellation was not preserved')
        check({row['signature']: row['hash32'] for row in registry.rows()} == expected, 'Cancelled client changed test database')
        write(output / 'fixture.json', {'state': 'passed', 'coverage_added': 0, 'signatures': expected,
              'real_forge_upload': True, 'real_duplicate_upload': True, 'real_denied_write': True,
              'real_client_timeout': True})
    finally:
        registry.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    images = sub.add_parser('images')
    images.add_argument('directory', type=Path)
    live = sub.add_parser('fixture')
    live.add_argument('source', type=Path)
    live.add_argument('output', type=Path)
    private = sub.add_parser('child')
    for name in ('output', 'hosts', 'bundle', 'cert', 'key', 'uid', 'gid', 'cwd', 'environment_file'):
        private.add_argument(name)
    private.add_argument('argv', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.mode == 'images':
        build_images(args.directory)
    elif args.mode == 'fixture':
        fixture(args.source, args.output)
    else:
        sys.exit(child(args.output, args.hosts, args.bundle, args.cert, args.key, args.uid, args.gid,
                       args.cwd, args.environment_file, args.argv))
