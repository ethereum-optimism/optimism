FROM node:11-slim

WORKDIR /server

COPY . /server
RUN yarn

EXPOSE 3000
CMD [ "npm", "start" ]
