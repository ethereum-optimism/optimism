"""Exercise real RPC pacing, original frames and bounded transport failures."""
from concurrent.futures import ThreadPoolExecutor
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
from pathlib import Path
import tempfile
import threading
import time
import unittest
import urllib.error
import urllib.request

SPEC=importlib.util.spec_from_file_location('proxy',Path(__file__).resolve().parents[1] / 'runtime' / 'l2-rpc-proxy.py')
P=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(P)


class RpcProxyTests(unittest.TestCase):
    def setUp(self):
        temporary=tempfile.TemporaryDirectory();self.addCleanup(temporary.cleanup)
        self.directory=Path(temporary.name)/'originals'

    def server(self, replies):
        observed=[]
        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                body=self.rfile.read(int(self.headers['Content-Length']))
                observed.append({'at':time.monotonic(),'body':body,'headers':dict(self.headers)})
                status,response=replies[min(len(observed)-1,len(replies)-1)]
                self.send_response(status);self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
            def log_message(self,*_):pass
        server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
        worker=threading.Thread(target=server.serve_forever,daemon=True);worker.start()
        self.addCleanup(server.server_close);self.addCleanup(worker.join,5);self.addCleanup(server.shutdown)
        return 'http://127.0.0.1:'+str(server.server_port)+'/',observed

    def request(self, endpoint, body):
        request=urllib.request.Request(endpoint,data=body,headers={'Content-Type':'application/json'})
        try:
            with urllib.request.urlopen(request,timeout=10) as response:return response.status,response.read()
        except urllib.error.HTTPError as failure:
            try:return failure.code,failure.read()
            finally:failure.close()

    def test_real_parallel_clients_share_one_proactive_budget_and_original_bytes(self):
        body=b'{ "jsonrpc": "2.0", "id": 41, "method": "eth_chainId", "params": [] }'
        response=b'{"jsonrpc":"2.0","id":41,"result":"0xa"}'
        upstream,observed=self.server([(200,response)])
        with P.serve(self.directory,upstream) as endpoint,ThreadPoolExecutor(max_workers=3) as workers:
            results=list(workers.map(lambda _:self.request(endpoint,body),range(3)))
        self.assertEqual(results,[(200,response)]*3)
        self.assertEqual(len(observed),3)
        self.assertTrue(all(b['at']-a['at']>=0.45 for a,b in zip(observed,observed[1:])))
        for i,row in enumerate(observed):
            self.assertEqual(row['body'],body);self.assertEqual((self.directory/f'request-{i}.json').read_bytes(),body)
            self.assertEqual((self.directory/f'request-{i}-attempt-1.json').read_bytes(),response)
            self.assertFalse(any(k.lower()=='authorization' for k in row['headers']))
        final=json.loads((self.directory/'final.json').read_bytes());self.assertEqual(final['requests'],3)
        self.assertEqual(final['sha256'],{p.name:P.digest(p) for p in self.directory.iterdir() if p.name!='final.json'})

    def test_transient_transport_retries_retain_every_body_and_do_not_rewrite_ids(self):
        response=b'{"jsonrpc":"2.0","id":9,"result":"0xa"}'
        upstream,observed=self.server([(429,b'original capacity denial'),(503,b'original unavailable'),(200,response)])
        with P.serve(self.directory,upstream,wait=lambda _:None,clock=lambda:0) as endpoint:
            self.assertEqual(self.request(endpoint,b'{"jsonrpc":"2.0","id":9,"method":"eth_chainId","params":[]}'),(200,response))
        self.assertEqual(len(observed),3)
        self.assertEqual([(self.directory/f'request-0-attempt-{i}.json').read_bytes() for i in range(1,4)],
                         [b'original capacity denial',b'original unavailable',response])
        self.assertEqual([json.loads((self.directory/f'request-0-attempt-{i}.metadata.json').read_bytes())['http_status'] for i in range(1,4)],[429,503,200])

    def test_denials_and_rpc_execution_errors_are_returned_once_without_retry(self):
        for status,response in ((403,b'original permanent denial'),(200,b'{"jsonrpc":"2.0","id":1,"error":{"code":3,"message":"execution reverted"}}')):
            with self.subTest(status=status):
                directory=self.directory/str(status);self.directory.mkdir(exist_ok=True)
                upstream,observed=self.server([(status,response)])
                with P.serve(directory,upstream) as endpoint:
                    self.assertEqual(self.request(endpoint,b'{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[]}'),(status,response))
                self.assertEqual(len(observed),1)

    def test_capacity_retries_are_bounded_and_the_last_failure_is_returned(self):
        upstream,observed=self.server([(429,b'original exhausted capacity')])
        with P.serve(self.directory,upstream,wait=lambda _:None,clock=lambda:0) as endpoint:
            self.assertEqual(self.request(endpoint,b'{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}'),(429,b'original exhausted capacity'))
        self.assertEqual(len(observed),5)
        self.assertEqual(len(list(self.directory.glob('request-0-attempt-*.metadata.json'))),5)

    def test_cancellation_prevents_queued_requests_from_reaching_upstream(self):
        self.directory.mkdir();stopped=threading.Event();stopped.set()
        def forbidden(*_,**__):self.fail('Canceled request reached upstream')
        transport=P.Transport(self.directory,'http://unreached/',P.POLICY,forbidden,time.monotonic,lambda _:None,stopped)
        self.assertEqual(transport.forward(b'{}'),(503,b''))
        metadata=json.loads((self.directory/'request-0-attempt-1.metadata.json').read_bytes())
        self.assertEqual(metadata['error'],'InterruptedError');self.assertIsNone(metadata['http_status'])


if __name__=='__main__':unittest.main()
