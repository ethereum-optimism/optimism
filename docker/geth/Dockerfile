FROM ethereum/client-go
COPY rollup-fullnode.json boot.key private_key.hex /etc/
COPY entrypoint.sh /bin
RUN chmod +x /bin/entrypoint.sh

EXPOSE 8545
ENTRYPOINT ["sh", "/bin/entrypoint.sh"]
# ENTRYPOINT []
