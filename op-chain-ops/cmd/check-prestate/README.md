# check-prestate

The `check-prestate` tool can be used to generate a status report for an absolute prestate hash.
It outputs JSON to stdout that contains useful information like its op-program version tag and commit of the monorepo,
its type (cannon32/64), and the underlying op-geth and superchain-registry commits.
It then also checks for each specified chain if the included chain configuration has changed
compared to the latest changes in the superchain-registry.

## Usage

The script internally clones op-geth and runs its `sync-superchain.sh` script,
so `git`, `jq`, `dasel`, and `zipinfo` need to be installed on your system.

Given an absolute prestate hash and a list of chains (with their canonical network names), the tool can be run like the following:
```sh
HASH=0x03ee2917da962ec266b091f4b62121dc9682bb0db534633707325339f99ee405
CHAINS=op-sepolia,ink-sepolia,base-sepolia,unichain-sepolia,soneium-minato-sepolia,op-mainnet,ink-mainnet,base-mainnet,unichain-mainnet,soneium-mainnet
go run . --prestate-hash $HASH --chains $CHAINS | tee check.json | pbcopy
```
This will write the output JSON report to file `check.json` and also copy it to your clipboard (on MacOS).

However, if there are diffs in the chain configurations, the different chain configs will be printed out, in full,
for each chain in the `"outdated-chains"` array, at JSON paths `diff.prestate` and `diff.latest`.
This makes it hard to tell the actual differences.
The included script `diff-check.zsh` can be used for post-processing to only show what actually changed.
It can be used like this:
```sh
./diff-check.zsh < check.json | pbcopy
```
after you have written the output of `check-prestate` to `check.json`.