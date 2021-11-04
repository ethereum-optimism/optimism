const fs = require("fs")
const { hasUncaughtExceptionCaptureCallback } = require("process")

async function deploy() {
  const MIPS = await ethers.getContractFactory("MIPS")
  const m = await MIPS.deploy()
  const mm = await ethers.getContractAt("MIPSMemory", await m.m())

  let startTrie = JSON.parse(fs.readFileSync("/tmp/cannon/golden.json"))
  let goldenRoot = startTrie["root"]
  console.log("goldenRoot is", goldenRoot)

  const Challenge = await ethers.getContractFactory("Challenge")
  const c = await Challenge.deploy(m.address, goldenRoot)

  return [c,m,mm]
}

async function getTrieNodesForCall(c, cdat, preimages) {
  let nodes = []
  while (1) {
    try {
      // TODO: make this eth call?
      // needs something like InitiateChallengeWithTrieNodesj
      let calldata = c.interface.encodeFunctionData("CallWithTrieNodes", [cdat, nodes])
      ret = await ethers.provider.call({
        to:c.address,
        data:calldata
      });
      console.log(ret)
      break
    } catch(e) {
      const missing = e.toString().split("'")[1]
      if (missing.length == 64) {
        console.log("requested node", missing)
        let node = preimages["0x"+missing]
        if (node === undefined) {
          throw("node not found")
        }
        const bin = Uint8Array.from(Buffer.from(node, 'base64').toString('binary'), c => c.charCodeAt(0))
        nodes.push(bin)
        continue
      } else {
        console.log(e)
        break
      }
    }
  }
  return nodes
}

module.exports = { deploy, getTrieNodesForCall }
