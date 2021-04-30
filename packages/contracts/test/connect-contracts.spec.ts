import { ethers } from 'hardhat'
import { Signer, Contract } from 'ethers'
import {
  connectL1Contracts,
  connectL2Contracts,
} from '../dist/connect-contracts.js'

describe('connectL1Contracts', () => {
  let user: Signer
  before(async () => {
    ;[user] = await ethers.getSigners()
  })

  it('should connect to all mainnet l1 contracts', async () => {
    const l1Contracts = await connectL1Contracts(user, 'mainnet')
    console.log(l1Contracts)
  })
})
