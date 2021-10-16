const { expect } = require("chai");

const trieAdd = {"root":"0xe5200ed7c7b2cdd673574f8fe42c5e448ed248766d4456dfa0fa1fda5f5ef9c2","preimages":{"0x02b8d50956bf99188941a96a6b62e5325e25fd361c64b9a5fdabcd096503f64c":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghF6tAADGIIQAAAAAgA==","0x1bcc822b269177eecb38ea1336f519ac32de68e9f90797e158998f4867c711eb":"+HGgL4Jb+u0gEWM4e9G4lO/GsyUEY/heVoGOAfiI04qPXfSgArjVCVa/mRiJQalqa2LlMl4l/TYcZLml/avNCWUD9kygaNY/x30waJPsd6PWg76b094l8vUmmL6XB1XdUFy+xfWAgICAgICAgICAgICAgA==","0x2f825bfaed201163387bd1b894efc6b3250463f85e56818e01f888d38a8f5df4":"+HHGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAADGIIQAAAAAgA==","0x3d95d160d966af02a751b950b626fe09a33cd20d4efd5e09605a1d77d6aea3b7":"5YMQAACgG8yCKyaRd+7LOOoTNvUZrDLeaOn5B5fhWJmPSGfHEes=","0x4439d93a074b4edd5b54b01da1a2de393b7b56b6172aee5eb3b021a1a20bc991":"5oQAAAAAoM6MuvdTbEcVOfd0sdLVE21XWGkXQyWTL4l5g0aHzGro","0x68d63fc77d306893ec77a3d683be9bd3de25f2f52698be970755dd505cbec5f5":"6cYghAAAAADGIIQAAAAAxiCEAAAAAMYghAAAAACAgICAgICAgICAgICA","0xce8cbaf7536c471539f774b1d2d5136d575869174325932f8979834687cc6ae8":"+FPGIIQ2EP/wxiCENBEAAcYghDwI///GIIQ1CP/9xiCENAkAA8YghAEJUCDGIIQtQgABxiCErgIACMYghK4RAATGIIQD4AAIxiCEAAAAAICAgICAgA==","0xe5200ed7c7b2cdd673574f8fe42c5e448ed248766d4456dfa0fa1fda5f5ef9c2":"+FGgRDnZOgdLTt1bVLAdoaLeOTt7VrYXKu5es7AhoaILyZGAgKA9ldFg2WavAqdRuVC2Jv4JozzSDU79XglgWh131q6jt4CAgICAgICAgICAgIA="}};

describe("MIPS contract", function () {
  it("add should work", async function () {
    const MIPS = await ethers.getContractFactory("MIPS")
    const m = await MIPS.deploy()
    const mm = await ethers.getContractAt("MIPSMemory", await m.m())

    for (k in trieAdd['preimages']) {
      const bin = Uint8Array.from(atob(trieAdd['preimages'][k]), c => c.charCodeAt(0))
      await mm.AddTrieNode(bin)
    }

    let root = trieAdd['root']
    for (let i = 0; i < 10; i++) {
      ret = await m.Step(root)
      const receipt = await ret.wait()
      root = receipt.logs[0].data
      console.log(i, root)
    }
  });
});