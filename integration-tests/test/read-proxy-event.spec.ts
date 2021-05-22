import { expect } from 'chai'
import { ethers } from 'hardhat'

/* Imports: External */
import { Contract, Wallet } from 'ethers'
import { OptimismEnv } from './shared/env'

describe('Reading events from proxy contracts', () => {
  let l2Wallet: Wallet
  before(async () => {
    const env = await OptimismEnv.new()
    l2Wallet = env.l2Wallet
  })

  // helper to query the transfers
  const _queryFilterTransfer = async (
    queryContract: Contract,
    filterContract: Contract
  ) => {
    // Get the filter
    const filter = filterContract.filters.Transfer(null, null, null)
    // Query the filter
    return queryContract.queryFilter(filter, 0, 'latest')
  }

  let ProxyERC20: Contract
  let ERC20: Contract

  beforeEach(async () => {
    // Set up our contract factories in advance.
    const Factory__ERC20 = await ethers.getContractFactory(
      'ChainlinkERC20',
      l2Wallet
    )
    const Factory__UpgradeableProxy = await ethers.getContractFactory(
      'UpgradeableProxy',
      l2Wallet
    )

    // Deploy the underlying ERC20 implementation.
    ERC20 = await Factory__ERC20.deploy()
    await ERC20.deployTransaction.wait()

    // Deploy the upgradeable proxy and execute the init function.
    ProxyERC20 = await Factory__UpgradeableProxy.deploy(
      ERC20.address,
      ERC20.interface.encodeFunctionData('init', [
        1000, // initial supply
        'Cool Token Name Goes Here', // token name
      ])
    )
    await ProxyERC20.deployTransaction.wait()
    ProxyERC20 = new ethers.Contract(
      ProxyERC20.address,
      ERC20.interface,
      l2Wallet
    )
  })

  it('should read transfer events from a proxy ERC20', async () => {
    // Make two transfers.
    const recipient = '0x0000000000000000000000000000000000000000'
    const transfer1 = await ProxyERC20.transfer(recipient, 1)
    await transfer1.wait()
    const transfer2 = await ProxyERC20.transfer(recipient, 1)
    await transfer2.wait()

    // Make sure events are being emitted in the right places.
    expect((await _queryFilterTransfer(ERC20, ERC20)).length).to.eq(0)
    expect((await _queryFilterTransfer(ERC20, ProxyERC20)).length).to.eq(0)
    expect((await _queryFilterTransfer(ProxyERC20, ERC20)).length).to.eq(2)
    expect((await _queryFilterTransfer(ProxyERC20, ProxyERC20)).length).to.eq(2)
  })
})
