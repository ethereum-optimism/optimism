/*

The contracts package provides Go bindings for our contracts.


The bindings are generated with `abigen`. `jq` is required to parse out the ABI section
from the hardhat artifacts.

*/

package contracts

//go:generate sh -c "jq .abi ../../packages/contracts/artifacts/contracts/L1/DepositFeed.sol/DepositFeed.json | abigen --pkg deposit --out deposit/deposit_feed_raw.go --abi -"
//go:generate sh -c "jq .abi ../../packages/contracts/artifacts/contracts/L2/L1Block.sol/L1Block.json | abigen --pkg l1block --out l1block/l1_block_info_raw.go --abi -"
