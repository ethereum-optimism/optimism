#!/bin/bash

set -e

VERBOSITY=${VERBOSITY:-6}
CHAIN_ID=${CHAIN_ID:-101}
DATADIR=${DATADIR:-.ethereum}
BLOCK_INTERVAL=${BLOCK_INTERVAL:-15}

mkdir -p ${DATADIR}

if [ ! -d ${DATADIR}/geth ]; then
  echo "${DATADIR}/geth not found; will re-init"

  # This genesis block includes one account used as the Clique sequencer,
  # followed by the 20 test accounts from the Hardhat l1_chain.

  cat > ${DATADIR}/l1_genesis.json <<END
{
  "config": {
    "chainId": $CHAIN_ID,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "muirGlacierBlock": 0,
    "londonBlock": null,
    "clique": {
      "period": ${BLOCK_INTERVAL},
      "epoch": 30000
    }
  },
  "difficulty": "1",
  "gasLimit": "8000000",
  "extradata": "0x00000000000000000000000000000000000000000000000000000000000000001dd5e0633ed04fdfac5890f0b77c2fce892f92d70000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "alloc": {
    "0x1dd5e0633ed04fdfac5890f0b77c2fce892f92d7": { "balance": "20000000000000000000000" },
    "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266": { "balance": "10000000000000000000000" },
    "0x70997970c51812dc3a010c7d01b50e0d17dc79c8": { "balance": "10000000000000000000000" },
    "0x3c44cdddb6a900fa2b585dd299e03d12fa4293bc": { "balance": "10000000000000000000000" },
    "0x90f79bf6eb2c4f870365e785982e1f101e93b906": { "balance": "10000000000000000000000" },
    "0x15d34aaf54267db7d7c367839aaf71a00a2c6a65": { "balance": "10000000000000000000000" },
    "0x9965507d1a55bcc2695c58ba16fb37d819b0a4dc": { "balance": "10000000000000000000000" },
    "0x976ea74026e726554db657fa54763abd0c3a0aa9": { "balance": "10000000000000000000000" },
    "0x14dc79964da2c08b23698b3d3cc7ca32193d9955": { "balance": "10000000000000000000000" },
    "0x23618e81e3f5cdf7f54c3d65f7fbc0abf5b21e8f": { "balance": "10000000000000000000000" },
    "0xa0ee7a142d267c1f36714e4a8f75612f20a79720": { "balance": "10000000000000000000000" },
    "0xbcd4042de499d14e55001ccbb24a551f3b954096": { "balance": "10000000000000000000000" },
    "0x71be63f3384f5fb98995898a86b02fb2426c5788": { "balance": "10000000000000000000000" },
    "0xfabb0ac9d68b0b445fb7357272ff202c5651694a": { "balance": "10000000000000000000000" },
    "0x1cbd3b2770909d4e10f157cabc84c7264073c9ec": { "balance": "10000000000000000000000" },
    "0xdf3e18d64bc6a983f673ab319ccae4f1a57c7097": { "balance": "10000000000000000000000" },
    "0xcd3b766ccdd6ae721141f452c550ca635964ce71": { "balance": "10000000000000000000000" },
    "0x2546bcd3c84621e976d8185a91a922ae77ecec30": { "balance": "10000000000000000000000" },
    "0xbda5747bfd65f08deb54cb465eb87d40e51b197e": { "balance": "10000000000000000000000" },
    "0xdd2fd4581271e230360230f9337d5c0430bf44c0": { "balance": "10000000000000000000000" },
    "0x8626f6940e2eb28930efb4cef49b2d1f2c9c1199": { "balance": "10000000000000000000000" }
  }
}
END

  geth --verbosity="$VERBOSITY"\
  --datadir ${DATADIR}\
  init ${DATADIR}/l1_genesis.json
  echo
fi

# The private keyfile used for the Clique sequencer account.
if [ ! -f .ethereum/keystore/sequencer.key ]; then
  echo "Creating keyfile"
  mkdir -p ${DATADIR}/keystore
  cat > ${DATADIR}/keystore/clique.key <<END
{"address":"1dd5e0633ed04fdfac5890f0b77c2fce892f92d7","crypto":{"cipher":"aes-128-ctr","ciphertext":"37cea4ef114c9ada7e1ca756156328008c62f947a17b5d27d3478b909080f5fc","cipherparams":{"iv":"ee0aea92cb6d3616992efccdbe9bab6e"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"bd4d2119acfeab24c9c68beb89999fbaa52e95bcc159d4f9287f9b16625749c9"},"mac":"24f1c60de2f4f07660d8231c2055b7f87ed360cf8d216573ed8cf170642ff724"},"id":"41106238-4890-4319-b33d-dca7ba97bd40","version":3}  
END
fi

echo "--- DEBUG ---"
env
cat ${DATADIR}/l1_genesis.json
echo
cat ${DATADIR}/keystore/clique.key
echo "--- DEBUG ---"

echo "Starting L1 geth"

# Password for the Clique account is empty. The private keys for the Hardhat
# accounts are not needed inside this container.
echo > ${DATADIR}/geth_passwords

exec geth \
  --verbosity="$VERBOSITY"\
  --datadir ${DATADIR}\
  --syncmode full\
  --gcmode archive\
  --nodiscover\
  --mine\
  --unlock 0x1dd5e0633ed04fdfac5890f0b77c2fce892f92d7\
  --password ${DATADIR}/geth_passwords\
  --allow-insecure-unlock\
  --http\
  --http.addr "0.0.0.0"\
  --http.api "admin,debug,web3,eth,txpool,personal,clique,miner,net"\
  --http.corsdomain "*"\
  --http.vhosts "*"\
  --txpool.pricelimit 0\
  --gpo.maxprice 0\
  --miner.gasprice 0\
  "$@"
