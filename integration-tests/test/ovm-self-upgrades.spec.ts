import { expect } from 'chai'

/* Imports: External */
import hre, { ethers } from 'hardhat'
import { Wallet, Contract, ContractFactory } from 'ethers'
import {
  getContractInterface,
  predeploys,
  ChugSplashAction,
  getChugSplashActionBundle,
  isSetStorageAction,
} from '@eth-optimism/contracts'
import { getRandomAddress } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { OptimismEnv } from './shared/env'
import { fundUser, l2Provider, OVM_ETH_ADDRESS } from './shared/utils'

const applyAndVerifyUpgrade = async (
  L2ChugSplashDeployer: Contract,
  actions: ChugSplashAction[]
) => {
  const bundle = getChugSplashActionBundle(actions)

  const tx1 = await L2ChugSplashDeployer.approveTransactionBundle(
    bundle.root,
    bundle.actions.length
  )
  await tx1.wait()

  for (const action of bundle.actions) {
    const tx2 = await L2ChugSplashDeployer.executeAction(
      action.action,
      action.proof
    )
    await tx2.wait()
  }

  for (const action of actions) {
    if (isSetStorageAction(action)) {
      expect(
        await l2Provider.getStorageAt(action.target, action.key)
      ).to.deep.equal(action.value)
    } else {
      expect(await l2Provider.getCode(action.target)).to.deep.equal(action.code)
    }
  }
}

describe.only('OVM Self-Upgrades', async () => {
  let l2Wallet: Wallet
  before(async () => {
    const env = await OptimismEnv.new()
    // For simplicity, this is the default wallet that (at least for now) controls upgrades when
    // running the system locally.
    l2Wallet = new ethers.Wallet('0x' + 'FF'.repeat(64), l2Provider)
    await fundUser(env.watcher, env.gateway, hre.ethers.utils.parseEther('10'), l2Wallet.address)
  })

  let L2ChugSplashDeployer: Contract
  before(async () => {
    L2ChugSplashDeployer = new Contract(
      predeploys.L2ChugSplashDeployer,
      getContractInterface('L2ChugSplashDeployer'),
      l2Wallet
    )
  })

  describe('setStorage and setCode are correctly applied according to geth RPC', () => {
    it('Should execute a basic storage upgrade', async () => {
      await applyAndVerifyUpgrade(L2ChugSplashDeployer, [
        {
          target: OVM_ETH_ADDRESS,
          key:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
          value:
            '0x6789123412341234123412341234123412341234123412341234678967896789',
        },
      ])
    })

    it('Should execute a basic upgrade overwriting existing deployed code', async () => {
      // Deploy a dummy contract to overwrite.
      const factory = await hre.ethers.getContractFactory('SimpleStorage')
      const contract = await factory.connect(l2Wallet).deploy()
      await contract.deployTransaction.wait()

      await applyAndVerifyUpgrade(L2ChugSplashDeployer, [
        {
          target: contract.address,
          code:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
        },
      ])
    })

    it('Should execute a basic code upgrade which is not overwriting an existing account', async () => {
      await applyAndVerifyUpgrade(L2ChugSplashDeployer, [
        {
          target: getRandomAddress(),
          code:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
        },
      ])
    })
  })

  describe('Contracts upgraded with setStorage and setCode behave as expected', () => {
    it('code with updated storage returns the new storage', async () => {
      const factory = await hre.ethers.getContractFactory('SimpleStorage')
      const contract = await factory.connect(l2Wallet).deploy()
      await contract.deployTransaction.wait()

      expect(await contract.value()).to.eq(ethers.constants.HashZero)

      const newValue = '0x' + '00'.repeat(31) + '01'

      await applyAndVerifyUpgrade(L2ChugSplashDeployer, [
        {
          target: contract.address,
          key: ethers.constants.HashZero,
          value: newValue,
        },
      ])

      const valueAfter = await contract.value()
      expect(valueAfter).to.eq(newValue)
    })

    it('code with an updated constant returns the new constant', async () => {
      const factory1 = await hre.ethers.getContractFactory('ReturnOne')
      const contract1 = await factory1.connect(l2Wallet).deploy()
      await contract1.deployTransaction.wait()

      const factory2 = await hre.ethers.getContractFactory('ReturnTwo')
      const contract2 = await factory2.connect(l2Wallet).deploy()
      await contract2.deployTransaction.wait()

      const one = await contract1.get()
      expect(one.toNumber()).to.eq(1)

      await applyAndVerifyUpgrade(L2ChugSplashDeployer, [
        {
          target: contract1.address,
          code: await l2Provider.getCode(contract2.address),
        },
      ])

      const two = await contract1.get()
      expect(two.toNumber()).to.eq(2)
    })
  })
})
