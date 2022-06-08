import { task, types } from 'hardhat/config'
import { ethers } from 'ethers'
import { LedgerSigner } from '@ethersproject/hardware-wallets'
import dotenv from 'dotenv'

import { prompt } from '../src/prompt'
dotenv.config()

// Hardcode the expected addresse
const addresses = {
  governanceToken: '0x4200000000000000000000000000000000000042',
}

task('deploy-token', 'Deploy governance token and its mint manager contracts')
  .addParam('mintManagerOwner', 'Owner of the mint manager')
  .addOptionalParam('useLedger', 'User ledger hardware wallet as signer')
  .addOptionalParam(
    'ledgerTokenDeployerPath',
    'Ledger key derivation path for the token deployer account',
    ethers.utils.defaultPath,
    types.string
  )
  .addParam(
    'pkDeployer',
    'Private key for main deployer account',
    process.env.PRIVATE_KEY_DEPLOYER
  )
  .addOptionalParam(
    'pkTokenDeployer',
    'Private key for the token deployer account',
    process.env.PRIVATE_KEY_TOKEN_DEPLOYER
  )
  .setAction(async (args, hre) => {
    console.log('Deploying token to', hre.network.name, 'network')
    // There cannot be two ledgers at the same time
    let tokenDeployer
    // Deploy the token
    if (args.useLedger) {
      // Token is deployed to a system address at `0x4200000000000000000000000000000000000042`
      // For that a dedicated deployer account is used
      tokenDeployer = new LedgerSigner(
        hre.ethers.provider,
        'default',
        args.ledgerTokenDeployerPath
      )
    } else {
      tokenDeployer = new hre.ethers.Wallet(args.pkTokenDeployer).connect(
        hre.ethers.provider
      )
    }

    // Create the MintManager Deployer
    const deployer = new hre.ethers.Wallet(args.pkDeployer).connect(
      hre.ethers.provider
    )

    // Get the sizes of the bytecode to check if the contracts
    // have already been deployed. Useful for an error partway through
    // the script
    const governanceTokenCode = await hre.ethers.provider.getCode(
      addresses.governanceToken
    )

    const addrTokenDeployer = await tokenDeployer.getAddress()
    console.log(`Using token deployer: ${addrTokenDeployer}`)

    const tokenDeployerBalance = await tokenDeployer.getBalance()
    if (tokenDeployerBalance.eq(0)) {
      throw new Error(`Token deployer has no balance`)
    }
    console.log(`Token deployer balance: ${tokenDeployerBalance.toString()}`)
    const nonceTokenDeployer = await tokenDeployer.getTransactionCount()
    console.log(`Token deployer nonce: ${nonceTokenDeployer}`)

    const GovernanceToken = await hre.ethers.getContractFactory(
      'GovernanceToken'
    )
    let governanceToken = GovernanceToken.attach(
      addresses.governanceToken
    ).connect(tokenDeployer)

    if (nonceTokenDeployer === 0 && governanceTokenCode === '0x') {
      await prompt('Ready to deploy. Does everything look OK?')
      // Deploy the GovernanceToken
      governanceToken = await GovernanceToken.connect(tokenDeployer).deploy()
      const tokenReceipt = await governanceToken.deployTransaction.wait()
      console.log('GovernanceToken deployed to:', tokenReceipt.contractAddress)

      if (tokenReceipt.contractAddress !== addresses.governanceToken) {
        console.log(
          `Expected governance token address ${addresses.governanceToken}`
        )
        console.log(`Got ${tokenReceipt.contractAddress}`)
        throw new Error(`Fatal error! Mismatch of governance token address`)
      }
    } else {
      console.log(
        `GovernanceToken already deployed at ${addresses.governanceToken}, skipping`
      )
      console.log(`Deployer nonce: ${nonceTokenDeployer}`)
      console.log(`Code size: ${governanceTokenCode.length}`)
    }

    const { mintManagerOwner } = args

    // Do the deployer things
    console.log('Deploying MintManager')
    const addr = await deployer.getAddress()
    console.log(`Using MintManager deployer: ${addr}`)

    const deployerBalance = await deployer.getBalance()
    if (deployerBalance.eq(0)) {
      throw new Error('Deployer has no balance')
    }
    console.log(`Deployer balance: ${deployerBalance.toString()}`)
    const deployerNonce = await deployer.getTransactionCount()
    console.log(`Deployer nonce: ${deployerNonce}`)
    await prompt('Does this look OK?')

    const MintManager = await hre.ethers.getContractFactory('MintManager')
    // Deploy the MintManager
    console.log(
      `Deploying MintManager with (${mintManagerOwner}, ${addresses.governanceToken})`
    )
    const mintManager = await MintManager.connect(deployer).deploy(
      mintManagerOwner,
      addresses.governanceToken
    )

    const receipt = await mintManager.deployTransaction.wait()
    console.log(`Deployed mint manager to ${receipt.contractAddress}`)
    let mmOwner = await mintManager.owner()
    const currTokenOwner = await governanceToken
      .attach(addresses.governanceToken)
      .owner()
    console.log(
      'About to transfer ownership of the token to the mint manager! This is irreversible.'
    )
    console.log(`Current token owner:   ${currTokenOwner}`)
    console.log(`Mint manager address:  ${mintManager.address}`)
    console.log(`Mint manager owner:    ${mmOwner}`)
    await prompt('Is this OK?')

    console.log('Transferring ownership...')
    // Transfer ownership of the token to the MintManager instance
    const tx = await governanceToken
      .attach(addresses.governanceToken)
      .transferOwnership(mintManager.address)
    await tx.wait()
    console.log(
      `Transferred ownership of governance token to ${mintManager.address}`
    )

    console.log('MintManager deployed to:', receipt.contractAddress)
    console.log('MintManager owner set to:', mintManagerOwner)
    console.log(
      'MintManager governanceToken set to:',
      addresses.governanceToken
    )
    console.log('### Token deployment complete ###')

    const tokOwner = await governanceToken
      .attach(addresses.governanceToken)
      .owner()
    if (tokOwner !== mintManager.address) {
      throw new Error(`GovernanceToken owner not set correctly`)
    }

    // Check that the deployment went as expected
    const govToken = await mintManager.governanceToken()
    if (govToken !== addresses.governanceToken) {
      throw new Error(`MintManager governance token not set correctly`)
    }
    mmOwner = await mintManager.owner()
    if (mmOwner !== mintManagerOwner) {
      throw new Error(`MintManager owner not set correctly`)
    }
    console.log('Validated MintManager config')
  })
