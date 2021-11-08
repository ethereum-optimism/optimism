const fs = require("fs")
const { deployed, getBlockRlp, getTrieNodesForCall } = require("../scripts/lib")

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

  // TODO: move this to lib, it's shared with the test
  let startTrie = JSON.parse(fs.readFileSync("/tmp/cannon/golden.json"))
  /*const assertionRoot = "0x1111111111111111111111111111111111111111111111111111111111111111"
  let finalTrie = JSON.parse(fs.readFileSync("/tmp/cannon/0_"+blockNumberN.toString()+"/checkpoint_final.json"))*/

  // fake for testing (it's the next block)
  const assertionRoot = "0xb135cb00efbc2341905eafc034eca0dcec40b039a1b28860bf7c309c872e5644"
  let finalTrie = JSON.parse(fs.readFileSync("/tmp/cannon/0_1171896/checkpoint_final.json"))

  let preimages = Object.assign({}, startTrie['preimages'], finalTrie['preimages']);
  const finalSystemState = finalTrie['root']

  let args = [blockNumberN, blockNp1Rlp, assertionRoot, finalSystemState, finalTrie['step']]
  let cdat = c.interface.encodeFunctionData("InitiateChallenge", args)
  let nodes = await getTrieNodesForCall(c, cdat, preimages)

  // run "on chain"
  for (n of nodes) {
    await mm.AddTrieNode(n)
  }
  let ret = await c.InitiateChallenge(...args)
  let receipt = await ret.wait()
  // ChallengeCreate event
  let challengeId = receipt.events[0].args['challengeId'].toNumber()
  console.log("new challenge with id", challengeId)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
