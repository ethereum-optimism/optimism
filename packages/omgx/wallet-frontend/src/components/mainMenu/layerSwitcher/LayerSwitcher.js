import React, { useState, useCallback } from 'react'
import { Box } from '@material-ui/system'
import { useSelector, useDispatch } from 'react-redux'
import * as S from './LayerSwitcher.styles.js'
import { selectLayer } from 'selectors/setupSelector'
import { setLayer } from 'actions/setupAction'
import { Typography } from '@material-ui/core'
import networkService from 'services/networkService'
import Button from 'components/button/Button';

import LayerIcon from 'components/icons/LayerIcon';

function LayerSwitcher({ walletEnabled, isButton = false }) {

  const dispatch = useDispatch()
  const [ showAllLayers, setShowAllLayers ] = useState(false)
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
    setShowAllLayers(false)
  }, [ dispatch, setShowAllLayers ])

  if (!!isButton) {
    return (<>
      <Button
        onClick={() => { dispatchSetLayer(otherLayer) }}
        size='medium'
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
