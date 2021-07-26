const {
  advanceBlocks,
  etherUnsigned,
  both,
  encodeParameters,
  etherMantissa,
  mineBlock,
  freezeTime,
  increaseTime
} = require('../../Utils/Ethereum');

const path = require('path');
const solparse = require('solparse');

const governorBravoPath = path.join(__dirname, '../../..', 'contracts', 'Governance/GovernorBravoInterfaces.sol');
const statesInverted = solparse
  .parseFile(governorBravoPath)
  .body
  .find(k => k.name === 'GovernorBravoDelegateStorageV1')
  .body
  .find(k => k.name == 'ProposalState')
  .members

const states = Object.entries(statesInverted).reduce((obj, [key, value]) => ({ ...obj, [value]: key }), {});

describe('GovernorBravo#state/1', () => {
  let comp, gov, root, acct, delay, timelock;

  beforeAll(async () => {
    await freezeTime(100);
    [root, acct, ...accounts] = accounts;
    comp = await deploy('Comp', [root]);
    delay = etherUnsigned(2 * 24 * 60 * 60).multipliedBy(2)
    timelock = await deploy('TimelockHarness', [root, delay]);
    gov = await deploy('GovernorBravoImmutable', [timelock._address, comp._address, root, 17280, 1, "100000000000000000000000"]);
    await send(gov, '_initiate');
    await send(timelock, "harnessSetAdmin", [gov._address])
    await send(comp, 'transfer', [acct, etherMantissa(4000000)]);
    await send(comp, 'delegate', [acct], { from: acct });
  });

  let trivialProposal, targets, values, signatures, callDatas;
  beforeAll(async () => {
    targets = [root];
    values = ["0"];
    signatures = ["getBalanceOf(address)"]
    callDatas = [encodeParameters(['address'], [acct])];
    await send(comp, 'delegate', [root]);
    await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"]);
    proposalId = await call(gov, 'latestProposalIds', [root]);
    trivialProposal = await call(gov, "proposals", [proposalId])
  })

  it("Invalid for proposal not found", async () => {
    await expect(call(gov, 'state', ["5"])).rejects.toRevert("revert GovernorBravo::state: invalid proposal id")
  })

  it("Pending", async () => {
    expect(await call(gov, 'state', [trivialProposal.id], {})).toEqual(states["Pending"])
  })

  it("Active", async () => {
    await mineBlock()
    await mineBlock()
    expect(await call(gov, 'state', [trivialProposal.id], {})).toEqual(states["Active"])
  })

  it("Canceled", async () => {
    await send(comp, 'transfer', [accounts[0], etherMantissa(4000000)]);
    await send(comp, 'delegate', [accounts[0]], { from: accounts[0] });
    await mineBlock()
    await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: accounts[0] })
    let newProposalId = await call(gov, 'proposalCount')

    // send away the delegates
    await send(comp, 'delegate', [root], { from: accounts[0] });
    await send(gov, 'cancel', [newProposalId])

    expect(await call(gov, 'state', [+newProposalId])).toEqual(states["Canceled"])
  })

  it("Defeated", async () => {
    // travel to end block
    await advanceBlocks(20000)

    expect(await call(gov, 'state', [trivialProposal.id])).toEqual(states["Defeated"])
  })

  it("Succeeded", async () => {
    await mineBlock()
    const { reply: newProposalId } = await both(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: acct })
    await mineBlock()
    await send(gov, 'castVote', [newProposalId, 1])
    await advanceBlocks(20000)

    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Succeeded"])
  })

  it("Queued", async () => {
    await mineBlock()
    const { reply: newProposalId } = await both(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: acct })
    await mineBlock()
    await send(gov, 'castVote', [newProposalId, 1])
    await advanceBlocks(20000)

    await send(gov, 'queue', [newProposalId], { from: acct })
    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Queued"])
  })

  it("Expired", async () => {
    await mineBlock()
    const { reply: newProposalId } = await both(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: acct })
    await mineBlock()
    await send(gov, 'castVote', [newProposalId, 1])
    await advanceBlocks(20000)

    await increaseTime(1)
    await send(gov, 'queue', [newProposalId], { from: acct })

    let gracePeriod = await call(timelock, 'GRACE_PERIOD')
    let p = await call(gov, "proposals", [newProposalId]);
    let eta = etherUnsigned(p.eta)

    await freezeTime(eta.plus(gracePeriod).minus(1).toNumber())

    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Queued"])

    await freezeTime(eta.plus(gracePeriod).toNumber())

    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Expired"])
  })

  it("Executed", async () => {
    await mineBlock()
    const { reply: newProposalId } = await both(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: acct })
    await mineBlock()
    await send(gov, 'castVote', [newProposalId, 1])
    await advanceBlocks(20000)

    await increaseTime(1)
    await send(gov, 'queue', [newProposalId], { from: acct })

    let gracePeriod = await call(timelock, 'GRACE_PERIOD')
    let p = await call(gov, "proposals", [newProposalId]);
    let eta = etherUnsigned(p.eta)

    await freezeTime(eta.plus(gracePeriod).minus(1).toNumber())

    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Queued"])
    await send(gov, 'execute', [newProposalId], { from: acct })

    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Executed"])

    // still executed even though would be expired
    await freezeTime(eta.plus(gracePeriod).toNumber())

    expect(await call(gov, 'state', [newProposalId])).toEqual(states["Executed"])
  })

})