const { expect } = require("chai");
const { execPath } = require("process");

const trieAdd = {"root":"0x22ffce7c56d926c2d8d6337d8917fa0e1880e1869e189c15385ead63c6c45b93","preimages":{"0x044371dc86fb8c621bc84b69dce16de366de1126777250888b17416d0bd11279":"+FPGIIQ8EL//xiCENhD/8MYghDQRAAHGIIQ8CP//xiCENQj//cYghDQJAAPGIIQBCVAgxiCELUIAAcYghK4CAAjGIISuEQAExiCEA+AACICAgICAgA==","0x0fdfcc24b1b21d78ef2b7c6503eb9354677743685c2d00a14a8b502a177911b0":"+HGgL4Jb+u0gEWM4e9G4lO/GsyUEY/heVoGOAfiI04qPXfSgLCZprT7WBOLipiwJxxI0vy09rw9iPR+x0p/Xz1p3X5WgaNY/x30waJPsd6PWg76b094l8vUmmL6XB1XdUFy+xfWAgICAgICAgICAgICAgA==","0x11228d4f4a028a9088e6ec0aa6513e0d4731d9dc488e2af1957e46ba80624a69":"5oQAAAAAoARDcdyG+4xiG8hLadzhbeNm3hEmd3JQiIsXQW0L0RJ5","0x22ffce7c56d926c2d8d6337d8917fa0e1880e1869e189c15385ead63c6c45b93":"+FGgESKNT0oCipCI5uwKplE+DUcx2dxIjirxlX5GuoBiSmmAgKBv5gezlmGtxjQAs8Du76D93mAxExw5qWgAZjJQp1xmfICAgICAgICAgICAgIA=","0x2c2669ad3ed604e2e2a62c09c71234bf2d3daf0f623d1fb1d29fd7cf5a775f95":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIRerQAAgA==","0x2f825bfaed201163387bd1b894efc6b3250463f85e56818e01f888d38a8f5df4":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAgA==","0x68d63fc77d306893ec77a3d683be9bd3de25f2f52698be970755dd505cbec5f5":"6cYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAACAgICAgICAgICAgICA","0x6fe607b39661adc63400b3c0eeefa0fdde6031131c39a96800663250a75c667c":"5YMQAACgD9/MJLGyHXjvK3xlA+uTVGd3Q2hcLQChSotQKhd5EbA="}};
const trieOracle = {"root":"0x26595bd4f73f14273d9f43f0c5253eef964f4581d5e293cded54d23f1cf3db5f","step":-1,"preimages":{"0x0fdfcc24b1b21d78ef2b7c6503eb9354677743685c2d00a14a8b502a177911b0":"+HGgL4Jb+u0gEWM4e9G4lO/GsyUEY/heVoGOAfiI04qPXfSgLCZprT7WBOLipiwJxxI0vy09rw9iPR+x0p/Xz1p3X5WgaNY/x30waJPsd6PWg76b094l8vUmmL6XB1XdUFy+xfWAgICAgICAgICAgICAgA==","0x16e7f9821e0a3a2fc92d337f6d269c4c0ddb4d5c10a04b49d368643c9818ec1d":"5YMQAACgoLKCLk7kk3rBB2CtW7YQizz670HG0TNCobpfs/1MJqs=","0x26595bd4f73f14273d9f43f0c5253eef964f4581d5e293cded54d23f1cf3db5f":"+FGgFuf5gh4KOi/JLTN/bSacTA3bTVwQoEtJ02hkPJgY7B2AgKBv5gezlmGtxjQAs8Du76D93mAxExw5qWgAZjJQp1xmfICAgICAgICAgICAgIA=","0x2c2669ad3ed604e2e2a62c09c71234bf2d3daf0f623d1fb1d29fd7cf5a775f95":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIRerQAAgA==","0x2f825bfaed201163387bd1b894efc6b3250463f85e56818e01f888d38a8f5df4":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAgA==","0x65c024ed3b68b3f86a44f6af5179e4ad055ac046ac2e954355d476c611947b16":"+HHGIISuCAAQxiCEPAhCpcYghDUI7F/GIISuCAAUxiCEPAgDu8YghDUI+iXGIISuCAAYxiCEPAhMsMYghDUIH63GIISuCAAcxiCEJAIPtMYghAAAAAzGIIQ8ETEAxiCEjigAAMYghCQMAAvGIIQBDGgjgA==","0x68d63fc77d306893ec77a3d683be9bd3de25f2f52698be970755dd505cbec5f5":"6cYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAACAgICAgICAgICAgICA","0x6fe607b39661adc63400b3c0eeefa0fdde6031131c39a96800663250a75c667c":"5YMQAACgD9/MJLGyHXjvK3xlA+uTVGd3Q2hcLQChSotQKhd5EbA=","0x8cd6ed962850eebdb5bf360b496b5ab3425659a8ba3d115d5bb71055981a6bc2":"+F/GIIQtogABxiCEjigABMYghDwMaGXGIIQ1jGxsxiCEAQxoI8YghC2jAAHGIIQAQxAkxiCEPBC//8YghDYQ//DGIIQ0EQABxiCErgIACMYghK4RAATGIIQD4AAIgICAgA==","0xa0b2822e4ee4937ac10760ad5bb6108b3cfaef41c6d13342a1ba5fb3fd4c26ab":"+HGgt8fQkZe4faINYK98tz2Czvb1LZ/LaDPtw0E2aioQ4lagZcAk7Ttos/hqRPavUXnkrQVawEasLpVDVdR2xhGUexagjNbtlihQ7r21vzYLSWtas0JWWai6PRFdW7cQVZgaa8KAgICAgICAgICAgICAgA==","0xb7c7d09197b87da20d60af7cb73d82cef6f52d9fcb6833edc341366a2a10e256":"+HHGIIQ8EDAAxiCENhAQAMYghDwIRxfGIIQ1CDKFxiCErggAAMYghDwIqNfGIIQ1CDQexiCErggABMYghDwIXpfGIIQ1CC/GxiCErggACMYghDwIdyjGIIQ1CGOExiCErggADMYghDwI+ALGIIQ1CPjvgA=="}};

