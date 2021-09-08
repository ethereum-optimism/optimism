#!/usr/bin/env node

const BlockMonitorService = require('../services/blockMonitor');
const stateRootMonitorService = require('../services/stateRootMonitor');
const exitMonitorService = require('../services/exitMonitor');
const l1BridgeMonitorService = require('../services/l1BridgeMonitor');
const { sleep } = require('@eth-optimism/core-utils');

const loop = async (func) => {
  while (true) {
    try {
      await func();
    } catch (error) {
      console.log('Unhandled exception during monitor service', {
        message: error.toString(),
        stack: error.stack,
        code: error.code,
      });
      await sleep(1000);
    }
  }
}

const main = async () => {
  // l1 bridge monitor
  const l1BridgeService = new l1BridgeMonitorService();
  await l1BridgeService.initConnection();

  loop(() => l1BridgeService.startL1BridgeMonitor());
  loop(() => l1BridgeService.startCrossDomainMessageMonitor());

  // liquidity pool
  const exitService = new exitMonitorService();
  await exitService.initConnection();

  loop(() => exitService.startExitMonitor())

  // state root
  const stateRootService = new stateRootMonitorService();
  await stateRootService.initConnection();

  loop(() => stateRootService.startStateRootMonitor())

  // block
  const blockService = new BlockMonitorService();
  await blockService.initConnection();
  await blockService.initScan();

  loop(() => blockService.startTransactionMonitor())
  loop(() => blockService.startCrossDomainMessageMonitor())
}

(async () => {
  main();
})().catch((err) => {
    console.log(err)
  process.exit(1)
})
