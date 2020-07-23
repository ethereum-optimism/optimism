# DB container & scripts for use in Rollup
Simple postgres image, docker-compose.yml and init scripts for the Rollup DB.

## Usage:
### Start container:
```
docker-compose -f docker-compose.postgres.yml up --build
``` 

### Start detached:
```
docker-compose -f docker-compose.postgres.yml up --build
```

### Kill when run detcahed:
```
docker ps
# grab CONTAINER ID
docker kill <container id here>
```

### Purge all data:
```
# Note this purges _all_ docker data
docker system prune -f
```

## Recommendations
Use PG Admin 4 to interact with the postgres server, downloadable [here](https://www.pgadmin.org/download/)
