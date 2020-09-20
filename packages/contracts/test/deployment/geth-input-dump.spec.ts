import { expect } from '../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'

import L2ToL1MessagePasser = require('../../artifacts/L2ToL1MessagePasser.json')
import L1MessageSender = require('../../artifacts/L1MessageSender.json')

/* Internal Imports */
import { deployAllContracts } from '../../src'
import {
  RollupDeployConfig,
  factoryToContractName,
} from '../../src/deployment/types'
import { Signer, Transaction } from 'ethers'
import { keccak256 } from '@ethersproject/keccak256'
import {
  GAS_LIMIT,
  DEFAULT_FORCE_INCLUSION_PERIOD_SECONDS,
} from '../test-helpers'

describe.skip('L2Geth Dumper Input Generator', () => {
  let wallet: Signer
  let sequencer: Signer
  let l1ToL2TransactionPasser: Signer
  before(async () => {
    ;[wallet, sequencer, l1ToL2TransactionPasser] = await ethers.getSigners()
  })

  it('write all the (simpilified) transactions to a file to be ingested into L2Geth', async () => {
    const config: RollupDeployConfig = {
      signer: wallet,
      rollupOptions: {
        gasLimit: GAS_LIMIT,
        forceInclusionPeriodSeconds: DEFAULT_FORCE_INCLUSION_PERIOD_SECONDS,
        ownerAddress: await wallet.getAddress(),
        sequencerAddress: await sequencer.getAddress(),
      },
    }

    const resolver = await deployAllContracts(config)

    // Pull all blocks and transactions
    const totalNumberOfBlocks = await wallet.provider.getBlockNumber()
    const transactions = []
    for (let i = 0; i < totalNumberOfBlocks; i++) {
      const blockTransactions = (
        await wallet.provider.getBlockWithTransactions(i)
      ).transactions
      blockTransactions.map((tx) => transactions.push(tx))
    }

    // Declare types and helpers for the GethDumpInput
    interface SimplifiedTx {
      from: string
      to: string
      data: string
    }
    interface GethDumpInput {
      simplifiedTxs: SimplifiedTx[]
      walletAddress: string
      executionManagerAddress: string
      stateManagerAddress: string,
      codeHashes: Object,
    }

    const getSimplifiedTx = (tx: Transaction): SimplifiedTx => {
      return {
        from: tx.from,
        to: tx.to ? tx.to : '0x' + '00'.repeat(20), // use ZERO_ADDRESS for null because that's the logic I wrote in geth
        data: tx.data,
      }
    }
    const simplifiedTxs: SimplifiedTx[] = transactions.map((tx) =>
      getSimplifiedTx(tx)
    )

    const gethDumpInput: GethDumpInput = {
      simplifiedTxs,
      walletAddress: await wallet.getAddress(),
      executionManagerAddress: resolver.contracts.executionManager.address,
      stateManagerAddress: resolver.contracts.stateManager.address,
      codeHashes: {
          l2ToL1MessagePasser: keccak256(L2ToL1MessagePasser.deployedBytecode),
          l1MessageSender: keccak256(L1MessageSender.deployedBytecode)
      }
    }

    for (const [name, contract] of Object.entries(resolver.contracts)) {
        const address = contract.address
        const code = await contract.provider.getCode(address)
        const hash = keccak256(code)
        gethDumpInput.codeHashes[name] = hash
    }

    // Write all the simplified transactions data to a file
    const fs = require('fs')
    const path = require('path')
    const filename = 'deployment-tx-data.json'
    fs.writeFile(filename, JSON.stringify(gethDumpInput), (err) => {
      if (err) {
        console.log(err)
      } else {
        console.log('Wrote deployment tx data to', filename)
      }
    })
  })
})
