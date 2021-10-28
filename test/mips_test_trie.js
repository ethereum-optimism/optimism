const { expect } = require("chai");

const trieAdd = {"root":"0x22ffce7c56d926c2d8d6337d8917fa0e1880e1869e189c15385ead63c6c45b93","preimages":{"0x044371dc86fb8c621bc84b69dce16de366de1126777250888b17416d0bd11279":"+FPGIIQ8EL//xiCENhD/8MYghDQRAAHGIIQ8CP//xiCENQj//cYghDQJAAPGIIQBCVAgxiCELUIAAcYghK4CAAjGIISuEQAExiCEA+AACICAgICAgA==","0x0fdfcc24b1b21d78ef2b7c6503eb9354677743685c2d00a14a8b502a177911b0":"+HGgL4Jb+u0gEWM4e9G4lO/GsyUEY/heVoGOAfiI04qPXfSgLCZprT7WBOLipiwJxxI0vy09rw9iPR+x0p/Xz1p3X5WgaNY/x30waJPsd6PWg76b094l8vUmmL6XB1XdUFy+xfWAgICAgICAgICAgICAgA==","0x11228d4f4a028a9088e6ec0aa6513e0d4731d9dc488e2af1957e46ba80624a69":"5oQAAAAAoARDcdyG+4xiG8hLadzhbeNm3hEmd3JQiIsXQW0L0RJ5","0x22ffce7c56d926c2d8d6337d8917fa0e1880e1869e189c15385ead63c6c45b93":"+FGgESKNT0oCipCI5uwKplE+DUcx2dxIjirxlX5GuoBiSmmAgKBv5gezlmGtxjQAs8Du76D93mAxExw5qWgAZjJQp1xmfICAgICAgICAgICAgIA=","0x2c2669ad3ed604e2e2a62c09c71234bf2d3daf0f623d1fb1d29fd7cf5a775f95":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIRerQAAgA==","0x2f825bfaed201163387bd1b894efc6b3250463f85e56818e01f888d38a8f5df4":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAgA==","0x68d63fc77d306893ec77a3d683be9bd3de25f2f52698be970755dd505cbec5f5":"6cYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAACAgICAgICAgICAgICA","0x6fe607b39661adc63400b3c0eeefa0fdde6031131c39a96800663250a75c667c":"5YMQAACgD9/MJLGyHXjvK3xlA+uTVGd3Q2hcLQChSotQKhd5EbA="}};

describe("MIPS contract", function () {
  it("add should work", async function () {
    const MIPS = await ethers.getContractFactory("MIPS")
    const m = await MIPS.deploy()
    const mm = await ethers.getContractAt("MIPSMemory", await m.m())

    for (k in trieAdd['preimages']) {
      const bin = Uint8Array.from(Buffer.from(trieAdd['preimages'][k], 'base64').toString('binary'), c => c.charCodeAt(0))
      await mm.AddTrieNode(bin)
    }

    let root = trieAdd['root']
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
  });
});