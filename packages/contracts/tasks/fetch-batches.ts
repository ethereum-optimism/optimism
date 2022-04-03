import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { BatchType, SequencerBatch } from '@eth-optimism/core-utils'

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

        // Add an extra field to the resulting json
        // so that the serialization sizes can be observed
        const json = batch.toJSON()
        json.sizes = {
          legacy: 0,
          zlib: 0,
        }

        // Create a copy of the batch to serialize in
        // the alternative format
        const copy = (SequencerBatch as any).fromHex(tx.data)
        if (batch.type === BatchType.ZLIB) {
          copy.type = BatchType.LEGACY
          json.sizes.legacy = copy.encode().length
          json.sizes.zlib = batch.encode().length
        } else {
          copy.type = BatchType.ZLIB
          json.sizes.zlib = copy.encode().length
          json.sizes.legacy = batch.encode().length
        }

        json.compressionRatio = json.sizes.zlib / json.sizes.legacy

        batches.push(json)
      }
    }

    console.log(JSON.stringify(batches, null, 2))
  })
