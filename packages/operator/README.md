# @pigi/operator
Deploy your own plasma chain!

## Prerequisites 
- `node.js` -- version 11.6.0
- `git` -- version 2.17.1
- Build essentials or similar (specifically `g++` -- version 7.3.1)

## Setup
To deploy a new Plasma Chain, use the following commands:
```
0) $ npm install plasma-chain [-g]  # install the plasma chain operator. Global flag is optional, if you don't
use global, just replace all of the following commands with `npm run plamsa-chain [command]`. If you can't install
globally without `sudo` then just use local!

1) $ plasma-chain account new  # create a new account

2) # On Rinkeby testnet, send your new Operator address ~0.5 ETH.
You can use a faucet to get test ETH for free here: https://faucet.rinkeby.io/

2.5) $ plasma-chain list # list all the plasma chains which others have deployed to the Plasma Network Registry 

3) $ plasma-chain deploy # deploy a new Plasma Chain.
Note you will be prompted for a unique Plasma Chain name & IP address.
If you are running on your laptop, just set the IP to `0.0.0.0` as you probably don't
want to reveal your IP to the public. However, if you are running in a data center and would
like to accept Plasma transactions & serve a block explorer to the public, go ahead and set an IP.

4) $ plasma-chain start # start your new Plasma Chain
You can also view your local block explorer at http:127.0.0.1:8000

5) [optional]
Open a new terminal. In this new terminal use the following command:
$ plasma-chain testSwarm # spam your Plasma Chain with tons of test transactions 😁

```

### To open your port, you may need to forward traffic from port 80 to port 3000
See: https://coderwall.com/p/plejka/forward-port-80-to-port-3000
