import styled from '@emotion/styled';
import { TableCell, TableRow } from '@material-ui/core';

export const CellTitle = styled.div`
font-weight: normal;
font-size: 24px;
line-height: 24px;
color: ${props => props.color ? props.color : '#fff'};
opacity: 0.9;
margin-bottom: 7px;
`
export const CellSubTitle = styled.div`
font-weight: normal;
font-size: 18px;
line-height: 112%;
color: rgba(255, 255, 255, 0.7);
opacity: 0.9;
`

export const StyledTableRow = styled(TableRow)`
    padding: 0px 30px;
    &.expand{
        background: rgba(255, 255, 255, 0.03);
        td:first-of-type { border-top-left-radius: 16px; }
        td:last-of-type { border-top-right-radius: 16px; }
        td {
            box-shadow: none;
        }
    }
    &:nth-last-of-type(2) {
        td {
            box-shadow: none;
        }
    }
    &.header{

        th {
            box-shadow: none;
        }
    }
    &.divider {
        height: 0px;
        td {
            padding: 0px;
        }
    }
    &.hidden {
        height: 0px;
        td {
            padding: 0px;
        }
        .value {
            marginLeft: 10px;
            color: rgba(255, 255, 255, 0.7);
        }
    }
    &.detail {
        background: rgba(255, 255, 255, 0.03);

        td:first-of-type { border-bottom-left-radius: 16px; }
        td:last-of-type { border-bottom-right-radius: 16px; }

        .value {
            marginLeft: 10px;
            color: rgba(255, 255, 255, 0.7);
        }
    }
`
export const StyledTableCell = styled(TableCell)`
    border: none;
    padding-top: 20px;
    color: ${props => props.color ? props.color : '#fff'};
    box-shadow: 0px 1px 0px rgb(255 255 255 / 5%)
`
