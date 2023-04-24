const fs = require("fs")
const { deployed, getTrieNodesForCall, getTrieAtStep } = require("../scripts/lib")

async function main() {
  let [c, m, mm] = await deployed()

  const challengeId = parseInt(process.env.ID)
  const blockNumberN = parseInt(process.env.BLOCK)
  const isChallenger = process.env.CHALLENGER == "1"

  let step = (await c.getStepNumber(challengeId)).toNumber()
  console.log("searching step", step, "at block", blockNumberN)

  if (!(await c.isSearching(challengeId))) {
    console.log("search is done")
    return
  }

  // see if it's proposed or not
  const proposed = await c.getProposedState(challengeId)
  const isProposing = proposed == "0x0000000000000000000000000000000000000000000000000000000000000000"
  if (isProposing != isChallenger) {
    console.log("bad challenger state")
    return
  }
  console.log("isProposing", isProposing)
  let thisTrie = getTrieAtStep(blockNumberN, step)
  const root = thisTrie['root']
  console.log("new root", root)

  let ret
  if (isProposing) {
    ret = await c.proposeState(challengeId, root)
  } else {
    ret = await c.respondState(challengeId, root)
  }
  let receipt = await ret.wait()
  console.log("done", receipt.blockNumber)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });