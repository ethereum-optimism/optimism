---
title: Pause and Unpause the Bridge
lang: en-US
---


## Why do it?

The `OptimismPortal` is a bridge contract that makes it possible to send messages between your L1 and your L2 OP Stack chain. 
The `OptimismPortal` is pausable as a backup safety mechanism that allows a specific `GUARDIAN` address to temporarily halt deposits and withdrawals to mitigate security issues if necessary. 
An OP Stack chain does not have to specify a usable `GUARDIAN` address if it does not want to make the `OptimismPortal` contract pausable, it can specify an address such as zero.


## Who can do it?

[`OptimismPortal`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L1/OptimismPortal.sol) has an immutable `GUARDIAN`. 
That address can call [`pause`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L1/OptimismPortal.sol#L166-L170) and [`unpause`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L1/OptimismPortal.sol#L175-L179).


### Changing the guardian

The guardian created by the setup script is the admin account.
This is sufficient for testing, but for a production system you would want the guardian to be a multisig with trusted security council.

The `GUARDIAN` variable is immutable, but the `OptimismPortal` contract sits behind a proxy, so the `GUARDIAN` can be modified by changing the `OptimismPortal` proxy to point to a new implementation contract.
You do this using the L1 [`ProxyAdmin`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/ProxyAdmin.sol) contract.

<!--
## Seeing it in action

1. Set these environment variables

   | Variable | Meaning
   | - | - |
   | `PRIV_KEY` | Private key for your ADMIN account
   | `ADMIN_ADDR` | Address of the ADMIN account
   | `PORTAL_ADDR` | Portal proxy address, get from `.../optimism/packages/contracts-bedrock/deployments/getting-started/OptimismPortalProxy.json`
   | `GOERLI_RPC` | URL for an RPC to the L1 Goerli network 

1.  For using Foundry, set `ETH_RPC_URL`.

    ```sh
    export ETH_RPC_URL=$GOERLI_RPC
    ```

1. Check the balance of the ADMIN account.
   If it is too low you will not be able to submit transactions.

   ```sh
   cast balance $ADMIN_ADDR
   ```

1. Send a deposit to L2.

   ```sh
   cast send --private-key $PRIV_KEY --value 1gwei $PORTAL_ADDR
   ```

   Note the transaction hash.

1. Pause the portal.

   ```sh
   cast send --private-key $PRIV_KEY $PORTAL_ADDR "pause()"
   ```

1. Send a deposit to L2.

   ```sh
   cast send --private-key $PRIV_KEY --value 1gwei $PORTAL_ADDR
   ```

   Note the transaction hash.

1. Wait ten minutes and see which transaction(s) have been relayed using the [SDK](../build/sdk.md). 
   Use [`getMessageStatus`](https://sdk.optimism.io/classes/crosschainmessenger#getMessageStatus) to get the information.



1. Unpause the portal.

   ```sh
   cast send --private-key $PRIV_KEY $PORTAL_ADDR "pause()"
   ```
-->
