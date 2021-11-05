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

function randint(n) {
  return Math.floor(Math.random() * n)
}

describe("MIPSMemory contract", function () {
  beforeEach(async function () {
    const MIPSMemory = await ethers.getContractFactory("MIPSMemory")
    mm = await MIPSMemory.deploy()
    await mm.AddTrieNode(new Uint8Array([0x80]))
  })
  it("write from new should work", async function() {
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
  it("write other three should work", async function() {
    let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"

    root = await write(mm, root, 0x7fffd00c, 1)
    root = await write(mm, root, 0x7fffd010, 2)
    root = await write(mm, root, 0x7fffcffc, 3)

    expect(await mm.ReadMemory(root, 0x7fffd00c)).to.equal(1)
    expect(await mm.ReadMemory(root, 0x7fffd010)).to.equal(2)
    expect(await mm.ReadMemory(root, 0x7fffcffc)).to.equal(3)
  })
  it("bug found fuzzing 1", async function() {
    let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
    root = await write(mm, root, 0, 0)
    root = await write(mm, root, 0, 1)
    root = await write(mm, root, 0, 2)
  })
  /*it("fuzzing should be okay", async function() {
    let root = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
    let kv = {}

    for (var i = 0; i < 100; i++) {
      const keys = Object.keys(kv)
      const choice = Math.random()
      if (choice < 0.5 || keys.length == 0) {
        // write new key
        const key = randint(0x100)*4
        const value = randint(0x100000000)
        console.log("writing", key, value)
        root = await write(mm, root, key, value)
        kv[key] = value
      } else if (choice > 0.7) {
        // read old key
        const idx = randint(keys.length)
        const key = keys[idx]
        console.log("reading", key)
        expect(await mm.ReadMemory(root, key)).to.equal(kv[key])
      } else {
        // rewrite old key
        const idx = randint(keys.length)
        const key = keys[idx]
        const value = randint(0x100000000)
        console.log("writing", key, value)
        root = await write(mm, root, key, value)
        kv[key] = value
      }
    }
  })*/
})
