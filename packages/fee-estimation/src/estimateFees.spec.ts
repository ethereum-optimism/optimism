/**
 * The first 2 test cases are good documentation of how to use this library
 */
import { vi, test, expect, beforeEach } from 'vitest'
import { formatEther } from 'viem/utils'
import {
  baseFee,
  decimals,
  estimateFees,
  gasPrice,
  getL1Fee,
  getL1GasUsed,
  getL2Client,
  l1BaseFee,
  overhead,
  scalar,
  version,
} from './estimateFees'
import {
  optimistABI,
  optimistAddress,
  l2StandardBridgeABI,
  l2StandardBridgeAddress,
} from '@eth-optimism/contracts-ts'
import { parseEther, parseGwei } from 'viem'

vi.mock('viem', async () => {
  const _viem = (await vi.importActual('viem')) as any
  return {
    ..._viem,
    // no way to get historical gas price
    createPublicClient: (...args: [any]) => {
      const client = _viem.createPublicClient(...args)
      client.getGasPrice = async () => parseGwei('0.00000042')
      return client
    },
  }
})

// using this optimist https://optimistic.etherscan.io/tx/0xaa291efba7ea40b0742e5ff84a1e7831a2eb6c2fc35001fa03ba80fd3b609dc9
const blockNumber = BigInt(107028270)
const optimistOwnerAddress =
  '0x77194aa25a06f932c10c0f25090f3046af2c85a6' as const
const functionDataBurn = {
  functionName: 'burn',
  // this is an erc721 abi
  abi: optimistABI,
  args: [BigInt(optimistOwnerAddress)],
  account: optimistOwnerAddress,
  to: optimistAddress[10],
  chainId: 10,
} as const
const functionDataBurnWithPriorityFees = {
  ...functionDataBurn,
  maxFeePerGas: parseGwei('2'),
  maxPriorityFeePerGas: parseGwei('2'),
} as const
// This tx
// https://optimistic.etherscan.io/tx/0xe6f3719be7327a991b9cb562ebf8d979cbca72bbdb2775f55a18274f4d0c9bbf
const functionDataWithdraw = {
  abi: l2StandardBridgeABI,
  functionName: 'withdraw',
  value: BigInt(parseEther('0.00000001')),
  account: '0x6387a88a199120aD52Dd9742C7430847d3cB2CD4',
  // currently a bug is making chain id 10 not exist
  to: l2StandardBridgeAddress[420],
  chainId: 10,
  args: [
    // l2 token address
    '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
    // amount
    BigInt(parseEther('0.00000001')),
    // l1 gas
    0,
    // extra data
    '0x0',
  ],
  maxFeePerGas: parseGwei('.2'),
  maxPriorityFeePerGas: parseGwei('.1'),
} as const

const clientParams = {
  chainId: functionDataBurn.chainId,
  rpcUrl: process.env.VITE_L2_RPC_URL ?? 'https://mainnet.optimism.io',
} as const

const viemClient = getL2Client(clientParams)

const paramsWithRpcUrl = {
  client: clientParams,
  blockNumber,
} as const
const paramsWithViemClient = {
  client: viemClient,
  viemClient,
  blockNumber,
} as const
const blockNumberWithdraw = BigInt(107046472)
const paramsWithRpcUrlWithdraw = {
  client: clientParams,
  blockNumber: blockNumberWithdraw,
} as const

beforeEach(() => {
  vi.resetAllMocks()
})

test('estimateFees should return correct fees', async () => {
  // burn
  const res = await estimateFees({ ...paramsWithRpcUrl, ...functionDataBurn })
  expect(res).toMatchInlineSnapshot('20573203833264n')
  expect(formatEther(res)).toMatchInlineSnapshot('"0.000020573203833264"')
  expect(
    await estimateFees({ ...paramsWithRpcUrl, ...functionDataBurn })
  ).toMatchInlineSnapshot('20573203833264n')
  expect(
    await estimateFees({ ...paramsWithViemClient, ...functionDataBurn })
  ).toMatchInlineSnapshot('20573203833264n')
  expect(
    await estimateFees({
      ...paramsWithRpcUrl,
      ...functionDataBurnWithPriorityFees,
    })
  ).toMatchInlineSnapshot('21536992690265n')
  // what is the l2 and l1 part of the fees for reference?
  const l1Fee = await getL1Fee({ ...paramsWithRpcUrl, ...functionDataBurn })
  const l2Fee = res - l1Fee
  expect(l1Fee).toMatchInlineSnapshot('20573185216764n')
  expect(formatEther(l1Fee)).toMatchInlineSnapshot('"0.000020573185216764"')
  expect(l2Fee).toMatchInlineSnapshot('18616500n')
  expect(formatEther(l2Fee)).toMatchInlineSnapshot('"0.0000000000186165"')

  // withdraw
  const res2 = await estimateFees({
    ...paramsWithRpcUrlWithdraw,
    ...functionDataWithdraw,
  })
  expect(res2).toMatchInlineSnapshot('62857090247510n')
  expect(
    await estimateFees({ ...paramsWithRpcUrlWithdraw, ...functionDataWithdraw })
  ).toMatchInlineSnapshot('62857090247510n')
  expect(
    await estimateFees({ ...paramsWithRpcUrlWithdraw, ...functionDataWithdraw })
  ).toMatchInlineSnapshot('62857090247510n')
  expect(
    await estimateFees({ ...paramsWithRpcUrlWithdraw, ...functionDataWithdraw })
  ).toMatchInlineSnapshot('62857090247510n')
  expect(formatEther(res2)).toMatchInlineSnapshot('"0.00006285709024751"')
  // what is the l2 and l1 part of the fees for reference?
  const l1Fee2 = await getL1Fee({
    ...paramsWithRpcUrlWithdraw,
    ...functionDataWithdraw,
  })
  const l2Fee2 = res2 - l1Fee
  expect(l1Fee2).toMatchInlineSnapshot('62857038894110n')
  expect(formatEther(l1Fee2)).toMatchInlineSnapshot('"0.00006285703889411"')
  expect(l2Fee2).toMatchInlineSnapshot('42283905030746n')
  expect(formatEther(l2Fee2)).toMatchInlineSnapshot('"0.000042283905030746"')
})

