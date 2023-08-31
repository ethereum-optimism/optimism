FROM node:18

WORKDIR /app

COPY ./ ./

RUN npm i -g pnpm
RUN pnpm install
RUN pnpm build

EXPOSE 2000

CMD [ "./start.sh" ]
