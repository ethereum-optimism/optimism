import { expect } from 'chai'
import { Wallet, utils, BigNumber, Contract, ContractFactory } from 'ethers'

import { OptimismEnv } from './shared/env'

import { getContractInterface } from '@eth-optimism/contracts'
import { l2Provider, OVM_ETH_ADDRESS } from './shared/utils'
import { ethers } from 'hardhat'

// TODO: use actual imported Chugsplash type

interface SetCodeInstruction {
  target: string // address
  code: string // bytes memory
}

interface SetStorageInstruction {
  target: string // address
  key: string // bytes32
  value: string // bytes32
}

type ChugsplashInstruction = SetCodeInstruction | SetStorageInstruction

// Just an array of the above two instruction types.
type ChugSplashInstructions = Array<ChugsplashInstruction>

const isSetStorageInstruction = (
  instr: ChugsplashInstruction
): instr is SetStorageInstruction => {
  return !instr['code']
}

describe('OVM Self-Upgrades', async () => {
  let env: OptimismEnv
  let l2Wallet: Wallet
  let OVM_UpgradeExecutor: Contract
  let Factory__ReturnOne: ContractFactory
  let DeployedBytecode__ReturnTwo: string
  let Factory__SimpleStorage: ContractFactory

  const applyChugsplashInstructions = async (
    instructions: ChugSplashInstructions
  ) => {
    for (const instruction of instructions) {
      let res
      if (isSetStorageInstruction(instruction)) {
        res = await OVM_UpgradeExecutor.setStorage(
          instruction.target,
          instruction.key,
          instruction.value
        )
      } else {
        res = await OVM_UpgradeExecutor.setCode(
          instruction.target,
          instruction.code
        )
      }
      await res.wait() // TODO: promise.all
    }
  }

  const checkChugsplashInstructionsApplied = async (
    instructions: ChugSplashInstructions
  ) => {
    for (const instruction of instructions) {
      // TODO: promise.all this for with a map for efficiency
      if (isSetStorageInstruction(instruction)) {
        const actualStorage = await l2Provider.getStorageAt(
          instruction.target,
          instruction.key
        )
        expect(actualStorage).to.deep.eq(instruction.value)
      } else {
        const actualCode = await l2Provider.getCode(instruction.target)
        expect(actualCode).to.deep.eq(instruction.code)
      }
    }
  }

  const applyAndVerifyUpgrade = async (
    instructions: ChugSplashInstructions
  ) => {
    await applyChugsplashInstructions(instructions)
    await checkChugsplashInstructionsApplied(instructions)
  }

  before(async () => {
    env = await OptimismEnv.new()
    l2Wallet = env.l2Wallet

    OVM_UpgradeExecutor = new Contract(
      '0x420000000000000000000000000000000000000a',
      getContractInterface('OVM_UpgradeExecutor', true),
      l2Wallet
    )

    Factory__ReturnOne = await ethers.getContractFactory('ReturnOne', l2Wallet)
    const Factory__ReturnTwo = await ethers.getContractFactory(
      'ReturnTwo',
      l2Wallet
    )
    const returnTwo = await (
      await Factory__ReturnTwo.deploy()
    ).deployTransaction.wait()
    DeployedBytecode__ReturnTwo = await l2Provider.getCode(
      returnTwo.contractAddress
    )

    Factory__SimpleStorage = await ethers.getContractFactory(
      'SimpleStorage',
      l2Wallet
    )
  })

  describe('setStorage and setCode are correctly applied according to geth RPC', () => {
    it('Should execute a basic storage upgrade', async () => {
      const basicStorageUpgrade: ChugSplashInstructions = [
        {
          target: OVM_ETH_ADDRESS,
          key:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
          value:
            '0x6789123412341234123412341234123412341234123412341234678967896789',
        },
      ]
      await applyAndVerifyUpgrade(basicStorageUpgrade)
    })

    it('Should execute a basic upgrade overwriting existing deployed code', async () => {
      const DummyContract = await (
        await ethers.getContractFactory('SimpleStorage', l2Wallet)
      ).deploy()
      await DummyContract.deployTransaction.wait()

      const basicCodeUpgrade: ChugSplashInstructions = [
        {
          target: DummyContract.address,
          code:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
        },
      ]
      await applyAndVerifyUpgrade(basicCodeUpgrade)
    })

    it('Should execute a basic code upgrade which is not overwriting an existing account', async () => {
      // TODO: fix me?  Currently breaks due to nil pointer dereference; triggerd by evm.StateDB.SetCode(...) in ovm_state_manager.go ?
      // More recent update: I cannot get this to error out any more.
      const emptyAccountCodeUpgrade: ChugSplashInstructions = [
        {
          target: '0x5678657856785678567856785678567856785678',
          code:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
        },
      ]
      await applyAndVerifyUpgrade(emptyAccountCodeUpgrade)
    })
  })

  describe('Contracts upgraded with setStorage and setCode behave as expected', () => {
    it('code with updated storage returns the new storage', async () => {
      const SimpleStorage: Contract = await Factory__SimpleStorage.deploy()
      await SimpleStorage.deployTransaction.wait()

      const valueBefore = await SimpleStorage.value()
      expect(valueBefore).to.eq(ethers.constants.HashZero)

      const newValue = '0x' + '00'.repeat(31) + '01'
      const storageVarUpgrade: ChugSplashInstructions = [
        {
          target: SimpleStorage.address,
          key: ethers.constants.HashZero,
          value: newValue,
        },
      ]

      await applyAndVerifyUpgrade(storageVarUpgrade)

      const valueAfter = await SimpleStorage.value()
      expect(valueAfter).to.eq(newValue)
    })

    it('code with an updated constant returns the new constant', async () => {
      const Returner = await Factory__ReturnOne.deploy()
      await Returner.deployTransaction.wait()
      const one = await Returner.get()
      expect(one.toNumber()).to.eq(1)

      const constantUpgrade: ChugSplashInstructions = [
        {
          target: Returner.address,
          code: DeployedBytecode__ReturnTwo,
        },
      ]

      await applyAndVerifyUpgrade(constantUpgrade)

      const two = await Returner.get()
      expect(two.toNumber()).to.eq(2)
    })
  })
})
