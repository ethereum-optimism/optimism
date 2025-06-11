# Write an Off-Chain Handler

The first step is to write our *off-chain handler*. This handler is responsible for receiving requests from the bundler along with their payloads and returning the appropriate responses.

## Next Example

Let's begin by allowing our handler to receive two numbers. It will perform both addition and subtraction on these numbers. If the result of the subtraction results in an underflow (i.e. if `b > a` in the expression `a-b`), the handler will respond with an underflow error. Create a `Python` file in which to copy all the following code blocks.

First, we'll need to import some `Python` libraries to help with our example:

```python
from web3 import Web3
from eth_abi import abi as ethabi
from offchain_utils import gen_response, parse_req
```

The first two, `Web3` and `eth_abi`, are established `PyPI` libraries. The third library, `offchain_utils`, was developed by the Boba Foundation. You can view the [library](https://github.com/bobanetwork/rundler-hc/blob/boba-develop/hybrid-compute/offchain/offchain_utils.py) on our Github.

Now, we can write our function. Let's start by initializing an `err_code` and a `resp` object with values in case of an exception:

```python
def offchain_addsub2(sk, src_addr, src_nonce, oo_nonce, payload, *args):
    print("  -> offchain_addsub2 handler called with subkey={} src_addr={} src_nonce={} oo_nonce={} payload={} extra_args={}".format(sk,
          src_addr, src_nonce, oo_nonce, payload, args))
    err_code = 1
    resp = Web3.to_bytes(text="unknown error")
```

Next, we'll add a `try` block. Inside of it, let's make use of some of our imported library functions to parse and decode our binary data to an array of two `unit32`s:

```python 
    try:
        req = parse_req(sk, src_addr, src_nonce, oo_nonce, payload)
        dec = ethabi.decode(['uint32', 'uint32'], req['reqBytes'])

    except Exception as e:
        print("DECODE FAILED", e)
```

With this decoded information, we can add and subtract the two numbers, re-encode the information (this time to an array of two `uint256`s), and check for an overflow. The entire `try` block should look like this:

```python
    try:
        req = parse_req(sk, src_addr, src_nonce, oo_nonce, payload)
        dec = ethabi.decode(['uint32', 'uint32'], req['reqBytes'])

        # s for sum, d for difference
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
```

Before we can return these objects, we need to transform them into a specific object. We can do so with the `gen_response()` function from our imported library from the Boba Foundation. Putting the whole thing together, our whole `offchain_addsub2()` looks like this:

``` python
def offchain_addsub2(sk, src_addr, src_nonce, oo_nonce, payload, *args):
    print("  -> offchain_addsub2 handler called with subkey={} src_addr={} src_nonce={} oo_nonce={} payload={} extra_args={}".format(sk,
          src_addr, src_nonce, oo_nonce, payload, args))
    err_code = 1
    resp = Web3.to_bytes(text="unknown error")

    try:
        req = parse_req(sk, src_addr, src_nonce, oo_nonce, payload)
        dec = ethabi.decode(['uint32', 'uint32'], req['reqBytes'])

        # s for sum, d for difference
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

We've now successfully implemented a function which can receive a request from the bundler, perform some calculation with its payload, and return a response. Proceed to the next section to learn more about setting up a server to run this function.
