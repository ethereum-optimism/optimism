import { expect } from 'chai'
import { ethers } from 'hardhat'
import { Wallet, Contract } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

import { OptimismEnv } from './shared/env'
import { l2Provider, OVM_ETH_ADDRESS } from './shared/utils'

interface SetCodeInstruction {
  target: string // address
  code: string // bytes memory
}

interface SetStorageInstruction {
  target: string // address
  key: string // bytes32
  value: string // bytes32
}

type ChugSplashInstruction = SetCodeInstruction | SetStorageInstruction

const isSetStorageInstruction = (
  instr: ChugSplashInstruction
): instr is SetStorageInstruction => {
  return !instr['code']
}

describe('OVM Self-Upgrades', async () => {
  let env: OptimismEnv
  let l2Wallet: Wallet
  let ChugSplashDeployer: Contract

  const applyChugsplashInstructions = async (
    instructions: ChugSplashInstruction[]
  ) => {
    for (const instruction of instructions) {
      let res: any
      if (isSetStorageInstruction(instruction)) {
        res = await ChugSplashDeployer.executeAction(
          {
            actionType: 1,
            target: instruction.target,
            data: ethers.utils.defaultAbiCoder.encode(
              ['bytes32', 'bytes32'],
              [instruction.key, instruction.value]
            ),
          },
          {
            actionIndex: 0,
            siblings: [],
          }
        )
      } else {
        res = await ChugSplashDeployer.executeAction(
          {
            actionType: 0,
            target: instruction.target,
            data: instruction.code,
          },
          {
            actionIndex: 0,
            siblings: [],
          }
        )
      }
      await res.wait() // TODO: promise.all
    }
  }

  const checkChugsplashInstructionsApplied = async (
    instructions: ChugSplashInstruction[]
  ) => {
    for (const instruction of instructions) {
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
    instructions: ChugSplashInstruction[]
  ) => {
    // TODO: Initialize the upgrade here.
    // TODO: Add proof data to each instruction
    await applyChugsplashInstructions(instructions)
    await checkChugsplashInstructionsApplied(instructions)
  }

  before(async () => {
    env = await OptimismEnv.new()
    l2Wallet = env.l2Wallet

    ChugSplashDeployer = new Contract(
      '0x420000000000000000000000000000000000000a',
      getContractInterface('ChugSplashDeployer', true),
      l2Wallet
    )
  })

  describe('setStorage and setCode are correctly applied', () => {
    it('Should execute a basic storage upgrade', async () => {
      await applyAndVerifyUpgrade([
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
      const DummyContract = await (
        await ethers.getContractFactory('SimpleStorage', l2Wallet)
      ).deploy()
      await DummyContract.deployTransaction.wait()

      await applyAndVerifyUpgrade([
        {
          target: DummyContract.address,
          code:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
        },
      ])
    })

    it('Should execute a basic code upgrade which is not overwriting an existing account', async () => {
      await applyAndVerifyUpgrade([
        {
          target: '0x5678657856785678567856785678567856785678',
          code:
            '0x1234123412341234123412341234123412341234123412341234123412341234',
        },
      ])
    })
  })
})
