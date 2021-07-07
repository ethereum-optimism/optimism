#!/usr/bin/env node

const ChainScannerService = require('../services/chain-scanner.service');

const main = async () => {
  const service = new ChainScannerService();
  await service.initChainScannerService();
  await service.startChainScannerService();
}

module.exports = main;