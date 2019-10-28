# immutability-eth-plugin docker

Spins up a stateless vault server with immutability-eth-plugin for testing purposes.  Once stopped all state is lost. Stdout goes to console window where the docker-compose up command was run. Ctrl-c kills the server.

Not for production use.

### Run and test the docker image to ensure the mounts are there

```bash
docker-compose -f docker-compose.yml up
```

In another window, ensure the plugins were installed:
```bash
curl -k --header "X-Vault-Token: totally-secure" https://127.0.0.1:8200/v1/sys/mounts
``` 
### Certs
    
If you want to change the certs take a look under config/gencerts.sh