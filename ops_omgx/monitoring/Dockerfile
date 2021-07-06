FROM node:12-alpine

# Create app directory
WORKDIR /usr/src/app

# Install app dependencies
# Wildcard is used to ensure both package and package lock JSON are copied
COPY package*.json ./

RUN npm ci

COPY . .
COPY ./dummy-transaction/entrypoint.sh /usr/local/bin/

RUN chmod 755 /usr/src/app
RUN chmod 755 /usr/local/bin/entrypoint.sh

CMD [ "entrypoint.sh" ]
