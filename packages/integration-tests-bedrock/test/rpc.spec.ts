/* Imports: External */
import { ContractFactory, Wallet } from 'ethers'

/* Imports: Internal */
import { expect } from './shared/setup'
import { defaultTransactionFactory } from './shared/utils'
import env from './shared/env'
import counterArtifact from '../artifacts/Counter.sol/Counter.json'

describe('RPCs', () => {
  let wallet: Wallet

  before(async () => {
    wallet = env.l2Wallet
  })

  it('eth_chainId', async () => {
    const network = await env.l2Provider.getNetwork()
    expect(network.chainId).to.equal(901)
  })

  describe('eth_sendRawTransaction', () => {
    it('should correctly process a funds transfer', async () => {
      const altWallet = await Wallet.createRandom().connect(env.l2Provider)
      const tx = defaultTransactionFactory()
      tx.to = altWallet.address
      const nonce = await wallet.getTransactionCount()
      const result = await wallet.sendTransaction(tx)

      expect(result.from).to.equal(wallet.address)
      expect(result.nonce).to.equal(nonce)
      expect(result.gasLimit.toNumber()).to.equal(tx.gasLimit)
      expect(result.data).to.equal(tx.data)
      expect(await altWallet.getBalance()).to.equal(tx.value)
    })

    it('should correctly process a contract creation', async () => {
      const factory = new ContractFactory(
        counterArtifact.abi,
        counterArtifact.bytecode.object
      ).connect(wallet)
      const counter = await factory.deploy({
        gasLimit: 1_000_000,
      })
      await counter.deployed()
      expect(await env.l2Provider.getCode(counter.address)).not.to.equal('0x')
    })
  })
})
