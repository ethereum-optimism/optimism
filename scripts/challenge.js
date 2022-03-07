const fs = require("fs")
const { basedir, deployed, getBlockRlp, getTrieNodesForCall } = require("../scripts/lib")

async function main() {
  let [c, m, mm] = await deployed()

  const blockNumberN = parseInt(process.env.BLOCK)
  if (isNaN(blockNumberN)) {
    throw "usage: BLOCK=<number> npx hardhat run challenge.js"
  }
  console.log("challenging block number", blockNumberN)
  // sadly this doesn't work on hosthat
  const blockNp1 = await network.provider.send("eth_getBlockByNumber", ["0x"+(blockNumberN+1).toString(16), false])
  console.log(blockNp1)
  const blockNp1Rlp = getBlockRlp(blockNp1)

  console.log(c.address, m.address, mm.address)

  // TODO: move this to lib, it's shared with the test
  let startTrie = JSON.parse(fs.readFileSync(basedir+"/golden.json"))

  const assertionRootBinary = fs.readFileSync(basedir+"/0_"+blockNumberN.toString()+"/output")
  var assertionRoot = "0x"
  for (var i=0; i<32; i++) {
    hex = assertionRootBinary[i].toString(16);
    assertionRoot += ("0"+hex).slice(-2);
  }
  console.log("asserting root", assertionRoot)
  let finalTrie = JSON.parse(fs.readFileSync(basedir+"/0_"+blockNumberN.toString()+"/checkpoint_final.json"))

  let preimages = Object.assign({}, startTrie['preimages'], finalTrie['preimages']);
  const finalSystemState = finalTrie['root']

  let args = [blockNumberN, blockNp1Rlp, assertionRoot, finalSystemState, finalTrie['step']]
  let cdat = c.interface.encodeFunctionData("initiateChallenge", args)
  let nodes = await getTrieNodesForCall(c, c.address, cdat, preimages)

  // run "on chain"
  for (n of nodes) {
    await mm.AddTrieNode(n)
  }
  let ret = await c.initiateChallenge(...args)
  let receipt = await ret.wait()
  // ChallengeCreated event
  let challengeId = receipt.events[0].args['challengeId'].toNumber()
  console.log("new challenge with id", challengeId)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
