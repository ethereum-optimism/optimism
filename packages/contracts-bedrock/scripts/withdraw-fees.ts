'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { LedgerSigner } from '@ethersproject/hardware-wallets'

import {
  constructFeeVaultMulticalls,
  estimateMulticall,
  executeMulticalls,
  multicall3Contract,
} from '../src/multicall3'
import { withdrawFeeVault } from '../src/fee-vault'

// Predeployed fee vaults
// See: https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/libraries/Predeploys.sol
const PREDEPLOY_FEE_VAULTS = [
  // Base Fee Vault
  '0x4200000000000000000000000000000000000019',
  // L1 Fee Vault
  '0x420000000000000000000000000000000000001A',
  // Sequencer Fee Vault
  '0x4200000000000000000000000000000000000011',
]

// Withdraw fees to l1
task('withdraw-fees')
  .addOptionalParam('dryRun', 'simulate withdrawing fees', false, types.boolean)
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
    'privateKey',
    'Private Key',
    process.env.CONTRACTS_DEPLOYER_KEY,
    types.string
  )
  .setAction(async (args) => {
    // Set up the provider
    const provider = new ethers.providers.JsonRpcProvider(args.contractsRpcUrl)

    // Create the tx signer
    let signer: ethers.Signer
    if (!args.useLedger) {
      if (!args.contractsDeployerKey) {
        throw new Error('Must pass --contracts-deployer-key')
      }
      signer = new ethers.Wallet(args.contractsDeployerKey).connect(provider)
    } else {
      signer = new LedgerSigner(provider, 'default', args.ledgerPath)
    }
    if (args.dryRun) {
      console.log('Performing dry run of fee withdrawal...')
    }

    // Create the multicall3 contract
    const multicall3 = multicall3Contract(signer)

    // For each fee vault, check if we should add to the multicall
    let withdrawable = []
    for (const address of PREDEPLOY_FEE_VAULTS) {
      const shouldWithdraw = await withdrawFeeVault(address, signer, provider)
      if (shouldWithdraw) {
        console.log(`[${address}] added to multicall3 call`)
        withdrawable = [...withdrawable, address]
      }
    }
    const calls = constructFeeVaultMulticalls(withdrawable, 'withdraw')
    console.log(`Constructed ${calls.length} multicall3 calls.`)

    // Get Signer Metadata
    const signerAddress = await signer.getAddress()
    const signerBalance = await provider.getBalance(signerAddress)
    const signerBalanceInETH = ethers.utils.formatEther(signerBalance)
    console.log(
      `Using L2 signer ${signerAddress} with a balance of ${signerBalanceInETH} ETH`
    )

    // Execute the multicall3
    if (args.dryRun) {
      await estimateMulticall(multicall3, calls)
    } else {
      await executeMulticalls(multicall3, calls)
    }
  })
