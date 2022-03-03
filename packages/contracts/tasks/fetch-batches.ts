import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { SequencerBatch } from '@eth-optimism/core-utils'

import { names } from '../src/address-names'
import { getContractFromArtifact } from '../src/deploy-utils'

// Need to export env vars
// CONTRACTS_TARGET_NETWORK
// CONTRACTS_DEPLOYER_KEY
// CONTRACTS_RPC_URL
task('fetch-batches')
  .addOptionalParam(
    'contractsRpcUrl',
    'Ethereum HTTP Endpoint',
    process.env.CONTRACTS_RPC_URL || 'http://127.0.0.1:8545',
    types.string
  )
  .addOptionalParam('start', 'Start block height', 0, types.int)
  .addOptionalParam('end', 'End block height', undefined, types.int)
  .setAction(async (args, hre) => {
    const provider = new ethers.providers.StaticJsonRpcProvider(
      args.contractsRpcUrl
    )

    let CanonicalTransactionChain = await getContractFromArtifact(
      hre,
      names.managed.contracts.CanonicalTransactionChain
    )
    CanonicalTransactionChain = CanonicalTransactionChain.connect(provider)

    const start = args.start
    let end = args.end
    if (!end) {
      end = await provider.getBlockNumber()
    }

    const batches = []

    for (let i = start; i <= end; i += 2001) {
      const tip = Math.min(i + 2000, end)
      console.error(`Querying events ${i}-${tip}`)

      const events = await CanonicalTransactionChain.queryFilter(
        CanonicalTransactionChain.filters.SequencerBatchAppended(),
        i,
        tip
      )

      for (const event of events) {
        const tx = await provider.getTransaction(event.transactionHash)
        const batch = (SequencerBatch as any).fromHex(tx.data)
        batches.push(batch.toJSON())
      }
    }

    console.log(JSON.stringify(batches, null, 2))
  })
