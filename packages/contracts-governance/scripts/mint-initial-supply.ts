import { task } from 'hardhat/config'
import { ethers } from 'ethers'
import dotenv from 'dotenv'

import { prompt } from '../src/prompt'

dotenv.config()

task('mint-initial-supply', 'Mints the initial token supply')
  .addParam('mintManagerAddr', 'Address of the mint manager')
  .addParam('amount', 'Amount to mint (IN WHOLE OP)', '4294967296')
  .addParam(
    'pkMinter',
    'Private key of the minter',
    process.env.PRIVATE_KEY_INITIAL_MINTER
  )
  .setAction(async (args, hre) => {
    const minter = new hre.ethers.Wallet(args.pkMinter).connect(
      hre.ethers.provider
    )
    const amount = args.amount
    const amountBase = ethers.utils.parseEther(amount)
    console.log('Please verify initial mint amount and recipient.')
    console.log('!!! THIS IS A ONE-WAY ACTION !!!')
    console.log('')
    console.log(`Amount:              ${args.amount}`)
    console.log(`Amount (base units): ${amountBase.toString()}`)
    console.log(`Recipient:           ${minter.address}`)
    console.log('')

    const govToken = await hre.ethers.getContractAt(
      'GovernanceToken',
      '0x4200000000000000000000000000000000000042'
    )

    const mintManager = (
      await hre.ethers.getContractAt('MintManager', args.mintManagerAddr)
    ).connect(minter)

    const permittedAfter = await mintManager.mintPermittedAfter()
    if (!permittedAfter.eq(0)) {
      throw new Error('Mint manager has already executed.')
    }

    const owner = await mintManager.owner()
    if (minter.address !== owner) {
      throw new Error(
        `Mint manager is owned by ${owner}, not ${minter.address}`
      )
    }

    const tokOwner = await govToken.owner()
    if (mintManager.address !== tokOwner) {
      throw new Error(
        `Gov token is owned by ${tokOwner}, not ${mintManager.address}`
      )
    }

    await prompt('Is this OK?')

    const tx = await mintManager.mint(minter.address, amountBase, {
      gasLimit: 3_000_000,
    })
    console.log(`Sent transaction ${tx.hash}`)
    await tx.wait()
    console.log('Successfully minted. Verifying...')

    const supply = await govToken.totalSupply()
    if (supply.eq(amountBase)) {
      console.log('Total supply verified.')
    } else {
      console.log(
        `Total supply invalid! Have: ${supply.toString()}, want: ${amountBase.toString()}.`
      )
    }

    const bal = await govToken.balanceOf(minter.address)
    if (bal.eq(amountBase)) {
      console.log('Balance verified.')
    } else {
      console.log(
        `Minter balance invalid! Have: ${bal.toString()}, want: ${amountBase.toString()}.`
      )
    }
  })
