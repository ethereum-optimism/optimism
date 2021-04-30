import { expect } from 'chai'

/* Imports: External */
import { ethers } from 'hardhat'
import { Wallet, Contract } from 'ethers'
import {
  getContractInterface,
  getChugSplashActionBundle,
  ChugSplashAction,
  isSetStorageAction,
  predeploys,
} from '@eth-optimism/contracts'

/* Imports: Internal */
import { OptimismEnv } from './shared/env'

const executeAndVerifyChugSplashBundle = async (
  ChugSplashDeployer: Contract,
  actions: ChugSplashAction[]
): Promise<void> => {
  const bundle = getChugSplashActionBundle(actions)

  const res1 = await ChugSplashDeployer.approveTransactionBundle(
    bundle.root,
    bundle.actions.length,
    {
      gasLimit: 8000000,
      gasPrice: 0,
    }
  )
  await res1.wait()

  for (const rawAction of bundle.actions) {
    const res2 = await ChugSplashDeployer.executeAction(
      rawAction.action,
      rawAction.proof,
      {
        gasPrice: 0,
      }
    )
    await res2.wait()

    const action = actions[rawAction.proof.actionIndex]
    if (isSetStorageAction(action)) {
      expect(
        await ChugSplashDeployer.provider.getStorageAt(
          action.target,
          action.key
        )
      ).to.deep.equal(action.value)
    } else {
      expect(
        await ChugSplashDeployer.provider.getCode(action.target)
      ).to.deep.equal(action.code)
    }
  }
}

describe.only('OVM Self-Upgrades', async () => {
  let env: OptimismEnv
  let l2Wallet: Wallet
  let ChugSplashDeployer: Contract
  before(async () => {
    env = await OptimismEnv.new()
    l2Wallet = env.l2Wallet
    ChugSplashDeployer = new Contract(
      predeploys.ChugSplashDeployer,
      getContractInterface('ChugSplashDeployer', true),
      l2Wallet
    )
  })

  describe('setStorage and setCode are correctly applied', () => {
    it('Should execute a basic storage upgrade', async () => {
      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: predeploys.OVM_ETH,
          key: `0x${'12'.repeat(32)}`,
          value: `0x${'32'.repeat(32)}`,
        },
      ])
    })

    it('Should execute a basic upgrade overwriting existing deployed code', async () => {
      const DummyContract = await (
        await ethers.getContractFactory('SimpleStorage', l2Wallet)
      ).deploy()
      await DummyContract.deployTransaction.wait()

      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: DummyContract.address,
          code: `0x${'12'.repeat(32)}`,
        },
      ])
    })

    it('Should execute a basic code upgrade which is not overwriting an existing account', async () => {
      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: `0x${'56'.repeat(20)}`,
          code: `0x${'12'.repeat(32)}`,
        },
      ])
    })

    it('should set code and set storage in the same bundle', async () => {
      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: `0x${'56'.repeat(20)}`,
          code: `0x${'12'.repeat(32)}`,
        },
        {
          target: `0x${'56'.repeat(20)}`,
          key: `0x${'12'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
      ])
    })

    it('should set code multiple times in the same bundle', async () => {
      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: `0x${'56'.repeat(20)}`,
          code: `0x${'12'.repeat(32)}`,
        },
        {
          target: `0x${'56'.repeat(20)}`,
          code: `0x${'34'.repeat(32)}`,
        },
        {
          target: `0x${'56'.repeat(20)}`,
          code: `0x${'56'.repeat(32)}`,
        },
      ])
    })

    it('should set storage multiple times in the same bundle', async () => {
      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: `0x${'56'.repeat(20)}`,
          key: `0x${'12'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
        {
          target: `0x${'56'.repeat(20)}`,
          key: `0x${'34'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
        {
          target: `0x${'56'.repeat(20)}`,
          key: `0x${'56'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
      ])
    })

    it.skip('should set storage multiple times with different addresses in the same bundle', async () => {
      await executeAndVerifyChugSplashBundle(ChugSplashDeployer, [
        {
          target: `0x${'57'.repeat(20)}`,
          key: `0x${'12'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
        {
          target: `0x${'58'.repeat(20)}`,
          key: `0x${'34'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
        {
          target: `0x${'59'.repeat(20)}`,
          key: `0x${'56'.repeat(32)}`,
          value: `0x${'12'.repeat(32)}`,
        },
      ])
    })
  })
})
