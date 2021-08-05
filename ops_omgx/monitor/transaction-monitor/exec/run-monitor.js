#!/usr/bin/env node

const MonitorService = require('../services/monitor.service');

const loop = async (func) => {
  while (true) {
    try {
      await func();
    } catch (error) {
      console.log('Unhandled exception during monitor transaction and cross domain message', {
        message: error.toString(),
        stack: error.stack,
        code: error.code,
      });
      break;
    }
  }
}

const main = async () => {
  const service = new MonitorService();
  await service.initConnection();
  await service.initScan();

  loop(() => service.startTransactionMonitor())
  loop(() => service.startCrossDomainMessageMonitor())
}

(async () => {
  main();
})().catch((err) => {
    console.log(err)
  process.exit(1)
})
