const { expect } = require("chai")
const fs = require("fs")
const { deploy, getTrieNodesForCall } = require("../scripts/lib")

describe("Challenge contract", function () {
  if (!fs.existsSync("/tmp/cannon/golden.json")) {
    console.log("golden file doesn't exist, skipping test")
    return
  }

  beforeEach(async function () {
    [c, m, mm] = await deploy()
  })
  it("challenge contract deploys", async function() {
    console.log("Challenge deployed at", c.address)
  })
  it("initiate challenge", async function() {
    // TODO: is there a better way to get the "HardhatNetworkProvider"?
    const hardhat = network.provider._wrapped._wrapped._wrapped._wrapped._wrapped
    const blockchain = hardhat._node._blockchain

    // get data
    const blockNumberN = (await ethers.provider.getBlockNumber())-2
    const blockNp1 = blockchain._data._blocksByNumber.get(blockNumberN+1)
    const blockNp1Rlp = blockNp1.header.serialize()

    const assertionRoot = "0x9e0261efe4509912b8862f3d45a0cb8404b99b239247df9c55871bd3844cebbd"
    let startTrie = JSON.parse(fs.readFileSync("/tmp/cannon/golden.json"))
    let finalTrie = JSON.parse(fs.readFileSync("/tmp/cannon/0_13284469/checkpoint_final.json"))
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

    // the real issue here is from step 0->1 when we write the input hash
    // TODO: prove the challenger wrong?
  }).timeout(120000)
})
