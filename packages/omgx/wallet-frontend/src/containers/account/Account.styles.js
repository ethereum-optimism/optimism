import { styled } from '@material-ui/core/styles'
import { Box, Card, CardContent, Typography } from "@material-ui/core";

export const WrapperHeading = styled(Box)`
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 30px;
  justify-content: space-between;
`;

export const TableHeading = styled(Box)(({ theme }) => ({
  padding: "10px",
  borderRadius: "6px",
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  backgroundColor: theme.palette.background.secondary,

  [theme.breakpoints.down('md')]: {
    marginBottom: "5px",
  },
  [theme.breakpoints.up('md')]: {
    marginBottom: "20px",
  },
}));

export const TableHeadingItem = styled(Typography)`
  width: 20%;
  gap: 5px;
  text-align: center;
`;

export const AccountWrapper = styled(Box)(({ theme }) => ({
  [theme.breakpoints.down('md')]: {
    backgroundColor: "transparent",
  },
  [theme.breakpoints.up('md')]: {
    backgroundColor: theme.palette.background.secondary,
    borderRadius: "10px",
    padding: "20px",
  },
}));

export const CardTag = styled(Card)(({ theme }) => ({
  display: 'flex',
  padding: '10px',
  border: '2px solid rgba(255, 255, 255, 0.2)',
  overflow: 'initial',
  maxHeight: '190px',
  [theme.breakpoints.up('md')]: {
    margin: '60px 0 30px 0'
  },
}));

export const CardContentTag = styled(CardContent)(({ theme }) => ({
  clipPath: 'polygon(0 0, 93% 0, 100% 100%, 0% 100%)',
  backgroundColor: theme.palette.background.secondary,
  borderRadius: '6px',
  flex: 12,
  [theme.breakpoints.down('md')]: {
    // backgroundColor: "transparent",
  },
}));

export const BalanceValue = styled(Typography)(({ theme }) => ({
  color: theme.palette.secondary.main,
  fontSize: '50px !important',
  fontWeight: 700
}));

export const CardInfo = styled(Typography)`
  opacity: 0.7;
  font-size: 20px !important;
`;

export const ContentGlass = styled(Box)(({ theme }) => ({
  transform: 'rotateZ(350deg)',
  position: 'relative',
  top: '-87px',
  left: '-12px',
  flex: 3,
  [theme.breakpoints.up('md')]: {
  top: '-83px',
  left: '-20px',
  },
  [theme.breakpoints.up('lg')]: {
  top: '-85px',
  left: '-13px',
  },
}));
