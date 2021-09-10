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
import { isEqual } from 'lodash';

import { getFarmInfo, getFee } from 'actions/farmAction'

import ListFarm from 'components/listFarm/listFarm'
import Tabs from 'components/tabs/Tabs'
import AlertIcon from 'components/icons/AlertIcon'
import networkService from 'services/networkService'

import * as S from './Farm.styles'
import { Box, Typography } from '@material-ui/core';
import PageHeader from 'components/pageHeader/PageHeader';
import { tableHeadList } from './tableHeadList';
import LayerSwitcher from 'components/mainMenu/layerSwitcher/LayerSwitcher';

class Farm extends React.Component {

  constructor(props) {

    super(props)

    const {
      totalFeeRate,
      userRewardFeeRate,
      poolInfo,
      userInfo,
    } = this.props.farm

    const {
      layer1,
      layer2
    } = this.props.balance


    let initialViewLayer = 'L1 Liquidity Pool'
    let initialLayer = 'L1LP'

    if(networkService.L1orL2 === 'L2') {
      initialViewLayer = 'L2 Liquidity Pool'
      initialLayer = 'L2LP'
    }

    this.state = {
      totalFeeRate,
      userRewardFeeRate,
      poolInfo,
      userInfo,
      layer1,
      layer2,
      lpChoice: initialLayer,
      poolTab: initialViewLayer
    }

  }

  componentDidMount() {

    const { totalFeeRate, userRewardFeeRate } = this.props.farm

    if (!totalFeeRate || !userRewardFeeRate) {
      this.props.dispatch(getFee())
    }

    this.props.dispatch(getFarmInfo())

  }

  componentDidUpdate(prevState) {

    const {
      totalFeeRate,
      userRewardFeeRate,
      poolInfo,
      userInfo,
    } = this.props.farm

    const {
      layer1,
      layer2
    } = this.props.balance

    if (prevState.farm.totalFeeRate !== totalFeeRate) {
      this.setState({ totalFeeRate })
    }

    if (prevState.farm.userRewardFeeRate !== userRewardFeeRate) {
      this.setState({ userRewardFeeRate })
    }

    if (!isEqual(prevState.farm.poolInfo, poolInfo)) {
      this.setState({ poolInfo })
    }

    if (!isEqual(prevState.farm.userInfo, userInfo)) {
      this.setState({ userInfo })
    }

    if (!isEqual(prevState.balance.layer1, layer1)) {
      this.setState({ layer1 })
    }

    if (!isEqual(prevState.balance.layer2, layer2)) {
      this.setState({ layer2 })
    }

  }


  getBalance(address, chain) {

    const { layer1, layer2 } = this.state;

    if (typeof (layer1) === 'undefined') return [0, 0]
    if (typeof (layer2) === 'undefined') return [0, 0]

    if (chain === 'L1') {
      let tokens = Object.entries(layer1)
      for (let i = 0; i < tokens.length; i++) {
        if (tokens[i][1].address.toLowerCase() === address.toLowerCase()) {
          return [tokens[i][1].balance, tokens[i][1].decimals]
        }
      }
    }
    else if (chain === 'L2') {
      let tokens = Object.entries(layer2)
      for (let i = 0; i < tokens.length; i++) {
        if (tokens[i][1].address.toLowerCase() === address.toLowerCase()) {
          return [tokens[i][1].balance, tokens[i][1].decimals]
        }
      }
    }

    return [0, 0]

  }

  handleChange = (event, t) => {
    if( t === 'L1 Liquidity Pool' )
      this.setState({ 
        lpChoice: 'L1LP',
        poolTab: t  
      })
    else if(t === 'L2 Liquidity Pool')
      this.setState({ 
        lpChoice: 'L2LP',
        poolTab: t 
      })
  }

  render() {
    const {
      // Pool
      poolInfo,
      // user
      userInfo,
      lpChoice,
      poolTab
    } = this.state;

    const { isMobile } = this.props

    const networkLayer = networkService.L1orL2
    return (
      <>
        <PageHeader title="Earn" />

        <Box sx={{ my: 3, width: '100%' }}>
          <Box sx={{ mb: 2 }}>
            <Tabs
              activeTab={poolTab}
              onClick={(t)=>this.handleChange(null, t)}
              aria-label="Liquidity Pool Tab"
              tabs={["L1 Liquidity Pool", "L2 Liquidity Pool"]}
            />
          </Box>

          {networkLayer === 'L2' && lpChoice === 'L1LP' &&
            <S.LayerAlert>
              <S.AlertInfo>
                <AlertIcon sx={{flex: 1}} />
                <S.AlertText
                  variant="body1"
                  component="p"
                >
                  Note: MetaMask is set to L2. To interact with the L1 liquidity pool, please switch MetaMask to L1.
                </S.AlertText>
              </S.AlertInfo>
              <LayerSwitcher isButton={true} size={isMobile ? "small" : "medium"}/>
            </S.LayerAlert>
          }

          {networkLayer === 'L1' && lpChoice === 'L2LP' &&
            <S.LayerAlert>
              <S.AlertInfo>
                <AlertIcon />
                <S.AlertText
                  variant="body2"
                  component="p"
                >
                  Note: MetaMask is set to L1. To interact with the L2 liquidity pool, please switch MetaMask to L2.
                </S.AlertText>
              </S.AlertInfo>
              <LayerSwitcher isButton={true} />
            </S.LayerAlert>
          }

          {!isMobile ? (
            <S.TableHeading>
              {tableHeadList.map((item) => {
                return (
                  <S.TableHeadingItem key={item.label} variant="body2" component="div">
                    {item.label}
                  </S.TableHeadingItem>
                )
              })}
            </S.TableHeading>
          ) : (null)}

          {lpChoice === 'L1LP' &&
            <Box>
              {Object.keys(poolInfo.L1LP).map((v, i) => {
                const ret = this.getBalance(v, 'L1')
                return (
                  <ListFarm
                    key={i}
                    poolInfo={poolInfo.L1LP[v]}
                    userInfo={userInfo.L1LP[v]}
                    L1orL2Pool={lpChoice}
                    balance={ret[0]}
                    decimals={ret[1]}
                    isMobile={isMobile}
                  />
                )
              })}
            </Box>}

          {lpChoice === 'L2LP' &&
            <Box>
              {Object.keys(poolInfo.L2LP).map((v, i) => {
                const ret = this.getBalance(v, 'L2')
                return (
                  <ListFarm
                    key={i}
                    poolInfo={poolInfo.L2LP[v]}
                    userInfo={userInfo.L2LP[v]}
                    L1orL2Pool={lpChoice}
                    balance={ret[0]}
                    decimals={ret[1]}
                    isMobile={isMobile}
                  />
                )
              })}
            </Box>
          }
        </Box>
      </>
    )
  }
}

const mapStateToProps = state => ({
  farm: state.farm,
  balance: state.balance
})

export default connect(mapStateToProps)(Farm)
