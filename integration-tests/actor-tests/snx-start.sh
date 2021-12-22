#!/bin/bash

npx hardhat test:integration:dual --deploy | tee output.txt
ADDRESSES=$(cat output.txt | grep -a '║')

echo "$ADDRESSES" | node -e '
const readline = require("readline");
const { stdin: input } = require("process");

const rl = readline.createInterface({ input });

const output = {
  l1: {},
  l2: {}
}

rl.on("line", (input) => {
  const columns = input.split("│")
  let [name, addr, url] = columns

  name = name.slice(2).trim()
  addr = addr.trim()

  if (url.includes("optimism")) {
    output.l2[name] = addr
  } else {
    output.l1[name] = addr
  }
})

rl.on("close", () => {
  console.log(JSON.stringify(output))
})
' | tee /synthetix/build/deployment.json

echo "Starting server."
python3 -m http.server \
    --bind "0.0.0.0" 8082 \
    --directory /synthetix/build
