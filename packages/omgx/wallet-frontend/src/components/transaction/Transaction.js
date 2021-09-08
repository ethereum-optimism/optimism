/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */

import React, { useState } from 'react'

import { Button as ButtonMUI, Grid, Typography } from '@material-ui/core'
import * as styles from './Transaction.module.scss'
import * as S from './Transaction.styles'
import { useTheme } from '@emotion/react'
import { selectNetwork } from 'selectors/setupSelector'
import { useSelector } from 'react-redux'
import { getAllNetworks } from 'util/masterConfig'
import Button from 'components/button/Button'

function Transaction({
  link,
  status,
  statusPercentage,
  subStatus,
  button,
  title,
  time,
  subTitle,
  chain,
  typeTX,
  blockNumber,
  tooltip = '',
  detail,
  oriChain,
  oriHash,
}) {

  const [dropDownBox, setDropDownBox] = useState(false)
  const [dropDownBoxInit, setDropDownBoxInit] = useState(false)
  const theme = useTheme()

  const currentNetwork = useSelector(selectNetwork())
  const nw = getAllNetworks()

  const chainLink = () => {
    let network = nw[currentNetwork]
    if (!!network && !!network[oriChain]) {
      if(oriChain === 'L1') {
        //go to etherscan
        return `${network[oriChain].transaction}${oriHash}`;
      } else {
        //the boba blockexplorer
        return `${network[oriChain].transaction}${oriHash}?network=${currentNetwork[0].toUpperCase()+currentNetwork.slice(1)}`;
      }
    }
    return '';
  }

  function renderDetailRedesign() {

    if (!detail) {
      return null
    }

    let prefix = 'L2'
    if( oriChain === 'L2') prefix = 'L1'

    return (

      <S.TableBody
        style={{ justifyContent: 'center' }}
      >
        <S.TableCell sx={{
          gap: '5px',
          width: '98% !important',
          padding: '10px',
          alignItems: 'flex-start !important',
        }}>
          <div
            className={dropDownBox ?
              styles.dropDownContainer : dropDownBoxInit ? styles.dropDownInit : styles.closeDropDown}
          >
            <Grid className={styles.dropDownContent} container spacing={1}>
              <div className={styles.mutedMI}>
                {prefix} Hash:&nbsp;
                  <a className={styles.href} href={detail.txLink} target="_blank" rel="noopener noreferrer">
                    {detail.hash}
                  </a>
              </div>
            </Grid>
            <Grid className={styles.dropDownContent} container spacing={1}>
              <div className={styles.mutedMI}>{prefix} Block:&nbsp;{detail.blockNumber}</div>
            </Grid>
            <Grid className={styles.dropDownContent} container spacing={1}>
              <div className={styles.mutedMI}>{prefix} Block Hash:&nbsp;{detail.blockHash}</div>
            </Grid>
            <Grid className={styles.dropDownContent} container spacing={1}>
              <div className={styles.mutedMI}>{prefix} From:&nbsp;{detail.from}</div>
            </Grid>
            <Grid className={styles.dropDownContent} container spacing={1}>
              <div className={styles.mutedMI}>{prefix} To:&nbsp;{detail.to}</div>
            </Grid>
          </div>
        </S.TableCell>
      </S.TableBody>)
  }

  return (
    <
      div style={{
        padding: '10px',
        borderRadius: '8px',
        background: theme.palette.background.secondary,
      }}
    >
      <S.TableBody>

        <S.TableCell
          sx={{ gap: '5px' }}
          style={{ width: '50%' }}
        >
          <Typography variant="h3">{chain}</Typography>
          <div className={styles.muted}>{time}</div>
          <div className={styles.muted}>{oriChain}&nbsp;Hash:&nbsp;
          <a
              href={chainLink()}
              target={'_blank'}
              rel='noopener noreferrer'
              style={{ color: theme.palette.mode === 'light' ? 'black' : 'white' }}
            >
              {oriHash}
          </a>
          </div>
          <div className={styles.muted}>{typeTX}</div>
          
          {!!detail &&
            <Typography
              variant="body2"
              sx={{
                cursor: 'pointer',
              }}
              onClick={() => {
                setDropDownBox(!dropDownBox)
                setDropDownBoxInit(false)
              }}
            >
              More Information
            </Typography>
          }
        </S.TableCell>

        <S.TableCell
          sx={{ gap: '5px' }}
          style={{ width: '20%' }}
        >
          <div className={styles.muted}>{blockNumber}</div>
        </S.TableCell>

        <S.TableCell sx={{ gap: "5px" }}>
          {button &&
            <Button
              variant="contained"
              color="primary"
              sx={{
                boder: '1.4px solid #506DFA',
                borderRadius: '8px',
                width: '180px'
              }}
              onClick={button.onClick}
            >
              {button.text}
            </Button>
          }

        </S.TableCell>
      </S.TableBody>
      {renderDetailRedesign()}
    </div>)

}

export default Transaction