test('baseFee should return the correct result', async () => {
  expect(await baseFee(paramsWithRpcUrl)).toMatchInlineSnapshot('64n')
  expect(await baseFee(paramsWithViemClient)).toMatchInlineSnapshot('64n')
})

test('decimals should return the correct result', async () => {
  expect(await decimals(paramsWithRpcUrl)).toMatchInlineSnapshot('6n')
  expect(await decimals(paramsWithViemClient)).toMatchInlineSnapshot('6n')
})

test('gasPrice should return the correct result', async () => {
  expect(await gasPrice(paramsWithRpcUrl)).toMatchInlineSnapshot('64n')
  expect(await gasPrice(paramsWithViemClient)).toMatchInlineSnapshot('64n')
})

test('getL1Fee should return the correct result', async () => {
  // burn
  expect(
    await getL1Fee({ ...paramsWithRpcUrl, ...functionDataBurn })
  ).toMatchInlineSnapshot('20573185216764n')
  expect(
    await getL1Fee({ ...paramsWithViemClient, ...functionDataBurn })
  ).toMatchInlineSnapshot('20573185216764n')
  expect(
    await getL1Fee({
      ...paramsWithViemClient,
      ...functionDataBurnWithPriorityFees,
    })
  ).toMatchInlineSnapshot('21536974073765n')
  expect(
    formatEther(
      await getL1Fee({ ...paramsWithViemClient, ...functionDataBurn })
    )
  ).toMatchInlineSnapshot('"0.000020573185216764"')
  // withdraw
  expect(
    await getL1Fee({ ...paramsWithRpcUrlWithdraw, ...functionDataWithdraw })
  ).toMatchInlineSnapshot('62857038894110n')
  expect(
    formatEther(
      await getL1Fee({ ...paramsWithRpcUrlWithdraw, ...functionDataWithdraw })
    )
  ).toMatchInlineSnapshot('"0.00006285703889411"')
})

test('getL1GasUsed should return the correct result', async () => {
  // burn
  expect(
    await getL1GasUsed({ ...paramsWithRpcUrl, ...functionDataBurn })
  ).toMatchInlineSnapshot('2220n')
  expect(
    await getL1GasUsed({ ...paramsWithViemClient, ...functionDataBurn })
  ).toMatchInlineSnapshot('2220n')
  expect(
    await getL1GasUsed({
      ...paramsWithViemClient,
      ...functionDataBurnWithPriorityFees,
    })
  ).toMatchInlineSnapshot('2324n')
  // withdraw
  expect(
    await getL1GasUsed({ ...paramsWithRpcUrlWithdraw, ...functionDataWithdraw })
  ).toMatchInlineSnapshot('2868n')
})

test('l1BaseFee should return the correct result', async () => {
  expect(await l1BaseFee(paramsWithRpcUrl)).toMatchInlineSnapshot(
    '13548538813n'
  )
  expect(await l1BaseFee(paramsWithViemClient)).toMatchInlineSnapshot(
    '13548538813n'
  )
})

test('overhead should return the correct result', async () => {
  expect(await overhead(paramsWithRpcUrl)).toMatchInlineSnapshot('188n')
  expect(await overhead(paramsWithViemClient)).toMatchInlineSnapshot('188n')
})

test('scalar should return the correct result', async () => {
  expect(await scalar(paramsWithRpcUrl)).toMatchInlineSnapshot('684000n')
  expect(await scalar(paramsWithViemClient)).toMatchInlineSnapshot('684000n')
})

test('version should return the correct result', async () => {
  expect(await version(paramsWithRpcUrl)).toMatchInlineSnapshot('"1.0.0"')
  expect(await version(paramsWithViemClient)).toMatchInlineSnapshot('"1.0.0"')
})
