const { keccak256 } = require("@ethersproject/keccak256");
const { expect } = require("chai");

describe("MIPSMemory contract", function () {
  it("Keccak should work", async function () {
    const [owner] = await ethers.getSigners();

    const MIPSMemory = await ethers.getContractFactory("MIPSMemory");
    const mm = await MIPSMemory.deploy();
    console.log("deployed at", mm.address, "by", owner.address);

    await mm.AddLargePreimageInit();
    console.log("preimage initted");

    const a = ["0x0100000000000000",0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0x80];
    await mm.AddLargePreimageUpdate(a);
    console.log("preimage updated");

    /*var tst1 = await mm.largePreimage(owner.address, 0);
    console.log(tst);*/

    const hash = await mm.AddLargePreimageFinal();
    console.log("comp hash is", hash);
    console.log("real hash is", keccak256(new Uint8Array(0)));
  });
});