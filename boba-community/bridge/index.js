const { ethers } = require('ethers')
const { keccak256 } = require('@ethersproject/keccak256')
const { defaultAbiCoder } = require('@ethersproject/abi')
require('dotenv').config()

const { SEPOLIA, GOERLI, MAINNET, ABI } = require('./constants')

const L1_NODE_URL = process.env.L1_NODE_URL
const PRIVATE_KEY = process.env.PRIVATE_KEY

const BOBA_SEPOLIA_ENDPOINT = 'https://l2.anchorage.boba.network'

if (!L1_NODE_URL) {
  throw new Error('L1_NODE_URL is not set')
}

if (!PRIVATE_KEY) {
  throw new Error('PRIVATE_KEY is not set')
}

const L1Web3Provider = new ethers.providers.JsonRpcProvider(L1_NODE_URL)
const L1Wallet = new ethers.Wallet(PRIVATE_KEY, L1Web3Provider)

const loadContracts = (addresses, l1Wallet, l2Wallet) => {
  const L2OutputOracle = new ethers.Contract(
    addresses.L2OUTPUTORACLE,
    new ethers.utils.Interface([
      'function latestBlockNumber() public view returns (uint256)',
      'function getL2OutputIndexAfter() public view returns (uint256)',
      'function getL2Output() public view returns (bytes32 outputRoot, uint128 timestamp, uint128 l2BlockNumber)',
    ]),
    l1Wallet
  )
  const L1StandardBridge = new ethers.Contract(
    addresses.L1STANDARDBRIDGE,
    new ethers.utils.Interface([
      'function depositETH(uint32 _minGasLimit, bytes _extraData) external payable',
      'function depositERC20(address _l1token, address _l2token, uint256 _amount, uint32 _minGasLimit, bytes _extraData) external',
    ]),
    l1Wallet
  )
  const OptimismPortal = new ethers.Contract(
    addresses.OPTIMISMPORTAL,
    ABI.OPTIMISMPORTAL,
    l1Wallet
  )
  const L1BOBA = new ethers.Contract(
    addresses.L1BOBA,
    new ethers.utils.Interface([
      'function approve(address _spender, uint256 _amount) external returns (bool)',
      'function balanceOf(address _account) external view returns (uint256)',
    ]),
    l1Wallet
  )
  const L2StandardBridge = new ethers.Contract(
    addresses.L2STANDARDBRIDGE,
    new ethers.utils.Interface([
      'function withdraw(address _l2Token, uint256 _amount, uint32 _minGasLimit, bytes _extraData) external',
    ]),
    l2Wallet
  )
  const L2ToL1MessagePasser = new ethers.Contract(
    addresses.L2TOL1MESSAGEPASSER,
    new ethers.utils.Interface([
      'event MessagePassed(uint256 indexed nonce, address indexed sender, address indexed target, uint256 value, uint256 gasLimit, bytes data, bytes32 withdrawalHash)',
    ]),
    l2Wallet
  )
  const L2BOBA = new ethers.Contract(
    addresses.L2BOBA,
    new ethers.utils.Interface([
      'function approve(address _spender, uint256 _amount) external returns (bool)',
      'function balanceOf(address _account) external view returns (uint256)',
    ]),
    l2Wallet
  )
  return {
    L1Wallet: l1Wallet,
    L2Wallet: l2Wallet,
    L2OutputOracle,
    OptimismPortal,
    L1StandardBridge,
    L1BOBA,
    L2StandardBridge,
    L2ToL1MessagePasser,
    L2BOBA,
    addresses,
  }
}

const getConfigurations = async () => {
  l1ChainId = (await L1Web3Provider.getNetwork()).chainId
  if (l1ChainId === 11155111) {
    console.log('Connected to Sepolia Testnet. Loading contracts...')
    const l2Wallet = L1Wallet.connect(
      new ethers.providers.JsonRpcProvider(BOBA_SEPOLIA_ENDPOINT)
    )
    return loadContracts(SEPOLIA, L1Wallet, l2Wallet)
  }
  console.error('!! Unsupported network !!')
  return
}

