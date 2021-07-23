import React from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';

import { openAlert, openError } from 'actions/uiAction';

import Button from 'components/button/Button'
import Input from 'components/input/Input'

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

import networkService from 'services/networkService';
import root from 'images/root.png';

import * as styles from './listNFT.module.scss';

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
      haveRights
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
      receiverAddress: '',
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
      haveRights 
    }
  }
  
  componentDidUpdate(prevState) {

    const { 
      name, layer, symbol, owner, 
      address, UUID, time, URL,
      oriChain, oriAddress, oriID,
      haveRights 
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

  }

  async handleMintAndSend() {

    const { receiverAddress, ownerName, tokenURI, address } = this.state;
    const networkStatus = await this.props.dispatch(networkService.confirmLayer('L2'))
    
    if (!networkStatus) {
      this.props.dispatch(openError('Please use L2 network.'))
      return
    }

    this.setState({ loading: true });

    const mintTX = await networkService.mintAndSendNFT(
      receiverAddress, 
      address,
      ownerName, 
      tokenURI
    )
    
    if (mintTX) {
      this.props.dispatch(openAlert(`You minted a new NFT for ${receiverAddress}. The owner's name is ${ownerName}.`));
    } else {
      this.props.dispatch(openError('NFT minting error'));
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
      receiverAddress,
      tokenURI,
      ownerName,
      newNFTsymbol,
      newNFTname,
      haveRights 
    } = this.state;

    return (
      <div className={styles.ListNFT}>

        <div 
          className={styles.topContainer} 
        >
          
          <div className={styles.Table1}>
            <img className={styles.Image} src={icon} alt="icon"/>
          </div>

          <div className={styles.Table2}>
            {UUID && <>
              <div className={styles.BasicText}>{name} ({symbol})</div>
              <div className={styles.BasicLightText}>Owner: {truncate(owner, 6, 4, '...')}</div>
              <div className={styles.BasicLightText}>UUID: {UUID}</div>
              <div className={styles.BasicLightText}>Address: {truncate(address, 6, 4, '...')}</div>
              <div className={styles.BasicLightText}>Time minted: {time}</div>
              <a style={{color: 'blue', paddingTop: '2px', fontSize: '0.7em', fontWeight: '700'}} href={URL}>DATASHEET</a>
            </> }
            {!UUID && <>
              <div className={styles.BasicText}>{name} ({symbol}) Factory</div><br/>
              <div className={styles.BasicLightText}>Owner: {truncate(owner, 6, 4, '...')}</div> 
              <div className={styles.BasicLightText}>Address: {truncate(address, 6, 4, '...')}</div>
              <div className={styles.BasicLightText}>Chain: {oriChain}</div>
              <div className={styles.BasicLightText}>Owner rights: {haveRights ? 'True' : 'False'}</div>
            </> }
          </div>

          <div 
            className={styles.Table3}
            onClick={()=>{this.setState({ dropDownBox: !dropDownBox, dropDownBoxInit: false })}}
          >
            <div className={styles.LinkText}>Actions</div>
            <ExpandMoreIcon className={styles.LinkButton} />
          </div>
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
            <img className={styles.Image} src={root} alt="root"/>
            <div className={styles.originRight}>
              <div className={styles.BasicText}>Root</div>
              <div className={styles.BasicLightText}>Address: {oriAddress}</div>
              <div className={styles.BasicLightText}>NFT: {oriID}</div>
              <div className={styles.BasicLightText}>Chain: {oriChain}</div>
            </div>
          </div>

          <div className={styles.boxContainer}>
          {!UUID && haveRights && <>
            <h3>Mint and Send</h3>
            <div 
              className={styles.BasicLightText}
            >To mint and send a new {name} NFT, please fill in the information and click "Mint and Send".</div><br/>
            <Input
              small={true}
              placeholder="Receiver Address (Ox.....)"
              onChange={i=>{this.setState({receiverAddress: i.target.value})}}
              value={receiverAddress}
            />
            <Input
              small={true}
              placeholder="NFT Owner Name (e.g. Henrietta Lacks)"
              onChange={i=>{this.setState({ownerName: i.target.value})}}
              value={ownerName}
            />
            <Input
              small={true}
              placeholder="NFT URL (e.g. https://jimb.stanford.edu)"
              onChange={i=>{this.setState({tokenURI: i.target.value})}}
              value={tokenURI}
            />
            <Button
              type='primary'
              size='small'
              disabled={!receiverAddress || !ownerName || !tokenURI}
              onClick={()=>{this.handleMintAndSend()}}
              loading={loading}
            >
              Mint and Send
            </Button>
          </>}  
          {UUID && <>
            <h3>Derive New NFT Factory</h3>
            <div className={styles.BasicLightText}
            >To derive a new NFT factory from this NFT, please fill in the information and click "Derive".</div><br/>
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
              Derive
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
  /*
  login: state.login,
  sell: state.sell,
  sellTask: state.sellTask,
  buy: state.buy,
  */
});

export default connect(mapStateToProps)(listNFT);