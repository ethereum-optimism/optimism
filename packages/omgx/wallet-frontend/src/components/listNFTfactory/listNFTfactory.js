import React from 'react'
import { connect } from 'react-redux'
import { isEqual } from 'lodash'

import { openAlert, openError } from 'actions/uiAction'

import Button from 'components/button/Button'
import Input from 'components/input/Input'

import Select from '@material-ui/core/Select'
import MenuItem from '@material-ui/core/MenuItem'

import networkService from 'services/networkService'

import * as styles from './listNFTfactory.module.scss'

import { Box } from '@material-ui/core'

class listNFTfactory extends React.Component {

  constructor(props) {

    super(props);

    const {
      name,
      symbol,
      layer,
    } = this.props;


    this.state = {

      //details of this contract
      name,
      symbol,
      address: '',
      layer,

      // loading
      loading: false,

      //for minting new NFTs
      receiverAddress: '',
      tokenURI: '',
    }
  }

  componentDidUpdate(prevState) {

    const {
      name, symbol, address, layer,
    } = this.props;

    if (!isEqual(prevState.name, name)) {
      this.setState({ name })
    }

    if (!isEqual(prevState.layer, layer)) {
      this.setState({ layer })
    }

    if (!isEqual(prevState.symbol, symbol)) {
      this.setState({ symbol })
    }

    if (!isEqual(prevState.address, address)) {
      this.setState({ address })
    }

  }

  async handleSimpleMintAndSend() {

    const { receiverAddress, tokenURI, address } = this.state;

    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'))

    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'))
      return
    }

    this.setState({ loading: true })

    const mintTX = await networkService.mintAndSendNFT(
      receiverAddress,
      address,
      tokenURI
    )

    if (mintTX) {
      this.props.dispatch(openAlert(`You minted a new NFT for ${receiverAddress}.`))
    } else {
      this.props.dispatch(openError('NFT minting error'))
    }

    this.props.closeMintModal();
    this.setState({ loading: false })
  }

  onSelectChange(value) {
    this.setState({ address: value })
  }

  render() {

    const {
      name,
      loading,
      receiverAddress, // for new NFTs
      tokenURI,        // for new NFTs
    } = this.state;

    return (
      <div className={styles.ListNFT}>

        <div className={styles.boxContainer}>
          <>
            <div className={styles.BasicLightText} style={{paddingBottom: '3px'}}>
              To mint and send a new {name} NFT, please fill in the information and click "Mint and Send".
            </div>
            <Box sx={{mb: 2}}>
              <Select
                value={this.state.address}
                onChange={(e) => {this.onSelectChange(e.target.value)}}
                placeholder="Select"
                displayEmpty
              >
                <MenuItem value="">Select</MenuItem>
                {Object.values(this.props.contracts).map((item) =>
                  <MenuItem
                    key={item.address}
                    value={item.address}>
                      {item.symbol}
                  </MenuItem>
                )}
              </Select>
            </Box>
            <Box sx={{mb: 2}}>
              <Input
                fullwidth
                placeholder="Receiver Address (Ox.....)"
                onChange={i=>{this.setState({receiverAddress: i.target.value})}}
                value={receiverAddress}
              />
            </Box>
            <Box sx={{mb: 2}}>
              <Input
                fullwidth
                placeholder="NFT URL (e.g. https://jimb.stanford.edu)"
                onChange={i=>{this.setState({tokenURI: i.target.value})}}
                value={tokenURI}
              />
            </Box>
            <Box sx={{mb: 2}}>
              <Button
                disabled={!receiverAddress || !tokenURI}
                onClick={()=>{this.handleSimpleMintAndSend()}}
                loading={loading}
                size="large"
                variant="contained"
              >
                Mint and Send
              </Button>
            </Box>
          </>
        </div>
      </div>
    )
  }
}

const mapStateToProps = state => ({
  nft: state.nft
})

export default connect(mapStateToProps)(listNFTfactory)
