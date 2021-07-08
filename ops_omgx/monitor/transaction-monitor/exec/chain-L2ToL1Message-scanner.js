const mainChainScanner = require('./chain-scanner');
const L1ToL2MessageScanner = require('./L2ToL1Message-scanner');

(async () => {
    mainChainScanner();
    L1ToL2MessageScanner();
})().catch((err) => {
    console.log(err)
    process.exit(1)
  })