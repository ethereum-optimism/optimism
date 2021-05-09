import React from 'react';
import Card from '@material-ui/core/Card';
import CardActionArea from '@material-ui/core/CardActionArea';
import CardActions from '@material-ui/core/CardActions';
import CardContent from '@material-ui/core/CardContent';
import CardHeader from '@material-ui/core/CardHeader';
import CardMedia from '@material-ui/core/CardMedia';
import Button from '@material-ui/core/Button';
import Typography from '@material-ui/core/Typography';

import * as styles from './NftCard.module.scss';

import cells from 'images/hela.jpg';

function NFTCard({ UUID, owner, URL, time, name, symbol }) {

  return (
    <Card className={styles.NFTroot}>
      <CardActionArea>
        <CardMedia
          className={styles.NFTmedia}
          image={cells}
          title="Cell line"
        />
        <CardContent>
          <Typography gutterBottom variant="h4" component="h2">
            {name}
          </Typography>
          <Typography gutterBottom variant="h5" component="h3">
            {symbol}
          </Typography>
          <Typography variant="body2" color="textSecondary" component="p">
            <span style={{fontWeight: '600'}}>NFT ID: </span>{UUID}<br/>
            <span style={{fontWeight: '600'}}>Owner: </span>{owner}<br/>
            <span style={{fontWeight: '600'}}>Time Minted: </span>{time}<br/>
          </Typography>
          <Typography variant="body2" color="textSecondary" component="p">
            <a style={{color: 'blue'}} 
              href={URL}
            >
              LINK to datasheet
            </a>
          </Typography>
        </CardContent>
      </CardActionArea>
      <CardActions>
        <Button size="small" color="primary">
          Transfer
        </Button>
        <Button size="small" color="primary">
          Delete
        </Button>
      </CardActions>
    </Card>
  );
}

export default NFTCard;