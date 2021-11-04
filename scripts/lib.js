const fs = require("fs")

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

module.exports = { deploy }
