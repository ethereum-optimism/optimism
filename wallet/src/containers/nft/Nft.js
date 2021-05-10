import React from 'react';
import { connect } from 'react-redux';
import Input from 'components/input/Input';
import NFTCard from 'components/nft/NftCard';
import Button from 'components/button/Button';
import { openError, openAlert } from 'actions/uiAction';
import networkService from 'services/networkService';
import { Grid } from '@material-ui/core/'

import * as styles from './Nft.module.scss';

class Nft extends React.Component {

  constructor(props) {

    super(props);

    const { nftList } = this.props;

    this.state = {
      NFTs: nftList,
      loading: false,
      receiverAddress: '',
      tokenID: 1,
      tokenURI: '' 
    }
  }

  componentDidMount() {
    //ToDo
  }

  componentDidUpdate(prevState) {
    //ToDo
  }

  async handleMintAndSend() {

    const { receiverAddress, tokenID, tokenURI } = this.state;

    const networkStatus = await this.props.dispatch(networkService.checkNetwork('L2'));
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });

    const mintTX = await networkService.mintAndSendNFT(
      receiverAddress, 
      tokenID, 
      tokenURI
    );
    
    if (mintTX) {
      this.props.dispatch(openAlert(`You minted a new NFT for ${receiverAddress}`));
    } else {
      this.props.dispatch(openError('NFT minting error'));
    }

    this.setState({ loading: false })
  }

  render() {

    const { 
      loading,
      receiverAddress,
      tokenID,
      tokenURI,
      NFTs, 
    } = this.state;

    return (

      <div className={styles.container}>
        <div className={styles.boxContainer}>
          
          <h2>Minter/Owner Functions</h2>
          
          <h3>Mint and Send (Contract Owner Only)</h3>
          
          <Input
            placeholder="Receiver Address (e.g. Ox.....)"
            onChange={i=>{this.setState({receiverAddress: i.target.value})}}
          />
          <Input
            placeholder="NFT Unique ID (e.g. 7)"
            onChange={i=>{this.setState({tokenID: i.target.value})}}
          />
          <Input
            placeholder="NFT URL (e.g. https://jimb.stanford.edu)"
            onChange={i=>{this.setState({tokenURI: i.target.value})}}
          />
          <Button
            className={styles.button}
            onClick={() => {this.handleMintAndSend()}}
            type='primary'
            loading={loading}
            disabled={!receiverAddress || !tokenID || !tokenURI}
          >
            Mint and Send
          </Button>
          
          <h3>My NFTs</h3>

          <div className={styles.root}>
            <Grid
              container
              spacing={2}
              direction="row"
              justify="flex-start"
              alignItems="flex-start"
            >
              {Object.keys(NFTs).map(elem => (
                <Grid item xs={12} sm={9} md={6} key={elem}>
                  <NFTCard
                    name={NFTs[elem].name}
                    symbol={NFTs[elem].symbol}
                    UUID={NFTs[elem].UUID}
                    owner={NFTs[elem].owner}
                    URL={NFTs[elem].url}
                    time={NFTs[elem].mintedTime}
                  >
                  </NFTCard>
                </Grid>
              ))}
            </Grid>
          </div>

        </div>

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  nftList: state.nftList
});

export default connect(mapStateToProps)(Nft);