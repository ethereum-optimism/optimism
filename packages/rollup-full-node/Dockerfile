FROM node:11

WORKDIR /server

COPY . /server
RUN yarn

EXPOSE 8545
CMD [ "yarn", "run", "server:fullnode" ]
