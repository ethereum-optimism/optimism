#!/usr/bin/env python3
"""Check archive inputs without logging URLs, credentials or provider bodies."""
import json
import os
import urllib.parse
import urllib.error
import urllib.request

NAMES = ('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL', 'OP_CI_SEPOLIA_L1_ARCHIVE_RPC_URL')

def check():
    for name in NAMES:
        value = os.environ.get(name, '')
        url = urllib.parse.urlparse(value)
        if url.scheme not in ('http', 'https') or not url.netloc:
            raise ValueError('An archive RPC input is unavailable or invalid')
        request = urllib.request.Request(value, headers={'Content-Type': 'application/json', 'User-Agent': 'Go-http-client/1.1'},
            data=json.dumps({'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBalance',
                             'params': ['0x0000000000000000000000000000000000000000', '0x1']}).encode())
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                result = json.load(response)
            if result.get('id') != 1 or not isinstance(result.get('result'), str) or not result['result'].startswith('0x') or result.get('error'):
                code = result.get('error', {}).get('code')
                reason = 'RPC error ' + str(code) if isinstance(code, int) else 'invalid JSON-RPC response'
                raise ValueError(reason)
        except Exception as error:
            # Report only fixed labels and numeric codes; provider exception text
            # can contain a credential-bearing URL or the response body.
            reason = ('HTTP ' + str(error.code)) if isinstance(error, urllib.error.HTTPError) else (str(error) if isinstance(error, ValueError) and str(error).startswith(('RPC error ', 'invalid JSON-RPC')) else type(error).__name__)
            raise ValueError(name + ' historical-state probe failed: ' + reason) from None
    print('Both archive RPC historical-state probes passed')

if __name__ == '__main__':
    try:
        check()
    except ValueError as error:
        raise SystemExit(str(error))
