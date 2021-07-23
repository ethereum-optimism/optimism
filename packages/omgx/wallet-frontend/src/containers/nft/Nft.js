import React from 'react'
import { connect } from 'react-redux'
import { isEqual } from 'lodash'

import ListNFT from 'components/listNFT/listNFT'

import Button from 'components/button/Button'
import Input from 'components/input/Input'
import networkService from 'services/networkService'
import { openError } from 'actions/uiAction'
import { addNFTContract } from 'actions/nftAction'

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

  async addNewNFT() {

    const { newAddress  } = this.state;

    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'))
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true })

    await addNFTContract( newAddress) 
    await networkService.addNFTFactoryNS( newAddress )

    this.setState({ newAddress: '' });

    this.setState({ loading: false })

  }

  render() {

    const { 
      list,
      factories,
      newAddress,
      loading 
    } = this.state;

    //console.log("Factories:",factories)

    const numberOfNFTs = Object.keys(list).length
    const numberOfFactories = Object.keys(factories).length

    return (

      <div className={styles.container}>
        
        <div className={styles.boxContainer}>

          <h2>Your NFT Factories</h2>

          {numberOfFactories > 0 && 
            <div className={styles.note}>
              Status: You have owner permissions and are authorized to mint new NFTs. 
              Select the desired NFT factory and click "Actions" to mint NFTs.
            </div> 
          }
          {numberOfFactories === 0 &&
            <div className={styles.note}>
              Status: You do not have owner permissions and you not 
              are authorized to mint new NFTs. 
              Obtain an NFT, and then you can derive new NFTs from it.
            </div> 
          }

          <div className={styles.TableContainer}>
            {Object.keys(factories).map((v, i) => {
              return (
                <ListNFT 
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
                  haveRights={factories[v].haveRights}
                />
              )
            })}
          </div>
        </div>

        <div className={styles.nftContainer}>
          
          <h2>Your NFTs</h2>

            <div className={styles.note}>
              To add an NFT, please fill in its contract address and click "Add".
            </div> 

            <div className={styles.ListNFT}>
              
              <div style={{fontSize: '1em', fontWeight: 700, paddingBottom: '5px'}}>Add NFT</div>

              <Input
                small={false}
                style={{width: '90%'}}
                placeholder="New NFT contract address (0x...)"
                onChange={i=>{this.setState({newAddress: i.target.value})}}
                value={newAddress}
              />
              
              <Button
                type='primary'
                size='small'
                disabled={!newAddress}
                onClick={()=>{this.addNewNFT()}}
                loading={loading}
              >
                Add NFT
              </Button>
              
          </div>

          {numberOfNFTs === 1 && 
            <div className={styles.note}>You have one NFT and it should be shown below.</div> 
          }
          {numberOfNFTs > 1 && 
            <div className={styles.note}>You have {numberOfNFTs} NFTs and they should be shown below.</div> 
          }
          {numberOfNFTs < 1 &&
            <div className={styles.note}>You do not have any NFTs.</div> 
          }

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
                />
              )
            })
          }

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