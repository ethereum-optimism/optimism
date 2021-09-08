import { styled } from '@material-ui/core/styles';
import { Box } from "@material-ui/core";

export const Wrapper = styled(Box)(({ theme }) => ({
  display: "flex",
  justifyContent: 'space-between',
  alignItems: 'center',
  margin: '40px 0',
  [theme.breakpoints.down('md')]: {
    marginTop: 0,
  },
  [theme.breakpoints.up('md')]: {
  },
}));
