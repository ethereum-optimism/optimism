import { ethers } from 'hardhat'

export const SECONDS_IN_1_DAY = 24 * 60 * 60
export const SECONDS_IN_365_DAYS = 365 * 24 * 60 * 60

export const getBlockTimestamp = async (blockNumber: number) => {
  const block = await ethers.provider.getBlock(blockNumber)
  return block.timestamp
}

export const fastForwardDays = async (numberOfDays: number) => {
  const latestBlock = await ethers.provider.getBlock('latest')
  const timestampAfterXDays =
    latestBlock.timestamp + numberOfDays * SECONDS_IN_1_DAY
  await ethers.provider.send('evm_setNextBlockTimestamp', [timestampAfterXDays])
  await ethers.provider.send('evm_mine', [])
}
