const { keccak256 } = require("@ethersproject/keccak256");
const { expect } = require("chai");
const { writeMemory } = require("../scripts/lib")

describe("MIPSMemory oracle", function () {
  beforeEach(async function () {
    const MIPSMemory = await ethers.getContractFactory("MIPSMemory")
    mm = await MIPSMemory.deploy()
    await mm.AddTrieNode(new Uint8Array([0x80]))
  })
  it("simple oracle", async function() {
    // add in the preimage at offset 4
    const data = [0x11,0x22,0x33,0x44,0xaa,0xbb,0xcc,0xdd]
    const hash = keccak256(data)
    await mm.AddPreimage(data, 4)

    // write the oracle selection address
    let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
    root = await writeMemory(mm, root, 0x30001000, hash, true)

    // length is 8
    expect(await mm.ReadMemory(root, 0x31000000)).to.equal(8)

    // offset 4 is 0xaabbccdd
    expect(await mm.ReadMemory(root, 0x31000008)).to.equal(0xaabbccdd)
  })
})