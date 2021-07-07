#!/usr/bin/env node

const main = require('./L2ToL1Message-scanner');

(async () => {
  await main()
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
