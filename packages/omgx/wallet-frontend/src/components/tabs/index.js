import React from 'react';
import {
    Tabs,
    Tab,
    Box,
} from '@material-ui/core';
import SearchIcon from 'components/icons/SearchIcon';

/**
 *
 * @param {
 *  optionList: ['string','string']
 * } param
 * @returns ReactComponent
 */

function StyledTabs({
    onChange,
    selectedTab,
    optionList = [],
    isSearch,
    onSearch,
}) {


    return (
        <Box
            sx={{
                display: 'flex',
                flex:1,
                justifyContent: 'flex-start',
                alignItems: 'center'
            }}
        >

            <Tabs value={selectedTab}
                indicatorColor="primary"
                textColor="primary"
                onChange={onChange}
                aria-label="transaction tabs"
            >
                {optionList.map((label) => (<Tab
                    key={label}
                    sx={{
                        maxWidth: 'unset',
                        minWidth: 'unset',
                        alignItems: 'flex-start',
                        margin: '0px 5px',
                        height: '24px',
                        fontWeight: 'normal',
                        fontSize: '24px',
                        lineHeight: '24px',
                        textTransform: 'capitalize'
                    }}
                    label={label} />))}
            </Tabs>
            {isSearch && <SearchIcon color="#F0A000" />}
        </Box>
    )

}

export default StyledTabs
