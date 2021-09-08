import React, { useState, useRef, useCallback, useEffect } from 'react'
import { openModal } from 'actions/uiAction'
import { Box } from '@material-ui/system'
import { useSelector, useDispatch } from 'react-redux'
import * as S from './NetworkSwitcher.styles.js'
import chevron from 'images/chevron.svg'
import { selectNetwork } from 'selectors/setupSelector'
import { setNetwork } from 'actions/setupAction'
import { getAllNetworks } from 'util/masterConfig'
import { Typography } from '@material-ui/core'
import networkService from 'services/networkService'
import WrongNetworkModal from 'containers/modals/wrongnetwork/WrongNetworkModal'
import { selectModalState } from 'selectors/uiSelector'

import NetworkIcon from 'components/icons/NetworkIcon'

function NetworkSwitcher({ walletEnabled }) {

  const dispatch = useDispatch()
  const dropdownNode = useRef(null)
  const [ showAllNetworks, setShowAllNetworks ] = useState(false)
  const masterConfig = useSelector(selectNetwork())

  const [ wrongNetwork, setWrongNetwork ] = useState(false)
  const wrongNetworkModalState = useSelector(selectModalState('wrongNetworkModal'))

  // defines the set of possible networks
  const networks = getAllNetworks()

  let allNetworks = []
  for (var prop in networks) allNetworks.push(prop)

  const dispatchSetNetwork = useCallback((network) => {
    dispatch(setNetwork(network))

    async function initializeAccounts () {
      const initialized = await networkService.initializeAccounts(network);
      if (initialized === 'wrongnetwork') {
        return setWrongNetwork(true)
      }
    }

    initializeAccounts()

    setShowAllNetworks(false)
  }, [ dispatch ])

  useEffect(() => {
    if (wrongNetwork) {
      dispatch(openModal('wrongNetworkModal'))
      localStorage.setItem('changeChain', false)
    }
  }, [ dispatch, wrongNetwork ]);

  function resetSelection () {
    //dispatchSetWalletMethod(null);
    //setWalletEnabled(false);
    //setAccountsEnabled(false);
  }

  return (
    <S.WalletPickerContainer>
      <WrongNetworkModal
        open={wrongNetworkModalState}
        onClose={resetSelection}
      />
      <S.WallerPickerWrapper>
        <S.Menu>
          <S.NetWorkStyle>
            <NetworkIcon />
            <S.Label variant="body2">Network</S.Label>
            <Box sx={{
              display: 'flex',
              margin: '10px 0 10px 15px',
              alignItems: 'center',
              gap: 2,
              position: 'relative',
            }}
            onClick={()=>{setShowAllNetworks(prev => !prev)}}>
              <Typography variant="body2" sx={{ display: 'flex', alignItems: 'center', cursor: 'pointer', textTransform: 'capitalize'}}>
                {masterConfig}
                <S.Chevron
                  open={showAllNetworks}
                  src={chevron}
                  alt='chevron'
                />
              </Typography>
              {showAllNetworks ? (
                <S.Dropdown ref={dropdownNode}>
                  {!!allNetworks.length && showAllNetworks && allNetworks.map((network) => (
                    <Typography
                      key={network}
                      onClick={()=>dispatchSetNetwork(network)}
                      color={masterConfig === network ? 'secondary' : 'white'}
                    >
                      {network}
                    </Typography>))
                  }
                </S.Dropdown>
              ) : null}
            </Box>
          </S.NetWorkStyle>
        </S.Menu>
      </S.WallerPickerWrapper>
    </S.WalletPickerContainer>
  )
};

export default NetworkSwitcher;
