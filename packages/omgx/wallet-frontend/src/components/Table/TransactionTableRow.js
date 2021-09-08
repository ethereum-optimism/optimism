import React, { useState } from 'react';
import {
    Box, Button, Collapse,
    Grid, Typography
} from '@material-ui/core';

import DownIcon from 'components/icons/DownIcon';
import L2ToL1Icon from 'components/icons/L2ToL1Icon';
import LinkIcon from 'components/icons/LinkIcon';
import UpIcon from 'components/icons/UpIcon';

import moment from 'moment';
import truncate from 'truncate-middle';

import {
    CellSubTitle, CellTitle,
    StyledTableCell,
    StyledTableRow
} from './table.styles';


function TransactionTableRow({ chainLink, index, ...data }) {

    const [expandRow, setExpandRow] = useState(false);

    return <React.Fragment
        key={index}
    >
        <StyledTableRow
            className={!!expandRow ? 'expand' : ''}
        >
            <StyledTableCell>
                <Grid
                    container
                    direction='row'
                    justify='space-between'
                    alignItems='center'
                >
                    <L2ToL1Icon />
                    <Box
                        sx={{
                            marginLeft: '30px',
                            display: 'flex',
                            flexDirection: 'column',
                        }}
                    >
                        <CellTitle> L2 - L1 Exit </CellTitle>
                        <CellSubTitle> Fast Offramp </CellSubTitle>
                    </Box>
                </Grid>
            </StyledTableCell>
            <StyledTableCell>
                <Grid
                    container
                    direction='row'
                    justify='space-between'
                    alignItems='center'
                >
                    <Box
                        sx={{
                            display: 'flex',
                            flexDirection: 'column',
                        }}
                    >
                        <CellTitle color="#06D3D3"> Swapped </CellTitle>
                        <CellSubTitle> {moment.unix(data.timeStamp).format('lll')} </CellSubTitle>
                    </Box>
                </Grid>
            </StyledTableCell>
            <StyledTableCell>
                <Grid
                    container
                    direction='row'
                    justify='space-between'
                    alignItems='center'
                >
                    <Box
                        sx={{
                            display: 'flex',
                            flexDirection: 'column',
                        }}
                    >
                        <CellTitle> {truncate(data.hash, 8, 6, '...')} </CellTitle>
                        <CellSubTitle> Block {data.blockNumber} </CellSubTitle>
                    </Box>
                </Grid>
            </StyledTableCell>
            <StyledTableCell>
                <Grid
                    container
                    direction='row'
                    justify='space-between'
                    alignItems='center'
                >
                    <a
                        href={chainLink(data)}
                        target={'_blank'}
                        rel='noopener noreferrer'
                    ><Button
                        startIcon={<LinkIcon />}
                        variant="outlined"
                        color="primary">
                            Advanced Details
                        </Button></a>
                </Grid>
            </StyledTableCell>
            <StyledTableCell>
                <Grid
                    sx={{
                        cursor: 'pointer'
                    }}
                    container
                    direction='row'
                    justify='space-between'
                    alignItems='center'
                    onClick={() => setExpandRow(!expandRow)}
                >

                    {data.l1Hash ?
                        !!expandRow ?
                            <UpIcon /> : <DownIcon /> : null
                    }
                </Grid>
            </StyledTableCell>
        </StyledTableRow>
        {
            data.l1Hash ?
                <StyledTableRow
                    className={!!expandRow ? "detail" : 'hidden'}
                >
                    <StyledTableCell
                        colSpan="5"
                    >
                        <Collapse
                            in={expandRow}
                        >

                            <Box
                                sx={{
                                    background: 'rgba(9, 22, 43, 0.5)',
                                    borderRadius: '12px',
                                    padding: '30px',
                                }}
                            >

                                <Grid container>
                                    <Grid item xs={12}>
                                        <Typography variant="body1">
                                            {truncate(data.l1Hash, 8, 6, '...')}
                                        </Typography>
                                    </Grid>
                                </Grid>
                                <Grid container>
                                    <Grid item xs={2}>
                                        <Typography variant="body1">
                                            L1 Block
                                        </Typography>
                                    </Grid>
                                    <Grid item xs={10}>
                                        <Typography variant="body1" className="value" >
                                            {data.l1BlockNumber}
                                        </Typography>
                                    </Grid>
                                </Grid>
                                <Grid container>
                                    <Grid item xs={2}>
                                        <Typography variant="body1">
                                            Block Has
                                        </Typography>
                                    </Grid>
                                    <Grid item xs={10}>
                                        <Typography variant="body1" className="value" >
                                            {data.l1BlockHash}
                                        </Typography>
                                    </Grid>
                                </Grid>
                                <Grid container>
                                    <Grid item xs={2}>
                                        <Typography variant="body1">
                                            L1 From
                                        </Typography>
                                    </Grid>
                                    <Grid item xs={10}>
                                        <Typography variant="body1" className="value" >
                                            {data.l1From}
                                        </Typography>
                                    </Grid>
                                </Grid>
                                <Grid container>
                                    <Grid item xs={2}>
                                        <Typography variant="body1">
                                            L1 To
                                        </Typography>
                                    </Grid>
                                    <Grid item xs={10}>
                                        <Typography variant="body1" className="value" >
                                            {data.l1To}
                                        </Typography>
                                    </Grid>
                                </Grid>
                            </Box>
                        </Collapse>
                    </StyledTableCell>
                </StyledTableRow> : null}
    </React.Fragment>

}

export default TransactionTableRow;


/*
blockHash: "0xacae9c9e92199ece44055bca7340c6df267b5aaa51c0b88eaa14b163667ef24c"
blockNumber: "9128892"
chain: "L1"
confirmations: "1186"
contractAddress: ""
cumulativeGasUsed: "140780"
from: "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
gas: "21000"
gasPrice: "5147441866"
gasUsed: "21000"
hash: "0x45bea808dac6618405d5161e7ea0d3611ac8f5ee535355059c6e46a259174276"
input: "0x"
isError: "0"
nonce: "2211"
timeStamp: "1629174050"
to: "0x52270d8234b864dcac9947f510ce9275a8a116db"
transactionIndex: "2"
txreceipt_status: "1"
typeTX: "0x52270d8234b864dcac9947f510ce9275a8a116db"
 */
