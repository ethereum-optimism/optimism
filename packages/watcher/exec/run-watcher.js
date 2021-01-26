#!/usr/bin/env node

const { Watcher } = require("../build/src")
const {
  providers: { JsonRpcProvider },
} = require('ethers');

const env = process.env
const L1_WEB3_URL = env.L1_WEB3_URL || 'http://localhost:9545'
const L2_WEB3_URL = env.L2_WEB3_URL || 'http://localhost:8545'

// Note: be sure to use the proxy for the L1xdomain
// messenger, not the implementation
const L1CrossDomainMessenger = env.L1CrossDomainMessenger
  || '0xfBE93ba0a2Df92A8e8D40cE00acCF9248a6Fc812'
const L2CrossDomainMessenger = env.L2CrossDomainMessenger
  || '0x4200000000000000000000000000000000000007'

const l1Provider = new JsonRpcProvider(L1_WEB3_URL)
const l2Provider = new JsonRpcProvider(L2_WEB3_URL)

const L2_TX_HASH = env.L2_TX_HASH

if (!L2_TX_HASH) {
  throw new Error('Must pass L2_TX_HASH')
}

;(async ()=> {
  const watcher = new Watcher({
    l1: {
      provider: l1Provider,
      messengerAddress: L1CrossDomainMessenger,
    },
    l2: {
      provider: l2Provider,
      messengerAddress: L2CrossDomainMessenger,
    }
  })

  const msgHashes = await watcher.getMessageHashesFromL2Tx(L2_TX_HASH)
  console.log(`Got ${msgHashes.length} messages`)
  for (const hash of msgHashes) {
    console.log(hash)
  }
  if (msgHashes.length > 0) {
    const receipt = await watcher.getL1TransactionReceipt(msgHashes[0])
    console.log(receipt)
  }
  process.exit(0)
})()
