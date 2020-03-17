FROM node:11

WORKDIR /ovm

COPY . /ovm
COPY ./tsconfig.docker.json /ovm/tsconfig.json
RUN yarn

EXPOSE 8545
ENTRYPOINT ["sh", "/ovm/bin/entrypoint.sh"]
