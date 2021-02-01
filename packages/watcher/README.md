# @eth-optimism/watcher

#### Watcher
Our `Watcher` allows you to retrieve all transaction hashes related to cross domain messages such as deposits and withdrawals. In order to use, first send a transaction which sends a cross domain message, for example a deposit from L1 into L2. After sending the deposit transaction and storing the transaction hash, use `getMessageHashesFromL1Tx(l1TxHash)` to get an array of the message hashes of all of the L1->L2 messages that were sent inside of that L1 tx (This will usually just be a single element array, but it can return multiple if one L1 transaction triggers multiple deposits). `getMessageHashesFromL2Tx(l2TxHash)` does the same for L2->L1 messages. `getL2TransactionReceipt(messageHash)` takes in an L1->L2 message hash and then after 2-5 minutes, returns the receipt of the L2 tx that the message ends up getting relayed in. `getL1TransactionReceipt(messageHash)` does the same for L2->L1 messages, except the delay is 7 days.

```typescript
import { Watcher } from '@eth-optimism/watcher'
import { JsonRpcProvider } from 'ethers/providers'

const watcher = new Watcher({
  l1: {
    provider: new JsonRpcProvider('INFURA_L1_URL'),
    messengerAddress: '0x...'
  },
  l2: {
    provider: new JsonRpcProvider('OPTIMISM_L2_URL'),
    messengerAddress: '0x...'
  }
})
const l1TxHash = (await depositContract.deposit(100)).hash
const [messageHash] = await watcher.getMessageHashesFromL1Tx(l1TxHash)
console.log('L1->L2 message hash:', messageHash)
const l2TxReceipt = await watcher.getL2TransactionReceipt(messageHash)
```
