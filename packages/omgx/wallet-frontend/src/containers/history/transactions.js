import StyledTable from 'components/table'
import React from 'react'

function Transactions({
    transactions,
    chainLink,
}) {

    return (
        <>
            <StyledTable
                chainLink={chainLink}
                isTransaction={true}
                tableData={transactions.slice(0, 200)} /// TODO: implement the scroll pagination.
            />
        </>)
}

export default Transactions
