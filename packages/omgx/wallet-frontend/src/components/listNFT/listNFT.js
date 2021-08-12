import React from 'react'
import { connect } from 'react-redux'
import { isEqual } from 'lodash'

import { openAlert, openError } from 'actions/uiAction'

import Button from 'components/button/Button'
import Input from 'components/input/Input'

import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

import networkService from 'services/networkService'

import { transfer } from 'actions/networkAction'

import * as styles from './listNFT.module.scss'

import truncate from 'truncate-middle'

class listNFT extends React.Component {
  
  constructor(props) {
    
    super(props);
    
    const {
      name,
      symbol,
      owner,
      address,
      layer,
      icon,
      UUID,
      time,
      URL,
      oriChain,
      oriAddress,
      oriID,
      haveRights,
      type,
      oriFeeRecipient
    } = this.props;


    this.state = {
      name,
      symbol,
      owner,
      address,
      layer,
      icon,
      UUID,
      time,
      URL,
      tokenURI: '',
      ownerName: '',
      //drop down box
      dropDownBox: false,
      dropDownBoxInit: true,
      // loading
      loading: false,
      newNFTsymbol: '',
      newNFTname: '',
      oriChain,
      oriAddress,
      oriID,
      haveRights,
      type,
      typeNew : 0,
      oriFeeRecipient 
    }
  }
  
  componentDidUpdate(prevState) {

    const { 
      name, layer, symbol, owner, 
      address, UUID, time, URL,
      oriChain, oriAddress, oriID,
      haveRights, type, oriFeeRecipient,
      typeNew 
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

    if (!isEqual(prevState.owner, owner)) {
      this.setState({ owner })
    }

    if (!isEqual(prevState.address, address)) {
      this.setState({ address })
    }

    if (!isEqual(prevState.UUID, UUID)) {
      this.setState({ UUID })
    }

    if (!isEqual(prevState.time, time)) {
      this.setState({ time })
    }

    if (!isEqual(prevState.URL, URL)) {
      this.setState({ URL })
    }

    if (!isEqual(prevState.oriChain, oriChain)) {
      this.setState({ oriChain })
    }

    if (!isEqual(prevState.oriAddress, oriAddress)) {
      this.setState({ oriAddress })
    }

    if (!isEqual(prevState.oriID, oriID)) {
      this.setState({ oriID })
    }

    if (!isEqual(prevState.haveRights, haveRights)) {
      this.setState({ haveRights })
    }

    if (!isEqual(prevState.type, type)) {
      this.setState({ type })
    }

    if (!isEqual(prevState.typeNew, typeNew)) {
      this.setState({ typeNew })
    }

    if (!isEqual(prevState.oriFeeRecipient, oriFeeRecipient)) {
      this.setState({ oriFeeRecipient })
    }

  }

  async handleMintAndSend() {

    const { receiverAddress, ownerName, tokenURI, address, typeNew, oriFeeRecipient } = this.state;
    
    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'))
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'))
      return
    }

    this.setState({ loading: true })

    const mintTX = await networkService.mintAndSendNFT(
      receiverAddress, 
      address,
      ownerName, 
      tokenURI,
      typeNew //can be 0, 1, or 2 - 0 denotes full rights
    )

    //for the payment, this is always in oETH, is always 0.01 oETH in magnitude (for now), and 
    //goes to the owner of the NFT that was the parent of the NFT you are sending to someone else 

    const ETHL2 = '0x4200000000000000000000000000000000000006'

    if (mintTX) {
      
      this.props.dispatch(openAlert(`You minted a new NFT for ${receiverAddress}. 
        The owner's name is ${ownerName}. 
        You will now be prompted to send a payment to the creator of the parent NFT`
      ))

      try {
        const transferResponse = await this.props.dispatch(
          transfer(oriFeeRecipient, 0.01, ETHL2)
        )
        if (transferResponse) {
          this.props.dispatch(openAlert('Payment submitted'))
        }
      } catch (err) {
        //guess not really?
      }

    } else {
      this.props.dispatch(openError('NFT minting error'))
    }
 
    this.setState({ loading: false })
  }

  async handleDeployDerivative() {

    const { newNFTsymbol, newNFTname, address, UUID  } = this.state;

    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'))
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'));
      return;
    }

    this.setState({ loading: true });

