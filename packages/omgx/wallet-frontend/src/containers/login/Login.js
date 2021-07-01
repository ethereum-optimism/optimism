/*
  Utility Functions for OMG Plasma 
  Copyright (C) 2021 Enya Inc. Palo Alto, CA

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import React from 'react';
import { connect } from 'react-redux';
import Button from 'components/button/Button';

import { 
  providePassword, 
  updateIsBeginner, 
  // verifyInvitationCode,
  login 
} from 'actions/loginAction';

import { 
  isItemOpenOrClosed, 
} from 'actions/sellAction';

import { 
  isBidOpenOrClosed, 
} from 'actions/buyAction';

import networkService from 'services/networkService';

import * as styles from './Login.module.scss';

/* Calculate a 32 bit FNV-1a hash
 * https://gist.github.com/vaiorabbit/5657561
 * Academic Ref.: http://isthe.com/chongo/tech/comp/fnv/
 *
 * @param {string} str the input value
 * @param {boolean} [asString=false] set to true to return the hash value as 
 *     8-digit hex string instead of an integer
 * @param {integer} [seed] optionally pass the hash of the previous chunk
 * @returns {integer | string}
 */
function hashFnv32a(str) {
  /*jshint bitwise:false */

  var i,l;
  var hval = 0x811c9dc5;

  for (i = 0, l = str.length; i < l; i++) {
    hval ^= str.charCodeAt(i);
    hval += (hval << 1) + (hval << 4) + (hval << 7) + (hval << 8) + (hval << 24);
  }

  return hval >>> 0;
}

class Login extends React.Component {

  constructor(props) {
    
    super(props);
    
    const { 
      FHEseed, 
      isBeginner,
      latestOpenitemID, 
      latestOpenBidID,
      // invitationCodeVerifyLoad, 
      // invitationCodeVerifyError,
      // invitationCodeGood,
    } = this.props.login;

    const { 
      itemOpenOrClosed, 
      itemOpenList, 
      itemOpenOrClosedLoad, 
      itemOpenOrClosedLoadIndicator,
      decryptedItemLoad, 
      decryptedItemError,
      // goodItemDecrypt,
    } = this.props.sell;

    this.state ={

      password: '', //UI purposes only
      FHEseed,      //this will get shared
      // Login data
      isBeginner, 
      // verify invitation code
      // invitationCodeVerifyLoad, 
      // invitationCodeVerifyError,
      // invitationCodeGood,

      latestOpenitemID, 
      latestOpenBidID,

      // Valid itemID
      itemOpenOrClosed, 
      itemOpenList, 
      itemOpenOrClosedLoad, 
      itemOpenOrClosedLoadIndicator,

      // Decrypt ask order
      decryptedItemLoad, 
      decryptedItemError, 
    }
  }

  componentDidMount() {
    this.props.dispatch(isItemOpenOrClosed()).then(itemOpenOrClosedData => {
      if (itemOpenOrClosedData) this.props.dispatch(updateIsBeginner(true));
    });
    this.props.dispatch(isBidOpenOrClosed());
  }

  componentDidUpdate(prevState) {

    const { 
      isBeginner, 
      FHEseed,
      // invitationCodeVerifyLoad, 
      // invitationCodeVerifyError,
      // invitationCodeGood,
    } = this.props.login;

    if (prevState.login.isBeginner !== isBeginner) {
      this.setState({ isBeginner });
    }

    if (prevState.login.FHEseed !== FHEseed) {
      this.props.dispatch(login());
    }

    // if (prevState.login.invitationCodeVerifyLoad !== invitationCodeVerifyLoad) {
    //   this.setState({ invitationCodeVerifyLoad });
    // }

    // if (prevState.login.invitationCodeVerifyError !== invitationCodeVerifyError) {
    //   this.setState({ invitationCodeVerifyError });
    // }

    // if (prevState.login.invitationCodeGood !== invitationCodeGood ||
    //     prevState.login.FHEseed !== FHEseed
    //   ) {
    //   this.setState({ invitationCodeGood, FHEseed });
    //   if (invitationCodeGood && FHEseed) {
    //     this.props.dispatch(login());
    //   }
    // }
  }

  handleProvidePassword(event) {
    this.setState({ password: event.target.value });
  }

  handlePasswordKeyDown(event) {
    const { invitationCode } = this.state;
    if (event.key === 'Enter' && invitationCode) {
      this.handleSubmit();
    }
  }

  handleSubmit() {
    const { /*invitationCode,*/ password } = this.state;
    //and prepare the seed
    const FHEseed = hashFnv32a(password);
    this.setState({ FHEseed });
    //and send the seed
    this.props.dispatch(providePassword( FHEseed ));
    //Let's check the invitation code
    // this.props.dispatch(verifyInvitationCode( invitationCode ));
  }

  render() {

    const {
      password, 
      // invitationCode, 
      isBeginner, 
      // invitationCodeGood,
      // invitationCodeVerifyLoad,
      // invitationCodeVerifyError,
    } = this.state;

    let submitButtonText = 'Use Varna';
    // let buttonLoading = false;
    
    // if (invitationCodeVerifyLoad) {
    //   submitButtonText = 'VERIFYING INVITATION CODE';
    //   buttonLoading = true;
    // }

    // if (invitationCodeGood) {
    //   submitButtonText = 'ALL SET - USE VARNA';
    //   buttonLoading = false;
    // }

    let noteText = '';

    if (!isBeginner) {
      noteText = `Please enter your password to decrypt your bids and items. If you 
      enter the wrong password, bids and items will remain encrypted and invisible. 
      If Varna is suddenly blank, that means that you used the wrong password.`;
    } else if (isBeginner) {
      noteText = `Welcome to Varna! Please set a good password, such as a string of 
      20 random characters. If you lose your password, then you will lose access to 
      your bids and listed items.`;
    }

    return (
      <div className={styles.Login}>
        <div className={styles.LoginInputContainer}>
          <div className={styles.Note}>
            {noteText}
          </div>
          <div className={styles.Note}>
            <br/>
            {`WARNING: There is no way to recover your password. If you lose your password, 
            you will have to relist your items and/or re-broadcast your bids.`}
          </div>
          <form>
          <h3>Password</h3>
          <input
            value={password}
            type="password"
            placeholder="Password"
            autoComplete="current-password"
            onChange={event=>{this.handleProvidePassword(event)}}
            onKeyDown={event=>{this.handlePasswordKeyDown(event)}}
            // disabled={buttonLoading}
          />
          {/* <h3>Invitation Code</h3>
          <input
            value={invitationCode}
            placeholder="Invitation Code"
            onChange={event=>{this.handleVerifyInvitationCode(event)}}
            disabled={buttonLoading}
          /> */}
          <Button
            onClick={()=>{this.handleSubmit()}}
            type='primary'
            style={{marginTop: '10px'}}
            // disabled={buttonLoading || !password || !invitationCode}
            disabled={!password || networkService.L1orL2 !== 'L2'}
            // loading={buttonLoading}
          >
            {submitButtonText}
          </Button>
          </form>
          {/* {invitationCodeVerifyError &&
            <div className={styles.ErrorMessageBox}>
              Invalid invitation code
            </div>
          } */}
          <br/>
        </div>
      </div>
    )
  }
}

const mapStateToProps = state => ({ 
  login: state.login,
  sell: state.sell,
  sellTask: state.sellTask,
  buy: state.buy,
});

export default connect(mapStateToProps)(Login);