async function addPreimages(mm, trie) {
  for (k in trie['preimages']) {
    const bin = Uint8Array.from(Buffer.from(trie['preimages'][k], 'base64').toString('binary'), c => c.charCodeAt(0))
    await mm.AddTrieNode(bin)
  }
  return trie['root']
}

describe("MIPS contract", function () {
  beforeEach(async function () {
    const MIPS = await ethers.getContractFactory("MIPS")
    m = await MIPS.deploy()
    mm = await ethers.getContractAt("MIPSMemory", await m.m())
  })

  it("add should work", async function () {
    let root = await addPreimages(mm, trieAdd)

    console.log("start", root)
    for (let i = 0; i < 12; i++) {
      ret = await m.Step(root)
      const receipt = await ret.wait()
      for (l of receipt.logs) {
        if (l.topics[0] == "0x86b89b5c9818dbbf520dd979a5f250d357508fe11b9511d4a43fd9bc6aa1be70") {
          root = l.data
        }
      }
      console.log(i, root)
    }
  }).timeout(40_000);

  it("oracle should work", async function () {
    let root = await addPreimages(mm, trieOracle)

    // "hello world" is the oracle
    await mm.AddPreimage(Buffer.from("hello world"), 0)

    console.log("start", root)
    let pc = 0, out1, out2
    while (pc != 0x5ead0000) {
      ret = await m.Step(root)
      const receipt = await ret.wait()
      for (l of receipt.logs) {
        if (l.topics[0] == "0x86b89b5c9818dbbf520dd979a5f250d357508fe11b9511d4a43fd9bc6aa1be70") {
          root = l.data
        }
      }
      pc = await mm.ReadMemory(root, 0xc0000080)
      out1 = await mm.ReadMemory(root, 0xbffffff4)
      out2 = await mm.ReadMemory(root, 0xbffffff8)
      console.log(root, pc, out1, out2)
    }
    expect(out1).to.equal(1)
    expect(out2).to.equal(1)
  }).timeout(200_000);

});
