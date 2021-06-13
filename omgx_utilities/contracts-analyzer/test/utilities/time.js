const { ethers } = require("hardhat")
const { BigNumber, providers } = require('ethers');
const { Provider } = require('./wallet');

async function advanceBlock() {
  const mine = await Provider.send("evm_mine", [])
  await mine.wait()
}

async function advanceBlockTo(blockNumber) {
  const block = await Provider.getBlockNumber()
  console.log({ block })
  for (let i = await Provider.getBlockNumber(); i < blockNumber; i++) {
    await advanceBlock()
  }
}

async function increase(value) {
  const increaseTime = await Provider.send("evm_increaseTime", [value.toNumber()])
  console.log(increaseTime)
  await increaseTime.wait()
  await advanceBlock()
}

async function latest() {
  const block = await Provider.getBlock("latest")
  return BigNumber.from(block.timestamp)
}

async function advanceTimeAndBlock(time) {
  await advanceTime(time)
  await advanceBlock()
}

async function advanceTime(time) {
  await Provider.send("evm_increaseTime", [time])
}

const duration = {
  seconds: function (val) {
    return BigNumber.from(val)
  },
  minutes: function (val) {
    return BigNumber.from(val).mul(this.seconds("60"))
  },
  hours: function (val) {
    return BigNumber.from(val).mul(this.minutes("60"))
  },
  days: function (val) {
    return BigNumber.from(val).mul(this.hours("24"))
  },
  weeks: function (val) {
    return BigNumber.from(val).mul(this.days("7"))
  },
  years: function (val) {
    return BigNumber.from(val).mul(this.days("365"))
  },
}

module.exports = {
  advanceBlock,
  advanceBlockTo,
  increase,
  latest,
  advanceTimeAndBlock,
  advanceTime,
  duration,
}