const { expect } = require("chai");

describe("Challenge contract", function () {
  beforeEach(async function () {
    // this mips can be reused for other challenges
    const MIPS = await ethers.getContractFactory("MIPS")
    const m = await MIPS.deploy()

    const Challenge = await ethers.getContractFactory("Challenge")
    // golden minigeth.bin hash
    c = await Challenge.deploy(m.address, "0x9c15aa86416a3a9d3b15188fc9f9be59626c1f83a33e5d63b58ca1bf0f8cef71")
  })
  it("challenge contract deploys", async function() {
    console.log("Challenge deployed at", c.address)
  })
})