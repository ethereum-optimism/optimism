FROM ethereum/client-go:latest

# Copies genesis block configurations
COPY ./docker/ethereum/genesis.json .

COPY ./scripts/ethereum.sh .

#Add permissions to run sh script
RUN chmod +x run.sh

EXPOSE 8545

# ENTRYPOINT ["./run.sh"]
# ENTRYPOINT ["/bin/sh", "-c"]