    const deployTX = await networkService.deployNewNFTContract(
      newNFTsymbol,
      newNFTname,
      address,
      UUID,
      'OMGX_Rinkeby_28'
    )
    
    if (deployTX) {
      this.props.dispatch(openAlert(`You have deployed a new NFT factory.`));
    } else {
      this.props.dispatch(openError('NFT factory deployment error'));
    }

    this.setState({ loading: false })
  }

  render() {

    const { 
      name,
      symbol,
      owner,
      address,
      icon,
      UUID,
      time,
      URL,
      oriChain,
      oriAddress,
      oriID, 
      dropDownBox, 
      dropDownBoxInit,
      loading,
      newNFTsymbol,
      newNFTname,
      type,
      oriFeeRecipient 
    } = this.state;

    let typeString = 'Commercial; derivatizable'

    if(type === 1){
      typeString = 'Commercial; no derivatives'
    } else if (type === 2) {
      typeString = 'Non-profit; no derivatives'
    }

    return (
      <div className={styles.ListNFT}>

        <img className={styles.Image} src={icon} alt="icon"/>

        <div 
          className={styles.topContainer}
        >
          
          {oriID === 'simple' && 
            <div className={styles.Table2}>
              <div className={styles.BasicText}>{name} ({symbol})</div>
              <div className={styles.BasicLightText}>Owner: {owner}</div>
              <div className={styles.BasicLightText}>UUID: {UUID}</div>
              <div className={styles.BasicLightText}>Address: {truncate(address, 6, 4, '...')}</div>
              <div className={styles.BasicLightText}>Time minted: {time}</div>
              <a className={styles.URILink} href={URL}>Link</a>
            </div>
          }

          {oriID !== 'simple' && <>
            <div className={styles.Table2}>
              <div className={styles.BasicText}>{name} ({symbol})</div>
              <div className={styles.BasicLightText}>Owner: {owner}</div>
              <div className={styles.BasicLightText}>UUID: {UUID}</div>
              <div className={styles.BasicLightText}>Address: {truncate(address, 6, 4, '...')}</div>
              <div className={styles.BasicLightText}>Time minted: {time}</div>
              <div className={styles.BasicLightText}>Type: {typeString}</div>
              <a className={styles.URILink} href={URL}>DATASHEET</a>
            </div>
          
            <div 
              className={styles.Table3}
              onClick={()=>{this.setState({ dropDownBox: !dropDownBox, dropDownBoxInit: false })}}
            >
              <div className={styles.LinkText}>Actions</div>
              <ExpandMoreIcon className={styles.LinkButton} />
            </div>
          </>
          }

        </div>

        {/*********************************************/
        /**************  Drop Down Box ****************/
        /**********************************************/
        }
        <div 
          className={dropDownBox ? 
            styles.dropDownContainer: dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}
        >
          
          <div className={styles.boxOrigin}>
            <div className={styles.BasicText}>Root</div>
            <div className={styles.BasicLightText}>Address: {oriAddress}</div>
            <div className={styles.BasicLightText}>NFT: {oriID}</div>
            <div className={styles.BasicLightText}>Chain: {oriChain}</div>
            <div className={styles.BasicLightText}>Fee recipient: {oriFeeRecipient}</div>
          </div>

          <div className={styles.boxContainer}>
          {(type === 0) && <>
            <h3>Derive New NFT Factory</h3>
            <div className={styles.BasicLightText}
            >To create a new NFT factory from this NFT, please fill in the information and click "Create New NFT Factory".</div><br/>
            <Input
              small={true}
              placeholder="NFT Symbol (e.g. TWST)"
              onChange={i=>{this.setState({newNFTsymbol: i.target.value})}}
              value={newNFTsymbol}
            />
            <Input
              small={true}
              placeholder="NFT Name (e.g. Twist Bio NFT)"
              onChange={i=>{this.setState({newNFTname: i.target.value})}}
              value={newNFTname}
            />
            <Button
              type='primary'
              size='small'
              disabled={!newNFTname || !newNFTsymbol}
              onClick={()=>{this.handleDeployDerivative()}}
              loading={loading}
            >
              Create New NFT Factory
            </Button>
          </>}  
          </div>
        </div>

      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  nft: state.nft
})

export default connect(mapStateToProps)(listNFT)