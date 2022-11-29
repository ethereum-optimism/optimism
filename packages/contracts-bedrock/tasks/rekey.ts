import { task } from 'hardhat/config'
import { hdkey } from 'ethereumjs-wallet'
import * as bip39 from 'bip39'

task('rekey', 'Generates a new set of keys for a test network').setAction(
  async () => {
    const mnemonic = bip39.generateMnemonic()
    const pathPrefix = "m/44'/60'/0'/0"
    const labels = [
      'l2OutputOracleProposer',
      'proxyAdminOwner',
      'optimismBaseFeeRecipient',
      'optimismL1FeeRecipient',
      'p2pSequencerAddress',
      'l2OutputOracleChallenger',
      'batchSenderAddress',
    ]

    const hdwallet = hdkey.fromMasterSeed(await bip39.mnemonicToSeed(mnemonic))
    let i = 0
    const out = {}
    console.log(`Mnemonic: ${mnemonic}`)
    for (const label of labels) {
      const wallet = hdwallet.derivePath(`${pathPrefix}/${i}`).getWallet()
      out[label] = `0x${wallet.getAddress().toString('hex')}`
      i++
    }
    console.log(JSON.stringify(out, null, '  '))
  }
)
