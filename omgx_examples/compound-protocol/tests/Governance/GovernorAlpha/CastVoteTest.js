const {
  address,
  etherMantissa,
  encodeParameters,
  mineBlock,
  unlockedAccount
} = require('../../Utils/Ethereum');
const EIP712 = require('../../Utils/EIP712');
const BigNumber = require('bignumber.js');
const chalk = require('chalk');

async function enfranchise(comp, actor, amount) {
  await send(comp, 'transfer', [actor, etherMantissa(amount)]);
  await send(comp, 'delegate', [actor], { from: actor });
}

describe("governorAlpha#castVote/2", () => {
  let comp, gov, root, a1, accounts;
  let targets, values, signatures, callDatas, proposalId;

  beforeAll(async () => {
    [root, a1, ...accounts] = saddle.accounts;
    comp = await deploy('Comp', [root]);
    gov = await deploy('GovernorAlpha', [address(0), comp._address, root]);

    targets = [a1];
    values = ["0"];
    signatures = ["getBalanceOf(address)"];
    callDatas = [encodeParameters(['address'], [a1])];
    await send(comp, 'delegate', [root]);
    await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"]);
    proposalId = await call(gov, 'latestProposalIds', [root]);
  });

  describe("We must revert if:", () => {
    it("There does not exist a proposal with matching proposal id where the current block number is between the proposal's start block (exclusive) and end block (inclusive)", async () => {
      await expect(
        call(gov, 'castVote', [proposalId, true])
      ).rejects.toRevert("revert GovernorAlpha::_castVote: voting is closed");
    });

    it("Such proposal already has an entry in its voters set matching the sender", async () => {
      await mineBlock();
      await mineBlock();

      await send(gov, 'castVote', [proposalId, true], { from: accounts[4] });
      await expect(
        gov.methods['castVote'](proposalId, true).call({ from: accounts[4] })
      ).rejects.toRevert("revert GovernorAlpha::_castVote: voter already voted");
    });
  });

  describe("Otherwise", () => {
    it("we add the sender to the proposal's voters set", async () => {
      await expect(call(gov, 'getReceipt', [proposalId, accounts[2]])).resolves.toPartEqual({hasVoted: false});
      await send(gov, 'castVote', [proposalId, true], { from: accounts[2] });
      await expect(call(gov, 'getReceipt', [proposalId, accounts[2]])).resolves.toPartEqual({hasVoted: true});
    });

    describe("and we take the balance returned by GetPriorVotes for the given sender and the proposal's start block, which may be zero,", () => {
      let actor; // an account that will propose, receive tokens, delegate to self, and vote on own proposal

      it("and we add that ForVotes", async () => {
        actor = accounts[1];
        await enfranchise(comp, actor, 400001);

        await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: actor });
        proposalId = await call(gov, 'latestProposalIds', [actor]);

        let beforeFors = (await call(gov, 'proposals', [proposalId])).forVotes;
        await mineBlock();
        await send(gov, 'castVote', [proposalId, true], { from: actor });

        let afterFors = (await call(gov, 'proposals', [proposalId])).forVotes;
        expect(new BigNumber(afterFors)).toEqual(new BigNumber(beforeFors).plus(etherMantissa(400001)));
      })

      it("or AgainstVotes corresponding to the caller's support flag.", async () => {
        actor = accounts[3];
        await enfranchise(comp, actor, 400001);

        await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: actor });
        proposalId = await call(gov, 'latestProposalIds', [actor]);;

        let beforeAgainsts = (await call(gov, 'proposals', [proposalId])).againstVotes;
        await mineBlock();
        await send(gov, 'castVote', [proposalId, false], { from: actor });

        let afterAgainsts = (await call(gov, 'proposals', [proposalId])).againstVotes;
        expect(new BigNumber(afterAgainsts)).toEqual(new BigNumber(beforeAgainsts).plus(etherMantissa(400001)));
      });
    });

    describe('castVoteBySig', () => {
      const Domain = (gov) => ({
        name: 'Compound Governor Alpha',
        chainId: 1, // await web3.eth.net.getId(); See: https://github.com/trufflesuite/ganache-core/issues/515
        verifyingContract: gov._address
      });
      const Types = {
        Ballot: [
          { name: 'proposalId', type: 'uint256' },
          { name: 'support', type: 'bool' }
        ]
      };

      it('reverts if the signatory is invalid', async () => {
        await expect(send(gov, 'castVoteBySig', [proposalId, false, 0, '0xbad', '0xbad'])).rejects.toRevert("revert GovernorAlpha::castVoteBySig: invalid signature");
      });

      it('casts vote on behalf of the signatory', async () => {
        await enfranchise(comp, a1, 400001);
        await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: a1 });
        proposalId = await call(gov, 'latestProposalIds', [a1]);;

        const { v, r, s } = EIP712.sign(Domain(gov), 'Ballot', { proposalId, support: true }, Types, unlockedAccount(a1).secretKey);

        let beforeFors = (await call(gov, 'proposals', [proposalId])).forVotes;
        await mineBlock();
        const tx = await send(gov, 'castVoteBySig', [proposalId, true, v, r, s]);
        expect(tx.gasUsed < 80000);

        let afterFors = (await call(gov, 'proposals', [proposalId])).forVotes;
        expect(new BigNumber(afterFors)).toEqual(new BigNumber(beforeFors).plus(etherMantissa(400001)));
      });
    });

    it("receipt uses one load", async () => {
      let actor = accounts[2];
      let actor2 = accounts[3];
      await enfranchise(comp, actor, 400001);
      await enfranchise(comp, actor2, 400001);
      await send(gov, 'propose', [targets, values, signatures, callDatas, "do nothing"], { from: actor });
      proposalId = await call(gov, 'latestProposalIds', [actor]);

      await mineBlock();
      await mineBlock();
      await send(gov, 'castVote', [proposalId, true], { from: actor });
      await send(gov, 'castVote', [proposalId, false], { from: actor2 });

      let trxReceipt = await send(gov, 'getReceipt', [proposalId, actor]);
      let trxReceipt2 = await send(gov, 'getReceipt', [proposalId, actor2]);

      await saddle.trace(trxReceipt, {
        constants: {
          "account": actor
        },
        preFilter: ({op}) => op === 'SLOAD',
        postFilter: ({source}) => !source || source.includes('receipts'),
        execLog: (log) => {
          let [output] = log.outputs;
          let votes = "000000000000000000000000000000000000000054b419003bdf81640000";
          let voted = "01";
          let support = "01";

          expect(output).toEqual(
            `${votes}${support}${voted}`
          );
        },
        exec: (logs) => {
          expect(logs.length).toEqual(1); // require only one read
        }
      });

      await saddle.trace(trxReceipt2, {
        constants: {
          "account": actor2
        },
        preFilter: ({op}) => op === 'SLOAD',
        postFilter: ({source}) => !source || source.includes('receipts'),
        execLog: (log) => {
          let [output] = log.outputs;
          let votes = "0000000000000000000000000000000000000000a968320077bf02c80000";
          let voted = "01";
          let support = "00";

          expect(output).toEqual(
            `${votes}${support}${voted}`
          );
        }
      });
    });
  });
});