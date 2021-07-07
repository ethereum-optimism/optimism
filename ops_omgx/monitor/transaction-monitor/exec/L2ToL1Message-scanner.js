#!/usr/bin/env node

const L2ToL1MessageScanner = require('../services/L2ToL1Message-scanner.service');

const main = async () => {
  const service = new L2ToL1MessageScanner();
  await service.initL2ToL1MessageScanner();
  await service.startL2ToL1MessageScanner();
}

module.exports = main;