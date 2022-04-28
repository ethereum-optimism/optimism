import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import {
  BatchType,
  SequencerBatch,
  calldataCost,
} from '@eth-optimism/core-utils'

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

        // Add extra fields to the resulting json
        // so that the serialization sizes and gas usage can be observed
        const json = batch.toJSON()
        json.sizes = {
          legacy: 0,
          zlib: 0,
        }
        json.gasUsage = {
          legacy: 0,
          zlib: 0,
        }

        // Create a copy of the batch to serialize in
        // the alternative format
        const copy = (SequencerBatch as any).fromHex(tx.data)
        let legacy: Buffer
        let zlib: Buffer
        if (batch.type === BatchType.ZLIB) {
          copy.type = BatchType.LEGACY
          legacy = copy.encode()
          zlib = batch.encode()
        } else {
          copy.type = BatchType.ZLIB
          zlib = copy.encode()
          legacy = batch.encode()
        }

        json.sizes.legacy = legacy.length
        json.sizes.zlib = zlib.length

        json.sizes.compressionRatio = json.sizes.zlib / json.sizes.legacy

        json.gasUsage.legacy = calldataCost(legacy).toNumber()
        json.gasUsage.zlib = calldataCost(zlib).toNumber()
        json.gasUsage.compressionRatio =
          json.gasUsage.zlib / json.gasUsage.legacy

        batches.push(json)
      }
    }

    console.log(JSON.stringify(batches, null, 2))
  })
