import React from 'react'
import { connect } from 'react-redux'
import { isEqual } from 'lodash'

import ListNFT from 'components/listNFT/listNFT'
import ListNFTfactory from 'components/listNFTfactory/listNFTfactory'

import * as styles from './Nft.module.scss'

import cellIcon from 'images/hela.jpg'
import factoryIcon from 'images/factory.png'

class Nft extends React.Component {

  constructor(props) {

    super(props);

    const { list, factories } = this.props.nft;

    this.state = {
      list,
      factories,
      loading: false,
      ownerName: '',
      tokenURI: '',
      newAddress: ''
    }
  }

  componentDidMount() {
    //ToDo
  }

  componentDidUpdate(prevState) {

    const { list, factories } = this.props.nft;
    
    if (!isEqual(prevState.nft.list, list)) {
     this.setState({ list });
    }

    if (!isEqual(prevState.nft.factories, factories)) {
     this.setState({ factories });
    }
 
  }

  render() {

    const { 
      list,
      factories
    } = this.state;

    const numberOfNFTs = Object.keys(list).length
    //const numberOfFactories = Object.keys(factories).length

    let rights = Object.keys(factories).map((v, i) => {
      return factories[v].haveRights
    }).filter(Boolean).length

    return (

      <div className={styles.container}>
        
        <div className={styles.boxContainer}>

          <h2>NFT Factories</h2>

          {rights > 0 && 
            <div className={styles.note}>
              Status: You have owner permissions for one or more NFT factories 
              and are authorized to mint new NFTs. Select the desired NFT factory 
              and click "Actions" to mint NFTs.
            </div> 
          }

          {rights === 0 &&
            <div className={styles.note}>
              Status: You do not have owner permissions and you not 
              are authorized to mint new NFTs. To create you own NFT 
              factory, obtain an NFT first.
            </div> 
          }

          <div className={styles.TableContainer}>
            {Object.keys(factories).map((v, i) => {
              if(factories[v].haveRights) {
              return (
                <ListNFTfactory 
                  key={i}
                  name={factories[v].name}
                  symbol={factories[v].symbol}
                  owner={factories[v].owner}
                  address={factories[v].address}
                  layer={factories[v].layer}
                  icon={factoryIcon}
                  oriChain={factories[v].originChain}
                  oriAddress={factories[v].originAddress}
                  oriID={factories[v].originID}
                  oriFeeRecipient={factories[v].originFeeRecipient}
                  haveRights={factories[v].haveRights}
                />
              )
              } else {
                return (<></>)
              }
            })}
          </div>
        </div>

        <div 
          className={styles.nftContainer}
        >
          
          <h2>Your NFTs</h2>

          {numberOfNFTs === 1 && 
            <div className={styles.note}>You have one NFT and it should be shown below.</div> 
          }
          {numberOfNFTs > 1 && 
            <div className={styles.note}>You have {numberOfNFTs} NFTs and they should be shown below.</div> 
          }
          {numberOfNFTs < 1 &&
            <div className={styles.note}>Scanning the blockchain for your NFTs...</div> 
          }
          <div className={styles.nftTiles} >
          {Object.keys(list).map((v, i) => {
            return (
              <ListNFT 
                key={i}
                name={list[v].name}
                symbol={list[v].symbol}
                owner={list[v].owner}
                address={list[v].address}
                layer={list[v].layer}
                icon={cellIcon}
                UUID={list[v].UUID}
                URL={list[v].url}
                time={list[v].mintedTime}
                oriChain={list[v].originChain}
                oriAddress={list[v].originAddress}
                oriID={list[v].originID}
                oriFeeRecipient={list[v].originFeeRecipient}
                type={list[v].type}
              />
            )
          })
          }
          </div>

        </div>

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  nft: state.nft,
  setup: state.setup
});

export default connect(mapStateToProps)(Nft);