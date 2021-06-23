import React from 'react';
import { connect } from 'react-redux';
import Input from 'components/input/Input';
import NFTCard from 'components/nft/NftCard';
import Button from 'components/button/Button';
import { openError, openAlert } from 'actions/uiAction';
import networkService from 'services/networkService';
import { Grid } from '@material-ui/core/'
import { isEqual } from 'lodash';

import * as styles from './Nft.module.scss';

class Nft extends React.Component {

  constructor(props) {

    super(props);

    const { nftList } = this.props;
    const { minter } = this.props.setup;

    //console.log(this.props)
    console.log(this.props.setup)

    this.state = {
      NFTs: nftList,
      minter: minter,
      loading: false,
      receiverAddress: '',
      ownerName: '',
      tokenURI: '' 
    }
  }

  componentDidMount() {
    //ToDo
  }

  componentDidUpdate(prevState) {
    const { nftList } = this.props;
    if (!isEqual(prevState.nftList, nftList)) {
      this.setState({ NFTs: nftList });
    }
  }

  async handleMintAndSend() {

    const { receiverAddress, ownerName, tokenURI } = this.state;

    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'));
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });

    const mintTX = await networkService.mintAndSendNFT(
      receiverAddress, 
      ownerName, 
      tokenURI
    );
    
    if (mintTX) {
      this.props.dispatch(openAlert(`You minted a new NFT for ${receiverAddress}. The owner's name is ${ownerName}.`));
    } else {
      this.props.dispatch(openError('NFT minting error'));
    }

    this.setState({ loading: false })
  }

  render() {

    const { 
      loading,
      receiverAddress,
      ownerName,
      tokenURI,
      NFTs,
      minter 
    } = this.state;

    const numberOfNFTs = Object.keys(NFTs).length;

    return (

      <div className={styles.container}>
        <div className={styles.boxContainer}>
          
          <h2>Minter/Owner Functions</h2>
                    
          {minter && 
            <div className={styles.note}>Status: You have owner permissions and are authorized to mint new NFTs. Once you have filled in all the information, click "Mint and Send".</div> 
          }
          {!minter &&
            <div className={styles.note}>Status: You do not have owner permissions and you not are authorized to mint new NFTs. The input fields are disabled.</div> 
          }

          <Input
            placeholder="Receiver Address (e.g. Ox.....)"
            onChange={i=>{this.setState({receiverAddress: i.target.value})}}
            value={receiverAddress}
            disabled={!minter}
          />
          <Input
            placeholder="NFT Owner Name (e.g. Henrietta Lacks)"
            onChange={i=>{this.setState({ownerName: i.target.value})}}
            value={ownerName}
            disabled={!minter}
          />
          <Input
            placeholder="NFT URL (e.g. https://jimb.stanford.edu)"
            onChange={i=>{this.setState({tokenURI: i.target.value})}}
            value={tokenURI}
            disabled={!minter}
          />
          <Button
            className={styles.button}
            onClick={() => {this.handleMintAndSend()}}
            type='primary'
            loading={loading}
            disabled={!receiverAddress || !ownerName || !tokenURI || !minter}
          >
            Mint and Send
          </Button>
          
          <h2>My NFTs</h2>

          {numberOfNFTs === 1 && 
            <div className={styles.note}>You have one NFT and it should be shown below.</div> 
          }
          {numberOfNFTs > 1 && 
            <div className={styles.note}>You have {numberOfNFTs} NFTs and they should be shown below.</div> 
          }
          {numberOfNFTs < 1 &&
            <div className={styles.note}>You do not have any NFTs.</div> 
          }

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
  nftList: state.nftList,
  setup: state.setup
});

export default connect(mapStateToProps)(Nft);