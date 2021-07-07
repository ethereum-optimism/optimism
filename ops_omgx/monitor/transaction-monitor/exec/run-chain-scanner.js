#!/usr/bin/env node

const main = require('./chain-scanner');

(async () => {
  await main()
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
