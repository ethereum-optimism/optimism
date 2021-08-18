import chai, { expect } from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import { Direction } from './shared/watcher-utils'

import L1NFTBridge from '../artifacts/contracts/bridges/OVM_L1NFTBridge.sol/OVM_L1NFTBridge.json'
import L2NFTBridge from '../artifacts-ovm/contracts/bridges/OVM_L2NFTBridge.sol/OVM_L2NFTBridge.json'
import L1ERC721Json from '../artifacts/contracts/L1ERC721.sol/L1ERC721.json'
import L2ERC721Json from '../artifacts-ovm/contracts/standards/L2StandardERC721.sol/L2StandardERC721.json'

import { OptimismEnv } from './shared/env'
import * as fs from 'fs'

describe('NFT Bridge Test', async () => {

  let Factory__L1ERC721: ContractFactory
  let Factory__L2ERC721: ContractFactory
  let L1Bridge: Contract
  let L2Bridge: Contract
  let L1ERC721: Contract
  let L2ERC721: Contract

  let env: OptimismEnv

  const DUMMY_TOKEN_ID = 1234

  before(async () => {

    env = await OptimismEnv.new()

    Factory__L1ERC721 = new ContractFactory(
        L1ERC721Json.abi,
        L1ERC721Json.bytecode,
        env.bobl1Wallet
    )

    Factory__L2ERC721 = new ContractFactory(
        L2ERC721Json.abi,
        L2ERC721Json.bytecode,
        env.bobl2Wallet
    )

    L1Bridge = new Contract(
        env.addressesOMGX.Proxy__L1NFTBridge,
        L1NFTBridge.abi,
        env.bobl1Wallet
    )

    L2Bridge = new Contract(
        env.addressesOMGX.Proxy__L2NFTBridge,
        L2NFTBridge.abi,
        env.bobl2Wallet
    )


    // deploy a test token each time if existing contracts are used for tests
    L1ERC721 = await Factory__L1ERC721.deploy(
        'Test',
        'TST'
    )

    await L1ERC721.deployTransaction.wait()

    L2ERC721 = await Factory__L2ERC721.deploy(L2Bridge.address, L1ERC721.address, 'Test', 'TST')

    await L2ERC721.deployTransaction.wait()
  })

  it('should deposit NFT to L2', async () => {
    // mint nft
    const mintTx = await L1ERC721.mint(env.bobl1Wallet.address, DUMMY_TOKEN_ID)
    await mintTx.wait()

    const approveTx = await L1ERC721.approve(L1Bridge.address, DUMMY_TOKEN_ID)
    await approveTx.wait()

    await env.waitForXDomainTransaction(
        L1Bridge.depositNFT(
          L1ERC721.address,
          L2ERC721.address,
          DUMMY_TOKEN_ID,
          9999999,
          utils.formatBytes32String((new Date().getTime()).toString())
        ),
        Direction.L1ToL2
      )

    const ownerL1 = await L1ERC721.ownerOf(DUMMY_TOKEN_ID)
    const ownerL2 = await L2ERC721.ownerOf(DUMMY_TOKEN_ID)

    expect(ownerL1).to.deep.eq(L1Bridge.address)
    expect(ownerL2).to.deep.eq(env.bobl2Wallet.address)
  })

  it('should be able to transfer NFT on L2', async () => {
    const transferTx = await L2ERC721.transferFrom(env.bobl2Wallet.address, env.alicel2Wallet.address, DUMMY_TOKEN_ID)
    await transferTx.wait()

    const ownerL2 = await L2ERC721.ownerOf(DUMMY_TOKEN_ID)
    expect(ownerL2).to.deep.eq(env.alicel2Wallet.address)
  })

  it('should not be able to withdraw non-owned NFT', async () => {

    await expect(
        L2Bridge.connect(env.bobl2Wallet).withdraw(
            L2ERC721.address,
            DUMMY_TOKEN_ID,
            9999999,
            utils.formatBytes32String((new Date().getTime()).toString())
        )
    ).to.be.reverted;
  })

  it('should withdraw NFT', async () => {

    await env.waitForXDomainTransaction(
        L2Bridge.connect(env.alicel2Wallet).withdraw(
          L2ERC721.address,
          DUMMY_TOKEN_ID,
          9999999,
          utils.formatBytes32String((new Date().getTime()).toString())
        ),
        Direction.L2ToL1
      )

      await expect(L2ERC721.ownerOf(DUMMY_TOKEN_ID)).to.be.reverted;

      const ownerL1 = await L1ERC721.ownerOf(DUMMY_TOKEN_ID)
      expect(ownerL1).to.be.deep.eq(env.alicel2Wallet.address)
  })

})
