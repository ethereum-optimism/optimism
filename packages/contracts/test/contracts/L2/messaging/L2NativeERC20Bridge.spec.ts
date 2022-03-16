import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'

describe('L2NativeERC20Bridge', () => {
  let alice: Signer
  let aliceAddress: string
  let bob: Signer
  let bobsAddress: string
  let l2MessengerImpersonator: Signer
  let Factory__L1StandardBridge: ContractFactory
  const INITIAL_TOTAL_SUPPLY = 100_000
  const ALICE_INITIAL_BALANCE = 50_000
  before(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    ;[alice, bob, l2MessengerImpersonator] = await ethers.getSigners()
    aliceAddress = await alice.getAddress()
    bobsAddress = await bob.getAddress()
    Factory__L1StandardBridge = await ethers.getContractFactory(
      'L1StandardBridge'
    )

    // get an L2ER20Bridge Interface
    //getContractInterface('IL2ERC20Bridge')
  })

  describe('initialise', () => {

  })

  describe('ERC20 deposits', () => {
    it('depositERC20() escrows the deposit amount and sends the correct deposit message', async () => {})

    it('depositERC20To() escrows the deposit amount and sends the correct deposit message', async () => {})

  })
})
