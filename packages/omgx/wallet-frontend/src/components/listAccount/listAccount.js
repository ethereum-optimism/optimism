import React from 'react'
import { connect } from 'react-redux'
import { logAmount } from 'util/amountConvert'
import { isEqual } from 'lodash'

import { openModal } from 'actions/uiAction'
import Button from 'components/button/Button'

import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

import { Box, Typography, Fade } from '@material-ui/core'
import * as S from './ListAccount.styles'
import EthereumIcon from 'components/icons/EthereumIcon'
import LayerSwitcher from 'components/mainMenu/layerSwitcher/LayerSwitcher'
import { getCoinImage } from 'util/coinImage'

class ListAccount extends React.Component {

  constructor(props) {
    super(props)
    const { token, chain, networkLayer, disabled } = this.props
    this.state = {
      token,
      chain,
      dropDownBox: false,
      networkLayer,
      disabled
    }
  }

  componentDidUpdate(prevState) {

    const { token, chain, networkLayer, disabled } = this.props;

    if (!isEqual(prevState.token, token)) {
      this.setState({ token });
    }

    if (!isEqual(prevState.chain, chain)) {
      this.setState({ chain });
    }

    if (!isEqual(prevState.networkLayer, networkLayer)) {
      this.setState({ networkLayer });
    }

    if (!isEqual(prevState.disabled, disabled)) {
      this.setState({ disabled });
    }

  }

  handleModalClick(modalName, token, fast) {
    this.props.dispatch(openModal(modalName, token, fast))
  }

  render() {

    const {
      token,
      chain,
      dropDownBox,
      networkLayer,
      disabled
    } = this.state;

    const enabled = (networkLayer === chain) ? true : false
    const logo = getCoinImage(token.symbol)

    return (
      <>
        <S.Content>
            <S.TableBody disabled={true}>

              <S.TableCell sx={{gap: "10px", justifyContent: "flex-start"}}>
                <img src={logo} alt="logo" width={42} height={42} />

                <S.TextTableCell enabled={`${enabled}`} variant="body2" component="div">
                  {token.symbol}
                </S.TextTableCell>
              </S.TableCell>

              <S.TableCell>
                <S.TextTableCell enabled={`${enabled}`} variant="body2" component="div" sx={{fontWeight:"700"}}>
                  {`${logAmount(token.balance, 18, 2)}`}
                </S.TextTableCell>
              </S.TableCell>

              <S.TableCell
                onClick={() => {
                  this.setState({
                    dropDownBox: !dropDownBox,
                    dropDownBoxInit: false
                  })
                }}
                sx={{cursor: "pointer", gap: "5px", justifyContent: "flex-end"}}
              >
                {chain === 'L1' &&
                  <S.TextTableCell enabled={`${enabled}`} variant="body2" component="div">
                    Deposit
                  </S.TextTableCell>
                }
                {chain === 'L2' &&
                  <S.TextTableCell enabled={`${enabled}`} variant="body2" component="div">
                    Transact
                  </S.TextTableCell>
                }
                <Box sx={{display: "flex", opacity: !enabled ? "0.4" : "1.0", transform: dropDownBox ? "rotate(-180deg)" : ""}}>
                  <ExpandMoreIcon sx={{width: "12px"}}/>
                </Box>
              </S.TableCell>
            </S.TableBody>

          {/*********************************************/
          /**************  Drop Down Box ****************/
          /**********************************************/
          }

          {dropDownBox ? (
          <Fade in={dropDownBox}>
            <S.DropdownWrapper>
              {!enabled && chain === 'L1' &&
                <S.AccountAlertBox>
                  <Box
                       sx={{
                         wordBreak: 'break-all',
                         flex: 1,
                         width: '75%'
                       }}
                     >
                       <Typography variant="body2" component="p" >
                         MetaMask is set to L2. To transact on L1, - SWITCH LAYER to L1
                       </Typography>
                     </Box>
                     <Box
                       sx={{
                         textAlign: 'center',
                         width: '25%'
                       }}
                     >
                       <LayerSwitcher isButton={true} />
                     </Box>
                </S.AccountAlertBox>
              }

              {!enabled && chain === 'L2' &&
                <S.AccountAlertBox>
                  <Box
                       sx={{
                         wordBreak: 'break-all',
                         flex: 1,
                         width: '75%'
                       }}
                     >
                       <Typography variant="body2" component="p" >
                         MetaMask is set to L1. To transact on L2, - SWITCH LAYER to L2
                       </Typography>
                     </Box>
                     <Box
                       sx={{
                         textAlign: 'center',
                         width: '25%'
                       }}
                     >
                       <LayerSwitcher isButton={true} />
                     </Box>
                </S.AccountAlertBox>
              }

              {enabled && chain === 'L1' &&
              <>
                <Button
                  onClick={()=>{this.handleModalClick('depositModal', token, false)}}
                  color='neutral'
                  variant="outlined"
                  disabled={disabled}
                  fullWidth
                >
                  Deposit
                </Button>

                <Button
                  onClick={()=>{this.handleModalClick('depositModal', token, true)}}
                  color='primary'
                  disabled={disabled}
                  variant="contained"
                  fullWidth
                >
                  Fast Deposit
                </Button>
              </>
              }

              {enabled && chain === 'L2' &&
                <>
                  <Button
                    onClick={()=>{this.handleModalClick('exitModal', token, false)}}
                    variant="outlined"
                    disabled={disabled}
                    fullWidth
                  >
                    Standard Exit
                  </Button>

                  <Button
                    onClick={()=>{this.handleModalClick('exitModal', token, true)}}
                    variant="contained"
                    disabled={disabled}
                    fullWidth
                  >
                    Fast Exit
                  </Button>

                  <Button
                    onClick={()=>{this.handleModalClick('transferModal', token, false)}}
                    variant="contained"
                    disabled={disabled}
                    fullWidth
                  >
                    Transfer
                  </Button>
                </>
              }
            </S.DropdownWrapper>
          </Fade>
          ) : null}
        </S.Content>
      </>
    )
  }
}

const mapStateToProps = state => ({
  //login: state.login,
  //sell: state.sell,
  //sellTask: state.sellTask,
  //buy: state.buy,
});

export default connect(mapStateToProps)(ListAccount);
