import { styled } from '@material-ui/core/styles'
import { Box, Typography } from "@material-ui/core";

export const TableHeading = styled(Box)(({ theme }) => ({
  padding: "20px",
  borderTopLeftRadius: "6px",
  borderTopRightRadius: "6px",
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  background: theme.palette.background.secondary,

  [theme.breakpoints.down('md')]: {
    marginBottom: "5px",
  },
}));

export const TableHeadingItem = styled(Typography)`
  width: 20%;
  gap: 5px;
  text-align: center;
  opacity: 0.7;
`;

export const LayerAlert = styled(Box)`
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 30px;
  padding: 27px 50px;
  border-radius: 8px;
  margin: 20px 0px;
  background: ${props => props.theme.palette.background.secondary};

  .info {
    display: flex;
    justify-content: space-around;
    align-items: center;
  }
`
