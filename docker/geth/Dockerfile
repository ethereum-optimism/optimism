FROM ethereum/client-go
RUN apk add --no-cache  openssl jq
COPY rollup-fullnode.json /etc/
COPY entrypoint.sh /bin
RUN chmod +x /bin/entrypoint.sh

EXPOSE 9545
ENTRYPOINT ["sh", "/bin/entrypoint.sh"]
