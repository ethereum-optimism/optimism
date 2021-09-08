/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */


import PageHeader from 'components/pageHeader/PageHeader';
import { PageContent } from 'pages/page.style';
import React from 'react';
import {
  Box, Paper, Typography
} from '@material-ui/core';
import { styled } from '@material-ui/core/styles';

const CustomPaper = styled(Paper)(({ theme }) => ({
  padding: theme.spacing(2),
  [theme.breakpoints.up('md')]: {
    padding: theme.spacing(10),
  },
}));

function LearnPage() {

  return (
    <PageContent>
      <PageHeader title="Learn" />
      <CustomPaper>
        <Typography variant="h1">What is Layer 2?</Typography>

        <Box sx={{ my: 3 }}>
          <Typography variant="h2" gutterBottom>
            Why should you use it
          </Typography>
          <Typography
            variant="body1"
          >
            Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit, sed quia non numquam eius modi tempora incidunt ut labore et dolore magnam aliquam quaerat voluptatem. Ut enim ad minima veniam, quis nostrum exercitationem ullam corporis suscipit laboriosam, nisi ut aliquid ex ea commodi consequatur? Quis autem vel eum iure reprehenderit qui in ea voluptate velit esse quam nihil molestiae consequatur, vel illum qui dolorem eum fugiat quo voluptas nulla pariatur?"
          </Typography>
        </Box>

        <Box sx={{ my: 3 }}>
          <Typography variant="h2" gutterBottom>
            How to use it
          </Typography>

          <Box sx={{ mb: 3 }}>
            <Typography variant="h3" gutterBottom>
              Step 1
            </Typography>
            <Typography variant="body1">
              Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi"
            </Typography>
          </Box>

          <Box sx={{ mb: 3 }}>
            <Typography variant="h3" gutterBottom>
              Step 2
            </Typography>
            <Typography variant="body1">
              Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi"
            </Typography>
          </Box>

          <Typography variant="h3" gutterBottom>
            Step 3
          </Typography>
          <Typography variant="body1">
            Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit
          </Typography>
        </Box>
      </CustomPaper>
    </PageContent>
  );

}

export default React.memo(LearnPage);
