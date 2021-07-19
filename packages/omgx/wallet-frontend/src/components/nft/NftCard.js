import React from 'react';
import Card from '@material-ui/core/Card';
import CardActionArea from '@material-ui/core/CardActionArea';
import CardActions from '@material-ui/core/CardActions';
import CardContent from '@material-ui/core/CardContent';
import CardMedia from '@material-ui/core/CardMedia';
import Button from '@material-ui/core/Button';
import Typography from '@material-ui/core/Typography';

import * as styles from './NftCard.module.scss';

import cells from 'images/hela.jpg';

function NFTCard({ UUID, owner, URL, time, name, symbol }) {

  return (
    <Card 
      className={styles.NFTroot}
      style={{background: '#F8F8F8'}}
    >
      <CardActionArea>
        <CardMedia
          className={styles.NFTmedia}
          image={cells}
          title="Cell line"
        />
        <CardContent 
          style={{padding: 0, margin: 7}}
        >
          <Typography variant="h5">
            {name}
          </Typography>
          <Typography variant="h6">
            {symbol}
          </Typography>
          <Typography variant="body2" color="textSecondary" component="p" style={{fontSize: '0.9em',marginBottom: '15px'}}>
            <span style={{fontWeight: '600'}}>NFT ID: </span>{UUID}<br/>
            <span style={{fontWeight: '600'}}>Owner: </span>{owner}<br/>
            <span style={{fontWeight: '600'}}>Time Minted:</span><br/>{time}<br/>
          </Typography>
          <Typography 
            variant="body2" 
            color="textSecondary" 
            component="p" 
          >
            <a style={{color: 'blue'}} 
              href={URL}
            >
              DATASHEET
            </a>
          </Typography>
        </CardContent>
      </CardActionArea>
      <CardActions style={{flexDirection: 'column', justifyContent: 'flex-begin'}}>
        <Button size="small" color="primary">
          Transfer Ownership
        </Button>
        <Button size="small" color="primary">
          Delete
        </Button>
        <Button size="small" color="primary">
          License derived work
        </Button>
      </CardActions>
    </Card>
  );
}

export default NFTCard;