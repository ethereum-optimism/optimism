import React from 'react'
import { connect } from 'react-redux'
import { isEqual } from 'lodash'

import ListNFT from 'components/listNFT/listNFT'
import ListNFTfactory from 'components/listNFTfactory/listNFTfactory'

import { openAlert, openError } from 'actions/uiAction'

import * as styles from './Nft.module.scss'

import cellIcon from 'images/hela.jpg'
import factoryIcon from 'images/factory.png'
import koiIcon from 'images/koi.png'

import networkService from 'services/networkService'

import Button from 'components/button/Button'
import Input from 'components/input/Input'

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
      newAddress: '',
      newNFTname: '',
      newNFTsymbol: '',
    }
  }

  componentDidMount() {
    //ToDo
  }

  componentDidUpdate(prevState) {

    const { list, factories } = this.props.nft;
    
    if (!isEqual(prevState.nft.list, list)) {
     this.setState({ list })
    }

    if (!isEqual(prevState.nft.factories, factories)) {
     this.setState({ factories })
    }
 
  }

  async handleDeployContract() {

    const { newNFTsymbol, newNFTname } = this.state;

    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'))
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network'))
      return;
    }

    this.setState({ loading: true })

    let originName = ''

    if(networkService.chainID === 28) {
      originName = 'OMGX_Rinkeby_28'
    } else if (networkService.chainID === 288) {
      originName = 'OMGX_Mainnet_288'
    } else {
      originName = 'OMGX_Other'
    }

    const deployTX = await networkService.deployNewNFTContract(
      newNFTsymbol,
      newNFTname,
      '0x0000000000000000000000000000000000000042',
      'simple',
      originName
    )
    
    if (deployTX) {
      this.props.dispatch(openAlert(`You have deployed a new NFT contract`))
    } else {
      this.props.dispatch(openError('NFT contract deployment error'))
    }

    this.setState({ loading: false })
  }

  render() {

    const { 
      list,
      factories,
      newNFTsymbol,
      newNFTname,
      loading
    } = this.state;

    const numberOfNFTs = Object.keys(list).length
    //const numberOfFactories = Object.keys(factories).length

    let rights = Object.keys(factories).map((v, i) => {
      return factories[v].haveRights
    }).filter(Boolean).length

    return (

      <div className={styles.container}>
        
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
            const key_UUID = `nft_` + i
            return (
              <ListNFT 
                key={key_UUID}
                name={list[v].name}
                symbol={list[v].symbol}
                owner={list[v].owner}
                address={list[v].address}
                layer={list[v].layer}
                icon={list[v].originID === 'simple' ? koiIcon : cellIcon}
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

        <div className={styles.boxContainer}>

          <h2>Mint your own NFTs</h2>
 
          <div className={styles.note}>
            To mint your own NFTs, you first need to deploy your NFT contract. Specify the NFT's name and symbol, and then click 
            "Deploy NFT contract".
          </div> 

          <Input
            small={true}
            placeholder="NFT Symbol (e.g. TWST)"
            onChange={i=>{this.setState({newNFTsymbol: i.target.value})}}
            value={newNFTsymbol}
          />
          <Input
            small={true}
            placeholder="NFT Name (e.g. Twist)"
            onChange={i=>{this.setState({newNFTname: i.target.value})}}
            value={newNFTname}
          />
          <Button
            type='primary'
            size='small'
            disabled={!newNFTname || !newNFTsymbol}
            onClick={()=>{this.handleDeployContract()}}
            loading={loading}
          >
            Deploy NFT contract
          </Button> 

          <div className={styles.TableContainer}>
            {Object.keys(factories).map((v, i) => {
              if(factories[v].haveRights && factories[v].originID === 'simple') {
                const key_UUID = `fac_` + i
                console.log(key_UUID)
                return (
                  <ListNFTfactory 
                    key={key_UUID}
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

        <div className={styles.boxContainer}>

          <h2>Derive an NFT Factory from another NFT (experimental)</h2>

          {rights > 0 && 
            <div className={styles.note}>
              In this tab, you can take an NFT you got from someone and derive a new family of NFTs from it.
              Think of this as creating a "child" NFT from a preceeding "parent" NFT. This is useful, for example, if you generate 
              creative content, and would like to license that content to others, and allow them to build on it, whilst still 
              receiving micropayments for your original contribution and work.
              Status: You have owner permissions for one or more NFT factories. Select the desired NFT factory 
              and click "Actions" to mint NFTs.
            </div> 
          }

          {rights === 0 &&
            <div className={styles.note}>
              In this tab, you can take an NFT you obtained from someone else and derive a new family of NFTs from it.
              Think of this as creating a "child" NFT from a preceeding "parent" NFT. This is useful, for example, if you generate 
              creative content, and would like to license that content to others and allow them to build on it, whilst still 
              receiving micropayments for your original contribution and work.
              Status: You do not have owner permissions. To create your own NFT factory, obtain an NFT first.
            </div> 
          }

          <div className={styles.TableContainer}>
            {Object.keys(factories).map((v, i) => {
              if(factories[v].haveRights && factories[v].originID !== 'simple') {
                const key_UUID = `fac_d_` + i
                return (
                  <ListNFTfactory 
                    key={key_UUID}
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

        

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  nft: state.nft,
  setup: state.setup
});

export default connect(mapStateToProps)(Nft);