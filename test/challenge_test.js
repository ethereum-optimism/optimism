const { expect } = require("chai");

// golden minigeth.bin hash
const goldenRoot = "0x9c15aa86416a3a9d3b15188fc9f9be59626c1f83a33e5d63b58ca1bf0f8cef71"

describe("Challenge contract", function () {
  beforeEach(async function () {
    // this mips can be reused for other challenges
    const MIPS = await ethers.getContractFactory("MIPS")
    const m = await MIPS.deploy()

    const Challenge = await ethers.getContractFactory("Challenge")
    c = await Challenge.deploy(m.address, goldenRoot)
  })
  it("challenge contract deploys", async function() {
    console.log("Challenge deployed at", c.address)
  })
  it("initiate challenge", async function() {
    // TODO: is there a better way to get the "HardhatNetworkProvider"?
    const hardhat = network.provider._wrapped._wrapped._wrapped._wrapped._wrapped
    const blockchain = hardhat._node._blockchain

    // get data
    const blockNumberN = (await ethers.provider.getBlockNumber())-1;
    const blockNp1 = blockchain._data._blocksByNumber.get(blockNumberN+1)
    const blockNp1Rlp = blockNp1.header.serialize()

    const assertionRoot = "0x1337133713371337133713371337133713371337133713371337133713371337"
    // TODO: compute a valid one of these
    const finalSystemState = "0x1337133713371337133713371337133713371337133713371337133713371337"

    await c.InitiateChallenge(blockNumberN, blockNp1Rlp, assertionRoot, finalSystemState, 1)

    //const blockHeaderNp1 = getBlockRlp(await ethers.provider.getBlock(blockNumberN+1));
    //console.log(blockNumberN, blockHeaderNp1);
  })
})