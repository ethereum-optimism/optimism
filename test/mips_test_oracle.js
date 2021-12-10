const { keccak256 } = require("@ethersproject/keccak256");
const { expect } = require("chai");

const chai = require("chai");
const { solidity } = require("ethereum-waffle");
chai.use(solidity);

const { writeMemory } = require("../scripts/lib")

async function loadPreimageAndSelect(mm, data, offset) {
  // add in the preimage at offset 4
  const hash = keccak256(data)
  await mm.AddPreimage(data, offset)

  // write the oracle selection address
  let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
  root = await writeMemory(mm, root, 0x30001000, hash, true)
  return root
}

describe("MIPSMemory oracle", function () {
  beforeEach(async function () {
    const MIPSMemory = await ethers.getContractFactory("MIPSMemory")
    mm = await MIPSMemory.deploy()
    await mm.AddTrieNode(new Uint8Array([0x80]))
  })
  it("simple oracle", async function() {
    root = await loadPreimageAndSelect(mm, [0x11,0x22,0x33,0x44,0xaa,0xbb,0xcc,0xdd], 4)

    // length is 8
    expect(await mm.ReadMemory(root, 0x31000000)).to.equal(8)

    // offset 4 is 0xaabbccdd
    expect(await mm.ReadMemory(root, 0x31000008)).to.equal(0xaabbccdd)

    // offset 0 isn't loaded
    await expect(mm.ReadMemory(root, 0x31000004)).to.be.reverted;
  })

  it("misaligned oracle", async function() {
    root = await loadPreimageAndSelect(mm, [0x11,0x22,0x33,0x44,0xaa,0xbb,0xcc], 4)
    expect(await mm.ReadMemory(root, 0x31000000)).to.equal(7)
    expect(await mm.ReadMemory(root, 0x31000008)).to.equal(0xaabbcc00)
  })
})