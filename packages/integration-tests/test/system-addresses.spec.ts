/* Imports: External */
import { BigNumber, Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import { futurePredeploys } from '@eth-optimism/contracts'
import { sleep } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { envConfig } from './shared/utils'

const SYSTEM_ADDRESSES = [futurePredeploys.System0, futurePredeploys.System1]

describe('System addresses', () => {
  let env: OptimismEnv

  let deployerWallets: Wallet[] = []

  let contracts: Contract[]

  let Factory__ERC20: ContractFactory

  before(async function () {
    if (!envConfig.RUN_SYSTEM_ADDRESS_TESTS) {
      console.log('Skipping system address tests.')
      this.skip()
      return
    }

    env = await OptimismEnv.new()

    deployerWallets = [
      new Wallet(process.env.SYSTEM_ADDRESS_0_DEPLOYER_KEY, env.l2Provider),
      new Wallet(process.env.SYSTEM_ADDRESS_1_DEPLOYER_KEY, env.l2Provider),
    ]
    for (const deployer of deployerWallets) {
      await env.l2Wallet.sendTransaction({
        to: deployer.address,
        value: utils.parseEther('0.1'),
      })
    }
    contracts = []

    Factory__ERC20 = await ethers.getContractFactory('ERC20', env.l2Wallet)
  })

  it('should have no code for the system addresses initially', async () => {
    for (const addr of SYSTEM_ADDRESSES) {
      const code = await env.l2Provider.getCode(addr, 'latest')
      expect(code).to.eq('0x')
    }
  })

  it('should deploy to the system address', async () => {
    for (let i = 0; i < deployerWallets.length; i++) {
      const contract = await Factory__ERC20.connect(deployerWallets[i]).deploy(
        100000000,
        'OVM Test',
        8,
        'OVM'
      )
      // have to use the receipt here since ethers calculates the
      // contract address on-the-fly
      const receipt = await contract.deployTransaction.wait()
      expect(receipt.contractAddress).to.eq(SYSTEM_ADDRESSES[i])

      const fetchedReceipt = await env.l2Provider.getTransactionReceipt(
        receipt.transactionHash
      )
      expect(fetchedReceipt.contractAddress).to.eq(SYSTEM_ADDRESSES[i])

      contracts.push(await ethers.getContractAt('ERC20', SYSTEM_ADDRESSES[i]))
    }
  })

  it('contracts deployed at the system addresses should function', async () => {
    expect(contracts.length).to.eq(2)

    for (let i = 0; i < contracts.length; i++) {
      const wallet = deployerWallets[i]
      const contract = contracts[i].connect(wallet)
      const code = await env.l2Provider.getCode(contract.address, 'latest')
      expect(code).not.to.eq('0x')

      const tx = await contract.transfer(env.l2Wallet.address, 1000)
      await tx.wait()
      const bal = await contract.balanceOf(env.l2Wallet.address)
      expect(bal).to.deep.equal(BigNumber.from(1000))
    }
  })

  it('should not deploy any additional contracts from the deployer at the system address', async () => {
    for (let i = 0; i < deployerWallets.length; i++) {
      const contract = await Factory__ERC20.connect(deployerWallets[i]).deploy(
        100000000,
        'OVM Test',
        8,
        'OVM'
      )
      await contract.deployed()
      const receipt = await contract.deployTransaction.wait()
      expect(receipt.contractAddress).not.to.eq(SYSTEM_ADDRESSES[i])
      expect(receipt.contractAddress).not.to.eq(null)
    }
  })

  const testReplica = async (otherProvider) => {
    const seqBlock = await env.l2Provider.getBlock('latest')
    while (true) {
      const verHeight = await otherProvider.getBlockNumber()
      if (verHeight >= seqBlock.number) {
        break
      }
      await sleep(200)
    }

    const verBlock = await otherProvider.getBlock(seqBlock.number)
    expect(verBlock).to.deep.eq(seqBlock)

    for (const addr of SYSTEM_ADDRESSES) {
      const seqCode = await env.l2Provider.getCode(addr)
      const verCode = await otherProvider.getCode(addr)
      expect(seqCode).to.eq(verCode)
    }
  }

  it('should be properly handled on verifiers', async function () {
    if (!envConfig.RUN_VERIFIER_TESTS) {
      this.skip()
      return
    }

    await testReplica(env.verifierProvider)
  })

  it('should be properly handled on replicas', async function () {
    if (!envConfig.RUN_REPLICA_TESTS) {
      this.skip()
      return
    }

    await testReplica(env.replicaProvider)
  })
})
