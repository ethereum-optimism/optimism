const { expect } = require("chai");

describe("MIPSMemory contract", function () {
  it("Keccak should work", async function () {
    const [owner] = await ethers.getSigners();

    const MIPSMemory = await ethers.getContractFactory("MIPSMemory");
    const mm = await MIPSMemory.deploy();
    console.log("deployed at", mm.address, "by", owner.address);

    await mm.AddLargePreimageInit();
    console.log("preimage initted");

    var a = [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];
    await mm.AddLargePreimageUpdate(a);
    console.log("preimage updated");

    var tst = await mm.largePreimage(owner.address, 0);
    console.log(tst);
  });
});