const bridgeETHToL2 = async () => {
  const config = await getConfigurations()
  if (!config) {
    return
  }
  const { L1Wallet, L2Wallet, addresses } = config
  console.log(`---------------------------------------------------`)

  const L1Balance = await L1Wallet.getBalance()
  const L2Balance = await L2Wallet.getBalance()
  console.log(`L1 Balance: ${ethers.utils.formatEther(L1Balance)}`)
  console.log(`L2 Balance: ${ethers.utils.formatEther(L2Balance)}`)
  console.log(`---------------------------------------------------`)

  const L1DepositAmount = ethers.utils.parseEther('0.01')
  if (L1Balance.lt(L1DepositAmount)) {
    console.error('Insufficient L1 balance')
    return
  }

  const L1Tx = await L1Wallet.sendTransaction({
    to: addresses.OPTIMISMPORTAL,
    value: L1DepositAmount,
  })
  await L1Tx.wait()
  console.log(
    `Deposited ${ethers.utils.formatEther(
      L1DepositAmount
    )} ETH to Optimism Portal`
  )

  while (true) {
    const postL2Balance = await L2Wallet.getBalance()
    if (!L2Balance.eq(postL2Balance)) {
      break
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }

  console.log(`---------------------------------------------------`)
  console.log(`ETH bridged to L2!`)
  const postL2Balance = await L2Wallet.getBalance()
  console.log(`L2 Balance: ${ethers.utils.formatEther(postL2Balance)}`)
}

const bridgeBOBAToL2 = async () => {
  const config = await getConfigurations()
  if (!config) {
    return
  }
  const { L1Wallet, L2Wallet, L1StandardBridge, L1BOBA, L2BOBA, addresses } = config
  console.log(`---------------------------------------------------`)

  const L1BOBABalance = await L1BOBA.balanceOf(L1Wallet.address)
  const L2BOBABalance = await L2BOBA.balanceOf(L2Wallet.address)
  console.log(`L1 BOBA Balance: ${ethers.utils.formatEther(L1BOBABalance)}`)
  console.log(`L2 BOBA Balance: ${ethers.utils.formatEther(L2BOBABalance)}`)
  console.log(`---------------------------------------------------`)

  const L1DepositAmount = ethers.utils.parseEther('10')
  if (L1BOBABalance.lt(L1DepositAmount)) {
    console.error('Insufficient L1 BOBA balance')
    return
  }

  const L1ApproveTx = await L1BOBA.approve(
    addresses.L1STANDARDBRIDGE,
    L1DepositAmount
  )
  await L1ApproveTx.wait()
  console.log(`Approved ${ethers.utils.formatEther(L1DepositAmount)} BOBA`)

  const L1DepositTx = await L1StandardBridge.depositERC20(
    L1BOBA.address,
    L2BOBA.address,
    L1DepositAmount,
    999999,
    '0x'
  )
  await L1DepositTx.wait()
  console.log(
    `Deposited ${ethers.utils.formatEther(
      L1DepositAmount
    )} BOBA to L1 Standard Bridge`
  )

  while (true) {
    const postL2BOBABalance = await L2BOBA.balanceOf(L2Wallet.address)
    if (!L2BOBABalance.eq(postL2BOBABalance)) {
      break
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }

  console.log(`---------------------------------------------------`)
  console.log(`BOBA bridged to L2!`)
  const postL2BOBABalance = await L2BOBA.balanceOf(L2Wallet.address)
  console.log(`L2 BOBA Balance: ${ethers.utils.formatEther(postL2BOBABalance)}`)
}

const bridgeETHToL1 = async () => {
  const config = await getConfigurations()
  if (!config) {
    return
  }
  const {
    L1Wallet,
    L2Wallet,
    L2OutputOracle,
    OptimismPortal,
    L2ToL1MessagePasser,
    addresses,
  } = config
  console.log(`---------------------------------------------------`)

  const L1Balance = await L1Wallet.getBalance()
  const L2Balance = await L2Wallet.getBalance()
  console.log(`L1 Balance: ${ethers.utils.formatEther(L1Balance)}`)
  console.log(`L2 Balance: ${ethers.utils.formatEther(L2Balance)}`)
  console.log(`---------------------------------------------------`)

  const L2WithdrawAmount = ethers.utils.parseEther('0.01')
  if (L2Balance.lt(L2WithdrawAmount)) {
    console.error('Insufficient L2 balance')
    return
  }

  const exitTx = await L2Wallet.sendTransaction({
    to: addresses.L2STANDARDBRIDGE,
    value: L2WithdrawAmount,
  })
  const receipt = await exitTx.wait()
  const L2BlockNumber = receipt.blockNumber
  console.log(
    `Sent ${ethers.utils.formatEther(
      L2WithdrawAmount
    )} ETH to L2 Standard Bridge at block ${L2BlockNumber}`
  )

  const logs = await L2ToL1MessagePasser.queryFilter(
    L2ToL1MessagePasser.filters.MessagePassed(),
    L2BlockNumber,
    L2BlockNumber
  )
  if (logs.length !== 1) {
    console.error('!! Unknown error: logs.length != 1 !!')
    return
  }

  const types = ['uint256', 'address', 'address', 'uint256', 'uint256', 'bytes']
  const encoded = defaultAbiCoder.encode(types, [
    logs[0].args.nonce,
    logs[0].args.sender,
    logs[0].args.target,
    logs[0].args.value,
    logs[0].args.gasLimit,
    logs[0].args.data,
  ])
  const slot = keccak256(encoded)
  console.log(`Calculated slot: ${slot}`)

  const withdrawalHash = logs[0].args.withdrawalHash
  console.log(`withdrawalHash: ${withdrawalHash}`)
  const messageSlot = ethers.utils.keccak256(
    ethers.utils.defaultAbiCoder.encode(
      ['bytes32', 'uint256'],
      [slot, ethers.constants.HashZero]
    )
  )

  if (withdrawalHash !== slot) {
    console.error('!! Unknown error: withdrawalHash !== slot !!')
    return
  }
  console.log(`Pass check! withdrawalHash === slot`)

  const proof = await L2Wallet.provider.send('eth_getProof', [
    '0x4200000000000000000000000000000000000016',
    [messageSlot],
    L2BlockNumber,
  ])
  console.log(`Got proof: ${JSON.stringify(proof)}`)

  console.log(`---------------------------------------------------`)
  console.log(`Waiting for L2 block to be published...`)

  let latestBlockOnL1 = await L2OutputOracle.latestBlockNumber()
  while (latestBlockOnL1 < L2BlockNumber) {
    await new Promise((resolve) => setTimeout(resolve, 1000))
    latestBlockOnL1 = await L2OutputOracle.latestBlockNumber()
    console.log(
      `Waiting for L2 block: ${L2BlockNumber} - Latest L2 block on L1: ${latestBlockOnL1}`
    )
  }

  console.log(`---------------------------------------------------`)
  console.log(`L2 block published!`)
  const l2OutputIndex = await L2OutputOracle.getL2OutputIndexAfter(blockNumber)
  const proposal = await L2OutputOracle.getL2Output(l2OutputIndex)
  const proposalBlockNumber = proposal.l2BlockNumber
  const proposalBlock = await L2Web3.send('eth_getBlockByNumber', [
    proposalBlockNumber.toNumber(),
    false,
  ])
  const hash = keccak256(
    defaultAbiCoder.encode(
      ['bytes32', 'bytes32', 'bytes32', 'bytes32'],
      [
        ethers.constants.HashZero,
        proposalBlock.stateRoot,
        proof.storageHash,
        proposalBlock.hash,
      ]
    )
  )
  console.log(`Calculated L2 proposal hash: ${hash}`)

  const proveTx = await OptimismPortal.proveWithdrawalTransaction(
    [
      logs[0].args.nonce,
      logs[0].args.sender,
      logs[0].args.target,
      logs[0].args.value,
      logs[0].args.gasLimit,
      logs[0].args.data,
    ],
    l2OutputIndex,
    [
      ethers.constants.HashZero,
      proposalBlock.stateRoot,
      proof.storageHash,
      proposalBlock.hash,
    ],
    estimate.storageProof[0].proof
  )
  await proveTx.wait()
  console.log(`Proved withdrawal transaction!`)

  const gasEstimationFinalSubmit = async () => {
    const gas = await OptimismPortal.estimateGas.finalizeWithdrawalTransaction([
      logs[0].args.nonce,
      logs[0].args.sender,
      logs[0].args.target,
      logs[0].args.value,
      logs[0].args.gasLimit,
      logs[0].args.data,
    ])
    console.log(
      'Gas estimation for finalizeWithdrawalTransaction',
      gas.toString()
    )
    await new Promise((resolve) => setTimeout(resolve, 2000))
  }

  while (true) {
    try {
      await gasEstimationFinalSubmit()
      break
    } catch (e) {
      console.log(
        'Failed to get gas estimation for finalizeWithdrawalTransaction'
      )
    }
  }

  const preL1ETHBalance = await L1Wallet.getBalance()
  const finalSubmitTx = await OptimismPortal.finalizeWithdrawalTransaction([
    logs[0].args.nonce,
    logs[0].args.sender,
    logs[0].args.target,
    logs[0].args.value,
    logs[0].args.gasLimit,
    logs[0].args.data,
  ])
  await finalSubmitTx.wait()
  console.log(`Finalized withdrawal transaction!`)
  const postL1ETHBalance = await L1Wallet.getBalance()
  console.log(
    `L1 ETH Balance: ${ethers.utils.formatEther(
      postL1ETHBalance
    )} (was ${ethers.utils.formatEther(preL1ETHBalance)})`
  )
}

const bridgeBOBAToL1 = async () => {
  const config = await getConfigurations()
  if (!config) {
    return
  }
  const {
    L1Wallet,
    L2Wallet,
    L2OutputOracle,
    OptimismPortal,
    L2StandardBridge,
    L2ToL1MessagePasser,
    L1BOBA,
    L2BOBA,
  } = config
  console.log(`---------------------------------------------------`)

  const L1BOBABalance = await L1BOBA.balanceOf(L1Wallet.address)
  const L2BOBABalance = await L2BOBA.balanceOf(L1Wallet.address)
  console.log(`L1 BOBA Balance: ${ethers.utils.formatEther(L1BOBABalance)}`)
  console.log(`L2 BOBA Balance: ${ethers.utils.formatEther(L2BOBABalance)}`)
  console.log(`---------------------------------------------------`)

  const L2WithdrawAmount = ethers.utils.parseEther('0.01')
  if (L2BOBABalance.lt(L2WithdrawAmount)) {
    console.error('Insufficient L2 balance')
    return
  }

  const approveTx = await L2BOBA.approve(
    L2StandardBridge.address,
    L2WithdrawAmount
  )
  await approveTx.wait()

  const exitTx = await L2StandardBridge.withdraw(
    L2BOBA.address,
    L2WithdrawAmount,
    999999,
    '0x'
  )
  const receipt = await exitTx.wait()
  const L2BlockNumber = receipt.blockNumber

  console.log(
    `Sent ${ethers.utils.formatEther(
      L2WithdrawAmount
    )} BOBA to L2 Standard Bridge at block ${L2BlockNumber}`
  )

  const logs = await L2ToL1MessagePasser.queryFilter(
    L2ToL1MessagePasser.filters.MessagePassed(),
    L2BlockNumber,
    L2BlockNumber
  )
  if (logs.length !== 1) {
    console.error('!! Unknown error: logs.length != 1 !!')
    return
  }

  const types = ['uint256', 'address', 'address', 'uint256', 'uint256', 'bytes']
  const encoded = defaultAbiCoder.encode(types, [
    logs[0].args.nonce,
    logs[0].args.sender,
    logs[0].args.target,
    logs[0].args.value,
    logs[0].args.gasLimit,
    logs[0].args.data,
  ])
  const slot = keccak256(encoded)
  console.log(`Calculated withdrawHash: ${slot}`)

  const withdrawalHash = logs[0].args.withdrawalHash
  console.log(`withdrawalHash: ${withdrawalHash}`)

  if (withdrawalHash !== slot) {
    console.error('!! Unknown error: withdrawalHash !== slot !!')
    return
  }
  console.log(`Pass check! withdrawalHash === slot`)

  const messageSlot = ethers.utils.keccak256(
    ethers.utils.defaultAbiCoder.encode(
      ['bytes32', 'uint256'],
      [slot, ethers.constants.HashZero]
    )
  )

  const proof = await L2Wallet.provider.send('eth_getProof', [
    '0x4200000000000000000000000000000000000016',
    [messageSlot],
    L2BlockNumber,
  ])
  console.log(`Got proof: ${JSON.stringify(proof)}`)

  console.log(`---------------------------------------------------`)
  console.log(`Waiting for L2 block to be published...`)

  let latestBlockOnL1 = await L2OutputOracle.latestBlockNumber()
  while (latestBlockOnL1 < L2BlockNumber) {
    await new Promise((resolve) => setTimeout(resolve, 1000))
    latestBlockOnL1 = await L2OutputOracle.latestBlockNumber()
    console.log(
      `Waiting for L2 block: ${L2BlockNumber} - Latest L2 block on L1: ${latestBlockOnL1}`
    )
  }

  console.log(`---------------------------------------------------`)
  console.log(`L2 block published!`)
  const l2OutputIndex = await L2OutputOracle.getL2OutputIndexAfter(blockNumber)
  const proposal = await L2OutputOracle.getL2Output(l2OutputIndex)
  const proposalBlockNumber = proposal.l2BlockNumber
  const proposalBlock = await L2Web3.send('eth_getBlockByNumber', [
    proposalBlockNumber.toNumber(),
    false,
  ])
  const hash = keccak256(
    defaultAbiCoder.encode(
      ['bytes32', 'bytes32', 'bytes32', 'bytes32'],
      [
        ethers.constants.HashZero,
        proposalBlock.stateRoot,
        proof.storageHash,
        proposalBlock.hash,
      ]
    )
  )
  console.log(`Calculated L2 proposal hash: ${hash}`)

  const proveTx = await OptimismPortal.proveWithdrawalTransaction(
    [
      logs[0].args.nonce,
      logs[0].args.sender,
      logs[0].args.target,
      logs[0].args.value,
      logs[0].args.gasLimit,
      logs[0].args.data,
    ],
    l2OutputIndex,
    [
      ethers.constants.HashZero,
      proposalBlock.stateRoot,
      proof.storageHash,
      proposalBlock.hash,
    ],
    estimate.storageProof[0].proof
  )
  await proveTx.wait()
  console.log(`Proved withdrawal transaction!`)

  const gasEstimationFinalSubmit = async () => {
    const gas = await OptimismPortal.estimateGas.finalizeWithdrawalTransaction([
      logs[0].args.nonce,
      logs[0].args.sender,
      logs[0].args.target,
      logs[0].args.value,
      logs[0].args.gasLimit,
      logs[0].args.data,
    ])
    console.log(
      'Gas estimation for finalizeWithdrawalTransaction',
      gas.toString()
    )
    await new Promise((resolve) => setTimeout(resolve, 2000))
  }

  while (true) {
    try {
      await gasEstimationFinalSubmit()
      break
    } catch (e) {
      console.log(
        'Failed to get gas estimation for finalizeWithdrawalTransaction'
      )
    }
  }

  const preL1BOBABalance = await L1BOBA.balanceOf(L1Wallet.address)
  const finalSubmitTx = await OptimismPortal.finalizeWithdrawalTransaction([
    logs[0].args.nonce,
    logs[0].args.sender,
    logs[0].args.target,
    logs[0].args.value,
    logs[0].args.gasLimit,
    logs[0].args.data,
  ])
  await finalSubmitTx.wait()
  console.log(`Finalized withdrawal transaction!`)
  const postL1BOBABalance = await L1BOBA.balanceOf(L1Wallet.address)
  console.log(
    `L1 BOBA Balance: ${ethers.utils.formatEther(
      postL1BOBABalance
    )} (was ${ethers.utils.formatEther(preL1BOBABalance)})`
  )
}

const handleCommand = async (command) => {
  if (command === 'depositETH') {
    await bridgeETHToL2()
  }
  if (command === 'depositBOBA') {
    await bridgeBOBAToL2()
  }
  if (command === 'withdrawETH') {
    await bridgeETHToL1()
  }
  if (command === 'withdrawBOBA') {
    await bridgeBOBAToL1()
  }
  console.log('Command not found')
}

const main = async () => {
  const args = process.argv.slice(2)
  if (args.length === 0) {
    console.log('Please specify a command')
    return
  }
  if (args.length === 1) {
    const command = args[0]
    await handleCommand(command)
  }
  if (args.length === 2) {
    for (arg of args) {
      await handleCommand(arg)
    }
  }
}

main()
