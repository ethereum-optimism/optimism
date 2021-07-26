const {
  etherBalance,
  etherGasCost,
  getContract
} = require('./Utils/Ethereum');

const {
  makeComptroller,
  makeCToken,
  makePriceOracle,
  pretendBorrow,
  borrowSnapshot
} = require('./Utils/Compound');

describe('Const', () => {
  it("does the right thing and not too expensive", async () => {
    const base = await deploy('ConstBase');
    const sub = await deploy('ConstSub');
    expect(await call(base, 'c')).toEqual("1");
    expect(await call(sub, 'c')).toEqual("2");
    expect(await call(base, 'ADD', [1])).toEqual("2");
    expect(await call(base, 'add', [1])).toEqual("2");
    expect(await call(sub, 'ADD', [1])).toEqual("2");
    expect(await call(sub, 'add', [1])).toEqual("3");

    const tx1 = await send(base, 'ADD', [1]);
    const tx2 = await send(base, 'add', [1]);
    const tx3 = await send(sub, 'ADD', [1]);
    const tx4 = await send(sub, 'add', [1]);
    expect(Math.abs(tx2.gasUsed - tx1.gasUsed) < 20);
    expect(Math.abs(tx4.gasUsed - tx3.gasUsed) < 20);
  });
});

describe('Structs', () => {
  it("only writes one slot", async () => {
    const structs1 = await deploy('Structs');
    const tx1_0 = await send(structs1, 'writeEach', [0, 1, 2, 3]);
    const tx1_1 = await send(structs1, 'writeEach', [0, 1, 2, 3]);
    const tx1_2 = await send(structs1, 'writeOnce', [0, 1, 2, 3]);

    const structs2 = await deploy('Structs');
    const tx2_0 = await send(structs2, 'writeOnce', [0, 1, 2, 3]);
    const tx2_1 = await send(structs2, 'writeOnce', [0, 1, 2, 3]);
    const tx2_2 = await send(structs2, 'writeEach', [0, 1, 2, 3]);

    expect(tx1_0.gasUsed < tx2_0.gasUsed); // each beats once
    expect(tx1_1.gasUsed < tx2_1.gasUsed); // each beats once
    expect(tx1_2.gasUsed > tx2_2.gasUsed); // each beats once
  });
});
