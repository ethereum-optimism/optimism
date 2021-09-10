
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

 import React, { useCallback } from 'react'

import { Box } from '@material-ui/system'
import { useSelector, useDispatch } from 'react-redux'
import * as S from './LayerSwitcher.styles.js'
import { selectLayer } from 'selectors/setupSelector'
import { setLayer } from 'actions/setupAction'
import { Typography } from '@material-ui/core'
import networkService from 'services/networkService'
import Button from 'components/button/Button'

import LayerIcon from 'components/icons/LayerIcon'

function LayerSwitcher({ walletEnabled, isButton = false, size }) {

  const dispatch = useDispatch()

  let layer = useSelector(selectLayer())

  if (networkService.L1orL2 !== layer) {
    //networkService.L1orL2 is always right...
    layer = networkService.L1orL2
  }

  let otherLayer = ''

  if(layer === 'L1') {
    otherLayer = 'L2'
  } else {
    otherLayer = 'L1'
  }

  const dispatchSetLayer = useCallback((layer) => {
    dispatch(setLayer(layer))
    networkService.switchChain(layer)
  }, [ dispatch ])

  if (!!isButton) {
    return (<>
      <Button
        onClick={() => { dispatchSetLayer(otherLayer) }}
        size={size}
        variant="contained"
        >
          SWITCH LAYER
      </Button>
    </>)
  }

  return (
    <S.WalletPickerContainer>
      <S.WalletPickerWrapper>
        <Box
          sx={{
            display: 'flex',
            width: '100%',
            alignItems: 'center'
          }}
        >
          <LayerIcon />
          <S.Label variant="body2">Layer</S.Label>
          <S.LayerSwitch
            onClick={()=>{dispatchSetLayer(otherLayer)}}
          >
            <Typography
              className={layer === 'L1' ? 'active': ''}
              variant="body2"
              component="span"
              color="white">
                1
            </Typography>
            <Typography
              className={layer === 'L2' ? 'active': ''}
              variant="body2"
              component="span"
              color="white">
                2
            </Typography>
          </S.LayerSwitch>
        </Box>
      </S.WalletPickerWrapper>
    </S.WalletPickerContainer>
  )
};

export default LayerSwitcher;
