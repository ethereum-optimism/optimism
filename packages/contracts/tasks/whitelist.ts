'use strict'

import fs from 'fs'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { LedgerSigner } from '@ethersproject/hardware-wallets'

import { getContractFactory } from '../src/contract-defs'
import { predeploys } from '../src/predeploys'

// Add accounts the the OVM_DeployerWhitelist
// npx hardhat whitelist --address 0x..
task('whitelist')
  .addOptionalParam('address', 'Address to whitelist', undefined, types.string)
  .addOptionalParam(
    'addressFile',
    'File containing addresses to whitelist separated by a newline',
    undefined,
    types.string
  )
  .addOptionalParam(
    'whitelistMode',
    '"enable" if you want to add the address(es) from the whitelist, "disable" if you want remove the address(es) from the whitelist',
    'enable',
    types.string
  )
  .addOptionalParam('transactionGasPrice', 'tx.gasPrice', undefined, types.int)
  .addOptionalParam(
    'useLedger',
    'use a ledger for signing',
    false,
    types.boolean
  )
  .addOptionalParam(
    'ledgerPath',
    'ledger key derivation path',
    ethers.utils.defaultPath,
    types.string
  )
  .addOptionalParam(
    'contractsRpcUrl',
    'Sequencer HTTP Endpoint',
    process.env.CONTRACTS_RPC_URL,
    types.string
  )
  .addOptionalParam(
    'contractsDeployerKey',
    'Private Key',
    process.env.CONTRACTS_DEPLOYER_KEY,
    types.string
  )
  .addOptionalParam(
    'contractAddress',
    'Address of DeployerWhitelist contract',
    predeploys.OVM_DeployerWhitelist,
    types.string
  )
  .setAction(async (args) => {
    if (args.whitelistMode !== 'enable' && args.whitelistMode !== 'disable') {
      throw new Error(`Whitelist mode must be either "enable" or "disable"`)
    }

    if (args.address === undefined && args.addressPath === undefined) {
      throw new Error(`Must provide either address or address-path`)
    }

    if (args.address !== undefined && args.addressPath !== undefined) {
      throw new Error(`Cannot provide both address and address-path`)
    }

    const provider = new ethers.providers.JsonRpcProvider(args.contractsRpcUrl)
    let signer: ethers.Signer
    if (!args.useLedger) {
      if (!args.contractsDeployerKey) {
        throw new Error('Must pass --contracts-deployer-key')
      }
      signer = new ethers.Wallet(args.contractsDeployerKey).connect(provider)
    } else {
      signer = new LedgerSigner(provider, 'default', args.ledgerPath)
    }

    const deployerWhitelist = getContractFactory('OVM_DeployerWhitelist')
      .connect(signer)
      .attach(args.contractAddress)

    const addr = await signer.getAddress()
    console.log(`Using signer: ${addr}`)
    const owner = await deployerWhitelist.owner()
    if (owner === '0x0000000000000000000000000000000000000000') {
      console.log(`Whitelist is disabled. Exiting early.`)
      return
    } else {
      console.log(`OVM_DeployerWhitelist owner: ${owner}`)
    }

    if (addr !== owner) {
      throw new Error(`Incorrect key. Owner ${owner}, Signer ${addr}`)
    }

    const addresses = []
    if (args.address !== undefined) {
      addresses.push(args.address)
    } else {
      const addressFile = fs.readFileSync(args.addressPath, 'utf8')
      for (const line of addressFile.split('\n')) {
        if (line !== '') {
          addresses.push(line)
        }
      }
    }

    for (const address of addresses) {
      console.log(`Changing whitelist status for address: ${address}`)
      console.log(`New whitelist status: ${args.whitelistMode}`)
      const res = await deployerWhitelist.setWhitelistedDeployer(
        address,
        args.whitelistMode === 'enable' ? true : false,
        { gasPrice: args.transactionGasPrice }
      )
      await res.wait()
    }
  })
