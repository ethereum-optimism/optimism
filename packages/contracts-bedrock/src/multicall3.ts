import { ethers } from 'ethers'

// Multicall3 abi
// Multicall3 address: 0xcA11bde05977b3631167028862bE2a173976CA11
// Multicall3 github: https://github.com/mds1/multicall
export const MULTICALL3_ABI = [
  'function aggregate(tuple(address target, bytes callData)[] calls) payable returns (uint256 blockNumber, bytes[] returnData)',
  'function aggregate3(tuple(address target, bool allowFailure, bytes callData)[] calls) payable returns (tuple(bool success, bytes returnData)[] returnData)',
  'function aggregate3Value(tuple(address target, bool allowFailure, uint256 value, bytes callData)[] calls) payable returns (tuple(bool success, bytes returnData)[] returnData)',
  'function blockAndAggregate(tuple(address target, bytes callData)[] calls) payable returns (uint256 blockNumber, bytes32 blockHash, tuple(bool success, bytes returnData)[] returnData)',
  'function getBasefee() view returns (uint256 basefee)',
  'function getBlockHash(uint256 blockNumber) view returns (bytes32 blockHash)',
  'function getBlockNumber() view returns (uint256 blockNumber)',
  'function getChainId() view returns (uint256 chainid)',
  'function getCurrentBlockCoinbase() view returns (address coinbase)',
  'function getCurrentBlockDifficulty() view returns (uint256 difficulty)',
  'function getCurrentBlockGasLimit() view returns (uint256 gaslimit)',
  'function getCurrentBlockTimestamp() view returns (uint256 timestamp)',
  'function getEthBalance(address addr) view returns (uint256 balance)',
  'function getLastBlockHash() view returns (bytes32 blockHash)',
  'function tryAggregate(bool requireSuccess, tuple(address target, bytes callData)[] calls) payable returns (tuple(bool success, bytes returnData)[] returnData)',
  'function tryBlockAndAggregate(bool requireSuccess, tuple(address target, bytes callData)[] calls) payable returns (uint256 blockNumber, bytes32 blockHash, tuple(bool success, bytes returnData)[] returnData)',
]

// Multicall3Call3 is a Multicall3-specific struct to pass into aggregate3.
interface Multicall3Call3 {
  target: string
  allowFailure: boolean
  callData: string
}

/**
 * Constructs a Multicall3 Contract instance with the provided optional signer or provider.
 *
 * @param signerOrProvider an optional signer or provider to pass into the ethers Contract object.
 * @returns A Multicall3 ethers Contract instance.
 */
export const multicall3Contract = (
  signerOrProvider?: ethers.providers.Provider | ethers.Signer
): ethers.Contract => {
  // Create interface
  const multicall3Interface = new ethers.utils.Interface(MULTICALL3_ABI)

  // Create multicall contract
  const MULTICALL3_ADDRESS = '0xcA11bde05977b3631167028862bE2a173976CA11'
  return new ethers.Contract(
    MULTICALL3_ADDRESS,
    multicall3Interface,
    signerOrProvider
  )
}

/**
 * Create Multicall3Call3 Objects from a list of FeeVault addresses
 *
 * @param targets A list of FeeVault addresses to construct multicall3 withdraw calls.
 * @param method The method to call on the FeeVault.
 * @returns A list of Multicall3Call3 objects.
 */
export const constructFeeVaultMulticalls = (
  targets: string[],
  method: string
): Multicall3Call3[] => {
  return targets.map((t) => {
    return {
      target: t,
      allowFailure: false,
      callData: new ethers.utils.Interface(MULTICALL3_ABI).encodeFunctionData(
        method
      ),
    }
  })
}

/**
 * Estimate Multicall3 aggregate3 gas cost.
 *
 * @param multicall3 A Multicall3 ethers Contract instance.
 * @param calls A list of Multicall3Call3 objects to pass into aggregate3.
 * @returns A gas estimate for the Multicall3 aggregate3 call.
 */
export const estimateMulticall = async (
  multicall3?: ethers.Contract,
  calls?: Multicall3Call3[]
) => {
  if (!multicall3) {
    console.log(`Missing Multicall3 contract... constructing new one.`)
    multicall3 = multicall3Contract()
  }
  console.log(`Dry running multicall3...`)
  const estimatedGas = await multicall3.estimateGas.aggregate3(
    calls ? calls : []
  )
  console.log(`Estimated gas: ${estimatedGas}`)
}

/**
 * Executes the multicall3 calls.
 *
 * @param multicall3 A Multicall3 ethers Contract instance.
 * @param calls A list of Multicall3Call3 objects to pass into aggregate3.
 * @returns The return value of the aggregate3 call.
 */
export const executeMulticalls = async (
  multicall3?: ethers.Contract,
  calls?: Multicall3Call3[]
) => {
  if (!multicall3) {
    console.log(`Missing Multicall3 contract... constructing new one.`)
    multicall3 = multicall3Contract()
  }
  const multicall = await multicall3.aggregate3(calls ? calls : [])
  console.log(
    `Withdrawals complete: https://optimistic.etherscan.io/tx/${multicall.hash}`
  )
  for (const call of calls) {
    console.log(
      `Complete withdrawal in 1 week here: https://optimistic.etherscan.io/address/${call.target}#withdrawaltxs`
    )
  }
}
