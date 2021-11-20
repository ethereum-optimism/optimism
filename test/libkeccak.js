const { keccak256 } = require("@ethersproject/keccak256");
const { expect } = require("chai");

const empty = [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];
const endEmpty = [0x1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,"0x8000000000000000"];

describe("MIPSMemory contract", function () {
  it("Keccak should work", async function () {
    const [owner] = await ethers.getSigners();

    const MIPSMemory = await ethers.getContractFactory("MIPSMemory");
    const mm = await MIPSMemory.deploy();
    console.log("deployed at", mm.address, "by", owner.address);

    await mm.AddLargePreimageInit(0);
    console.log("preimage initted");

    // empty
    expect(await mm.AddLargePreimageFinal(endEmpty)).to.equal(keccak256(new Uint8Array(0)));

    // block size is 136
    await mm.AddLargePreimageUpdate(empty);

    const hash = await mm.AddLargePreimageFinal(endEmpty);
    console.log("preimage updated");

    /*var tst1 = await mm.largePreimage(owner.address, 0);
    console.log(tst);*/

    const realhash = keccak256(new Uint8Array(136));
    console.log("comp hash is", hash);
    console.log("real hash is", realhash);
    expect(hash).to.equal(realhash);
  });
});