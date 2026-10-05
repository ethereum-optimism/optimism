#!/usr/bin/env python3
"""Pace anonymous fork RPC transport and retain every original attempt."""
from contextlib import contextmanager
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import threading
import time
import urllib.error
import urllib.request


POLICY = {'requests_per_second': 2, 'max_attempts': 5, 'timeout_seconds': 30,
          'retry_http_statuses': [429, 500, 502, 503, 504], 'backoff_seconds': [2, 4, 8, 16]}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write(path, value):
    path.write_text(json.dumps(value, sort_keys=True, indent=2) + '\n')


class Transport:
    def __init__(self, directory, upstream, policy, opener, clock, wait, stopped):
        self.directory, self.upstream, self.policy = directory, upstream, policy
        self.opener, self.clock, self.wait = opener, clock, wait
        self.stopped = stopped
        self.lock = threading.Lock(); self.next_request = 0; self.count = 0

    def forward(self, body):
        # Hold the lock across attempts: retry traffic must obey the same global
        # budget as new traffic, with at most one upstream request in flight.
        with self.lock:
            number = self.count; self.count += 1
            prefix = self.directory / ('request-' + str(number))
            request_path = prefix.with_suffix('.json'); request_path.write_bytes(body)
            write(prefix.with_suffix('.metadata.json'), {'upstream': self.upstream,
                  'request_sha256': digest(request_path)})
            for attempt in range(1, self.policy['max_attempts'] + 1):
                self.wait(max(0, self.next_request - self.clock()))
                started = time.time(); status, raw, error = None, b'', None
                self.next_request = self.clock() + 1 / self.policy['requests_per_second']
                try:
                    if self.stopped.is_set(): raise InterruptedError('RPC transport stopped')
                    request = urllib.request.Request(self.upstream, data=body,
                        headers={'Content-Type': 'application/json', 'User-Agent': 'Go-http-client/1.1'})
                    with self.opener(request, timeout=self.policy['timeout_seconds']) as response:
                        status, raw = response.status, response.read()
                except urllib.error.HTTPError as failure:
                    try: status, raw, error = failure.code, failure.read(), 'HTTP ' + str(failure.code)
                    finally: failure.close()
                except OSError as failure: error = type(failure).__name__
                response_path = self.directory / ('request-' + str(number) + '-attempt-' + str(attempt) + '.json')
                response_path.write_bytes(raw)
                write(response_path.with_suffix('.metadata.json'), {'http_status': status, 'error': error,
                      'started_at': started, 'elapsed_seconds': time.time() - started,
                      'response_sha256': digest(response_path)})
                if self.stopped.is_set(): return 503, raw
                if status is not None and status not in self.policy['retry_http_statuses']:
                    return status, raw
                if attempt == self.policy['max_attempts']:
                    return status or 502, raw
                self.wait(self.policy['backoff_seconds'][attempt - 1])


@contextmanager
def serve(directory, upstream, policy=None, opener=None, clock=None, wait=None):
    directory = Path(directory); directory.mkdir()
    policy = dict(POLICY if policy is None else policy)
    stopped = threading.Event()
    transport = Transport(directory, upstream, policy, opener or urllib.request.urlopen,
                          clock or time.monotonic, wait or stopped.wait, stopped)

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            try:
                size = int(self.headers['Content-Length'])
                if self.path != '/' or not 0 < size <= 16 * 1024 * 1024:
                    self.send_error(400); return
                body = self.rfile.read(size)
                if len(body) != size:
                    self.send_error(400); return
                # Validate encoding without rewriting request bytes or RPC IDs.
                json.loads(body)
                status, response = transport.forward(body)
                self.send_response(status); self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(response))); self.end_headers()
                try: self.wfile.write(response)
                except OSError: pass  # A canceled client still leaves complete upstream originals.
            except (OSError, ValueError, KeyError):
                self.send_error(502)

        def log_message(self, *_): pass

    # server_close waits for active requests so no evidence is sealed while a
    # request is still writing. Forge's process group is reaped before exit.
    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    worker = threading.Thread(target=server.serve_forever, daemon=True); worker.start()
    write(directory / 'settings.json', {'upstream': upstream, 'policy': policy})
    try:
        yield 'http://127.0.0.1:' + str(server.server_port) + '/'
    finally:
        stopped.set()
        server.shutdown(); server.server_close(); worker.join()
        write(directory / 'final.json', {'requests': transport.count,
              'sha256': {str(p.relative_to(directory)): digest(p) for p in sorted(directory.iterdir()) if p.is_file()}})
