import WalletPicker from 'components/walletpicker/WalletPicker';
import React from 'react';
import {Route} from 'react-router-dom'


function Layout({
    enabled, 
    onEnable,
    ...routeProps
}) {
    if(enabled) {
        return <Route {...routeProps} key={routeProps.path} />
    }

    return (<WalletPicker enabled={enabled} onEnable={onEnable} />)
}


export default Layout;