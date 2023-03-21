---
title: Modifying Predeployed Contracts
lang: en-US
---

::: warning 🚧 OP Stack Hacks are explicitly things that you can do with the OP Stack that are *not* currently intended for production use

OP Stack Hacks are not for the faint of heart. You will not be able to receive significant developer support for OP Stack Hacks — be prepared to get your hands dirty and to work without support.

:::


OP Stack blockchains have a number of [predeployed contracts](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/constants.ts) that provide important functionality. 
Most of those contracts are proxies that can be upgraded using the `proxyAdminOwner` which was configured when the network was initially deployed.

The predeploys are controlled from a predeploy called [`ProxyAdmin`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/ProxyAdmin.sol), whose address is `0x4200000000000000000000000000000000000018`. 
The function to call is [`upgrade(address,address)`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/ProxyAdmin.sol#L211-L229).
The first parameter is the proxy to upgrade, and the second is the address of a new implementation.

For example, the legacy `L1BlockNumber` contract is at `0x420...013`. 
To disable this function, we'll set the implementation to `0x00...00`.
We do this using the [Foundry](https://book.getfoundry.sh/) command `cast`.

1. We'll need several constants.

    - Set these addresses as variables in your terminal.

        ```sh
        L1BLOCKNUM=0x4200000000000000000000000000000000000013
        PROXY_ADMIN=0x4200000000000000000000000000000000000018
        ZERO_ADDR=0x0000000000000000000000000000000000000000
        ```

    - Set `PRIVKEY` to the private key of your ADMIN account.

    - Set `ETH_RPC_URL`. If you're on the computer that runs the blockchain, use this command.

        ```sh
        export ETH_RPC_URL=http://localhost:8545
        ```

1. Verify `L1BlockNumber` works correctly.
   See that when you call the contract you get a block number, and twelve seconds later you get the next one (block time on L1 is twelve seconds).

   ```sh
   cast call $L1BLOCKNUM 'number()' | cast --to-dec
   sleep 12 && cast call $L1BLOCKNUM 'number()' | cast --to-dec
   ```

1. Get the current implementation for the contract.

   ```sh
   L1BLOCKNUM_IMPLEMENTATION=`cast call $L1BLOCKNUM "implementation()" | sed 's/000000000000000000000000//'`
   echo $L1BLOCKNUM_IMPLEMENTATION 
   ```

1. Change the implementation to the zero address   

   ```sh
   cast send --private-key $PRIVKEY $PROXY_ADMIN "upgrade(address,address)" $L1BLOCKNUM $ZERO_ADDR
   ```

1. See that the implementation is address zero, and that calling it fails.

   ```sh
   cast call $L1BLOCKNUM 'implementation()'
   cast call $L1BLOCKNUM 'number()'
   ```

1. Fix the predeploy by returning it to the previous implementation, and verify it works. 


   ```sh
   cast send --private-key $PRIVKEY $PROXY_ADMIN "upgrade(address,address)" $L1BLOCKNUM $L1BLOCKNUM_IMPLEMENTATION
   cast call $L1BLOCKNUM 'number()' | cast --to-dec
   ```