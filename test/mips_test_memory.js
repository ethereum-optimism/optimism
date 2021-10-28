const { expect } = require("chai");

async function write(mm, root, addr, data) {
  ret = await mm.WriteMemoryWithReceipt(root, addr, data)
  const receipt = await ret.wait()
  for (l of receipt.logs) {
    if (l.topics[0] == "0x86b89b5c9818dbbf520dd979a5f250d357508fe11b9511d4a43fd9bc6aa1be70") {
      root = l.data
    }
  }
  console.log("new hash", root)
  return root
}

describe("MIPSMemory contract", function () {
  beforeEach(async function () {
    const MIPSMemory = await ethers.getContractFactory("MIPSMemory")
    mm = await MIPSMemory.deploy()
  })
  it("write from new should work", async function() {
    await mm.AddTrieNode(new Uint8Array([0x80]))
    let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"

    root = await write(mm, root, 0, 1)
    root = await write(mm, root, 4, 2)

    expect(await mm.ReadMemory(root, 0)).to.equal(1)
    expect(await mm.ReadMemory(root, 4)).to.equal(2)
  })
  it("write three should work", async function() {
    await mm.AddTrieNode(new Uint8Array([0x80]))
    let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"

    root = await write(mm, root, 0, 1)
    root = await write(mm, root, 4, 2)
    root = await write(mm, root, 0x40, 3)

    expect(await mm.ReadMemory(root, 0)).to.equal(1)
    expect(await mm.ReadMemory(root, 4)).to.equal(2)
    expect(await mm.ReadMemory(root, 0x40)).to.equal(3)
  })
})
