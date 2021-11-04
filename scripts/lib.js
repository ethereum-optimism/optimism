const goldenRoot = "0x1f6285b6d372ee187815a8580d1af3ab348cea34abbee18a8e13272454a4c4af"

async function deploy() {
  const MIPS = await ethers.getContractFactory("MIPS")
  const m = await MIPS.deploy()
  const mm = await ethers.getContractAt("MIPSMemory", await m.m())

  const Challenge = await ethers.getContractFactory("Challenge")
  const c = await Challenge.deploy(m.address, goldenRoot)

  return [c,m,mm]
}

module.exports = { deploy }
