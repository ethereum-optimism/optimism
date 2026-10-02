#!/usr/bin/env python3
"""Check archive inputs without logging URLs, credentials or provider bodies."""
import json
import os
import urllib.parse
import urllib.request

NAMES = ('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL', 'OP_CI_SEPOLIA_L1_ARCHIVE_RPC_URL')

def check():
    for name in NAMES:
        value = os.environ.get(name, '')
        url = urllib.parse.urlparse(value)
        if url.scheme not in ('http', 'https') or not url.netloc:
            raise ValueError('An archive RPC input is unavailable or invalid')
        request = urllib.request.Request(value, headers={'Content-Type': 'application/json'},
            data=json.dumps({'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBalance',
                             'params': ['0x0000000000000000000000000000000000000000', '0x1']}).encode())
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                result = json.load(response)
            if result.get('id') != 1 or not isinstance(result.get('result'), str) or not result['result'].startswith('0x') or result.get('error'):
                raise ValueError('Invalid archive response')
        except Exception:
            # Provider exceptions can contain the credential-bearing URL.
            raise ValueError('An archive RPC input did not answer the historical-state probe') from None
    print('Both archive RPC historical-state probes passed')

if __name__ == '__main__':
    try:
        check()
    except ValueError as error:
        raise SystemExit(str(error))
