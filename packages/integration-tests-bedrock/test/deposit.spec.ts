/* Imports: External */
import { Contract, ContractFactory, utils, Wallet } from 'ethers'
import { awaitCondition } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { defaultTransactionFactory } from './shared/utils'
import env from './shared/env'
import counterArtifact from '../forge-artifacts/Counter.sol/Counter.json'
import multiDepositorArtifact from '../forge-artifacts/MultiDepositor.sol/MultiDepositor.json'

describe('Deposits', () => {
  let portal: Contract

  before(() => {
    portal = env.optimismPortal.connect(env.l1Wallet)
  })

  it('should deposit value', async () => {
    const recipWallet = Wallet.createRandom().connect(env.l2Provider)
    const tx = defaultTransactionFactory()
    tx.value = utils.parseEther('1.337')
    tx.to = recipWallet.address
    const result = await portal.depositTransaction(
      tx.to,
      tx.value,
      '3000000',
      false,
      [],
      {
        value: tx.value,
      }
    )
    await result.wait()

    await awaitCondition(async () => {
      const bal = await recipWallet.getBalance()
      return bal.eq(tx.value)
    })
  })

  it('should support multiple deposits in a single tx', async () => {
    const recipWallet = Wallet.createRandom().connect(env.l2Provider)
    const value = utils.parseEther('0.1')
    const factory = new ContractFactory(
      multiDepositorArtifact.abi,
      multiDepositorArtifact.bytecode.object
    ).connect(env.l1Wallet)
    const contract = await factory.deploy(portal.address)
    await contract.deployed()
    const tx = await contract.deposit(recipWallet.address, {
      value,
    })
    await tx.wait()

    await awaitCondition(async () => {
      const bal = await recipWallet.getBalance()
      return bal.eq('3000')
    })
  }).timeout(60_000)

  it.skip('should deposit a contract creation', async () => {
    const value = utils.parseEther('0.1')
    const factory = new ContractFactory(
      counterArtifact.abi,
      counterArtifact.bytecode.object
    )
    const tx = await factory.getDeployTransaction()
    const result = await portal.depositTransaction(
      `0x${'0'.repeat(40)}`,
      '0',
      '3000000',
      true,
      tx.data,
      {
        value,
      }
    )
    await result.wait()
    const l2Nonce = await env.l2Wallet.getTransactionCount()
    const addr = utils.getContractAddress({
      from: env.l2Wallet.address,
      nonce: l2Nonce,
    })

    await awaitCondition(async () => {
      const code = await env.l2Provider.getCode(addr)
      return code === counterArtifact.bytecode.object
    })
  })
})
