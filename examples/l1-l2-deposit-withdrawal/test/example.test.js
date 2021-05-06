/* External Imports */
const { ethers } = require('hardhat')
const { expect } = require('chai')
const { Watcher } = require('@eth-optimism/core-utils')
const { getContractFactory } = require('@eth-optimism/contracts')

/* Internal Imports */
const factory = (name, ovm = false) => {
  const artifact = require(`../artifacts${ovm ? '-ovm' : ''}/contracts/${name}.sol/${name}.json`)
  return new ethers.ContractFactory(artifact.abi, artifact.bytecode)
}
const factory__L1_ERC20 = factory('ERC20')
const factory__L2_ERC20 = factory('L2DepositedERC20', true)
const factory__L1_ERC20Gateway = getContractFactory('OVM_L1ERC20Gateway')


describe(`L1 <> L2 Deposit and Withdrawal`, () => {
  // Set up our RPC provider connections.
  const l1RpcProvider = new ethers.providers.JsonRpcProvider('http://127.0.0.1:9545')

  // Constructor arguments for `ERC20.sol`
  const INITIAL_SUPPLY = 1234
  const L1_ERC20_NAME = 'L1 ERC20'

  // L1 messenger address depends on the deployment, this is default for our local deployment.
  const l1MessengerAddress = '0x59b670e9fA9D0A427751Af201D676719a970857b'
  // L2 messenger address is always the same.
  const l2MessengerAddress = '0x4200000000000000000000000000000000000007'
  const L2_ERC20_NAME = 'L2 ERC20'

  // Set up our wallets (using a default private key with 10k ETH allocated to it).
  // Need two wallets objects, one for interacting with L1 and one for interacting with L2.
  // Both will use the same private key.
  const key = '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'
  const l1Wallet = new ethers.Wallet(key, l1RpcProvider)
  const l2Wallet = new ethers.Wallet(key, ethers.provider)

  // Tool that helps watches and waits for messages to be relayed between L1 and L2.
  const watcher = new Watcher({
    l1: {
      provider: l1Wallet.provider,
      messengerAddress: l1MessengerAddress
    },
    l2: {
      provider: l2Wallet.provider,
      messengerAddress: l2MessengerAddress
    }
  })

  let L1_ERC20, L2_ERC20, L1_ERC20Gateway

  before(`deploy contracts`, async () => {
    // Deploy an ERC20 token on L1.
    L1_ERC20 = await factory__L1_ERC20.connect(l1Wallet).deploy(
      INITIAL_SUPPLY,
      L1_ERC20_NAME,
    )

    await L1_ERC20.deployTransaction.wait()

    // Deploy the paired ERC20 token to L2.
    L2_ERC20 = await factory__L2_ERC20.connect(l2Wallet).deploy(
      l2MessengerAddress,
      L2_ERC20_NAME,
    )

    await L2_ERC20.deployTransaction.wait()

    // Create a gateway that connects the two contracts.
    L1_ERC20Gateway = await factory__L1_ERC20Gateway.connect(l1Wallet).deploy(
      L1_ERC20.address,
      L2_ERC20.address,
      l1MessengerAddress,
    )
    await L1_ERC20Gateway.deployTransaction.wait()

    // initialize the erc20
    const tx = await L2_ERC20.init(L1_ERC20Gateway.address)
    await tx.wait()

    const l1Balance = await L1_ERC20.balanceOf(l1Wallet.address)
    const l2Balance = await L2_ERC20.balanceOf(l1Wallet.address)
    expect(l1Balance).to.eq(INITIAL_SUPPLY)
    expect(l2Balance).to.eq(0)
  })

  it("deposit followed by withdrawal", async () => {
    const amount = 1234

    let tx = await L1_ERC20.approve(L1_ERC20Gateway.address, amount)
    await tx.wait()

    tx = await L1_ERC20Gateway.deposit(amount)
    await tx.wait()

    // Wait for the message to be relayed to L2.
    let [ msgHash ] = await watcher.getMessageHashesFromL1Tx(tx.hash)
    const l2TxReceipt = await watcher.getL2TransactionReceipt(msgHash)
    expect(l2TxReceipt.to).to.eq(l2MessengerAddress)

    let l1Balance = await L1_ERC20.balanceOf(l1Wallet.address)
    let l2Balance = await L2_ERC20.balanceOf(l1Wallet.address)
    expect(l1Balance).to.eq(0)
    expect(l2Balance).to.eq(amount)

    // Burn the tokens on L2 and ask the L1 contract to unlock on our behalf.
    tx = await L2_ERC20.withdraw(amount)
    await tx.wait()

    // wait for the msg to get relayed
    ;[ msgHash ] = await watcher.getMessageHashesFromL2Tx(tx.hash)
    const l1TxReceipt = await watcher.getL1TransactionReceipt(msgHash)
    expect(l1TxReceipt.to).to.eq(l1MessengerAddress)

    // check balances
    l1Balance = await L1_ERC20.balanceOf(l1Wallet.address)
    l2Balance = await L2_ERC20.balanceOf(l1Wallet.address)
    expect(l1Balance).to.eq(amount)
    expect(l2Balance).to.eq(0)
  })
})
