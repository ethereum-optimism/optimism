# Write the Offchain Handler

The first step is to write our *offchain handler*. This handler is responsible for receiving requests from the bundler along with their payloads and returning the appropriate responses.

## Simple Example

Let's begin with a simple example where the handler receives two numbers. It will perform both addition and subtraction on these numbers. If the result of the subtraction results in an overflow (i.e. the first number is greater than the second), the handler will respond with an underflow error.

```python
from web3 import Web3
from eth_abi import abi as ethabi
from offchain_utils import gen_response, parse_req


def offchain_addsub2(sk, src_addr, src_nonce, oo_nonce, payload, *args):
    print("  -> offchain_addsub2 handler called with subkey={} src_addr={} src_nonce={} oo_nonce={} payload={} extra_args={}".format(sk,
          src_addr, src_nonce, oo_nonce, payload, args))
    err_code = 1
    resp = Web3.to_bytes(text="unknown error")

    try:
        req = parse_req(sk, src_addr, src_nonce, oo_nonce, payload)
        dec = ethabi.decode(['uint32', 'uint32'], req['reqBytes'])

        if dec[0] >= dec[1]:
            s = dec[0] + dec[1]
            d = dec[0] - dec[1]
            resp = ethabi.encode(['uint256', 'uint256'], [s, d])
            err_code = 0
        else:
            print("offchain_addsub2 underflow error", dec[0], dec[1])
            resp = Web3.to_bytes(text="underflow error")
    except Exception as e:
        print("DECODE FAILED", e)

    return gen_response(req, err_code, resp)
```

First, we initialize an `err_code` and a `resp` object with values in case of an exception:

``` python
err_code = 1
resp = Web3.to_bytes(text="unknown error")
```

In the `try-block` we can parse the request with the help of this function:

```python
def parse_req(sk, src_addr, src_nonce, oo_nonce, payload):
    req = dict()
    req['skey'] = Web3.to_bytes(hexstr=sk)
    req['srcAddr'] = Web3.to_checksum_address(src_addr)
    req['srcNonce'] = Web3.to_int(hexstr=src_nonce)
    req['opNonce'] = Web3.to_int(hexstr=oo_nonce)
    req['reqBytes'] = Web3.to_bytes(hexstr=payload)
    return req
```

We can also decode the `reqBytes` to an array of `[uin32, uint32]` since we want to receive two numbers.

``` python
dec = ethabi.decode(['uint32', 'uint32'], req['reqBytes'])
```

Now we can perform our custom logic on the received values:

```python
if dec[0] >= dec[1]:
```

Assuming a successful calculation, we overwrite the previously created `err_code` and `resp` variables with successful values:

```python
resp = ethabi.encode(['uint256', 'uint256'], [s, d])
err_code = 0
```

Note that, similarly to passing in two `uint32`s to the `decode()` function, we let the `encode()` function know to decode two `uint256`s.

In case of an underflow error, we encode a string containing a text to let the user know what happened:

```python
resp = Web3.to_bytes(text="underflow error")
```

Before we can return these objects, we need to transform them into a specific object:

``` python
def gen_response(req, err_code, resp_payload):
    resp2 = ethabi.encode(['address', 'uint256', 'uint32', 'bytes'], [req['srcAddr'], req['srcNonce'], err_code, resp_payload])

    enc1 = ethabi.encode(['bytes32', 'bytes'], [req['skey'], resp2])

    p_enc1 = "0x" + selector("PutResponse(bytes32,bytes)") + \
        Web3.to_hex(enc1)[2:]  # dfc98ae8

    enc2 = ethabi.encode(['address', 'uint256', 'bytes'], [
                         Web3.to_checksum_address(HelperAddr), 0, Web3.to_bytes(hexstr=p_enc1)])
    p_enc2 = "0x" + selector("execute(address,uint256,bytes)") + \
        Web3.to_hex(enc2)[2:]  # b61d27f6

    limits = {
        'verificationGasLimit': "0x10000",
        'preVerificationGas': "0x10000",
    }
    callGas = 705*len(resp_payload) + 170000

    print("callGas calculation", len(resp_payload), 4+len(enc2), callGas)

    p = ethabi.encode([
        'address',
        'uint256',
        'bytes32',
        'bytes32',
        'uint256',
        'uint256',
        'uint256',
        'uint256',
        'uint256',
        'bytes32',
    ], [
        HybridAcctAddr,
        req['opNonce'],
        Web3.keccak(Web3.to_bytes(hexstr='0x')),  # initCode
        Web3.keccak(Web3.to_bytes(hexstr=p_enc2)),
        callGas,
        Web3.to_int(hexstr=limits['verificationGasLimit']),
        Web3.to_int(hexstr=limits['preVerificationGas']),
        0,  # maxFeePerGas
        0,  # maxPriorityFeePerGas
        Web3.keccak(Web3.to_bytes(hexstr='0x')),  # paymasterANdData
    ])

    ooHash = Web3.keccak(ethabi.encode(['bytes32', 'address', 'uint256'], [
                         Web3.keccak(p), EntryPointAddr, HC_CHAIN]))

    signAcct = eth_account.account.Account.from_key(hc1_key)
    eMsg = eth_account.messages.encode_defunct(ooHash)
    sig = signAcct.sign_message(eMsg)

    success = (err_code == 0)
    print("Method returning success={} response={} signature={}".format(
        success, Web3.to_hex(resp_payload), Web3.to_hex(sig.signature)))
    return ({
        "success": success,
        "response": Web3.to_hex(resp_payload),
        "signature": Web3.to_hex(sig.signature)
    })
```


We no have successfuly implemented a function which can receive a request from the bundler, perform some calculation with its payload, and return a response.
