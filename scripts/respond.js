const fs = require("fs")
const { deployed, getTrieNodesForCall } = require("../scripts/lib")

async function main() {
  let [c, m, mm] = await deployed()

  const challengeId = parseInt(process.env.ID)

  let step = (await c.getStepNumber(challengeId)).toNumber()
  console.log("searching step", step)

  // see if it's proposed or not
  //await c.getProposedState(challengeId)

  const blockNumberN = parseInt(process.env.BLOCK)
  let thisTrie = JSON.parse(fs.readFileSync("/tmp/cannon/0_"+blockNumberN.toString()+"/checkpoint_"+step.toString()+".json"))
  const root = thisTrie['root']
  console.log("new root", root)

  if (process.env.PROPOSE == "1") {
    let ret = await c.ProposeState(challengeId, root)
    let receipt = await ret.wait()
    console.log(receipt)
  }

  if (process.env.RESPOND == "1") {
    let ret = await c.RespondState(challengeId, root)
    let receipt = await ret.wait()
    console.log(receipt)
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });