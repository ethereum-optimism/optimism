import { expect } from 'chai'
import { Wallet, utils, BigNumber, Contract } from 'ethers'
import { Direction } from './shared/watcher-utils'

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
  let ChugSplashDeployer: Contract

  const applyChugsplashInstructions = async (
    instructions: ChugSplashInstructions
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

    ChugSplashDeployer = new Contract(
      '0x420000000000000000000000000000000000000a',
      getContractInterface('ChugSplashDeployer', true),
      l2Wallet
    )
  })

  describe('setStorage and setCode are correctly applied', () => {
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
})
