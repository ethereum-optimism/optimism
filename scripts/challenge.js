const { deployed, getBlockRlp } = require("../scripts/lib")

async function main() {
  let [c, m, mm] = await deployed()

  const blockNumberN = parseInt(process.env.BLOCK)
  if (isNaN(blockNumberN)) {
    throw "usage: challenge.js <block number>"
  }
  console.log("challenging block number", blockNumberN)
  // sadly this doesn't work on hosthat
  const blockNp1 = await network.provider.send("eth_getBlockByNumber", ["0x"+(blockNumberN+1).toString(16), true])
  console.log(blockNp1)
  const blockNp1Rlp = getBlockRlp(blockNp1)

  console.log(c.address, m.address, mm.address)
  // TODO: finish this
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
