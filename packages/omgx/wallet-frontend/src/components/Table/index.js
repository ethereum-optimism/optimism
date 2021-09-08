import {
    Grid,
    Table,
    TableBody,
    TableContainer,
    TableHead
} from '@material-ui/core';
import SortIcon from 'components/icons/SortIcon';
import React from 'react';
import {
    StyledTableCell,
    StyledTableRow
} from './table.styles';
import TransactionTableRow from './TransactionTableRow';


function StyledTable({
    tableHeadList,
    isTransaction,
    tableData,
    chainLink,
}) {

    return (
        <TableContainer
            sx={{
                marginTop: '30px',
                textAlign: 'left',
                width: '100%',
                background: 'linear-gradient(132.17deg, rgba(255, 255, 255, 0.019985) 0.24%, rgba(255, 255, 255, 0.03) 94.26%)',
                borderRadius: '8px',
                padding: '0px 0px 20px',
                height: 'calc(100vh - 250px)'
            }}
        >
            <Table stickyHeader>
                <TableHead
                    sx={{
                        padding: '0px 55px',
                        background: 'linear-gradient(132.17deg, rgba(255, 255, 255, 0.019985) 0.24%, rgba(255, 255, 255, 0.03) 94.26%)',
                    }}
                >
                    <StyledTableRow
                        className="header"
                    >
                        {tableHeadList && tableHeadList.length > 0 ?
                            tableHeadList.map((head) => {
                                return (
                                    <StyledTableCell
                                        key={head.label}
                                        color="rgba(255, 255, 255, 0.7)">
                                        <Grid
                                            container
                                            direction='row'
                                            justify='space-between'
                                            alignItems='center'
                                        >
                                            <span>{head.label}</span>
                                            {head.isSort ? <SortIcon /> : null}
                                        </Grid>
                                    </StyledTableCell>)
                            })
                            : null
                        }
                    </StyledTableRow>
                </TableHead>
                <TableBody>
                    {tableData && tableData.length > 0 ?
                        tableData.map((item,index) => {
                            if (isTransaction) {
                                return <TransactionTableRow
                                    index={index}
                                    chainLink={chainLink}
                                    {...item} />
                            }
                        })
                        : null}
                </TableBody>

            </Table>
        </TableContainer>
    )
}

export default StyledTable;
