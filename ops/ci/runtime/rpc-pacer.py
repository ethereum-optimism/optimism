#!/usr/bin/env python3
"""Forward JSON-RPC requests to one upstream URL at a bounded rate.

The public OP Mainnet RPC limits requests per IP address. At forge's own rate it answers
HTTP 429 for longer than forge retries, and the fork tests fail in setUp. The pacer starts
at most RATE requests per second. It does not retry: the client receives each upstream
status and body unchanged, and owns the retries.
"""
from contextlib import contextmanager
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import threading
import time
import urllib.error
import urllib.request

# A hosted run made the 420 requests of the fork tests at this rate without one HTTP 429.
RATE = 2
FORWARDED = ('Content-Type', 'User-Agent')


@contextmanager
def serve(upstream, rate=RATE, clock=time.monotonic, wait=time.sleep):
    """Yield the URL of a local endpoint that forwards each POST body to `upstream`."""
    lock = threading.Lock()
    following = [clock()]

    def pace():
        # Each request takes the next start time, so concurrent clients share one budget.
        with lock:
            start = max(following[0], clock())
            following[0] = start + 1 / rate
        wait(max(0, start - clock()))

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            body = self.rfile.read(int(self.headers['Content-Length']))
            request = urllib.request.Request(upstream, data=body, headers={
                name: self.headers[name] for name in FORWARDED if self.headers[name]})
            pace()
            try:
                with urllib.request.urlopen(request, timeout=30) as response:
                    status, answer = response.status, response.read()
            except urllib.error.HTTPError as failure:
                with failure: status, answer = failure.code, failure.read()
            # No answer from the upstream. Forge retries HTTP 503.
            except (OSError, http.client.HTTPException): status, answer = 503, b''
            self.send_response(status)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(answer)))
            self.end_headers()
            self.wfile.write(answer)

        def log_message(self, *_): pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    try: yield 'http://127.0.0.1:' + str(server.server_port)
    finally:
        server.shutdown(); server.server_close(); worker.join()
